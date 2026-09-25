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

type LineInspectionService interface {
	List(context.Context, dto.PageQuery) (repository.Page[model.LineInspection], error)
	Get(context.Context, uint) (model.LineInspection, error)
	Create(context.Context, dto.CreateLineInspection, string, string) (model.LineInspection, error)
	Update(context.Context, uint, dto.UpdateLineInspection, string, string) (model.LineInspection, error)
	Transition(context.Context, uint, dto.TransitionRequest, string, string) (model.LineInspection, error)
	Delete(context.Context, uint, string, string) error
	StatusCounts(context.Context) (map[string]int64, error)
}

type lineInspectionService struct {
	repository repository.LineInspectionRepository
	clearances repository.SafetyClearanceRepository
	security   SecurityService
}

func NewLineInspectionService(repo repository.LineInspectionRepository, clearances repository.SafetyClearanceRepository, security SecurityService) LineInspectionService {
	return &lineInspectionService{repository: repo, clearances: clearances, security: security}
}

func (s *lineInspectionService) List(ctx context.Context, query dto.PageQuery) (repository.Page[model.LineInspection], error) {
	return s.repository.List(ctx, query)
}

func (s *lineInspectionService) Get(ctx context.Context, id uint) (model.LineInspection, error) {
	return s.repository.Get(ctx, id)
}

func (s *lineInspectionService) Create(ctx context.Context, input dto.CreateLineInspection, actor, requestID string) (model.LineInspection, error) {
	if err := validateLineInspectionBusinessFields(input.Code, input.Name, input.PlanCode, input.LinePosition, input.Inspector); err != nil {
		return model.LineInspection{}, err
	}
	status := model.LineInspectionInitialStatus
	blockedReason := ""
	if constants.IsBlockingInspection(input.DefectLevel, input.Conclusion) {
		status = string(constants.InspectionStateBlocking)
		blockedReason = deriveBlockedReason(input.DefectLevel, input.Conclusion, "")
	}
	item := model.LineInspection{
		BaseModel: model.BaseModel{
			Code: strings.ToUpper(strings.TrimSpace(input.Code)), Name: strings.TrimSpace(input.Name),
			Status: status, Version: 1, Description: strings.TrimSpace(input.Description),
		},
		PlanCode:      strings.ToUpper(strings.TrimSpace(input.PlanCode)),
		LinePosition:  strings.TrimSpace(input.LinePosition),
		InspectedAt:   input.InspectedAt.UTC(),
		Inspector:     strings.TrimSpace(input.Inspector),
		DefectLevel:   input.DefectLevel,
		Conclusion:    input.Conclusion,
		BlockedReason: blockedReason,
	}
	if err := s.repository.Create(ctx, &item); err != nil {
		return model.LineInspection{}, fmt.Errorf("create 缆绳检查: %w", err)
	}
	s.supersedePrevious(ctx, &item, actor, requestID)
	if item.Status == string(constants.InspectionStateBlocking) {
		s.returnPendingClearances(ctx, &item, actor, requestID)
	}
	_ = s.security.Audit(ctx, actor, requestID, "create", "LineInspection", item.ID, "", item.Status, "created 缆绳检查")
	return item, nil
}

func (s *lineInspectionService) Update(ctx context.Context, id uint, input dto.UpdateLineInspection, actor, requestID string) (model.LineInspection, error) {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return model.LineInspection{}, err
	}
	if err := validateLineInspectionBusinessFields(current.Code, input.Name, input.PlanCode, input.LinePosition, input.Inspector); err != nil {
		return model.LineInspection{}, err
	}
	current.Name = strings.TrimSpace(input.Name)
	current.Description = strings.TrimSpace(input.Description)
	current.PlanCode = strings.ToUpper(strings.TrimSpace(input.PlanCode))
	current.LinePosition = strings.TrimSpace(input.LinePosition)
	current.InspectedAt = input.InspectedAt.UTC()
	current.Inspector = strings.TrimSpace(input.Inspector)
	current.DefectLevel = input.DefectLevel
	current.Conclusion = input.Conclusion
	// Terminal records keep their historical conclusion; live records escalate
	// to blocking when the edited defect or conclusion requires it. Leaving the
	// blocking state always takes the explicit 换绳复查 transition.
	if current.Status == string(constants.InspectionStateOpen) && constants.IsBlockingInspection(current.DefectLevel, current.Conclusion) {
		current.Status = string(constants.InspectionStateBlocking)
		current.BlockedReason = deriveBlockedReason(current.DefectLevel, current.Conclusion, "")
	}
	current.Version = input.ExpectedVersion + 1
	current.UpdatedAt = time.Now().UTC()
	if err := s.repository.Update(ctx, id, input.ExpectedVersion, &current); err != nil {
		return model.LineInspection{}, fmt.Errorf("update 缆绳检查: %w", err)
	}
	if current.Status == string(constants.InspectionStateBlocking) {
		s.returnPendingClearances(ctx, &current, actor, requestID)
	}
	_ = s.security.Audit(ctx, actor, requestID, "update", "LineInspection", id, current.Status, current.Status, "updated business fields")
	return s.repository.Get(ctx, id)
}

func (s *lineInspectionService) Transition(ctx context.Context, id uint, input dto.TransitionRequest, actor, requestID string) (model.LineInspection, error) {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return model.LineInspection{}, err
	}
	target := strings.TrimSpace(input.Status)
	if !constants.CanTransition(constants.LineInspectionTransitions, current.Status, target) {
		return model.LineInspection{}, fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, current.Status, target)
	}
	before := current.Status
	now := time.Now().UTC()
	switch target {
	case string(constants.InspectionStateBlocking):
		current.BlockedReason = deriveBlockedReason(current.DefectLevel, current.Conclusion, input.Reason)
	case string(constants.InspectionStateResolved):
		// 现场换绳复查通过：固定复查结论并记录复查人，许可随后可重新提交。
		current.Conclusion = constants.ConclusionRecheckPass
		current.ResolvedBy = actor
		current.ResolvedAt = &now
	}
	current.Status = target
	current.Version = input.ExpectedVersion + 1
	current.UpdatedAt = now
	if err := s.repository.Update(ctx, id, input.ExpectedVersion, &current); err != nil {
		return model.LineInspection{}, fmt.Errorf("transition 缆绳检查: %w", err)
	}
	if target == string(constants.InspectionStateBlocking) {
		s.returnPendingClearances(ctx, &current, actor, requestID)
	}
	if err := s.security.Audit(ctx, actor, requestID, "transition", "LineInspection", id, before, target, input.Reason); err != nil {
		return model.LineInspection{}, fmt.Errorf("persist transition audit: %w", err)
	}
	return s.repository.Get(ctx, id)
}

func (s *lineInspectionService) Delete(ctx context.Context, id uint, actor, requestID string) error {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repository.Delete(ctx, id); err != nil {
		return err
	}
	return s.security.Audit(ctx, actor, requestID, "delete", "LineInspection", id, current.Status, "deleted", "soft deleted 缆绳检查")
}

func (s *lineInspectionService) StatusCounts(ctx context.Context) (map[string]int64, error) {
	return s.repository.CountByStatus(ctx)
}

// supersedePrevious closes the earlier live records of the same line position
// so the newest inspection carries the current conclusion while the history
// stays queryable.
func (s *lineInspectionService) supersedePrevious(ctx context.Context, item *model.LineInspection, actor, requestID string) {
	previous, err := s.repository.ListActiveByPosition(ctx, item.PlanCode, item.LinePosition, item.ID)
	if err != nil {
		return
	}
	for _, record := range previous {
		version := record.Version
		before := record.Status
		record.Status = string(constants.InspectionStateSuperseded)
		record.Version = version + 1
		record.UpdatedAt = time.Now().UTC()
		if err := s.repository.Update(ctx, record.ID, version, &record); err != nil {
			continue
		}
		_ = s.security.Audit(ctx, actor, requestID, "transition", "LineInspection", record.ID, before, record.Status,
			fmt.Sprintf("被新检查 %s 取代", item.Code))
	}
}

// returnPendingClearances sends every clearance of the mooring plan that is
// waiting for the second confirmation back to its submitter. The hard stop
// itself is enforced again when the clearance is submitted or confirmed.
func (s *lineInspectionService) returnPendingClearances(ctx context.Context, item *model.LineInspection, actor, requestID string) {
	clearances, err := s.clearances.ListSubmittedPendingByPlan(ctx, item.PlanCode)
	if err != nil {
		return
	}
	reason := fmt.Sprintf("缆绳检查 %s 阻断：%s", item.Code, item.BlockedReason)
	for _, clearance := range clearances {
		version := clearance.Version
		clearance.SubmittedBy = ""
		clearance.SubmittedAt = nil
		clearance.ReturnedReason = reason
		clearance.Version = version + 1
		clearance.UpdatedAt = time.Now().UTC()
		if err := s.clearances.Update(ctx, clearance.ID, version, &clearance); err != nil {
			continue
		}
		_ = s.security.Audit(ctx, actor, requestID, "clearance_return", "SafetyClearance", clearance.ID, clearance.Status, clearance.Status, reason)
	}
}

func deriveBlockedReason(defectLevel, conclusion, fallback string) string {
	reasons := make([]string, 0, 2)
	switch defectLevel {
	case constants.DefectBrokenStrand:
		reasons = append(reasons, "断股缺陷")
	case constants.DefectWearOverLimit:
		reasons = append(reasons, "磨损超限")
	}
	if conclusion == constants.ConclusionReplacePending {
		reasons = append(reasons, "待换绳")
	}
	if len(reasons) == 0 {
		return strings.TrimSpace(fallback)
	}
	return strings.Join(reasons, "，")
}

func validateLineInspectionBusinessFields(code, name, planCode, linePosition, inspector string) error {
	if strings.TrimSpace(code) == "" || strings.TrimSpace(name) == "" || strings.TrimSpace(planCode) == "" ||
		strings.TrimSpace(linePosition) == "" || strings.TrimSpace(inspector) == "" {
		return ErrInvalidInput
	}
	return nil
}
