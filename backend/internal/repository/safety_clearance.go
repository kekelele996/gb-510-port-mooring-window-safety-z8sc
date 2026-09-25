package repository

import (
	"context"
	"strings"

	"github.com/blueship581/port-mooring-window-safety/backend/internal/dto"
	"github.com/blueship581/port-mooring-window-safety/backend/internal/model"
	"gorm.io/gorm"
)

// SafetyClearanceRepository owns all persistence operations for 安全许可.
type SafetyClearanceRepository interface {
	List(context.Context, dto.PageQuery) (Page[model.SafetyClearance], error)
	Get(context.Context, uint) (model.SafetyClearance, error)
	Create(context.Context, *model.SafetyClearance) error
	Update(context.Context, uint, uint, *model.SafetyClearance) error
	Delete(context.Context, uint) error
	CountByStatus(context.Context) (map[string]int64, error)
	ListSubmittedPendingByPlan(context.Context, string) ([]model.SafetyClearance, error)
}

type safetyClearanceRepository struct {
	store *Store[model.SafetyClearance]
	db    *gorm.DB
}

func NewSafetyClearanceRepository(db *gorm.DB) SafetyClearanceRepository {
	return &safetyClearanceRepository{store: NewStore[model.SafetyClearance](db), db: db}
}

func (r *safetyClearanceRepository) List(ctx context.Context, q dto.PageQuery) (Page[model.SafetyClearance], error) {
	return r.store.List(ctx, q)
}
func (r *safetyClearanceRepository) Get(ctx context.Context, id uint) (model.SafetyClearance, error) {
	return r.store.Get(ctx, id)
}
func (r *safetyClearanceRepository) Create(ctx context.Context, item *model.SafetyClearance) error {
	return r.store.Create(ctx, item)
}
func (r *safetyClearanceRepository) Update(ctx context.Context, id, version uint, item *model.SafetyClearance) error {
	return r.store.Update(ctx, id, version, item)
}
func (r *safetyClearanceRepository) Delete(ctx context.Context, id uint) error {
	return r.store.Delete(ctx, id)
}
func (r *safetyClearanceRepository) CountByStatus(ctx context.Context) (map[string]int64, error) {
	return r.store.CountByStatus(ctx)
}

// ListSubmittedPendingByPlan returns the clearances of one mooring plan that
// are waiting for the second confirmation; a blocking line inspection must
// send them back to the submitter.
func (r *safetyClearanceRepository) ListSubmittedPendingByPlan(ctx context.Context, planCode string) ([]model.SafetyClearance, error) {
	items := make([]model.SafetyClearance, 0)
	err := r.db.WithContext(ctx).
		Where("plan_code = ? AND status = ? AND submitted_by <> ''",
			strings.ToUpper(strings.TrimSpace(planCode)), "pending").
		Order("id ASC").Find(&items).Error
	return items, err
}
