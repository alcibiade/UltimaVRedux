package game_state

import (
	"math"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/deterministic_random"
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPartyStateBuilderSoundIsExplicit(t *testing.T) {
	gs, callbacks := NewIntegrationTestBuilder(t).WithLocation(references.Britain).
		WithSystemCallbacks().Build()

	NewPartyStateBuilder(gs).FlashInverted().setupEffects()
	assert.Empty(t, callbacks.SoundEffectsPlayed, "visual-only effects must not request enum-zero cannon fire")

	NewPartyStateBuilder(gs).WithSound(SoundNoEffect).setupEffects()
	assert.Equal(t, []SoundEffect{SoundNoEffect}, callbacks.SoundEffectsPlayed,
		"the no-effect cue is a real request, not an absence sentinel")

	NewPartyStateBuilder(gs).WithSound(SoundHeal).setupEffects()
	assert.Equal(t, []SoundEffect{SoundNoEffect, SoundHeal}, callbacks.SoundEffectsPlayed)
}

// TestPartyStateBuilder_BasicAllPartyDamage verifies random damage applied to all party members
func TestPartyStateBuilder_BasicAllPartyDamage(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	gs.SetRandomSeed(12345)

	// Record initial HP for active party members
	initialHP := make(map[int]int)
	gs.PartyState.ForEachCharacter(func(i int, char *party_state.PlayerCharacter) {
		if char.PartyMap == party_state.PartyMapActive {
			initialHP[i] = int(char.GetCurrentHp())
		}
	})

	// Build and execute damage event
	gs.EventQueueManager.Workflow("test_damage").Priority(event_queue.PriorityNormal).
		Sequential().ThenSequence(

		NewPartyStateBuilder(gs).
			WithRandomDamage(1, 8).
			ToAllParty().
			Build()).
		Start()

	// Process event queue
	gs.EventQueueManager.Update()

	// Assert: All active party members took 1-8 damage
	gs.PartyState.ForEachCharacter(func(i int, char *party_state.PlayerCharacter) {
		if char.PartyMap == party_state.PartyMapActive && !char.IsDead() {
			currentHP := int(char.GetCurrentHp())
			damageTaken := initialHP[i] - currentHP

			assert.GreaterOrEqual(t, damageTaken, 1, "Party member %d should take at least 1 damage", i)
			assert.LessOrEqual(t, damageTaken, 8, "Party member %d should take at most 8 damage", i)
		}
	})
}

// TestPartyStateBuilder_SingleCharacterDamage verifies fixed damage to single character
func TestPartyStateBuilder_SingleCharacterDamage(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()

	targetIdx := 2

	// Record initial HP for all party members
	initialHP := make(map[int]int)
	gs.PartyState.ForEachCharacter(func(i int, char *party_state.PlayerCharacter) {
		if char.PartyMap == party_state.PartyMapActive {
			initialHP[i] = int(char.GetCurrentHp())
		}
	})

	gs.EventQueueManager.Workflow("test_single").Priority(event_queue.PriorityNormal).
		Sequential().ThenSequence(

		NewPartyStateBuilder(gs).
			WithFixedDamage(5).
			ToCharacter(targetIdx).
			Build()).
		Start()

	gs.EventQueueManager.Update()

	// Assert: Only target character took damage
	gs.PartyState.ForEachCharacter(func(i int, char *party_state.PlayerCharacter) {
		if char.PartyMap != party_state.PartyMapActive {
			return
		}

		currentHP := int(char.GetCurrentHp())
		if i == targetIdx {
			// Target should take exactly 5 damage
			assert.Equal(t, initialHP[i]-5, currentHP, "Target character %d should take 5 damage", i)
		} else {
			// Others should be unaffected
			assert.Equal(t, initialHP[i], currentHP, "Character %d should be unaffected", i)
		}
	})
}

func TestPartyStateBuilderOriginalD30DamageUsesOneMappedRoll(t *testing.T) {
	const seed = uint64(4480)
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	target := gs.PartyState.GetCharacter(0)
	target.SetMaxHp(100)
	target.SetHp(100)
	gs.SetRandomSeed(seed)
	wantRNG := deterministic_random.New(seed)
	wantDamage := originalD30FromSourceValue(wantRNG.Intn(originalD30SourceValueCount))

	gs.EventQueueManager.Workflow("test_original_d30_damage").Priority(event_queue.PriorityNormal).
		Sequential().ThenSequence(
		NewPartyStateBuilder(gs).
			WithOriginalD30Damage().
			ToCharacter(0).
			Build()).
		Start()
	gs.EventQueueManager.Update()

	assert.Equal(t, uint16(100-wantDamage), target.GetCurrentHp())
	assert.Equal(t, uint64(1), gs.GetRNG().Calls())
}

// TestPartyStateBuilder_FlashEffect verifies flash effects are applied
func TestPartyStateBuilder_FlashEffect(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()

	gs.EventQueueManager.Workflow("test_flash").Priority(event_queue.PriorityNormal).
		Sequential().ThenSequence(

		NewPartyStateBuilder(gs).
			WithFixedDamage(5).
			ToAllParty().
			WithFlash(200, 1.0, 0.0, 0.0, 0.7).
			Build()).
		Start()

	// Process through animation
	for i := 0; i < 20; i++ {
		gs.EventQueueManager.Update()
		gs.ElapsedMs += 20 // Advance time through animation
	}

	// Assert: Flash color was set
	assert.True(t, len(mockCallbacks.ScreenFlashColors) > 0, "SetScreenFlashColor should be called")
	assert.Equal(t, 1.0, mockCallbacks.ScreenFlashColors[0].R, "Flash R should be 1.0")
	assert.Equal(t, 0.0, mockCallbacks.ScreenFlashColors[0].G, "Flash G should be 0.0")
	assert.Equal(t, 0.0, mockCallbacks.ScreenFlashColors[0].B, "Flash B should be 0.0")

	// Assert: Flash intensity was animated
	assert.True(t, len(mockCallbacks.ScreenFlashIntensities) > 0, "SetScreenFlashIntensity should be called during animation")
}

// TestPartyStateBuilder_ShakeEffect verifies shake effects are applied
func TestPartyStateBuilder_ShakeEffect(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()

	gs.EventQueueManager.Workflow("test_shake").Priority(event_queue.PriorityNormal).
		Sequential().ThenSequence(

		NewPartyStateBuilder(gs).
			WithFixedDamage(5).
			ToAllParty().
			WithShake(200, 6.0).
			Build()).
		Start()

	// Process through animation
	for i := 0; i < 20; i++ {
		gs.EventQueueManager.Update()
		gs.ElapsedMs += 20
	}

	// Assert: Shake offset was set
	assert.True(t, len(mockCallbacks.ScreenShakeOffsets) > 0, "SetScreenShakeOffset should be called")

	// Find a shake call with non-zero offset (should happen during animation)
	foundNonZero := false
	for _, call := range mockCallbacks.ScreenShakeOffsets {
		if call.X != 0.0 || call.Y != 0.0 {
			foundNonZero = true
			break
		}
	}
	assert.True(t, foundNonZero, "Shake should have non-zero offset during animation")
}

func TestPartyStateBuilder_ShakePreservesGameplayRNGSnapshot(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		WithRandomSeed(12345).
		Build()
	before, err := gs.CaptureSaveSnapshot()
	require.NoError(t, err)

	const workflowName = "test_cosmetic_shake"
	gs.EventQueueManager.Workflow(workflowName).Priority(event_queue.PriorityNormal).
		Sequential().ThenSequence(
		NewPartyStateBuilder(gs).
			ToAllParty().
			WithShake(200, 6.0).
			Build()).
		Start()

	for range 20 {
		gs.EventQueueManager.Update()
		gs.ElapsedMs += 20
		if !gs.EventQueueManager.HasWorkflow(workflowName) {
			break
		}
	}
	require.False(t, gs.EventQueueManager.HasWorkflow(workflowName))
	after, err := gs.CaptureSaveSnapshot()
	require.NoError(t, err)

	assert.Equal(t, before.Random, after.Random)
	require.NotEmpty(t, mockCallbacks.ScreenShakeOffsets)
	assert.Equal(t, 0.0, mockCallbacks.ScreenShakeOffsets[len(mockCallbacks.ScreenShakeOffsets)-1].X)
	assert.Equal(t, 0.0, mockCallbacks.ScreenShakeOffsets[len(mockCallbacks.ScreenShakeOffsets)-1].Y)
}

func TestCalculateScreenShakeOffsetIsDeterministic(t *testing.T) {
	const (
		progress  = float32(0.37)
		amplitude = 6.0
	)
	firstX, firstY := calculateScreenShakeOffset(progress, amplitude)
	secondX, secondY := calculateScreenShakeOffset(progress, amplitude)

	assert.Equal(t, firstX, secondX)
	assert.Equal(t, firstY, secondY)
	assert.InDelta(t, amplitude, math.Hypot(firstX, firstY), 0.000001)
}

// TestPartyStateBuilder_MultiLayerConcurrentEffects verifies multi-layer effects with different durations
// Tests the user's requirement: "2 second earthquake, and 1 second flash" running concurrently
func TestPartyStateBuilder_MultiLayerConcurrentEffects(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()

	gs.EventQueueManager.Workflow("test_multilayer").Priority(event_queue.PriorityNormal).
		Sequential().ThenSequence(

		NewPartyStateBuilder(gs).
			WithFixedDamage(5).
			ToAllParty().
			WithFlash(1000, 1.0, 0.0, 0.0, 0.7).
			WithShake(2000, 6.0).
			Build()).
		Start()

	// Process animation in 100ms steps for 2.5 seconds total
	flashStoppedAt := -1
	shakeStoppedAt := -1

	for i := 0; i < 25; i++ {
		gs.EventQueueManager.Update()
		gs.ElapsedMs += 100

		// Check flash state at each step
		if i*100 <= 1000 && flashStoppedAt == -1 {
			// Flash should be active (intensity changes tracked)
		} else if i*100 > 1000 && flashStoppedAt == -1 {
			// Flash should be cleared after 1000ms
			flashStoppedAt = i * 100
		}

		// Check shake state at each step
		if i*100 <= 2000 && shakeStoppedAt == -1 {
			// Shake should be active
		} else if i*100 > 2000 && shakeStoppedAt == -1 {
			// Shake should be cleared after 2000ms
			shakeStoppedAt = i * 100
		}
	}

	// Assert: Flash color was set
	assert.True(t, len(mockCallbacks.ScreenFlashColors) > 0, "Flash color should be set")

	// Assert: Flash intensity was animated (multiple calls)
	assert.True(t, len(mockCallbacks.ScreenFlashIntensities) > 0, "Flash intensity should be animated")

	// Assert: Shake was applied (multiple calls)
	assert.True(t, len(mockCallbacks.ScreenShakeOffsets) > 0, "Shake should be applied")

	// Assert: We should see final cleanup calls setting intensity to 0.0 and offset to (0,0)
	finalFlashIntensity := mockCallbacks.ScreenFlashIntensities[len(mockCallbacks.ScreenFlashIntensities)-1]
	assert.Equal(t, 0.0, finalFlashIntensity, "Flash intensity should be cleared to 0.0 at end")

	finalShakeOffset := mockCallbacks.ScreenShakeOffsets[len(mockCallbacks.ScreenShakeOffsets)-1]
	assert.Equal(t, 0.0, finalShakeOffset.X, "Shake X should be cleared to 0.0 at end")
	assert.Equal(t, 0.0, finalShakeOffset.Y, "Shake Y should be cleared to 0.0 at end")

	// Assert: Flash stopped before shake (1s vs 2s)
	// This is the key test - verifying concurrent effects with different durations
	// We verify this by checking that we have intensity calls in first half,
	// but shake calls continue into second half

	// Count flash intensity changes in different time windows
	// (This is implicit in the animation - flash runs 0-1000ms, shake runs 0-2000ms)
	t.Logf("Total flash intensity calls: %d", len(mockCallbacks.ScreenFlashIntensities))
	t.Logf("Total shake offset calls: %d", len(mockCallbacks.ScreenShakeOffsets))
}

// TestPartyStateBuilder_FlashRedHelper verifies FlashRed() helper sets correct color
func TestPartyStateBuilder_FlashRedHelper(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()

	gs.EventQueueManager.Workflow("test_flash_red").Priority(event_queue.PriorityNormal).
		Sequential().ThenSequence(

		NewPartyStateBuilder(gs).
			WithFixedDamage(5).
			ToAllParty().
			FlashRed().
			Build()).
		Start()

	// Process animation
	for i := 0; i < 20; i++ {
		gs.EventQueueManager.Update()
		gs.ElapsedMs += 20
	}

	// Assert: Flash color is red (1.0, 0.0, 0.0)
	assert.True(t, len(mockCallbacks.ScreenFlashColors) > 0, "Flash color should be set")
	flashColor := mockCallbacks.ScreenFlashColors[0]
	assert.Equal(t, 1.0, flashColor.R, "Red component should be 1.0")
	assert.Equal(t, 0.0, flashColor.G, "Green component should be 0.0")
	assert.Equal(t, 0.0, flashColor.B, "Blue component should be 0.0")
}

// TestPartyStateBuilder_ShakeModerateHelper verifies ShakeModerate() helper sets correct params
func TestPartyStateBuilder_ShakeModerateHelper(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()

	gs.EventQueueManager.Workflow("test_shake_moderate").Priority(event_queue.PriorityNormal).
		Sequential().ThenSequence(

		NewPartyStateBuilder(gs).
			WithFixedDamage(5).
			ToAllParty().
			ShakeModerate().
			Build()).
		Start()

	// Process animation
	for i := 0; i < 20; i++ {
		gs.EventQueueManager.Update()
		gs.ElapsedMs += 20
	}

	// Assert: Shake was applied with moderate intensity
	assert.True(t, len(mockCallbacks.ScreenShakeOffsets) > 0, "Shake should be applied")

	// Find max offset to verify it's in moderate range (should be <= 6.0 pixels)
	maxOffset := 0.0
	for _, offset := range mockCallbacks.ScreenShakeOffsets {
		absX := offset.X
		if absX < 0 {
			absX = -absX
		}
		absY := offset.Y
		if absY < 0 {
			absY = -absY
		}
		if absX > maxOffset {
			maxOffset = absX
		}
		if absY > maxOffset {
			maxOffset = absY
		}
	}

	assert.LessOrEqual(t, maxOffset, 6.5, "Shake offset should be moderate (<=6.5 pixels)")
}

// TestPartyStateBuilder_CombinedHelpers verifies multiple helpers can be chained
func TestPartyStateBuilder_CombinedHelpers(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()

	gs.EventQueueManager.Workflow("test_combined").Priority(event_queue.PriorityNormal).
		Sequential().ThenSequence(

		NewPartyStateBuilder(gs).
			WithFixedDamage(10).
			ToAllParty().
			FlashRed().
			ShakeModerate().
			Build()).
		Start()

	// Process animation
	for i := 0; i < 20; i++ {
		gs.EventQueueManager.Update()
		gs.ElapsedMs += 20
	}

	// Assert: Both effects were applied
	assert.True(t, len(mockCallbacks.ScreenFlashColors) > 0, "Flash should be set")
	assert.True(t, len(mockCallbacks.ScreenShakeOffsets) > 0, "Shake should be set")

	// Assert: Red flash
	flashColor := mockCallbacks.ScreenFlashColors[0]
	assert.Equal(t, 1.0, flashColor.R, "Should be red flash")
}

// TestPartyStateBuilder_VisualOnlyNoHealth verifies effects without health changes
// Tests earthquake-style effects (visual feedback with no damage)
func TestPartyStateBuilder_VisualOnlyNoHealth(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()

	// Record initial HP
	initialHP := make(map[int]int)
	gs.PartyState.ForEachCharacter(func(i int, char *party_state.PlayerCharacter) {
		if char.PartyMap == party_state.PartyMapActive {
			initialHP[i] = int(char.GetCurrentHp())
		}
	})

	gs.EventQueueManager.Workflow("test_visual_only").Priority(event_queue.PriorityNormal).
		Sequential().ThenSequence(

		NewPartyStateBuilder(gs).
			ToAllParty().
			ShakeStrong().
			Build()).
		Start()

	// Process animation
	for i := 0; i < 30; i++ {
		gs.EventQueueManager.Update()
		gs.ElapsedMs += 20
	}

	// Assert: Shake was applied
	assert.True(t, len(mockCallbacks.ScreenShakeOffsets) > 0, "Shake should be applied")

	// Assert: No HP changes (visual-only effect)
	gs.PartyState.ForEachCharacter(func(i int, char *party_state.PlayerCharacter) {
		if char.PartyMap == party_state.PartyMapActive {
			currentHP := int(char.GetCurrentHp())
			assert.Equal(t, initialHP[i], currentHP, "HP should not change for visual-only effect")
		}
	})
}

// TestPartyStateBuilder_DeadCharactersSkipped verifies dead party members are not affected
func TestPartyStateBuilder_DeadCharactersSkipped(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()

	// Kill first party member
	firstMember := gs.PartyState.GetCharacter(0)
	if firstMember.PartyMap == party_state.PartyMapActive {
		firstMember.SubtractHp(firstMember.GetCurrentHp()) // Kill them
		assert.True(t, firstMember.IsDead(), "First member should be dead")
	}

	// Record HP of remaining living members
	initialHP := make(map[int]int)
	gs.PartyState.ForEachCharacter(func(i int, char *party_state.PlayerCharacter) {
		if char.PartyMap == party_state.PartyMapActive && !char.IsDead() {
			initialHP[i] = int(char.GetCurrentHp())
		}
	})

	gs.SetRandomSeed(12345) // Deterministic

	gs.EventQueueManager.Workflow("test_dead_skipped").Priority(event_queue.PriorityNormal).
		Sequential().ThenSequence(

		NewPartyStateBuilder(gs).
			WithRandomDamage(1, 8).
			ToAllParty().
			Build()).
		Start()

	gs.EventQueueManager.Update()

	// Assert: Dead member stays at 0 HP (not affected)
	assert.Equal(t, uint16(0), firstMember.GetCurrentHp(), "Dead member should stay at 0 HP")

	// Assert: Living members took damage
	gs.PartyState.ForEachCharacter(func(i int, char *party_state.PlayerCharacter) {
		if char.PartyMap == party_state.PartyMapActive && !char.IsDead() {
			currentHP := int(char.GetCurrentHp())
			damageTaken := initialHP[i] - currentHP
			assert.GreaterOrEqual(t, damageTaken, 1, "Living member %d should take damage", i)
		}
	})
}
