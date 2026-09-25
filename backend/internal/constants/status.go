package constants

// Shared status values are mirrored in frontend/src/types/status.ts. Keeping
// the lists explicit makes state-machine drift visible during code review.

type CallState string

const (
	CallStatePlanned  CallState = "planned"
	CallStateApproach CallState = "approach"
	CallStateMoored   CallState = "moored"
	CallStateDeparted CallState = "departed"
)

var AllCallState = []string{"planned", "approach", "moored", "departed"}

type ClearanceState string

const (
	ClearanceStatePending    ClearanceState = "pending"
	ClearanceStateCleared    ClearanceState = "cleared"
	ClearanceStateRestricted ClearanceState = "restricted"
	ClearanceStateExpired    ClearanceState = "expired"
)

var AllClearanceState = []string{"pending", "cleared", "restricted", "expired"}

type InspectionState string

const (
	InspectionStateOpen       InspectionState = "open"
	InspectionStateBlocking   InspectionState = "blocking"
	InspectionStateResolved   InspectionState = "resolved"
	InspectionStateSuperseded InspectionState = "superseded"
)

var AllInspectionState = []string{"open", "blocking", "resolved", "superseded"}

// DefectLevel values mirror frontend/src/types/status.ts ALL_DEFECT_LEVEL.
const (
	DefectNone          = "none"
	DefectMinorWear     = "minor_wear"
	DefectWearOverLimit = "wear_over_limit"
	DefectBrokenStrand  = "broken_strand"
)

var AllDefectLevel = []string{"none", "minor_wear", "wear_over_limit", "broken_strand"}

// InspectionConclusion values mirror frontend/src/types/status.ts ALL_INSPECTION_CONCLUSION.
const (
	ConclusionPass           = "pass"
	ConclusionMonitor        = "monitor"
	ConclusionReplacePending = "replace_pending"
	ConclusionRecheckPass    = "recheck_pass"
)

var AllInspectionConclusion = []string{"pass", "monitor", "replace_pending", "recheck_pass"}

// IsBlockingInspection reports whether a defect level or disposal conclusion
// must hold back the associated safety clearance: 断股、超限磨损或待换绳。
func IsBlockingInspection(defectLevel, conclusion string) bool {
	return defectLevel == DefectBrokenStrand || defectLevel == DefectWearOverLimit || conclusion == ConclusionReplacePending
}

var VesselCallTransitions = map[string]map[string]bool{
	"planned":  {"approach": true, "moored": true},
	"approach": {"moored": true, "departed": true, "planned": true},
	"moored":   {"departed": true, "approach": true},
	"departed": {"moored": true},
}

var MooringPlanTransitions = map[string]map[string]bool{
	"draft":      {"review": true, "approved": true},
	"review":     {"approved": true, "superseded": true, "draft": true},
	"approved":   {"superseded": true, "review": true},
	"superseded": {"approved": true},
}

var WeatherWindowTransitions = map[string]map[string]bool{
	"forecast":   {"safe": true, "restricted": true},
	"safe":       {"restricted": true, "expired": true, "forecast": true},
	"restricted": {"expired": true, "safe": true},
	"expired":    {"restricted": true},
}

var SafetyClearanceTransitions = map[string]map[string]bool{
	"pending":    {"cleared": true, "restricted": true},
	"cleared":    {"restricted": true, "expired": true, "pending": true},
	"restricted": {"expired": true, "cleared": true},
	"expired":    {"restricted": true},
}

var LineInspectionTransitions = map[string]map[string]bool{
	"open":       {"blocking": true, "resolved": true, "superseded": true},
	"blocking":   {"resolved": true, "superseded": true},
	"resolved":   {},
	"superseded": {},
}

func CanTransition(graph map[string]map[string]bool, from, to string) bool {
	targets, exists := graph[from]
	return exists && targets[to]
}
