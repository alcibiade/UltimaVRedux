// Integration tests for combat flee, victory, and defeat mechanics (Phase 4.3).
//
// These tests validate:
//   - Flee: walking off the 11x11 map edge removes a PC from combat
//   - Victory: all monsters dead → Victory flag set, Escape exits combat
//   - Defeat: all PCs dead/fled with monsters remaining → "BATTLE IS LOST!"
//   - Evacuated: all PCs fled, no monsters → combat ends cleanly
//
// Original references:
//   - battlefield escape in original combat action rules
//   - remaining-allegiance count in original combat action rules
//   - battlefield exit in original command rules
//   - countdown scheduler victory/defeat checks in original combat scheduler
//
// See ref:COMBAT-0011.
package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// ─── helpers ────────────────────────────────────────────────────────────────

// setupCombatForFlee creates a combat scene with one monster present.
// The active PC is moved to the specified edge position so the next
// directional move will walk off the grid.
func setupCombatForFlee(
	t *testing.T,
	pcEdgePos references.Position,
	monsterPos references.Position,
	monsterType references.MonsterType,
) (*GameState, *MockSystemCallbacks, *CombatMonster) {
	t.Helper()

	gs, mock := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		WithRandomSeed(42).
		Build()

	err := gs.TransitionToCombatMap(0, references.BritanniaCombatTerritory, CombatEntryNorth)
	require.NoError(t, err)
	require.NotNil(t, gs.CombatScene)

	// Clear existing monsters
	clearCombatMonstersForTest(gs.CombatScene)

	// Place PC at the edge position
	activeIdx := gs.CombatScene.GetActiveCombatantIndex()
	gs.CombatScene.SetPartyMemberPosition(activeIdx, pcEdgePos)

	// Add one monster
	enemyRefs := *gs.GameReferences.EnemyReferences
	ref := &enemyRefs[monsterType]
	rng := newFixedRNG(1)
	monster := NewCombatMonster(monsterType, ref, monsterPos, rng)
	gs.CombatScene.AddMonsterToQueue(monster)

	mock.Reset()
	return gs, mock, monster
}

// setupCombatNoMonsters creates a combat scene with zero monsters.
// The active PC is moved to the specified edge position.
func setupCombatNoMonsters(
	t *testing.T,
	pcEdgePos references.Position,
) (*GameState, *MockSystemCallbacks) {
	t.Helper()

	gs, mock := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		WithRandomSeed(42).
		Build()

	err := gs.TransitionToCombatMap(0, references.BritanniaCombatTerritory, CombatEntryNorth)
	require.NoError(t, err)
	require.NotNil(t, gs.CombatScene)

	// Clear all monsters
	clearCombatMonstersForTest(gs.CombatScene)

	// Place PC at the edge
	activeIdx := gs.CombatScene.GetActiveCombatantIndex()
	gs.CombatScene.SetPartyMemberPosition(activeIdx, pcEdgePos)

	mock.Reset()
	return gs, mock
}

// ─── Flee Tests ─────────────────────────────────────────────────────────────

// TestCombatFlee_PCWalksOffEdge verifies that a PC at the left edge moving
// left produces CombatMoveFled, marks PartyActive false, and marks the
// queue entry as fled.
func TestCombatFlee_PCWalksOffEdge(t *testing.T) {
	// PC at left edge (x=0), monster in center
	monsterPos := references.Position{X: 5, Y: 5}
	gs, _, _ := setupCombatForFlee(t,
		references.Position{X: 0, Y: 5},
		monsterPos,
		references.MonsterOrc,
	)

	activeIdx := gs.CombatScene.GetActiveCombatantIndex()
	require.True(t, gs.CombatScene.IsPartyMemberActive(activeIdx),
		"PC should be active before fleeing")

	// Move left — off the map edge
	result := gs.MoveCombatCharacter(references.Left)
	assert.Equal(t, CombatMoveFled, result, "Moving off the left edge should return CombatMoveFled")

	// PC should be deactivated
	assert.False(t, gs.CombatScene.IsPartyMemberActive(activeIdx),
		"PC should be inactive after fleeing")

	// Queue entry should be marked fled
	entry := gs.CombatScene.TurnQueue.GetEntry(activeIdx, true)
	require.NotNil(t, entry, "Queue entry should exist")
	assert.Equal(t, turnQueueFled, entry.status, "Queue entry should have fled status")
}

// TestCombatFlee_PublishesEventWithMonstersRemaining prevents flee feedback
// from regressing to a battlefield-obscuring action dialog.
func TestCombatFlee_PublishesEventWithMonstersRemaining(t *testing.T) {
	gs, mock, _ := setupCombatForFlee(t,
		references.Position{X: 0, Y: 5},
		references.Position{X: 5, Y: 5},
		references.MonsterOrc,
	)

	activeIdx := gs.CombatScene.GetActiveCombatantIndex()
	gs.MoveCombatCharacter(references.Left)

	require.NotEmpty(t, mock.CombatEvents)
	event := mock.CombatEvents[len(mock.CombatEvents)-1]
	assert.Equal(t, CombatEventFled, event.Kind)
	assert.True(t, event.Attacker.IsPartyMember)
	assert.Equal(t, activeIdx, event.Attacker.Index)
}

// TestCombatFlee_PublishesEventWithoutMonsters verifies the same durable event
// path when leaving an already-cleared battlefield.
func TestCombatFlee_PublishesEventWithoutMonsters(t *testing.T) {
	gs, mock := setupCombatNoMonsters(t, references.Position{X: 0, Y: 5})

	activeIdx := gs.CombatScene.GetActiveCombatantIndex()
	result := gs.MoveCombatCharacter(references.Left)
	assert.Equal(t, CombatMoveFled, result, "Should still be CombatMoveFled with no monsters")

	require.NotEmpty(t, mock.CombatEvents)
	event := mock.CombatEvents[len(mock.CombatEvents)-1]
	assert.Equal(t, CombatEventFled, event.Kind)
	assert.Equal(t, activeIdx, event.Attacker.Index)
}

// TestCombatFlee_AllEdgeDirections verifies fleeing works from all four edges.
func TestCombatFlee_AllEdgeDirections(t *testing.T) {
	cases := []struct {
		name string
		pos  references.Position
		dir  references.Direction
	}{
		{"LeftEdge", references.Position{X: 0, Y: 5}, references.Left},
		{"RightEdge", references.Position{X: 10, Y: 5}, references.Right},
		{"TopEdge", references.Position{X: 5, Y: 0}, references.Up},
		{"BottomEdge", references.Position{X: 5, Y: 10}, references.Down},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gs, _, _ := setupCombatForFlee(t,
				tc.pos,
				references.Position{X: 5, Y: 5},
				references.MonsterOrc,
			)

			result := gs.MoveCombatCharacter(tc.dir)
			assert.Equal(t, CombatMoveFled, result,
				"%s: moving %s from (%d,%d) should cause flee",
				tc.name, tc.dir.GetDirectionCompassName(), tc.pos.X, tc.pos.Y)
		})
	}
}

// ─── Victory Tests ──────────────────────────────────────────────────────────

// TestCombatVictory_AllMonstersKilled verifies that killing all monsters
// sets the Victory flag and checkCombatEndConditions returns CombatVictory.
func TestCombatVictory_AllMonstersKilled(t *testing.T) {
	gs, _, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)

	// Equip Glass Sword for guaranteed kill
	activeIdx := gs.CombatScene.GetActiveCombatantIndex()
	character := gs.PartyState.GetCharacter(activeIdx)
	character.SetEquippedItem(references.SlotWeapon, references.GlassSword)
	character.SetEquippedItem(references.SlotHelm, references.NoEquipment)
	character.SetEquippedItem(references.SlotShield, references.NoEquipment)

	// Kill the monster
	attackCombatDirection(t, gs, references.Down)
	require.True(t, monster.IsDead(), "Monster should be dead from Glass Sword")

	// Check end conditions — should detect victory
	reason := gs.checkCombatEndConditions()
	assert.Equal(t, CombatVictory, reason, "Should return CombatVictory when all monsters are dead")
	assert.True(t, gs.CombatScene.Victory, "Victory flag should be set")
}

// TestCombatVictory_FriendlySummonDoesNotBlockVictory documents that only
// active hostile monsters determine whether combat has been won.
func TestCombatVictory_FriendlySummonDoesNotBlockVictory(t *testing.T) {
	gs, _, hostile := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)

	partyLoyal := NewCombatMonster(
		references.MonsterOrc,
		hostile.EnemyRef(),
		references.Position{X: 7, Y: 7},
		newFixedRNG(1),
	)
	partyLoyal.SetLoyalty(CombatMonsterLoyaltyParty)
	gs.CombatScene.AddMonsterToQueue(partyLoyal)
	hostile.SetCurrentHP(0)

	reason := gs.checkCombatEndConditions()
	assert.Equal(t, CombatVictory, reason, "a living friendly summon must not block victory")
	assert.True(t, gs.CombatScene.Victory)
	assert.Equal(t, 1, gs.CombatScene.MonsterManager.CountActive(), "friendly remains active for rendering and collision")
	assert.Equal(t, 0, gs.CombatScene.MonsterManager.CountActiveHostile())
}

// TestCombatVictory_SecondCheckReturnsNotEnded verifies that after Victory is
// set, subsequent calls to checkCombatEndConditions return CombatNotEnded
// (idempotent — won't re-trigger victory).
func TestCombatVictory_SecondCheckReturnsNotEnded(t *testing.T) {
	gs, _, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)

	activeIdx := gs.CombatScene.GetActiveCombatantIndex()
	character := gs.PartyState.GetCharacter(activeIdx)
	character.SetEquippedItem(references.SlotWeapon, references.GlassSword)
	character.SetEquippedItem(references.SlotHelm, references.NoEquipment)
	character.SetEquippedItem(references.SlotShield, references.NoEquipment)

	attackCombatDirection(t, gs, references.Down)
	require.True(t, monster.IsDead())

	// First check sets Victory
	reason1 := gs.checkCombatEndConditions()
	assert.Equal(t, CombatVictory, reason1)

	// Second check — already victorious, returns NotEnded
	reason2 := gs.checkCombatEndConditions()
	assert.Equal(t, CombatNotEnded, reason2,
		"Second check should return CombatNotEnded (victory already set)")
}

// TestCombatVictory_EscapeKeyExitsCombat verifies that pressing Escape after
// victory returns true and transitions out of combat.
func TestCombatVictory_EscapeKeyExitsCombat(t *testing.T) {
	gs, _, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)

	activeIdx := gs.CombatScene.GetActiveCombatantIndex()
	character := gs.PartyState.GetCharacter(activeIdx)
	character.SetEquippedItem(references.SlotWeapon, references.GlassSword)
	character.SetEquippedItem(references.SlotHelm, references.NoEquipment)
	character.SetEquippedItem(references.SlotShield, references.NoEquipment)

	attackCombatDirection(t, gs, references.Down)
	require.True(t, monster.IsDead())

	// Set Victory flag (normally done by AdvanceCombatTurn calling checkCombatEndConditions)
	gs.CombatScene.Victory = true

	// Escape should now exit combat
	result := gs.ActionEscapeCombatMap()
	assert.True(t, result, "Escape should succeed after victory")

	// Should have transitioned out of combat
	assert.Nil(t, gs.CombatScene, "CombatScene should be nil after escaping")
}

// ─── Defeat Tests ───────────────────────────────────────────────────────────

// TestCombatDefeat_AllPCsFledWithMonsters verifies that when all PCs flee
// while monsters remain, checkCombatEndConditions returns CombatDefeat
// and transitions out of combat with "BATTLE IS LOST!" message.
func TestCombatDefeat_AllPCsFledWithMonsters(t *testing.T) {
	gs, mock, _ := setupCombatForFlee(t,
		references.Position{X: 0, Y: 5},
		references.Position{X: 5, Y: 5},
		references.MonsterOrc,
	)

	// Deactivate all PCs except the first active one (already at edge)
	activeIdx := gs.CombatScene.GetActiveCombatantIndex()
	for i := range 6 {
		if i != activeIdx {
			gs.CombatScene.PartyActive[i] = false
			gs.CombatScene.TurnQueue.MarkFled(i, true)
		}
	}

	// Flee the last remaining PC
	gs.MoveCombatCharacter(references.Left)

	mock.Reset() // Clear flee messages, we want to check defeat message separately

	// Check end conditions — should detect defeat
	reason := gs.checkCombatEndConditions()
	assert.Equal(t, CombatDefeat, reason,
		"Should return CombatDefeat when all PCs fled with monsters remaining")

	// Should show "BATTLE IS LOST!"
	assert.True(t, mock.MessagesContain("BATTLE IS LOST!"),
		"Should show 'BATTLE IS LOST!' on defeat, got: %v", mock.Messages)

	// Should have transitioned out of combat
	assert.Nil(t, gs.CombatScene, "CombatScene should be nil after defeat")
}

// TestCombatEvacuated_AllPCsFledNoMonsters verifies that when all PCs flee
// and no monsters are present, checkCombatEndConditions returns CombatEvacuated
// and transitions out of combat (no defeat message).
func TestCombatEvacuated_AllPCsFledNoMonsters(t *testing.T) {
	gs, mock := setupCombatNoMonsters(t, references.Position{X: 0, Y: 5})

	// Deactivate all PCs except the active one
	activeIdx := gs.CombatScene.GetActiveCombatantIndex()
	for i := range 6 {
		if i != activeIdx {
			gs.CombatScene.PartyActive[i] = false
			gs.CombatScene.TurnQueue.MarkFled(i, true)
		}
	}

	// Flee the last PC
	gs.MoveCombatCharacter(references.Left)

	mock.Reset()

	// Check end conditions — should detect evacuated
	reason := gs.checkCombatEndConditions()
	assert.Equal(t, CombatEvacuated, reason,
		"Should return CombatEvacuated when all PCs fled with no monsters")

	// Should NOT show "BATTLE IS LOST!" — this is evacuation, not defeat
	assert.False(t, mock.MessagesContain("BATTLE IS LOST!"),
		"Should NOT show defeat message on evacuation, got: %v", mock.Messages)

	// Should have transitioned out of combat
	assert.Nil(t, gs.CombatScene, "CombatScene should be nil after evacuation")
}

// ─── Escape Command Tests ───────────────────────────────────────────────────

// TestCombatEscape_NotYetWhenNoVictory verifies that invoking Escape before
// victory shows "Not yet!" and returns false.
func TestCombatEscape_NotYetWhenNoVictory(t *testing.T) {
	gs, mock, _ := setupCombatForFlee(t,
		references.Position{X: 5, Y: 5},
		references.Position{X: 3, Y: 3},
		references.MonsterOrc,
	)

	// Victory is false (default)
	require.False(t, gs.CombatScene.Victory, "Victory should be false initially")

	result := gs.ActionEscapeCombatMap()
	assert.False(t, result, "Escape should fail when no victory")
	assert.True(t, mock.MessagesContain("Not yet!"),
		"Should show 'Not yet!' when trying to escape without victory, got: %v", mock.Messages)

	// Combat should still be active
	assert.NotNil(t, gs.CombatScene, "CombatScene should still be active")
}

// ─── Movement Regression Tests ──────────────────────────────────────────────

// TestCombatMovement_BlockedNotFled verifies that blocked movement within the
// map returns CombatMoveBlocked (not CombatMoveFled) — regression check.
func TestCombatMovement_BlockedNotFled(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		WithRandomSeed(42).
		Build()

	err := gs.TransitionToCombatMap(0, references.BritanniaCombatTerritory, CombatEntryNorth)
	require.NoError(t, err)

	// Place PC in the center — any movement should be within bounds
	activeIdx := gs.CombatScene.GetActiveCombatantIndex()
	gs.CombatScene.SetPartyMemberPosition(activeIdx, references.Position{X: 5, Y: 5})

	// Place another active PC in the way (one tile to the right)
	// Find another active PC to block with
	blockerIdx := -1
	for i := range 6 {
		if i != activeIdx && gs.CombatScene.IsPartyMemberActive(i) {
			blockerIdx = i
			break
		}
	}

	if blockerIdx >= 0 {
		gs.CombatScene.SetPartyMemberPosition(blockerIdx, references.Position{X: 6, Y: 5})

		// Try to move into the blocker
		result := gs.MoveCombatCharacter(references.Right)
		assert.Equal(t, CombatMoveBlocked, result,
			"Moving into another PC should return CombatMoveBlocked, not CombatMoveFled")
	} else {
		t.Log("Only one active PC — skipping blocker collision test")
	}

	// Moving into the center should always be valid (open tile on combat map 0)
	gs.CombatScene.SetPartyMemberPosition(activeIdx, references.Position{X: 5, Y: 5})
	result := gs.MoveCombatCharacter(references.Down)

	// Should be either moved or blocked (depending on terrain), but NOT fled
	assert.NotEqual(t, CombatMoveFled, result,
		"Moving within map bounds should never return CombatMoveFled")
}
