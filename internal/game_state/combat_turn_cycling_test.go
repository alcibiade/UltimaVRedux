// Tests for combat turn cycling, solo mode, and movement collision.
//
// These tests guard against regressions in:
//   - Dex-based turn ordering via CombatTurnQueue (replaces round-robin)
//   - Solo mode: only the solo character receives input (others auto-pass)
//   - Monster collision: party members cannot walk onto monster-occupied tiles
//   - Turn cycling wraps around correctly based on dex priority
package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// ─── Turn Cycling Tests ─────────────────────────────────────────────────────

// TestCombatTurnCycling_NormalMode_AdvancesThroughActiveMembers verifies that
// AdvanceCombatTurn in normal (non-solo) mode uses the dex-based turn queue
// and cycles through all active party members.
// With the dex-based queue, turn order is determined by dexterity values,
// not party slot order. This test verifies that all active PCs get turns
// and that the queue advances correctly.
func TestCombatTurnCycling_NormalMode_AdvancesThroughActiveMembers(t *testing.T) {
	gs, _ := newBritainCombatTestState(t, 42)
	require.NotNil(t, gs.CombatScene)

	// Solo mode should NOT be active by default
	assert.False(t, gs.CombatScene.IsSoloMoveActive(),
		"solo mode should not be active after combat initialization")

	// Count active party members
	var activeIndices []int
	for i := range 6 {
		if gs.CombatScene.IsPartyMemberActive(i) {
			activeIndices = append(activeIndices, i)
		}
	}
	require.GreaterOrEqual(t, len(activeIndices), 2,
		"need at least 2 active party members for meaningful turn cycling test")

	// First turn should be the first active member (set during NewCombatState)
	assert.Equal(t, activeIndices[0], gs.CombatScene.GetCurrentTurnMemberIndex(),
		"initial turn should be first active party member")

	// Advance multiple times and collect which PCs get turns.
	// With the dex-based queue, the order depends on dex values — we don't
	// assert a specific order, but we DO assert:
	// 1. Every advance produces a valid active PC index
	// 2. The activePlayerIndex stays in sync
	turnsSeen := make(map[int]int) // entityIndex -> count

	for step := 0; step < len(activeIndices)*3; step++ {
		gs.AdvanceCombatTurn()
		turnIdx := gs.CombatScene.GetCurrentTurnMemberIndex()
		assert.True(t, gs.CombatScene.IsPartyMemberActive(turnIdx),
			"step %d: turn index %d should be an active party member", step, turnIdx)
		assert.Equal(t, turnIdx, gs.GetActivePlayerIndex(),
			"step %d: activePlayerIndex should match CurrentTurnIndex", step, turnIdx)
		turnsSeen[turnIdx]++
	}

	// All active PCs should have gotten at least one turn
	for _, idx := range activeIndices {
		assert.Greater(t, turnsSeen[idx], 0,
			"active party member %d should have gotten at least one turn", idx)
	}
}

// TestCombatTurnCycling_NormalMode_SyncsActivePlayerIndex verifies that
// AdvanceCombatTurn updates activePlayerIndex to match CurrentTurnIndex,
// keeping UI character highlighting in sync with whose turn it is.
func TestCombatTurnCycling_NormalMode_SyncsActivePlayerIndex(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		WithRandomSeed(42).
		Build()
	require.True(t, gs.SetActivePlayer(1))

	err := gs.TransitionToCombatMap(0, references.BritanniaCombatTerritory, CombatEntryNorth)
	require.NoError(t, err)

	assert.Equal(t, -1, gs.GetActivePlayerIndex(),
		"overworld selection must not become combat selection")
	assert.False(t, gs.CombatScene.IsSoloMoveActive(),
		"combat must begin in round-robin mode")

	// The first authoritative PC turn owns its ordinary current-actor highlight.
	gs.AdvanceCombatTurn()
	newTurn := gs.CombatScene.GetCurrentTurnMemberIndex()
	assert.Equal(t, newTurn, gs.GetActivePlayerIndex(),
		"after advancing turn, activePlayerIndex should match new CurrentTurnIndex")
}

// TestAdvanceCombatTurn_PCTurnPromptUsesMessageOutput verifies that the original
// combat turn text stays in the scrolling transcript without opening an action
// feedback popup. Popups overlap during movement because every move ends a turn.
func TestAdvanceCombatTurn_PCTurnPromptUsesMessageOutput(t *testing.T) {
	gs, mockCallbacks := newBritainCombatTestState(t, 42)

	actionFeedbackCalls := 0
	gs.SystemCallbacks.UI.ShowActionFeedback = func(_ string, _ bool) {
		actionFeedbackCalls++
	}
	mockCallbacks.Reset()

	gs.AdvanceCombatTurn()

	activePCIndex := gs.CombatScene.GetCurrentTurnMemberIndex()
	require.Len(t, mockCallbacks.Messages, 1)
	assert.Equal(t, gs.combatPCTurnPrompt(activePCIndex), mockCallbacks.Messages[0])
	assert.Equal(t, []string{""}, mockCallbacks.CommandPrompts,
		"combat turn header must be followed by the original command arrow")
	assert.Zero(t, actionFeedbackCalls, "combat turn prompt must not open a popup")
}

// ─── Solo Mode Tests ────────────────────────────────────────────────────────

// TestCombatSoloMode_OnlySoloCharacterGetsTurn verifies that in solo mode,
// AdvanceCombatTurn advances through the dex queue but auto-passes all PCs
// except the solo-selected character. The solo character always ends up as
// the CurrentTurnIndex (matching original combat scheduler active_player behavior).
// See ref:COMBAT-0002.
func TestCombatSoloMode_OnlySoloCharacterGetsTurn(t *testing.T) {
	gs, _ := newBritainCombatTestState(t, 42)

	// Find an active party member to use as solo character
	soloIdx := -1
	for i := range 6 {
		if gs.CombatScene.IsPartyMemberActive(i) {
			soloIdx = i
		}
	}
	require.GreaterOrEqual(t, soloIdx, 0, "need at least one active party member")

	// Enable solo mode for this character
	gs.CombatScene.SetSoloMoveCharacter(soloIdx)
	assert.True(t, gs.CombatScene.IsSoloMoveActive())

	// AdvanceCombatTurn should always land on the solo character
	for i := range 5 {
		gs.AdvanceCombatTurn()
		assert.Equal(t, soloIdx, gs.CombatScene.GetCurrentTurnMemberIndex(),
			"advance %d: solo mode should always give turn to solo character %d", i, soloIdx)
	}
}

func TestCombatSoloModePresentedTurnsFastForwardSkippedPC(t *testing.T) {
	gs, mockCallbacks := newBritainCombatTestState(t, 42)

	var activeMembers []int
	for i := range party_state.MaxActiveParty {
		if gs.CombatScene.IsPartyMemberActive(i) {
			activeMembers = append(activeMembers, i)
		}
	}
	require.GreaterOrEqual(t, len(activeMembers), 2)
	skippedPC := activeMembers[0]
	selectedPC := activeMembers[1]
	gs.CombatScene.SetSoloMoveCharacter(selectedPC)

	for _, entry := range gs.CombatScene.TurnQueue.entries {
		entry.turnCountdown = turnCountdownBase
		if entry.Combatant.IsPC() && (entry.EntityIndex == skippedPC || entry.EntityIndex == selectedPC) {
			entry.turnCountdown = 1
		}
	}
	gs.CombatScene.TurnQueue.currentIndex = 0

	gs.CombatScene.presentTurns = true
	require.True(t, gs.ActivateProtectionEffect(3))
	actionCountBefore := gs.CombatScene.TurnQueue.actionCount
	pcsBefore, monstersBefore := gs.CombatScene.TurnQueue.CountRemaining()
	mockCallbacks.Reset()

	gs.AdvanceCombatTurn()

	assert.False(t, gs.CombatScene.HasScheduledCombatStep())
	assert.Equal(t, CombatTurnIdentity{IsPartyMember: true, EntityIndex: selectedPC},
		gs.CombatScene.GetCurrentTurnIdentity())
	assert.Equal(t, actionCountBefore+2, gs.CombatScene.TurnQueue.actionCount)
	assert.Equal(t, 2, gs.ActiveDurationEffect.TurnsRemaining,
		"the skipped PC must still resolve authoritative end-action effects")
	pcsAfter, monstersAfter := gs.CombatScene.TurnQueue.CountRemaining()
	assert.Equal(t, pcsBefore, pcsAfter)
	assert.Equal(t, monstersBefore, monstersAfter)
	assert.Equal(t, []string{gs.combatPCTurnPrompt(selectedPC)}, mockCallbacks.Messages)
	assert.Equal(t, []string{""}, mockCallbacks.CommandPrompts)
}

func TestCombatSoloModeFastForwardStopsForHostileTurn(t *testing.T) {
	gs, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	activeMembers := gs.ActiveCombatPartyMembers()
	require.GreaterOrEqual(t, len(activeMembers), 2)
	skippedPC, selectedPC := activeMembers[0], activeMembers[1]
	gs.CombatScene.SetSoloMoveCharacter(selectedPC)
	monsterIndex := NoActiveMonster
	for _, entry := range gs.CombatScene.TurnQueue.entries {
		entry.turnCountdown = turnCountdownBase
		switch {
		case entry.Combatant.IsPC() && entry.EntityIndex == skippedPC:
			entry.turnCountdown = 1
		case entry.Combatant.IsPC() && entry.EntityIndex == selectedPC:
			entry.turnCountdown = 2
		case !entry.Combatant.IsPC() && monsterIndex == NoActiveMonster:
			monsterIndex = entry.EntityIndex
			entry.turnCountdown = 1
		}
	}
	require.NotEqual(t, NoActiveMonster, monsterIndex)
	gs.CombatScene.TurnQueue.currentIndex = 0
	gs.CombatScene.presentTurns = true

	gs.AdvanceCombatTurn()
	step, scheduled := gs.ReviewScheduledCombatStep()

	require.True(t, scheduled)
	assert.Equal(t, CombatPresentationMonsterTurn, step.Kind)
	assert.Equal(t, CombatTurnIdentity{EntityIndex: monsterIndex}, step.Actor)
	assert.NotEqual(t, selectedPC, gs.CombatScene.GetCurrentTurnIdentity().EntityIndex,
		"active-player selection must not leapfrog a hostile turn")
}

func TestCombatSoloModeFastForwardsSkippedPCAfterVictory(t *testing.T) {
	gs, _ := newBritainCombatTestState(t, 42)
	activeMembers := gs.ActiveCombatPartyMembers()
	require.GreaterOrEqual(t, len(activeMembers), 2)
	skippedPC, selectedPC := activeMembers[0], activeMembers[1]
	gs.CombatScene.MonsterManager.Clear()
	gs.CombatScene.TurnQueue.RemoveAllMonsters()
	gs.CombatScene.Victory = true
	gs.CombatScene.SetSoloMoveCharacter(selectedPC)
	for _, entry := range gs.CombatScene.TurnQueue.entries {
		entry.turnCountdown = turnCountdownBase
		if entry.Combatant.IsPC() && (entry.EntityIndex == skippedPC || entry.EntityIndex == selectedPC) {
			entry.turnCountdown = 1
		}
	}
	gs.CombatScene.TurnQueue.currentIndex = 0
	gs.CombatScene.presentTurns = true
	pcsBefore, monstersBefore := gs.CombatScene.TurnQueue.CountRemaining()

	gs.AdvanceCombatTurn()

	assert.False(t, gs.CombatScene.HasScheduledCombatStep())
	assert.Equal(t, CombatTurnIdentity{IsPartyMember: true, EntityIndex: selectedPC},
		gs.CombatScene.GetCurrentTurnIdentity())
	assert.True(t, gs.CombatScene.Victory)
	pcsAfter, monstersAfter := gs.CombatScene.TurnQueue.CountRemaining()
	assert.Equal(t, pcsBefore, pcsAfter)
	assert.Equal(t, monstersBefore, monstersAfter)
}

// TestCombatSoloMode_CannotMoveSelectedCharacterOutOfTurn prevents number-key
// selection from granting action authority before the DEX queue reaches that PC.
func TestCombatSoloMode_CannotMoveSelectedCharacterOutOfTurn(t *testing.T) {
	gs, mock := newBritainCombatTestState(t, 42)

	// Find two active party members
	var activeMembers []int
	for i := range 6 {
		if gs.CombatScene.IsPartyMemberActive(i) {
			activeMembers = append(activeMembers, i)
		}
	}
	require.GreaterOrEqual(t, len(activeMembers), 2,
		"need at least 2 active party members for solo mode test")

	// CurrentTurnIndex is on the first active member
	turnMember := activeMembers[0]
	assert.Equal(t, turnMember, gs.CombatScene.GetCurrentTurnMemberIndex())

	// Enable solo mode for a DIFFERENT member
	soloMember := activeMembers[1]
	gs.CombatScene.SetSoloMoveCharacter(soloMember)

	// Record positions before move
	turnMemberPosBefore := gs.CombatScene.GetPartyMemberPosition(turnMember)
	soloMemberPosBefore := gs.CombatScene.GetPartyMemberPosition(soloMember)

	// Try to move before the queue reaches the selected character.
	// We try all 4 directions to find one that works (terrain may block some)
	moved := false
	for _, dir := range []references.Direction{
		references.Up, references.Down,
		references.Left, references.Right,
	} {
		if gs.MoveCombatCharacter(dir) == CombatMoveMoved {
			moved = true
			break
		}
	}

	if moved {
		// The selected future character must not move out of turn.
		soloMemberPosAfter := gs.CombatScene.GetPartyMemberPosition(soloMember)
		assert.Equal(t, soloMemberPosBefore, soloMemberPosAfter,
			"selected character %d must wait for its queue turn", soloMember)

		// The authoritative current actor performs the movement.
		turnMemberPosAfter := gs.CombatScene.GetPartyMemberPosition(turnMember)
		assert.NotEqual(t, turnMemberPosBefore, turnMemberPosAfter,
			"current turn character %d should retain action authority", turnMember)
		require.NotEmpty(t, mock.CombatEvents)
		moveEvent := mock.CombatEvents[len(mock.CombatEvents)-1]
		assert.Equal(t, CombatEventMove, moveEvent.Kind)
		assert.True(t, moveEvent.Attacker.IsPartyMember)
		assert.Equal(t, turnMember, moveEvent.Attacker.Index)
		assert.Equal(t, turnMemberPosBefore, moveEvent.From)
		assert.Equal(t, turnMemberPosAfter, moveEvent.To)
	} else {
		t.Log("Could not find a valid move direction for solo character — terrain blocked all 4 directions")
	}
}

// TestCombatSoloMode_ClearRestoresNormalCycling verifies that clearing solo
// mode allows AdvanceCombatTurn to resume normal dex-based cycling where
// all active PCs get turns (not just the solo character).
func TestCombatSoloMode_ClearRestoresNormalCycling(t *testing.T) {
	gs, _ := newBritainCombatTestState(t, 42)

	turnBefore := gs.CombatScene.GetCurrentTurnMemberIndex()

	// Enable then disable solo mode
	gs.CombatScene.SetSoloMoveCharacter(2)
	assert.True(t, gs.CombatScene.IsSoloMoveActive())

	gs.CombatScene.ClearSoloMoveMode()
	assert.False(t, gs.CombatScene.IsSoloMoveActive())
	nextMember := -1
	for _, entry := range gs.CombatScene.TurnQueue.entries {
		entry.turnCountdown = turnCountdownBase
		if nextMember < 0 && entry.Combatant.IsPC() && entry.EntityIndex != turnBefore {
			nextMember = entry.EntityIndex
			entry.turnCountdown = 1
		}
	}
	require.NotEqual(t, -1, nextMember)

	// Normal mode admits the next due party member rather than auto-passing it.
	gs.AdvanceCombatTurn()
	turnAfter := gs.CombatScene.GetCurrentTurnMemberIndex()
	assert.Equal(t, nextMember, turnAfter)
}

func TestCombatSoloModeDoesNotAutoPassPossessedPC(t *testing.T) {
	gs, _ := newBritainCombatTestState(t, 42)
	active := gs.ActiveCombatPartyMembers()
	require.GreaterOrEqual(t, len(active), 2)
	selected, possessed := active[0], active[1]
	gs.CombatScene.SetSoloMoveCharacter(selected)
	gs.CombatScene.PartyPossessed[possessed] = true
	gs.PartyState.GetCharacter(possessed).SetStatus(party_state.Charmed)
	for _, entry := range gs.CombatScene.TurnQueue.entries {
		entry.turnCountdown = turnCountdownBase
		if entry.Combatant.IsPC() && entry.EntityIndex == possessed {
			entry.turnCountdown = 1
		}
	}
	gs.CombatScene.presentTurns = true

	gs.AdvanceCombatTurn()
	step, scheduled := gs.ReviewScheduledCombatStep()

	require.True(t, scheduled)
	assert.Equal(t, CombatPresentationMonsterTurn, step.Kind)
	assert.Equal(t, CombatTurnIdentity{IsPartyMember: true, EntityIndex: possessed}, step.Actor)
}

// ─── Monster Collision Tests ────────────────────────────────────────────────

// TestCombatMovement_BlockedByMonster verifies that isCombatPositionValid
// rejects positions occupied by active monsters, preventing party members
// from walking onto monster tiles.
func TestCombatMovement_BlockedByMonster(t *testing.T) {
	gs, _ := newBritainCombatTestState(t, 42)

	manager := gs.CombatScene.MonsterManager
	manager.Clear()

	// Spawn a monster at a known position
	enemyRefs := *gs.GameReferences.EnemyReferences
	rng := newFixedRNG(1)
	monsterPos := references.Position{X: 5, Y: 3}
	ref := &enemyRefs[references.MonsterOrc]
	m := NewCombatMonster(references.MonsterOrc, ref, monsterPos, rng)
	manager.AddMonster(m)

	// The monster position should be considered invalid for party movement
	assert.False(t, gs.isCombatPositionValid(&monsterPos, 0),
		"position occupied by monster should be invalid for party movement")

	// A position without a monster should still be potentially valid (terrain permitting)
	emptyPos := references.Position{X: 5, Y: 5} // center of map
	// We can't guarantee terrain is passable, but monster collision should not block
	if gs.IsPassable(&emptyPos) && !gs.CombatScene.isPartyMemberAtPosition(emptyPos, -1) {
		assert.True(t, gs.isCombatPositionValid(&emptyPos, -1),
			"empty passable position should be valid for movement")
	}
}

// TestCombatMovement_DeadMonsterDoesNotBlock verifies that killed monsters
// do not block movement — only active monsters should be collision checked.
func TestCombatMovement_DeadMonsterDoesNotBlock(t *testing.T) {
	gs, _ := newBritainCombatTestState(t, 42)

	manager := gs.CombatScene.MonsterManager
	manager.Clear()

	// Spawn a monster and immediately kill it
	enemyRefs := *gs.GameReferences.EnemyReferences
	rng := newFixedRNG(1)
	monsterPos := references.Position{X: 5, Y: 5}
	ref := &enemyRefs[references.MonsterOrc]
	m := NewCombatMonster(references.MonsterOrc, ref, monsterPos, rng)
	manager.AddMonster(m)

	// Kill the monster
	m.SetCurrentHP(0)
	assert.True(t, m.IsDead())

	// Position should no longer be occupied (dead monster doesn't block)
	assert.False(t, manager.IsPositionOccupied(monsterPos),
		"dead monster should not block position")
}
