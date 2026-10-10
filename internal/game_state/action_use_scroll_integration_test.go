// Integration tests for Scroll usage workflows with real game data.
// These tests validate scroll effects, consumption, and location restrictions.
package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// TestScrollRelHur_WindChange tests the Rel Hur (Wind Change) scroll effect.
// Scroll 1: Changes wind direction on overworld only.
// Uses workflow pattern: useScroll starts workflow → direction input → CompleteWindChangeScroll finishes.
func TestScrollRelHur_WindChange(t *testing.T) {
	t.Run("SucceedsOnLargeMap_WorkflowPattern", func(t *testing.T) {
		gs, mockCallbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.BritanniaUnderworld).
			WithPlayerAt(100, 100).
			WithSystemCallbacks().
			Build()

		if gs == nil {
			return // Test was skipped due to missing game data
		}

		mockCallbacks.Reset()
		gs.SetRandomSeed(12345) // Deterministic wind direction

		// Give party scroll
		gs.PartyState.Inventory.Scrolls.Set(references.ScrollRelHur, 1)
		initialScrolls := gs.PartyState.Inventory.Scrolls.Get(references.ScrollRelHur)

		// Step 1: Start the workflow (shows prompt, waits for direction)
		result := gs.useScroll(references.ScrollRelHur)

		// Workflow started successfully
		assert.True(t, result, "Scroll workflow should start on large map")
		assert.True(t, mockCallbacks.StartWindChangeScrollWorkflowCalled,
			"Should trigger StartWindChangeScrollWorkflow callback")

		// Scroll NOT consumed yet (waiting for direction input)
		assert.Equal(t, initialScrolls, gs.PartyState.Inventory.Scrolls.Get(references.ScrollRelHur),
			"Scroll should NOT be consumed before direction input")

		// Step 2: Complete the workflow with direction (simulates player input)
		completeResult := gs.CompleteWindChangeScroll(references.Up)
		assert.True(t, completeResult, "CompleteWindChangeScroll should succeed")

		// NOW scroll should be consumed
		assert.Equal(t, initialScrolls-1, gs.PartyState.Inventory.Scrolls.Get(references.ScrollRelHur),
			"Should consume scroll after direction input")

		// Should show combined spell invocation and wind direction in popup
		assert.True(t, len(mockCallbacks.TimedOverlayMessages) > 0,
			"Should show timed overlay message")
		assert.Contains(t, mockCallbacks.TimedOverlayMessages[0], "Rel Hur!",
			"Popup should contain spell incantation")

		// Should have advanced time
		assert.Equal(t, 1, mockCallbacks.TimeAdvanced()[0], "Should advance time by 1 minute")

		t.Logf("Wind change scroll used successfully with direction workflow")
	})

	t.Run("WorksOnSmallMap", func(t *testing.T) {
		gs, mockCallbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.LordBritishsCastle).
			WithPlayerAt(12, 12).
			WithSystemCallbacks().
			Build()

		if gs == nil {
			return // Test was skipped due to missing game data
		}

		mockCallbacks.Reset()

		// Give party scroll
		gs.PartyState.Inventory.Scrolls.Set(references.ScrollRelHur, 1)
		initialScrolls := gs.PartyState.Inventory.Scrolls.Get(references.ScrollRelHur)

		result := gs.useScroll(references.ScrollRelHur)

		assert.True(t, result, "Town context permits this scroll (ref:CONSUME-0003)")

		// Should NOT have consumed scroll
		assert.Equal(t, initialScrolls, gs.PartyState.Inventory.Scrolls.Get(references.ScrollRelHur),
			"Direction input precedes scroll commitment")

		// Should show location restriction feedback (via ShowActionFeedback which adds to Messages with alsoAddToOutput=true)
		assert.False(t, mockCallbacks.MessagesContain("Not here!"))

		// Should NOT advance time
		assert.Equal(t, 0, len(mockCallbacks.TimeAdvanced()), "Should not advance time on failure")
	})
}

// TestScrollInQuasWis_View tests the In Quas Wis (View) scroll effect.
// Scroll 4: Shows overhead map view on implemented overworld or town maps.
func TestScrollInQuasWis_View(t *testing.T) {
	t.Run("SucceedsOnLargeMap", func(t *testing.T) {
		gs, mockCallbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.BritanniaUnderworld).
			WithPlayerAt(100, 100).
			WithSystemCallbacks().
			Build()

		if gs == nil {
			return // Test was skipped due to missing game data
		}

		mockCallbacks.Reset()

		// Give party scroll
		gs.PartyState.Inventory.Scrolls.Set(references.ScrollInQuasWis, 1)
		initialScrolls := gs.PartyState.Inventory.Scrolls.Get(references.ScrollInQuasWis)

		result := gs.useScroll(references.ScrollInQuasWis)

		// Should succeed
		assert.True(t, result, "View scroll should succeed on large map")

		// Should have consumed scroll
		assert.Equal(t, initialScrolls-1, gs.PartyState.Inventory.Scrolls.Get(references.ScrollInQuasWis),
			"Should consume exactly one scroll")

		// Should have triggered ShowGemViewInteraction callback
		assert.True(t, mockCallbacks.ShowGemViewInteractionCalled,
			"Should trigger ShowGemViewInteraction callback")

		// Should have advanced time
		assert.Equal(t, 1, mockCallbacks.TimeAdvanced()[0], "Should advance time by 1 minute")
	})

	t.Run("SucceedsOnSmallMap", func(t *testing.T) {
		gs, mockCallbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.LordBritishsCastle).
			WithPlayerAt(12, 12).
			WithSystemCallbacks().
			Build()

		if gs == nil {
			return // Test was skipped due to missing game data
		}

		mockCallbacks.Reset()

		// Give party scroll
		gs.PartyState.Inventory.Scrolls.Set(references.ScrollInQuasWis, 1)
		initialScrolls := gs.PartyState.Inventory.Scrolls.Get(references.ScrollInQuasWis)

		result := gs.useScroll(references.ScrollInQuasWis)

		assert.True(t, result, "View scroll should succeed on town maps")

		assert.Equal(t, initialScrolls-1, gs.PartyState.Inventory.Scrolls.Get(references.ScrollInQuasWis),
			"Should consume exactly one scroll")

		assert.True(t, mockCallbacks.ShowGemViewInteractionCalled,
			"Should trigger ShowGemViewInteraction on town maps")

		require.Len(t, mockCallbacks.TimeAdvanced(), 1)
		assert.Equal(t, 1, mockCallbacks.TimeAdvanced()[0])
	})

	t.Run("GenericExecutorDefersDungeonViewToCommandWorkflow", func(t *testing.T) {
		gs, mockCallbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.Deceit).
			WithPlayerAt(3, 3).
			WithSystemCallbacks().
			Build()
		if gs == nil {
			return
		}
		mockCallbacks.Reset()
		gs.PartyState.Inventory.Scrolls.Set(references.ScrollInQuasWis, 1)

		assert.False(t, gs.useScroll(references.ScrollInQuasWis))
		assert.Equal(t, uint16(1), gs.PartyState.Inventory.Scrolls.Get(references.ScrollInQuasWis))
		assert.False(t, mockCallbacks.ShowGemViewInteractionCalled)
		assert.Empty(t, mockCallbacks.TimeAdvanced())
		assert.True(t, mockCallbacks.MessagesContain("Not here!"))
	})
}

// TestScrollLocationRestrictions tests all scroll location restrictions.
func TestScrollLocationRestrictions(t *testing.T) {
	t.Run("ScrollVasLor_AllowedEverywhere", func(t *testing.T) {
		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.LordBritishsCastle).
			WithPlayerAt(12, 12).
			WithSystemCallbacks().
			Build()

		if gs == nil {
			return
		}

		// Light scroll should be valid on small map
		assert.True(t, gs.isScrollLocationValid(references.ScrollVasLor),
			"Light scroll should be allowed on small map")
	})

	t.Run("ScrollRelHur_SurfaceMaps", func(t *testing.T) {
		// Unlike the prepared spell, the scroll also permits towns.
		gs1, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.LordBritishsCastle).
			WithPlayerAt(12, 12).
			WithSystemCallbacks().
			Build()

		if gs1 == nil {
			return
		}

		assert.True(t, gs1.isScrollLocationValid(references.ScrollRelHur),
			"Wind scroll should be allowed on small map")

		// Test large map (should succeed)
		gs2, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.BritanniaUnderworld).
			WithPlayerAt(100, 100).
			WithSystemCallbacks().
			Build()

		if gs2 == nil {
			return
		}

		assert.True(t, gs2.isScrollLocationValid(references.ScrollRelHur),
			"Wind scroll should be allowed on large map")
	})

	t.Run("ScrollInQuasWis_SurfaceExecutor", func(t *testing.T) {
		gs1, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.LordBritishsCastle).
			WithPlayerAt(12, 12).
			WithSystemCallbacks().
			Build()

		if gs1 == nil {
			return
		}

		assert.True(t, gs1.isScrollLocationValid(references.ScrollInQuasWis),
			"View scroll should be allowed on town maps")

		// Test large map (should succeed)
		gs2, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.BritanniaUnderworld).
			WithPlayerAt(100, 100).
			WithSystemCallbacks().
			Build()

		if gs2 == nil {
			return
		}

		assert.True(t, gs2.isScrollLocationValid(references.ScrollInQuasWis),
			"View scroll should be allowed on large map")

		gs3, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Deceit).
			WithPlayerAt(3, 3).
			WithSystemCallbacks().
			Build()
		if gs3 == nil {
			return
		}
		assert.False(t, gs3.isScrollLocationValid(references.ScrollInQuasWis),
			"Generic scroll execution must defer dungeon View to its command workflow")
	})

	t.Run("ScrollInManiCorp_NonCombatOnly", func(t *testing.T) {
		// Test small map (should succeed - non-combat)
		gs1, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.LordBritishsCastle).
			WithPlayerAt(12, 12).
			WithSystemCallbacks().
			Build()

		if gs1 == nil {
			return
		}

		assert.True(t, gs1.isScrollLocationValid(references.ScrollInManiCorp),
			"Resurrection scroll should be allowed in non-combat")

		// Test large map (should succeed - non-combat)
		gs2, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.BritanniaUnderworld).
			WithPlayerAt(100, 100).
			WithSystemCallbacks().
			Build()

		if gs2 == nil {
			return
		}

		assert.True(t, gs2.isScrollLocationValid(references.ScrollInManiCorp),
			"Resurrection scroll should be allowed on large map")
	})

	t.Run("ScrollKalXenCorp_CombatOnly", func(t *testing.T) {
		// Test small map (should fail - not in combat)
		gs1, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.LordBritishsCastle).
			WithPlayerAt(12, 12).
			WithSystemCallbacks().
			Build()

		if gs1 == nil {
			return
		}

		assert.False(t, gs1.isScrollLocationValid(references.ScrollKalXenCorp),
			"Summon scroll should NOT be allowed outside combat")

		// Test large map (should fail - not in combat)
		gs2, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.BritanniaUnderworld).
			WithPlayerAt(100, 100).
			WithSystemCallbacks().
			Build()

		if gs2 == nil {
			return
		}

		assert.False(t, gs2.isScrollLocationValid(references.ScrollKalXenCorp),
			"Summon scroll should NOT be allowed outside combat")
	})
}

// TestScrollConsumption tests that scrolls are consumed correctly.
// Wind Change scroll uses workflow pattern - consumption happens after direction input.
func TestScrollConsumption(t *testing.T) {
	t.Run("OnlyConsumedOnSuccess_WorkflowScroll", func(t *testing.T) {
		gs, mockCallbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.BritanniaUnderworld).
			WithPlayerAt(100, 100).
			WithSystemCallbacks().
			Build()

		if gs == nil {
			return
		}

		mockCallbacks.Reset()
		gs.SetRandomSeed(12345)

		// Give party multiple scrolls
		gs.PartyState.Inventory.Scrolls.Set(references.ScrollRelHur, 3)

		// Start workflow
		result := gs.useScroll(references.ScrollRelHur)
		assert.True(t, result, "Scroll workflow should start")
		assert.Equal(t, uint16(3), gs.PartyState.Inventory.Scrolls.Get(references.ScrollRelHur),
			"Should still have 3 scrolls before direction input")

		// Complete workflow with direction
		gs.CompleteWindChangeScroll(references.Up)
		assert.Equal(t, uint16(2), gs.PartyState.Inventory.Scrolls.Get(references.ScrollRelHur),
			"Should have 2 scrolls left after successful use")
	})

	t.Run("LastScrollConsumed_WorkflowScroll", func(t *testing.T) {
		gs, mockCallbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.BritanniaUnderworld).
			WithPlayerAt(100, 100).
			WithSystemCallbacks().
			Build()

		if gs == nil {
			return
		}

		mockCallbacks.Reset()
		gs.SetRandomSeed(12345)

		// Give party exactly one scroll
		gs.PartyState.Inventory.Scrolls.Set(references.ScrollRelHur, 1)

		// Start and complete workflow
		result := gs.useScroll(references.ScrollRelHur)
		assert.True(t, result, "Last scroll workflow should start")
		gs.CompleteWindChangeScroll(references.Down)
		assert.Equal(t, uint16(0), gs.PartyState.Inventory.Scrolls.Get(references.ScrollRelHur),
			"Should have 0 scrolls left after using last one")
	})

	t.Run("ViewScroll_ImmediateConsumption", func(t *testing.T) {
		// In Quas Wis (View) scroll still uses immediate consumption pattern
		gs, mockCallbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.BritanniaUnderworld).
			WithPlayerAt(100, 100).
			WithSystemCallbacks().
			Build()

		if gs == nil {
			return
		}

		mockCallbacks.Reset()

		// Give party View scroll
		gs.PartyState.Inventory.Scrolls.Set(references.ScrollInQuasWis, 1)

		// Use scroll - should consume immediately
		result := gs.useScroll(references.ScrollInQuasWis)
		assert.True(t, result, "View scroll should succeed")
		assert.Equal(t, uint16(0), gs.PartyState.Inventory.Scrolls.Get(references.ScrollInQuasWis),
			"View scroll should be consumed immediately")
	})
}

func TestNonTimeStopScrollsFollowNormalRoutesAtCastAbsorptionLocations(t *testing.T) {
	tests := []struct {
		name         string
		scroll       references.Scroll
		wantAccepted bool
		wantQuantity uint16
	}{
		{name: "Vas Lor", scroll: references.ScrollVasLor, wantAccepted: true},
		{name: "Rel Hur", scroll: references.ScrollRelHur, wantAccepted: true, wantQuantity: 1},
		{name: "In Sanct", scroll: references.ScrollInSanct, wantAccepted: true},
		{name: "In An", scroll: references.ScrollInAn, wantAccepted: true},
		{name: "In Quas Wis", scroll: references.ScrollInQuasWis, wantAccepted: true},
		{name: "Kal Xen Corp", scroll: references.ScrollKalXenCorp, wantQuantity: 1},
		{name: "In Mani Corp", scroll: references.ScrollInManiCorp, wantAccepted: true, wantQuantity: 1},
	}

	for _, location := range []references.Location{references.Stonegate, references.PalaceOfBlackthorn} {
		for _, test := range tests {
			t.Run(location.String()+"/"+test.name, func(t *testing.T) {
				state, callbacks := NewIntegrationTestBuilder(t).
					WithLocation(location).
					WithPlayerAt(12, 12).
					WithSystemCallbacks().
					Build()
				state.PartyState.Inventory.Scrolls.Set(test.scroll, 1)
				callbacks.Reset()

				accepted := state.useScroll(test.scroll)

				assert.Equal(t, test.wantAccepted, accepted)
				assert.Equal(t, test.wantQuantity, state.PartyState.Inventory.Scrolls.Get(test.scroll))
				if test.scroll == references.ScrollRelHur {
					// Original item-use rules permit surface maps without the prepared Cast absorption gate.
					// See ref:CONSUME-0003.
					assert.True(t, callbacks.StartWindChangeScrollWorkflowCalled)
					require.True(t, state.CompleteWindChangeScroll(references.Up))
					assert.Zero(t, state.PartyState.Inventory.Scrolls.Get(test.scroll))
				}
				assert.False(t, callbacks.MessagesContain("Absorbed!"))
				assert.False(t, callbacks.SoundEffectWasPlayed(SoundAbsorb))
			})
		}
	}
}

func TestTimeStopScrollFollowsNormalRouteAtPalace(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.PalaceOfBlackthorn).
		WithSystemCallbacks().
		Build()
	state.PartyState.Inventory.Scrolls.Set(references.ScrollAnTym, 1)
	callbacks.Reset()

	require.True(t, state.useScroll(references.ScrollAnTym))
	assert.Zero(t, state.PartyState.Inventory.Scrolls.Get(references.ScrollAnTym))
	assert.True(t, callbacks.StartTimeStopScrollWorkflowCalled)
	assert.False(t, callbacks.MessagesContain("Absorbed!"))
	assert.False(t, callbacks.SoundEffectWasPlayed(SoundAbsorb))
}

// TestScrollDurationEffects tests duration-based scroll effects (Protection, Negate Magic).
// Note: VasLor (Light) and AnTym (Time Stop) use workflow callback patterns with visual effects,
// so they are tested separately in TestVasLorScrollTriggersWorkflow and TestAnTymScrollTriggersWorkflow.
func TestScrollDurationEffects(t *testing.T) {
	// Get expected durations from JSON via test helper
	invRefs := references.NewInventoryItemsReferences()

	testCases := []struct {
		name           string
		scroll         references.Scroll
		expectedEffect DurationEffectType
		expectedTurns  int
		expectedMsg    string
	}{
		// VasLor excluded - uses workflow callback pattern (tested in TestVasLorScrollTriggersWorkflow)
		// AnTym excluded - uses workflow callback pattern (tested in TestAnTymScrollTriggersWorkflow)
		{"InSanct_Protection", references.ScrollInSanct, DurationEffectProtection, invRefs.GetScrollEffectDuration(references.ScrollInSanct), "Protection!"},
		{"InAn_NegateMagic", references.ScrollInAn, DurationEffectNegateMagic, invRefs.GetScrollEffectDuration(references.ScrollInAn), "Negate magic!"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gs, mockCallbacks := NewIntegrationTestBuilder(t).
				WithLocation(references.BritanniaUnderworld). // Large map
				WithPlayerAt(100, 100).
				WithSystemCallbacks().
				Build()

			if gs == nil {
				return
			}

			mockCallbacks.Reset()

			// Give party scroll
			gs.PartyState.Inventory.Scrolls.Set(tc.scroll, 1)
			initialScrolls := gs.PartyState.Inventory.Scrolls.Get(tc.scroll)

			result := gs.useScroll(tc.scroll)

			// Should succeed
			assert.True(t, result, "%s scroll should succeed", tc.name)

			// Should consume scroll
			assert.Equal(t, initialScrolls-1, gs.PartyState.Inventory.Scrolls.Get(tc.scroll),
				"Should consume scroll on success")

			// Should activate duration effect
			assert.True(t, gs.HasActiveDurationEffect(tc.expectedEffect),
				"Should activate %s effect", tc.name)

			// Should have correct duration
			assert.Equal(t, tc.expectedTurns, gs.ActiveDurationEffect.TurnsRemaining,
				"Should have correct duration for %s", tc.name)

			// Should show activation message
			assert.True(t, mockCallbacks.MessagesContain(tc.expectedMsg),
				"Should show '%s' message", tc.expectedMsg)
		})
	}
}

// TestScrollDurationEffectRefresh tests that using the same scroll refreshes the shared effect.
// Note: VasLor uses workflow callback pattern; this test uses InSanct (Protection) which calls ActivateProtectionEffect directly.
func TestScrollDurationEffectRefresh(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		Build()

	if gs == nil {
		return
	}

	// Get expected duration from JSON
	expectedDuration := gs.GameReferences.InventoryItemReferences.GetScrollEffectDuration(references.ScrollInSanct)

	// Give party two Protection scrolls (InSanct calls ActivateProtectionEffect directly)
	gs.PartyState.Inventory.Scrolls.Set(references.ScrollInSanct, 2)

	// Use first scroll - should activate effect
	result1 := gs.useScroll(references.ScrollInSanct)
	assert.True(t, result1, "First Protection scroll should succeed")
	assert.True(t, gs.HasActiveDurationEffect(DurationEffectProtection), "Protection effect should be active")
	assert.Equal(t, expectedDuration, gs.ActiveDurationEffect.TurnsRemaining)

	mockCallbacks.Reset()

	gs.ActiveDurationEffect.TurnsRemaining = 1

	// Use second scroll - original duration installation refreshes the same effect.
	// See ref:CONSUME-0003.
	result2 := gs.useScroll(references.ScrollInSanct)
	assert.True(t, result2, "Second Protection scroll should refresh the effect")
	assert.True(t, gs.HasActiveDurationEffect(DurationEffectProtection), "Protection effect should remain active")
	assert.Equal(t, expectedDuration, gs.ActiveDurationEffect.TurnsRemaining)
	assert.True(t, mockCallbacks.MessagesContain("Protection!"), "Should show activation feedback")

	// Refreshing still consumes the scroll.
	assert.Equal(t, 0, int(gs.PartyState.Inventory.Scrolls.Get(references.ScrollInSanct)),
		"Should have consumed both scrolls")
}

// TestVasLorScrollTriggersWorkflow tests that VasLor scroll triggers the visual effects workflow.
func TestVasLorScrollTriggersWorkflow(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		Build()

	if gs == nil {
		return
	}

	// Give party a Light scroll
	gs.PartyState.Inventory.Scrolls.Set(references.ScrollVasLor, 1)

	// Use scroll - should trigger workflow callback
	result := gs.useScroll(references.ScrollVasLor)
	assert.True(t, result, "Light scroll should succeed")
	assert.True(t, mockCallbacks.StartLightScrollWorkflowCalled, "Should trigger StartLightScrollWorkflow callback")

	// Scroll should be consumed
	assert.Equal(t, 0, int(gs.PartyState.Inventory.Scrolls.Get(references.ScrollVasLor)),
		"Scroll should be consumed")
}

// TestAnTymScrollTriggersWorkflow tests that AnTym (Time Stop) scroll triggers the visual effects workflow.
// The actual time stop effect is activated by the workflow (in gamescene_use_scroll.go).
func TestAnTymScrollTriggersWorkflow(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		Build()

	if gs == nil {
		return
	}

	// Give party a Time Stop scroll
	gs.PartyState.Inventory.Scrolls.Set(references.ScrollAnTym, 1)

	// Use scroll - should trigger workflow callback
	result := gs.useScroll(references.ScrollAnTym)
	assert.True(t, result, "Time Stop scroll should succeed")
	assert.True(t, mockCallbacks.StartTimeStopScrollWorkflowCalled, "Should trigger StartTimeStopScrollWorkflow callback")

	// Scroll should be consumed
	assert.Equal(t, 0, int(gs.PartyState.Inventory.Scrolls.Get(references.ScrollAnTym)),
		"Scroll should be consumed")
}

// TestInManiCorpScrollTriggersWorkflow tests that In Mani Corp (Resurrection) scroll triggers the workflow.
// The resurrection scroll does NOT consume at game state level - consumption is conditional on success
// and handled by the workflow (if user selects a valid dead character).
func TestInManiCorpScrollTriggersWorkflow(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain). // Small map - resurrection is non-combat only
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	if gs == nil {
		return
	}

	// Kill a party member so resurrection can be used
	if char := gs.PartyState.GetCharacter(1); char != nil {
		char.SetHp(0)
		char.SetStatus(party_state.Dead)
	}

	// Give party a resurrection scroll
	gs.PartyState.Inventory.Scrolls.Set(references.ScrollInManiCorp, 1)
	initialScrolls := gs.PartyState.Inventory.Scrolls.Get(references.ScrollInManiCorp)

	// Use scroll - should trigger workflow callback
	result := gs.useScroll(references.ScrollInManiCorp)
	assert.True(t, result, "Resurrection scroll should succeed (trigger workflow)")
	assert.True(t, mockCallbacks.StartResurrectionScrollWorkflowCalled,
		"Should trigger StartResurrectionScrollWorkflow callback")

	// Scroll should NOT be consumed at game state level (workflow handles conditional consumption)
	assert.Equal(t, int(initialScrolls), int(gs.PartyState.Inventory.Scrolls.Get(references.ScrollInManiCorp)),
		"Scroll should NOT be consumed at game state level (workflow handles it)")
}

// TestInManiCorpScrollNoDeadMembersStillOffersTargetSelection verifies that
// ineffective targets remain visible warnings instead of blocking Use.
func TestInManiCorpScrollNoDeadMembersStillOffersTargetSelection(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	if gs == nil {
		return
	}

	// Give party a resurrection scroll (no one is dead)
	gs.PartyState.Inventory.Scrolls.Set(references.ScrollInManiCorp, 1)
	initialScrolls := gs.PartyState.Inventory.Scrolls.Get(references.ScrollInManiCorp)

	// Use scroll - should defer commitment until the target workflow confirms.
	result := gs.useScroll(references.ScrollInManiCorp)
	assert.True(t, result)
	assert.True(t, mockCallbacks.StartResurrectionScrollWorkflowCalled)

	// Scroll should NOT be consumed
	assert.Equal(t, int(initialScrolls), int(gs.PartyState.Inventory.Scrolls.Get(references.ScrollInManiCorp)),
		"Scroll should not be consumed before target confirmation")
}

func TestCommitResurrectionScrollUseConsumesConfirmedIneffectiveAndDeadTargets(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	gs.PartyState.Inventory.Scrolls.Set(references.ScrollInManiCorp, 2)

	assert.Equal(t, ResurrectionScrollNoEffect, gs.CommitResurrectionScrollUse(0))
	assert.Equal(t, uint16(1), gs.PartyState.Inventory.Scrolls.Get(references.ScrollInManiCorp))
	target := gs.PartyState.GetCharacter(1)
	target.SetStatus(party_state.Dead)
	target.SetHp(0)
	assert.Equal(t, ResurrectionScrollSuccess, gs.CommitResurrectionScrollUse(1))
	assert.Zero(t, gs.PartyState.Inventory.Scrolls.Get(references.ScrollInManiCorp))
	assert.True(t, target.IsGood())
}

// TestScrollDurationEffectExpiration tests that duration effects expire after the correct number of turns.
func TestScrollDurationEffectExpiration(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		Build()

	if gs == nil {
		return
	}

	// Use a short duration for testing - directly activate with 3 turns
	gs.ActivateNegateMagicEffect(3)
	assert.True(t, gs.HasActiveDurationEffect(DurationEffectNegateMagic))
	assert.Equal(t, 3, gs.ActiveDurationEffect.TurnsRemaining)

	mockCallbacks.Reset()

	// Process 2 turns - should still be active
	gs.processDurationEffects()
	assert.Equal(t, 2, gs.ActiveDurationEffect.TurnsRemaining)
	gs.processDurationEffects()
	assert.Equal(t, 1, gs.ActiveDurationEffect.TurnsRemaining)
	assert.True(t, gs.HasActiveDurationEffect(DurationEffectNegateMagic))

	// Process final turn - should expire and trigger callback
	gs.processDurationEffects()
	assert.False(t, gs.HasActiveDurationEffect(DurationEffectNegateMagic),
		"Effect should have expired after 3 turns")
	assert.True(t, mockCallbacks.SpellExpiredPopupShown,
		"Should have shown spell expired popup")
	assert.Equal(t, "Negate Magic", mockCallbacks.SpellExpiredName,
		"Should have shown correct spell name in popup")
}

// TestTimeStopFreezesBehavior tests that when time stop is active:
// 1. Time doesn't advance (game clock frozen)
// 2. Duration effects still decrement (spell wears off)
// See ref:CONSUME-0003.
func TestTimeStopFreezesBehavior(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		Build()

	if gs == nil {
		return
	}

	// Record initial time
	initialMinute := gs.DateTime.Minute
	initialHour := gs.DateTime.Hour

	// Activate time stop with short duration for testing
	gs.ActivateTimeStopEffect(5)
	assert.True(t, gs.HasActiveDurationEffect(DurationEffectTimeStop), "Should have time stop active")
	assert.Equal(t, 5, gs.ActiveDurationEffect.TurnsRemaining)

	// Finish a turn - time should NOT advance but duration should decrement
	gs.FinishTurn()

	// Time should NOT have advanced
	assert.Equal(t, initialMinute, gs.DateTime.Minute, "Time should not advance when time stop is active")
	assert.Equal(t, initialHour, gs.DateTime.Hour, "Hour should not change when time stop is active")

	// But duration should have decremented
	assert.Equal(t, 4, gs.ActiveDurationEffect.TurnsRemaining, "Duration effect should still decrement")

	// Process several more turns
	gs.FinishTurn()
	gs.FinishTurn()
	gs.FinishTurn()

	// Time still shouldn't have advanced
	assert.Equal(t, initialMinute, gs.DateTime.Minute, "Time should still not advance after multiple turns")

	// Duration should be at 1 now
	assert.Equal(t, 1, gs.ActiveDurationEffect.TurnsRemaining, "Duration should be 1 after 4 turns")

	// Final turn - time stop expires
	gs.FinishTurn()
	assert.False(t, gs.HasActiveDurationEffect(DurationEffectTimeStop), "Time stop should have expired")

	// Now finish another turn - time SHOULD advance
	minuteBefore := gs.DateTime.Minute
	gs.FinishTurn()
	// Time should now advance (minute may wrap, so just check it changed)
	assert.NotEqual(t, minuteBefore, gs.DateTime.Minute, "Time should advance after time stop expires")
}
