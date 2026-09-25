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

func TestRopeInspectionReturnsAndBlocksClearance(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&model.MooringPlan{}, &model.SafetyClearance{}, &model.RopeInspection{}, &model.AuditLog{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	ctx := context.Background()

	planRepository := repository.NewMooringPlanRepository(db)
	clearanceRepository := repository.NewSafetyClearanceRepository(db)
	ropeRepository := repository.NewRopeInspectionRepository(db)
	security := NewSecurityService(repository.NewSecurityRepository(db), config.Config{})
	clearanceSvc := NewSafetyClearanceService(clearanceRepository, security, ropeRepository)
	ropeSvc := NewRopeInspectionService(ropeRepository, planRepository, clearanceRepository, security)

	plan := model.MooringPlan{
		BaseModel: model.BaseModel{Code: "MP-T", Name: "Test mooring plan", Status: "approved", Version: 1},
		Facility:  "Berth T", Owner: "operations", Category: "test", RiskLevel: "medium", EffectiveAt: time.Now().UTC(),
	}
	if err := planRepository.Create(ctx, &plan); err != nil {
		t.Fatalf("create plan: %v", err)
	}
	clearance := model.SafetyClearance{
		BaseModel: model.BaseModel{Code: "SC-T", Name: "Test clearance", Status: "pending", Version: 1},
		Facility:  "Berth T", Owner: "operations", Category: "test", RiskLevel: "medium",
		EffectiveAt: time.Now().UTC(), RelatedCode: "WW-T", WindowVersion: 3, PlanCode: "MP-T",
	}
	if err := clearanceRepository.Create(ctx, &clearance); err != nil {
		t.Fatalf("create clearance: %v", err)
	}

	submitted, err := clearanceSvc.Transition(ctx, clearance.ID, dto.TransitionRequest{
		Status: "cleared", ExpectedVersion: 1, Reason: "operator safety submission", WindowVersion: 3,
	}, "operator", model.RoleOperator, "request-submit")
	if err != nil || submitted.SubmittedBy != "operator" {
		t.Fatalf("operator submission failed: %+v err=%v", submitted, err)
	}

	inspection, err := ropeSvc.Create(ctx, dto.CreateRopeInspection{
		Code: "RI-T1", PlanCode: "MP-T", LinePosition: "艏缆-1", InspectedAt: time.Now().UTC(),
		Inspector: "operator", DefectLevel: "broken_strand", Conclusion: "replace_pending", Notes: "现场发现断股",
	}, "operator", "request-inspect")
	if err != nil {
		t.Fatalf("create blocking inspection: %v", err)
	}
	if inspection.Status != "open" || inspection.BlockedReason == "" {
		t.Fatalf("blocking inspection must stay open with a reason: %+v", inspection)
	}

	returned, err := clearanceRepository.Get(ctx, clearance.ID)
	if err != nil {
		t.Fatalf("reload clearance: %v", err)
	}
	if returned.SubmittedBy != "" || returned.Status != "pending" {
		t.Fatalf("pending clearance must be returned, got %+v", returned)
	}
	if returned.BlockedInspectionID != inspection.ID || !strings.Contains(returned.BlockedReason, "RI-T1") {
		t.Fatalf("returned clearance must reference the blocking inspection: %+v", returned)
	}

	_, err = clearanceSvc.Transition(ctx, clearance.ID, dto.TransitionRequest{
		Status: "cleared", ExpectedVersion: returned.Version, Reason: "resubmit while blocked", WindowVersion: 3,
	}, "operator", model.RoleOperator, "request-resubmit")
	if !errors.Is(err, ErrRopeInspectionBlocked) {
		t.Fatalf("expected rope inspection blocking error, got %v", err)
	}

	followUp, err := ropeSvc.Create(ctx, dto.CreateRopeInspection{
		Code: "RI-T2", PlanCode: "MP-T", LinePosition: "艏缆-1", InspectedAt: time.Now().UTC(),
		Inspector: "reviewer", DefectLevel: "none", Conclusion: "passed", Notes: "换绳后复查通过",
	}, "reviewer", "request-recheck")
	if err != nil {
		t.Fatalf("create passing re-inspection: %v", err)
	}
	if followUp.Status != "closed" {
		t.Fatalf("passing re-inspection must be closed, got %s", followUp.Status)
	}
	open, err := ropeRepository.ListOpenByPlan(ctx, "MP-T")
	if err != nil || len(open) != 0 {
		t.Fatalf("expected no open inspections after passing re-check, got %+v err=%v", open, err)
	}

	latest, err := clearanceRepository.Get(ctx, clearance.ID)
	if err != nil {
		t.Fatalf("reload clearance before resubmission: %v", err)
	}
	resubmitted, err := clearanceSvc.Transition(ctx, clearance.ID, dto.TransitionRequest{
		Status: "cleared", ExpectedVersion: latest.Version, Reason: "resubmit after rope replacement", WindowVersion: 3,
	}, "operator", model.RoleOperator, "request-resubmit-2")
	if err != nil {
		t.Fatalf("resubmission after passing re-inspection: %v", err)
	}
	if resubmitted.SubmittedBy != "operator" || resubmitted.BlockedReason != "" || resubmitted.BlockedInspectionID != 0 {
		t.Fatalf("resubmission must clear the blocking record: %+v", resubmitted)
	}

	current, err := ropeSvc.Current(ctx, "MP-T")
	if err != nil || len(current) != 1 || current[0].Code != "RI-T2" || current[0].Conclusion != "passed" {
		t.Fatalf("current conclusion must be the passing re-inspection: %+v err=%v", current, err)
	}
}

func TestRopeInspectionRejectsUnknownPlan(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&model.MooringPlan{}, &model.SafetyClearance{}, &model.RopeInspection{}, &model.AuditLog{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	security := NewSecurityService(repository.NewSecurityRepository(db), config.Config{})
	ropeSvc := NewRopeInspectionService(
		repository.NewRopeInspectionRepository(db),
		repository.NewMooringPlanRepository(db),
		repository.NewSafetyClearanceRepository(db),
		security,
	)
	_, err = ropeSvc.Create(context.Background(), dto.CreateRopeInspection{
		Code: "RI-X", PlanCode: "MP-MISSING", LinePosition: "艏缆-1", InspectedAt: time.Now().UTC(),
		Inspector: "operator", DefectLevel: "none", Conclusion: "passed",
	}, "operator", "request-unknown-plan")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input for unknown mooring plan, got %v", err)
	}
}
