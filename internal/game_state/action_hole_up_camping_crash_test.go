package game_state

import (
	"testing"
)

// TestCampingWithWanderer_DetailedExecution provides detailed execution trace
func TestCampingWithWanderer_DetailedExecution(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		Build()

	gs.SetRandomSeed(12345)

	t.Logf("\n=== STARTING CAMPING WITH WANDERER ===")
	t.Logf("Hours: 9, Guard: 0, ForceWanderer: true, ForceAmbush: false")

	success := gs.ActionHoleUpLargeMapWithDebug(9, 0, true, false)
	if !success {
		t.Fatalf("Failed to start camping")
	}

	// Track state changes
	previousCampingActive := true
	eventQueueStarted := false

	for i := 0; i < 10000; i++ {
		currentCampingActive := gs.CampingScene != nil

		// Detect state transitions
		if !eventQueueStarted && currentCampingActive {
			eventQueueStarted = true
			t.Logf("\n[Update %d] Event queue started, CampingScene active", i)
		}

		if previousCampingActive && !currentCampingActive {
			t.Logf("\n[Update %d] *** CAMPING COMPLETED - CampingScene became nil ***", i)
			t.Logf("Total updates to completion: %d", i)
			return
		}

		// Update simulation
		gs.EventQueueManager.Update()
		gs.ElapsedMs += 16

		previousCampingActive = currentCampingActive
	}

	t.Fatalf("Camping did not complete after 10000 updates")
}
