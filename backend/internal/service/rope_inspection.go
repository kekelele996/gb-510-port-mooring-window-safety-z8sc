package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/blueship581/port-mooring-window-safety/backend/internal/constants"
	"github.com/blueship581/port-mooring-window-safety/backend/internal/dto"
	"github.com/blueship581/port-mooring-window-safety/backend/internal/model"
	"github.com/blueship581/port-mooring-window-safety/backend/internal/repository"
)

type RopeInspectionService interface {
	List(context.Context, dto.PageQuery) (repository.Page[model.RopeInspection], error)
	Current(context.Context, string) ([]model.RopeInspection, error)
	Create(context.Context, dto.CreateRopeInspection, string, string) (model.RopeInspection, error)
	StatusCounts(context.Context) (map[string]int64, error)
}

type ropeInspectionService struct {
	repository  repository.RopeInspectionRepository
	mooringPlan repository.MooringPlanRepository
	clearance   repository.SafetyClearanceRepository
	security    SecurityService
}

func NewRopeInspectionService(repo repository.RopeInspectionRepository, mooringPlan repository.MooringPlanRepository, clearance repository.SafetyClearanceRepository, security SecurityService) RopeInspectionService {
	return &ropeInspectionService{repository: repo, mooringPlan: mooringPlan, clearance: clearance, security: security}
}

func (s *ropeInspectionService) List(ctx context.Context, query dto.PageQuery) (repository.Page[model.RopeInspection], error) {
	return s.repository.List(ctx, query)
}

func (s *ropeInspectionService) Current(ctx context.Context, planCode string) ([]model.RopeInspection, error) {
	return s.repository.LatestPerLine(ctx, planCode)
}

func (s *ropeInspectionService) Create(ctx context.Context, input dto.CreateRopeInspection, actor, requestID string) (model.RopeInspection, error) {
	planCode := strings.ToUpper(strings.TrimSpace(input.PlanCode))
	linePosition := strings.TrimSpace(input.LinePosition)
	inspector := strings.TrimSpace(input.Inspector)
	if linePosition == "" || inspector == "" {
		return model.RopeInspection{}, ErrInvalidInput
	}
	if _, err := s.mooringPlan.GetByCode(ctx, planCode); err != nil {
		return model.RopeInspection{}, fmt.Errorf("%w: mooring plan %s does not exist", ErrInvalidInput, planCode)
	}
	blocking := constants.IsBlockingRopeInspection(input.DefectLevel, input.Conclusion)
	status := string(constants.RopeInspectionStateClosed)
	blockedReason := ""
	if blocking {
		status = string(constants.RopeInspectionStateOpen)
		blockedReason = buildRopeBlockedReason(linePosition, input.DefectLevel, input.Conclusion)
	}
	item := model.RopeInspection{
		BaseModel: model.BaseModel{
			Code: strings.ToUpper(strings.TrimSpace(input.Code)),
			Name: fmt.Sprintf("%s %s 缆绳检查", planCode, linePosition),
			Status: status, Version: 1, Description: strings.TrimSpace(input.Notes),
		},
		PlanCode: planCode, LinePosition: linePosition,
		InspectedAt: input.InspectedAt.UTC(), Inspector: inspector,
		DefectLevel: input.DefectLevel, Conclusion: input.Conclusion,
		BlockedReason: blockedReason,
	}
	if err := s.repository.Create(ctx, &item); err != nil {
		return model.RopeInspection{}, fmt.Errorf("create 缆绳检查: %w", err)
	}
	_ = s.security.Audit(ctx, actor, requestID, "create", "RopeInspection", item.ID, "", item.Status, "created 缆绳检查")
	if blocking {
		if err := s.returnPendingClearances(ctx, &item, actor, requestID); err != nil {
			return model.RopeInspection{}, err
		}
	} else {
		s.resolveLine(ctx, &item, actor, requestID)
	}
	return item, nil
}

// returnPendingClearances 退回该方案下所有待复核许可，并记录对应检查与阻断原因。
func (s *ropeInspectionService) returnPendingClearances(ctx context.Context, inspection *model.RopeInspection, actor, requestID string) error {
	clearances, err := s.clearance.ListSubmittedPendingByPlan(ctx, inspection.PlanCode)
	if err != nil {
		return fmt.Errorf("list pending clearances for 缆绳检查: %w", err)
	}
	now := time.Now().UTC()
	for _, clearance := range clearances {
		expectedVersion := clearance.Version
		clearance.SubmittedBy = ""
		clearance.SubmittedAt = nil
		clearance.BlockedReason = fmt.Sprintf("检查 %s：%s", inspection.Code, inspection.BlockedReason)
		clearance.BlockedInspectionID = inspection.ID
		clearance.Version = expectedVersion + 1
		clearance.UpdatedAt = now
		if err := s.clearance.Update(ctx, clearance.ID, expectedVersion, &clearance); err != nil {
			return fmt.Errorf("return 安全许可 %d: %w", clearance.ID, err)
		}
		if err := s.security.AuditWithWindowVersion(ctx, actor, requestID, "clearance_return", "SafetyClearance", clearance.ID, "pending", "pending", clearance.BlockedReason, clearance.WindowVersion); err != nil {
			return fmt.Errorf("persist clearance return audit: %w", err)
		}
	}
	return nil
}

// resolveLine 换绳复查通过后闭环同一缆位所有未闭环的检查记录。
func (s *ropeInspectionService) resolveLine(ctx context.Context, inspection *model.RopeInspection, actor, requestID string) {
	resolved, err := s.repository.ResolveOpenByLine(ctx, inspection.PlanCode, inspection.LinePosition, time.Now().UTC(), actor)
	if err != nil {
		return
	}
	for _, item := range resolved {
		_ = s.security.Audit(ctx, actor, requestID, "rope_resolve", "RopeInspection", item.ID, "open", "closed",
			fmt.Sprintf("复查 %s 通过，闭环 %s", inspection.Code, item.Code))
	}
}

func (s *ropeInspectionService) StatusCounts(ctx context.Context) (map[string]int64, error) {
	return s.repository.CountByStatus(ctx)
}

func buildRopeBlockedReason(linePosition, defectLevel, conclusion string) string {
	defect := constants.RopeDefectLabels[defectLevel]
	if defect == "" {
		defect = defectLevel
	}
	label := constants.RopeConclusionLabels[conclusion]
	if label == "" {
		label = conclusion
	}
	return fmt.Sprintf("%s %s，处置结论：%s", linePosition, defect, label)
}
