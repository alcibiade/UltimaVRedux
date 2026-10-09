package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestCombatRenderFrameSharesPreviewAndRefreshesAtNextFrame(t *testing.T) {
	scene, identity := setupCombatAutoAttackInputTest(t)
	state := scene.gameState
	character := state.PartyState.GetCharacter(identity.EntityIndex)
	character.SetEquippedItem(references.SlotWeapon, references.Bow)
	character.SetEquippedItem(references.SlotShield, references.NoEquipment)
	state.PartyState.Inventory.Equipment.Set(references.Arrows, 2)
	rngCalls := state.GetRNG().Calls()
	t.Cleanup(scene.endCombatRenderFrame)

	scene.beginCombatRenderFrame()
	preview := scene.combatAttackPreview()
	require.Len(t, preview.Targets, 1)
	assert.Equal(t, 0.0, testing.AllocsPerRun(20, func() {
		scene.combatAttackPreview()
	}), "HUD and battlefield must not rebuild an already prepared preview")
	scene.endCombatRenderFrame()
	assert.Empty(t, scene.combatRenderFrame)

	require.True(t, state.BeginCombatAttack())
	scene.secondaryKeyState = CombatAttackTargetInput
	state.CombatScene.AttackTargeting.Cursor = preview.Targets[0].Position
	state.PartyState.Inventory.Equipment.Set(references.Arrows, 0)
	scene.beginCombatRenderFrame()
	preview = scene.combatAttackPreview()
	require.NotNil(t, preview.Aim)
	assert.Equal(t, game_state.CombatShotNoAmmo, preview.Aim.Status)
	assert.Empty(t, preview.Targets)
	scene.endCombatRenderFrame()

	scene.inputBlocked = true
	scene.beginCombatRenderFrame()
	assert.Empty(t, scene.combatAttackPreview())
	scene.endCombatRenderFrame()
	scene.inputBlocked = false
	state.CombatScene = nil
	scene.beginCombatRenderFrame()
	assert.Empty(t, scene.combatAttackPreview())
	scene.endCombatRenderFrame()
	assert.Equal(t, rngCalls, state.GetRNG().Calls())
}

func TestCombatRenderFrameInsectStacksPreserveIdentityAndDisclosure(t *testing.T) {
	scene := newCombatConcealmentTestState(t)
	state := scene.gameState
	combat := state.CombatScene
	combat.MonsterManager.Clear()
	combat.TurnQueue.RemoveAllMonsters()
	var positions []references.Position
	var expected [][]int
	ref := (*state.GameReferences.EnemyReferences)[references.MonsterInsect]
	for count := 1; count <= 5; count++ {
		position := references.Position{X: references.Coordinate(count), Y: 5}
		positions = append(positions, position)
		setCombatConcealmentTerrain(state, position, indexes.BrickFloor)
		var indices []int
		for range count {
			monster := game_state.NewCombatMonster(references.MonsterInsect, &ref, position, state.AsRandomProvider())
			monster.SetLoyalty(game_state.CombatMonsterLoyaltyParty)
			indices = append(indices, combat.AddMonsterToQueue(monster))
		}
		if count < 2 || count > 4 {
			indices = nil
		}
		expected = append(expected, indices)
	}
	rngCalls := state.GetRNG().Calls()
	t.Cleanup(scene.endCombatRenderFrame)
	scene.beginCombatRenderFrame()
	for index, position := range positions {
		assert.Equal(t, expected[index], scene.combatPartyInsectStack(position))
	}
	assert.Equal(t, 0.0, testing.AllocsPerRun(20, func() {
		for _, position := range positions {
			scene.combatPartyInsectStack(position)
		}
	}), "sprite and health passes must share prepared memberships")
	scene.endCombatRenderFrame()
	assert.Empty(t, scene.combatRenderFrame)

	// Changes between rendered frames must not retain stale pips or identities.
	indices := expected[3]
	combat.MonsterManager.GetMonster(indices[0]).SetInvisible(true)
	combat.MonsterManager.GetMonster(indices[1]).SetLoyalty(game_state.CombatMonsterLoyaltyMonster)
	scene.beginCombatRenderFrame()
	assert.Equal(t, indices[2:], scene.combatPartyInsectStack(positions[3]))
	scene.endCombatRenderFrame()
	setCombatConcealmentTerrain(state, positions[3], indexes.BlackSquare)
	scene.beginCombatRenderFrame()
	assert.Empty(t, scene.combatPartyInsectStack(positions[3]))
	scene.endCombatRenderFrame()
	state.CombatScene = nil
	scene.beginCombatRenderFrame()
	assert.Empty(t, scene.combatPartyInsectStack(positions[3]))
	scene.endCombatRenderFrame()
	assert.Equal(t, rngCalls, state.GetRNG().Calls())
}

func TestAbsentCombatMarginSkipsArtworkAndResetsTint(t *testing.T) {
	scene, _ := newCombatUseTestScene(t)
	position := references.Position{X: -2, Y: -2}
	require.True(t, scene.gameState.IsOutOfBounds(position))
	require.Equal(t, indexes.NoSprites, scene.gameState.GetOuterTileForCurrentLocation())
	// Neither an atlas nor a render target is needed for absent terrain.
	scene.spriteSheet = &sprites.SpriteSheet{}
	options := &ebiten.DrawImageOptions{}
	scene.refreshStaticMapTiles(&position, scene.gameState.GetCurrentLayeredMap(), options)
	assert.Equal(t, ebiten.ColorScale{}, options.ColorScale, "edge tint must not leak into the next tile")
	assert.Nil(t, scene.unscaledMapImage)
}
