package main

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/ui/mainscreen"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

func TestCombatRosterTextProjectsKnownOriginalAndIndependentMembers(t *testing.T) {
	scene, _ := newCombatUseTestScene(t)
	state := scene.gameState
	combat := state.CombatScene
	source := combat.GetCurrentTurnIdentity()
	other := (source.EntityIndex + 1) % party_state.MaxCharactersInParty
	combat.PartyActive[other] = true
	combat.PartyPossessed[source.EntityIndex] = true
	combat.PartySleeping[source.EntityIndex] = true
	combat.PartyUnderground[source.EntityIndex] = true
	combat.PartyPossessed[other] = true
	_, disclosed := state.ProjectCombatActor(source, 0)
	require.False(t, disclosed, "the battlefield projection intentionally hides underground actors")
	before := snapshotBlockedCombatState(t, state)
	overlay := combatRosterText(state)
	assert.Equal(t, mainscreen.CharacterCombatText{InCombat: true, Possessed: true, Sleeping: true}, overlay[source.EntityIndex])
	assert.Equal(t, mainscreen.CharacterCombatText{InCombat: true, Possessed: true}, overlay[other])
	assert.Equal(t, before, snapshotBlockedCombatState(t, state), "roster projection must be read-only")
	combat.PartyPossessed[source.EntityIndex] = false
	updated := combatRosterText(state)
	assert.False(t, updated[source.EntityIndex].Possessed)
	assert.True(t, updated[other].Possessed)
	assert.True(t, overlay[source.EntityIndex].Possessed, "previous projections are detached")
	combat.PartyActive[other] = false
	assert.Equal(t, mainscreen.CharacterCombatText{InCombat: true}, combatRosterText(state)[other],
		"retired originals must not retain stale possession flags")
	state.TransitionFromCombatMap()
	assert.Equal(t, [party_state.MaxCharactersInParty]mainscreen.CharacterCombatText{}, combatRosterText(state))
}

func TestCombatRosterTextDoesNotInheritClonePossessionOrSharedSleep(t *testing.T) {
	scene, _ := newCombatUseTestScene(t)
	state := scene.gameState
	combat := state.CombatScene
	source := combat.GetCurrentTurnIdentity()
	character := state.PartyState.GetCharacter(source.EntityIndex)
	character.SetStatus(party_state.Sleep)
	combat.PartyPossessed[source.EntityIndex] = true
	combat.PartySleeping[source.EntityIndex] = true
	require.True(t, state.CastCombatInQuasXen(source.EntityIndex, source))
	var clone game_state.CombatTurnIdentity
	combat.ForEachPCClone(func(identity game_state.CombatTurnIdentity, _ references.Position) { clone = identity })
	require.Positive(t, clone.Slot)
	combat.PartyPossessed[source.EntityIndex] = false
	combat.PartySleeping[source.EntityIndex] = false
	combat.CurrentTurnIdentity = clone
	before := snapshotBlockedCombatState(t, state)
	assert.Equal(t, mainscreen.CharacterCombatText{InCombat: true}, combatRosterText(state)[source.EntityIndex])
	assert.True(t, combat.IsCombatPCPossessed(clone))
	assert.True(t, combat.IsCombatPCSleeping(clone))
	assert.Equal(t, party_state.Sleep, character.GetStatus(), "shared roster sleep cannot describe the awake original")
	assert.Equal(t, before, snapshotBlockedCombatState(t, state))
	combat.PartyActive[source.EntityIndex] = false
	assert.Equal(t, mainscreen.CharacterCombatText{InCombat: true}, combatRosterText(state)[source.EntityIndex],
		"a surviving clone must not replace the original roster identity")
}

func TestCombatRosterDrawAppliesAndClearsPossessionWithoutRefresh(t *testing.T) {
	scene := newCombatConcealmentTestState(t)
	state := scene.gameState
	source := state.CombatScene.GetCurrentTurnIdentity()
	character := state.PartyState.GetCharacter(source.EntityIndex)
	character.SetStatus(party_state.Good)
	require.True(t, state.SetActivePlayer(source.EntityIndex))
	const width = 1280
	scene.characterSummary = mainscreen.NewCharacterSummaryForWidth(scene.spriteSheet, width)
	t.Cleanup(scene.characterSummary.Dispose)
	panel := calculateRightRailLayout(image.Rect(0, 0, width, width*9/16)).characterContent
	scene.characterSummary.SetBounds(panel)
	screen := ebiten.NewImageWithOptions(panel, &ebiten.NewImageOptions{Unmanaged: true})
	t.Cleanup(screen.Deallocate)
	draw := func() []byte {
		screen.Clear()
		scene.drawCharacterSummary(screen)

		return combatRosterPixels(screen, panel)
	}
	baseline := draw()
	baselineBlue := countExactColor(screen, ucolor.CombatPossessed)
	before := *character
	state.CombatScene.PartyPossessed[source.EntityIndex] = true
	require.NotEqual(t, baseline, draw())
	assert.Greater(t, countExactColor(screen, ucolor.CombatPossessed), baselineBlue,
		"actual rendered text must use the readable light-blue possession colour")
	assert.Equal(t, before, *character)
	state.CombatScene.PartyPossessed[source.EntityIndex] = false
	require.Equal(t, baseline, draw(), "next draw must restore text without a stats callback")
	state.CombatScene.PartyPossessed[source.EntityIndex] = true
	draw()
	state.TransitionFromCombatMap()
	// Exit restores the pre-combat selection. Compare with the same selection
	// used for the baseline so this assertion isolates clearing the text overlay.
	require.True(t, state.SetActivePlayer(source.EntityIndex))
	assert.Equal(t, baseline, draw(), "combat exit must clear the previously drawn overlay")
	assert.Equal(t, before, *character)
}
