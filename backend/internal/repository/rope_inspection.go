package repository

import (
	"context"
	"strings"
	"time"

	"github.com/blueship581/port-mooring-window-safety/backend/internal/dto"
	"github.com/blueship581/port-mooring-window-safety/backend/internal/model"
	"gorm.io/gorm"
)

// RopeInspectionRepository owns all persistence operations for 缆绳检查.
type RopeInspectionRepository interface {
	List(context.Context, dto.PageQuery) (Page[model.RopeInspection], error)
	Create(context.Context, *model.RopeInspection) error
	LatestPerLine(context.Context, string) ([]model.RopeInspection, error)
	ListOpenByPlan(context.Context, string) ([]model.RopeInspection, error)
	ResolveOpenByLine(context.Context, string, string, time.Time, string) ([]model.RopeInspection, error)
	CountByStatus(context.Context) (map[string]int64, error)
}

type ropeInspectionRepository struct {
	store *Store[model.RopeInspection]
	db    *gorm.DB
}

func NewRopeInspectionRepository(db *gorm.DB) RopeInspectionRepository {
	return &ropeInspectionRepository{store: NewStore[model.RopeInspection](db), db: db}
}

func (r *ropeInspectionRepository) List(ctx context.Context, q dto.PageQuery) (Page[model.RopeInspection], error) {
	page, pageSize := normalizePage(q.Page, q.PageSize)
	db := r.db.WithContext(ctx).Model(&model.RopeInspection{})
	if search := strings.TrimSpace(strings.ToLower(q.Search)); search != "" {
		wildcard := "%" + search + "%"
		db = db.Where("LOWER(code) LIKE ? OR LOWER(line_position) LIKE ? OR LOWER(inspector) LIKE ?", wildcard, wildcard, wildcard)
	}
	if status := strings.TrimSpace(q.Status); status != "" {
		db = db.Where("status = ?", status)
	}
	if planCode := strings.TrimSpace(strings.ToUpper(q.PlanCode)); planCode != "" {
		db = db.Where("plan_code = ?", planCode)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return Page[model.RopeInspection]{}, err
	}
	items := make([]model.RopeInspection, 0)
	err := db.Order("inspected_at DESC, id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error
	return Page[model.RopeInspection]{Items: items, Total: total, Page: page, PageSize: pageSize}, err
}

func (r *ropeInspectionRepository) Create(ctx context.Context, item *model.RopeInspection) error {
	return r.store.Create(ctx, item)
}

// LatestPerLine returns the current conclusion for every 缆位, optionally scoped
// to one mooring plan. MAX(id) tracks the newest record because ids are monotonic.
func (r *ropeInspectionRepository) LatestPerLine(ctx context.Context, planCode string) ([]model.RopeInspection, error) {
	sub := r.db.Model(&model.RopeInspection{}).Select("MAX(id) AS id")
	if planCode = strings.TrimSpace(strings.ToUpper(planCode)); planCode != "" {
		sub = sub.Where("plan_code = ?", planCode)
	}
	sub = sub.Group("plan_code, line_position")
	items := make([]model.RopeInspection, 0)
	err := r.db.WithContext(ctx).Where("id IN (?)", sub).Order("plan_code, line_position").Find(&items).Error
	return items, err
}

// ListOpenByPlan returns every still-open (blocking) inspection of one plan.
func (r *ropeInspectionRepository) ListOpenByPlan(ctx context.Context, planCode string) ([]model.RopeInspection, error) {
	items := make([]model.RopeInspection, 0)
	err := r.db.WithContext(ctx).
		Where("plan_code = ? AND status = ?", strings.ToUpper(strings.TrimSpace(planCode)), "open").
		Order("inspected_at DESC, id DESC").Find(&items).Error
	return items, err
}

// ResolveOpenByLine closes every open inspection for one 缆位 after a passing
// re-inspection and returns the records that were closed for auditing.
func (r *ropeInspectionRepository) ResolveOpenByLine(ctx context.Context, planCode, linePosition string, resolvedAt time.Time, resolvedBy string) ([]model.RopeInspection, error) {
	open := make([]model.RopeInspection, 0)
	if err := r.db.WithContext(ctx).
		Where("plan_code = ? AND line_position = ? AND status = ?", planCode, linePosition, "open").
		Find(&open).Error; err != nil {
		return nil, err
	}
	if len(open) == 0 {
		return open, nil
	}
	ids := make([]uint, 0, len(open))
	for _, item := range open {
		ids = append(ids, item.ID)
	}
	err := r.db.WithContext(ctx).Model(&model.RopeInspection{}).Where("id IN ?", ids).
		Updates(map[string]any{"status": "closed", "resolved_at": resolvedAt, "resolved_by": resolvedBy, "updated_at": resolvedAt}).Error
	return open, err
}

func (r *ropeInspectionRepository) CountByStatus(ctx context.Context) (map[string]int64, error) {
	return r.store.CountByStatus(ctx)
}
