package model

import "time"

// RopeInspection models 缆绳检查 as an append-only history scoped to a mooring
// plan and a specific 缆位. The newest record per line position is the current
// conclusion; any open blocking record prevents the linked safety clearance
// from being released until a passing re-inspection closes it.
type RopeInspection struct {
	BaseModel
	PlanCode      string     `json:"planCode" gorm:"size:64;index"`
	LinePosition  string     `json:"linePosition" gorm:"size:80;index"`
	InspectedAt   time.Time  `json:"inspectedAt"`
	Inspector     string     `json:"inspector" gorm:"size:80;index"`
	DefectLevel   string     `json:"defectLevel" gorm:"size:32;index"`
	Conclusion    string     `json:"conclusion" gorm:"size:40;index"`
	BlockedReason string     `json:"blockedReason" gorm:"size:500"`
	ResolvedAt    *time.Time `json:"resolvedAt"`
	ResolvedBy    string     `json:"resolvedBy" gorm:"size:80"`
}

func (item *RopeInspection) GetBase() *BaseModel { return &item.BaseModel }

func (item RopeInspection) TableName() string { return "rope_inspections" }
