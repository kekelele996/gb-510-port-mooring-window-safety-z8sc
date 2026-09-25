package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/blueship581/port-mooring-window-safety/backend/internal/config"
	"github.com/blueship581/port-mooring-window-safety/backend/internal/dto"
	"github.com/blueship581/port-mooring-window-safety/backend/internal/model"
	"github.com/blueship581/port-mooring-window-safety/backend/internal/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupLineInspectionFixture(t *testing.T) (LineInspectionService, SafetyClearanceService, SecurityService, repository.SafetyClearanceRepository) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&model.SafetyClearance{}, &model.LineInspection{}, &model.AuditLog{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	clearanceRepository := repository.NewSafetyClearanceRepository(db)
	inspectionRepository := repository.NewLineInspectionRepository(db)
	security := NewSecurityService(repository.NewSecurityRepository(db), config.Config{})
	inspections := NewLineInspectionService(inspectionRepository, clearanceRepository, security)
	clearances := NewSafetyClearanceService(clearanceRepository, inspectionRepository, security)
	return inspections, clearances, security, clearanceRepository
}

func createClearanceForPlan(t *testing.T, repo repository.SafetyClearanceRepository, code, planCode string) model.SafetyClearance {
	t.Helper()
	item := model.SafetyClearance{
		BaseModel: model.BaseModel{Code: code, Name: "Clearance " + code, Status: model.SafetyClearanceInitialStatus, Version: 1},
		Facility:  "Berth A", Owner: "operations", Category: "test", RiskLevel: "medium",
		EffectiveAt: time.Now().UTC(), Evidence: "checked", RelatedCode: "WW-TEST", PlanCode: planCode, WindowVersion: 1,
	}
	if err := repo.Create(context.Background(), &item); err != nil {
		t.Fatalf("create clearance: %v", err)
	}
	return item
}

func blockingInspectionInput(code, planCode string) dto.CreateLineInspection {
	return dto.CreateLineInspection{
		Code: code, Name: "Inspection " + code, PlanCode: planCode, LinePosition: "船首左舷1#缆",
		InspectedAt: time.Now().UTC(), Inspector: "operator",
		DefectLevel: "broken_strand", Conclusion: "replace_pending",
	}
}

func TestBlockingInspectionReturnsPendingClearanceAndBlocksRelease(t *testing.T) {
	inspections, clearances, security, clearanceRepository := setupLineInspectionFixture(t)
	ctx := context.Background()
	clearance := createClearanceForPlan(t, clearanceRepository, "SC-BLOCK", "MP-BLOCK")

	submitted, err := clearances.Transition(ctx, clearance.ID, dto.TransitionRequest{
		Status: "cleared", ExpectedVersion: 1, Reason: "operator safety submission", WindowVersion: 3,
	}, "operator", model.RoleOperator, "request-submit")
	if err != nil {
		t.Fatalf("submit clearance: %v", err)
	}
	if submitted.SubmittedBy != "operator" {
		t.Fatalf("expected submission to be recorded: %+v", submitted)
	}

	inspection, err := inspections.Create(ctx, blockingInspectionInput("LI-BLOCK", "MP-BLOCK"), "operator", "request-inspect")
	if err != nil {
		t.Fatalf("create blocking inspection: %v", err)
	}
	if inspection.Status != "blocking" || inspection.BlockedReason == "" {
		t.Fatalf("expected blocking inspection with reason: %+v", inspection)
	}

	returned, err := clearanceRepository.Get(ctx, clearance.ID)
	if err != nil {
		t.Fatalf("reload clearance: %v", err)
	}
	if returned.SubmittedBy != "" || returned.SubmittedAt != nil {
		t.Fatalf("expected pending clearance to be returned to submitter: %+v", returned)
	}
	if !strings.Contains(returned.ReturnedReason, "LI-BLOCK") {
		t.Fatalf("expected returned reason to reference the inspection: %+v", returned)
	}

	_, err = clearances.Transition(ctx, returned.ID, dto.TransitionRequest{
		Status: "cleared", ExpectedVersion: returned.Version, Reason: "resubmit while blocked", WindowVersion: 3,
	}, "operator", model.RoleOperator, "request-resubmit")
	if !errors.Is(err, ErrClearanceBlocked) {
		t.Fatalf("expected clearance blocked error, got %v", err)
	}

	resolved, err := inspections.Transition(ctx, inspection.ID, dto.TransitionRequest{
		Status: "resolved", ExpectedVersion: inspection.Version, Reason: "现场换绳完成，复查通过",
	}, "operator", "request-resolve")
	if err != nil {
		t.Fatalf("resolve inspection: %v", err)
	}
	if resolved.Status != "resolved" || resolved.Conclusion != "recheck_pass" || resolved.ResolvedBy != "operator" || resolved.ResolvedAt == nil {
		t.Fatalf("expected recheck pass resolution: %+v", resolved)
	}

	resubmitted, err := clearances.Transition(ctx, returned.ID, dto.TransitionRequest{
		Status: "cleared", ExpectedVersion: returned.Version, Reason: "resubmit after rope replacement", WindowVersion: 3,
	}, "operator", model.RoleOperator, "request-resubmit-2")
	if err != nil {
		t.Fatalf("resubmit clearance after recheck pass: %v", err)
	}
	if resubmitted.SubmittedBy != "operator" || resubmitted.ReturnedReason != "" {
		t.Fatalf("expected clean resubmission: %+v", resubmitted)
	}

	final, err := clearances.Transition(ctx, resubmitted.ID, dto.TransitionRequest{
		Status: "cleared", ExpectedVersion: resubmitted.Version, Reason: "independent safety review", WindowVersion: 3,
	}, "reviewer", model.RoleReviewer, "request-review")
	if err != nil {
		t.Fatalf("confirm clearance after recheck pass: %v", err)
	}
	if final.Status != "cleared" || final.ConfirmedBy != "reviewer" {
		t.Fatalf("unexpected final state: %+v", final)
	}

	logs, total, err := security.ListAudits(ctx, 1, 50, "")
	if err != nil {
		t.Fatalf("list audits: %v", err)
	}
	if total < 3 || len(logs) < 3 {
		t.Fatalf("expected submission, return and resolution audits, total=%d", total)
	}
	foundReturn := false
	for _, audit := range logs {
		if audit.Action == "clearance_return" && audit.EntityType == "SafetyClearance" && audit.EntityID == clearance.ID {
			foundReturn = true
		}
	}
	if !foundReturn {
		t.Fatalf("expected clearance_return audit entry: %+v", logs)
	}
}

func TestPassingInspectionDoesNotBlockClearance(t *testing.T) {
	inspections, clearances, _, clearanceRepository := setupLineInspectionFixture(t)
	ctx := context.Background()
	clearance := createClearanceForPlan(t, clearanceRepository, "SC-PASS", "MP-PASS")

	input := blockingInspectionInput("LI-PASS", "MP-PASS")
	input.DefectLevel = "minor_wear"
	input.Conclusion = "monitor"
	inspection, err := inspections.Create(ctx, input, "operator", "request-inspect")
	if err != nil {
		t.Fatalf("create passing inspection: %v", err)
	}
	if inspection.Status != "open" || inspection.BlockedReason != "" {
		t.Fatalf("expected open inspection without blocked reason: %+v", inspection)
	}

	submitted, err := clearances.Transition(ctx, clearance.ID, dto.TransitionRequest{
		Status: "cleared", ExpectedVersion: 1, Reason: "operator safety submission", WindowVersion: 5,
	}, "operator", model.RoleOperator, "request-submit")
	if err != nil {
		t.Fatalf("submit clearance with passing inspection: %v", err)
	}
	if submitted.SubmittedBy != "operator" {
		t.Fatalf("expected submission to proceed: %+v", submitted)
	}
}

func TestNewInspectionSupersedesPreviousBlockingRecord(t *testing.T) {
	inspections, clearances, _, clearanceRepository := setupLineInspectionFixture(t)
	ctx := context.Background()
	clearance := createClearanceForPlan(t, clearanceRepository, "SC-SUPERSEDE", "MP-SUPERSEDE")

	blocking, err := inspections.Create(ctx, blockingInspectionInput("LI-OLD", "MP-SUPERSEDE"), "operator", "request-inspect-1")
	if err != nil {
		t.Fatalf("create blocking inspection: %v", err)
	}
	if _, err = clearances.Transition(ctx, clearance.ID, dto.TransitionRequest{
		Status: "cleared", ExpectedVersion: 1, Reason: "submit while blocked", WindowVersion: 2,
	}, "operator", model.RoleOperator, "request-submit"); !errors.Is(err, ErrClearanceBlocked) {
		t.Fatalf("expected clearance blocked error, got %v", err)
	}

	recheck := blockingInspectionInput("LI-NEW", "MP-SUPERSEDE")
	recheck.DefectLevel = "none"
	recheck.Conclusion = "recheck_pass"
	followUp, err := inspections.Create(ctx, recheck, "operator", "request-inspect-2")
	if err != nil {
		t.Fatalf("create recheck inspection: %v", err)
	}
	if followUp.Status != "open" {
		t.Fatalf("expected recheck pass inspection to stay open: %+v", followUp)
	}

	previous, err := inspections.Get(ctx, blocking.ID)
	if err != nil {
		t.Fatalf("reload previous inspection: %v", err)
	}
	if previous.Status != "superseded" {
		t.Fatalf("expected previous inspection to be superseded: %+v", previous)
	}

	if _, err = clearances.Transition(ctx, clearance.ID, dto.TransitionRequest{
		Status: "cleared", ExpectedVersion: 1, Reason: "resubmit after superseding recheck", WindowVersion: 2,
	}, "operator", model.RoleOperator, "request-submit-2"); err != nil {
		t.Fatalf("expected clearance to be submittable after recheck pass: %v", err)
	}
}

func TestLineInspectionRejectsInvalidTransition(t *testing.T) {
	inspections, _, _, _ := setupLineInspectionFixture(t)
	ctx := context.Background()
	input := blockingInspectionInput("LI-GRAPH", "MP-GRAPH")
	input.DefectLevel = "none"
	input.Conclusion = "pass"
	inspection, err := inspections.Create(ctx, input, "operator", "request-inspect")
	if err != nil {
		t.Fatalf("create inspection: %v", err)
	}
	if _, err = inspections.Transition(ctx, inspection.ID, dto.TransitionRequest{
		Status: "deleted", ExpectedVersion: inspection.Version, Reason: "invalid target",
	}, "operator", "request-invalid"); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected invalid transition error, got %v", err)
	}
}
