package repository

import (
	"context"
	"strings"

	"github.com/blueship581/port-mooring-window-safety/backend/internal/dto"
	"github.com/blueship581/port-mooring-window-safety/backend/internal/model"
	"gorm.io/gorm"
)

// LineInspectionRepository owns all persistence operations for 缆绳检查.
type LineInspectionRepository interface {
	List(context.Context, dto.PageQuery) (Page[model.LineInspection], error)
	Get(context.Context, uint) (model.LineInspection, error)
	Create(context.Context, *model.LineInspection) error
	Update(context.Context, uint, uint, *model.LineInspection) error
	Delete(context.Context, uint) error
	CountByStatus(context.Context) (map[string]int64, error)
	ActiveBlocking(context.Context, string) ([]model.LineInspection, error)
	ListActiveByPosition(context.Context, string, string, uint) ([]model.LineInspection, error)
}

type lineInspectionRepository struct {
	store *Store[model.LineInspection]
	db    *gorm.DB
}

func NewLineInspectionRepository(db *gorm.DB) LineInspectionRepository {
	return &lineInspectionRepository{store: NewStore[model.LineInspection](db), db: db}
}

func (r *lineInspectionRepository) List(ctx context.Context, q dto.PageQuery) (Page[model.LineInspection], error) {
	page, pageSize := normalizePage(q.Page, q.PageSize)
	db := r.db.WithContext(ctx).Model(&model.LineInspection{})
	if search := strings.TrimSpace(strings.ToLower(q.Search)); search != "" {
		wildcard := "%" + search + "%"
		db = db.Where("LOWER(code) LIKE ? OR LOWER(name) LIKE ? OR LOWER(line_position) LIKE ?", wildcard, wildcard, wildcard)
	}
	if status := strings.TrimSpace(q.Status); status != "" {
		db = db.Where("status = ?", status)
	}
	if planCode := strings.ToUpper(strings.TrimSpace(q.PlanCode)); planCode != "" {
		db = db.Where("plan_code = ?", planCode)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return Page[model.LineInspection]{}, err
	}
	items := make([]model.LineInspection, 0)
	err := db.Order("inspected_at DESC, id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error
	return Page[model.LineInspection]{Items: items, Total: total, Page: page, PageSize: pageSize}, err
}

func (r *lineInspectionRepository) Get(ctx context.Context, id uint) (model.LineInspection, error) {
	return r.store.Get(ctx, id)
}
func (r *lineInspectionRepository) Create(ctx context.Context, item *model.LineInspection) error {
	return r.store.Create(ctx, item)
}
func (r *lineInspectionRepository) Update(ctx context.Context, id, version uint, item *model.LineInspection) error {
	return r.store.Update(ctx, id, version, item)
}
func (r *lineInspectionRepository) Delete(ctx context.Context, id uint) error {
	return r.store.Delete(ctx, id)
}
func (r *lineInspectionRepository) CountByStatus(ctx context.Context) (map[string]int64, error) {
	return r.store.CountByStatus(ctx)
}

// ActiveBlocking returns the inspections currently holding back the safety
// clearances of one mooring plan (断股、超限磨损或待换绳).
func (r *lineInspectionRepository) ActiveBlocking(ctx context.Context, planCode string) ([]model.LineInspection, error) {
	items := make([]model.LineInspection, 0)
	err := r.db.WithContext(ctx).
		Where("plan_code = ? AND status = ?", strings.ToUpper(strings.TrimSpace(planCode)), "blocking").
		Order("inspected_at DESC, id DESC").Find(&items).Error
	return items, err
}

// ListActiveByPosition returns the open or blocking inspections of one line
// position so a newer record can supersede them. excludeID skips the record
// that triggered the lookup.
func (r *lineInspectionRepository) ListActiveByPosition(ctx context.Context, planCode, linePosition string, excludeID uint) ([]model.LineInspection, error) {
	items := make([]model.LineInspection, 0)
	err := r.db.WithContext(ctx).
		Where("plan_code = ? AND line_position = ? AND status IN ? AND id <> ?",
			strings.ToUpper(strings.TrimSpace(planCode)), strings.TrimSpace(linePosition), []string{"open", "blocking"}, excludeID).
		Order("inspected_at DESC, id DESC").Find(&items).Error
	return items, err
}
