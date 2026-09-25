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

// Rope inspection 缆绳检查 shared enumerations. Mirrors live in
// frontend/src/types/status.ts; blocking rules must stay identical on both sides.

type RopeInspectionState string

const (
	RopeInspectionStateOpen   RopeInspectionState = "open"
	RopeInspectionStateClosed RopeInspectionState = "closed"
)

var AllRopeInspectionState = []string{"open", "closed"}

const (
	RopeDefectNone          = "none"
	RopeDefectWearMinor     = "wear_minor"
	RopeDefectWearOverlimit = "wear_overlimit"
	RopeDefectBrokenStrand  = "broken_strand"
)

var AllRopeDefectLevel = []string{"none", "wear_minor", "wear_overlimit", "broken_strand"}

const (
	RopeConclusionPassed         = "passed"
	RopeConclusionMonitor        = "monitor"
	RopeConclusionReplacePending = "replace_pending"
)

var AllRopeConclusion = []string{"passed", "monitor", "replace_pending"}

var RopeDefectLabels = map[string]string{
	RopeDefectNone:          "无缺陷",
	RopeDefectWearMinor:     "轻微磨损",
	RopeDefectWearOverlimit: "超限磨损",
	RopeDefectBrokenStrand:  "断股",
}

var RopeConclusionLabels = map[string]string{
	RopeConclusionPassed:         "通过",
	RopeConclusionMonitor:        "观察使用",
	RopeConclusionReplacePending: "待换绳",
}

var RopeInspectionTransitions = map[string]map[string]bool{
	"open":   {"closed": true},
	"closed": {},
}

// IsBlockingRopeInspection reports whether a defect level / conclusion pair must
// block the linked safety clearance: 断股、超限磨损或待换绳均阻断放行。
func IsBlockingRopeInspection(defectLevel, conclusion string) bool {
	return defectLevel == RopeDefectWearOverlimit || defectLevel == RopeDefectBrokenStrand || conclusion == RopeConclusionReplacePending
}

func CanTransition(graph map[string]map[string]bool, from, to string) bool {
	targets, exists := graph[from]
	return exists && targets[to]
}
