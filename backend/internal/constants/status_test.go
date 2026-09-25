package constants

import "testing"

func TestVesselCallTransitionGraph(t *testing.T) {
	if !CanTransition(VesselCallTransitions, "planned", "approach") {
		t.Fatalf("expected planned -> approach transition to be allowed")
	}
	if CanTransition(VesselCallTransitions, "planned", "unknown") {
		t.Fatal("unknown status must never be accepted")
	}
}

func TestLineInspectionTransitionGraph(t *testing.T) {
	if !CanTransition(LineInspectionTransitions, "blocking", "resolved") {
		t.Fatalf("expected blocking -> resolved transition to be allowed")
	}
	if !CanTransition(LineInspectionTransitions, "open", "blocking") {
		t.Fatalf("expected open -> blocking transition to be allowed")
	}
	if CanTransition(LineInspectionTransitions, "resolved", "blocking") {
		t.Fatal("resolved inspections must stay terminal")
	}
	if CanTransition(LineInspectionTransitions, "superseded", "open") {
		t.Fatal("superseded inspections must stay terminal")
	}
}

func TestIsBlockingInspection(t *testing.T) {
	for _, defect := range []string{"broken_strand", "wear_over_limit"} {
		if !IsBlockingInspection(defect, "pass") {
			t.Fatalf("expected defect %s to block", defect)
		}
	}
	if !IsBlockingInspection("none", "replace_pending") {
		t.Fatal("pending rope replacement must block")
	}
	if IsBlockingInspection("minor_wear", "monitor") {
		t.Fatal("minor wear under observation must not block")
	}
	if IsBlockingInspection("none", "recheck_pass") {
		t.Fatal("passed recheck must not block")
	}
}
