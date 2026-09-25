package dto

import "time"

// CreateRopeInspection is the public write contract for 缆绳检查. Status is
// derived by the service from the defect level and conclusion so callers can
// never mark a blocking inspection as closed.
type CreateRopeInspection struct {
	Code         string    `json:"code" binding:"required,min=2,max=64"`
	PlanCode     string    `json:"planCode" binding:"required,max=64"`
	LinePosition string    `json:"linePosition" binding:"required,min=1,max=80"`
	InspectedAt  time.Time `json:"inspectedAt" binding:"required"`
	Inspector    string    `json:"inspector" binding:"required,max=80"`
	DefectLevel  string    `json:"defectLevel" binding:"required,oneof=none wear_minor wear_overlimit broken_strand"`
	Conclusion   string    `json:"conclusion" binding:"required,oneof=passed monitor replace_pending"`
	Notes        string    `json:"notes" binding:"max=1000"`
}
