package model

import "time"

// LineInspection models 缆绳检查 as an independently versioned aggregate. Each
// record belongs to a mooring plan (PlanCode) and a specific line position
// (LinePosition); the latest record per position carries the current
// conclusion while older records keep the inspection history.
type LineInspection struct {
	BaseModel
	PlanCode      string     `json:"planCode" gorm:"size:64;index"`
	LinePosition  string     `json:"linePosition" gorm:"size:120;index"`
	InspectedAt   time.Time  `json:"inspectedAt"`
	Inspector     string     `json:"inspector" gorm:"size:80;index"`
	DefectLevel   string     `json:"defectLevel" gorm:"size:32;index"`
	Conclusion    string     `json:"conclusion" gorm:"size:32;index"`
	BlockedReason string     `json:"blockedReason" gorm:"size:500"`
	ResolvedBy    string     `json:"resolvedBy" gorm:"size:80"`
	ResolvedAt    *time.Time `json:"resolvedAt"`
}

func (item *LineInspection) GetBase() *BaseModel { return &item.BaseModel }

func (item LineInspection) TableName() string { return "line_inspections" }

var LineInspectionInitialStatus = "open"
