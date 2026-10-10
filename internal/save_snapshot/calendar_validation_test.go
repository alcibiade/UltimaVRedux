package save_snapshot

import "testing"

func TestValidateAcceptsThirteenthCalendarMonth(t *testing.T) {
	t.Parallel()

	snapshot := validSnapshot()
	snapshot.World.Calendar.Month = 13
	snapshot.World.LastWandererMonth = 13

	if issues := Validate(snapshot); len(issues) != 0 {
		t.Fatalf("Validate() issues = %#v, want month 13 accepted", issues)
	}
}

func TestValidateRejectsFourteenthCalendarMonth(t *testing.T) {
	t.Parallel()

	snapshot := validSnapshot()
	snapshot.World.Calendar.Month = 14

	issues := Validate(snapshot)
	if !hasIssue(issues, "world.calendar", "out_of_range") {
		t.Fatalf("Validate() issues = %#v, want month 14 rejection", issues)
	}
}

func TestValidateRejectsFourteenthWandererMonth(t *testing.T) {
	t.Parallel()

	snapshot := validSnapshot()
	snapshot.World.LastWandererMonth = 14

	issues := Validate(snapshot)
	if !hasIssue(issues, "world", "out_of_range") {
		t.Fatalf("Validate() issues = %#v, want wanderer month 14 rejection", issues)
	}
}
