package dto

import "time"

// CreateLineInspection is the public write contract for 缆绳检查. Status is
// derived from the defect level and disposal conclusion so callers cannot
// bypass the blocking rules.
type CreateLineInspection struct {
	Code         string    `json:"code" binding:"required,min=2,max=64"`
	Name         string    `json:"name" binding:"required,min=2,max=160"`
	Description  string    `json:"description" binding:"max=1000"`
	PlanCode     string    `json:"planCode" binding:"required,max=64"`
	LinePosition string    `json:"linePosition" binding:"required,min=1,max=120"`
	InspectedAt  time.Time `json:"inspectedAt" binding:"required"`
	Inspector    string    `json:"inspector" binding:"required,max=80"`
	DefectLevel  string    `json:"defectLevel" binding:"required,oneof=none minor_wear wear_over_limit broken_strand"`
	Conclusion   string    `json:"conclusion" binding:"required,oneof=pass monitor replace_pending recheck_pass"`
}

type UpdateLineInspection struct {
	ExpectedVersion uint      `json:"expectedVersion" binding:"required"`
	Name            string    `json:"name" binding:"required,min=2,max=160"`
	Description     string    `json:"description" binding:"max=1000"`
	PlanCode        string    `json:"planCode" binding:"required,max=64"`
	LinePosition    string    `json:"linePosition" binding:"required,min=1,max=120"`
	InspectedAt     time.Time `json:"inspectedAt" binding:"required"`
	Inspector       string    `json:"inspector" binding:"required,max=80"`
	DefectLevel     string    `json:"defectLevel" binding:"required,oneof=none minor_wear wear_over_limit broken_strand"`
	Conclusion      string    `json:"conclusion" binding:"required,oneof=pass monitor replace_pending recheck_pass"`
}
