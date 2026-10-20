package main

import (
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func TestOutdoorLivingRetreatProductionDrainCompletesSlowStepAfterSummary(t *testing.T) {
	scene, state, _ := newSmashedInputTestScene(t, 0, 123)
	scene.spriteSheet = sharedPresentationTestSpriteSheet()
	now := time.Unix(100, 0)
	scene.startTime, scene.now = now, func() time.Time { return now }
	state.SystemCallbacks.Combat = game_state.NewCombatCallbacks(scene.recordCombatEvent, scene.recordCombatConclusion)
	state.TransitionToLocation(references.BritanniaUnderworld, 0, references.Position{X: 100, Y: 100}, game_state.TransitionFullLocation, false)
	state.CurrentNPCAIController.GetNpcs().ReplaceWith(nil)
	state.DebugOptions.MonsterGen = false
	state.DateTime.Hour, state.DateTime.Minute = 12, 0
	state.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(references.NoPartyVehicle, state.MapState.PlayerLocation.Position, 0)
	for index := range state.PartyState.GetCharacterCount() {
		member := state.PartyState.GetCharacter(index)
		member.PartyMap = party_state.PartyMapAtInn(references.Britain)
		member.SetStatus(party_state.Good)
		member.SetMaxHp(100)
		member.SetHp(100)
		member.SetEquippedItem(references.SlotRing, references.NoEquipment)
	}
	avatar := state.PartyState.GetAvatar()
	avatar.PartyMap, avatar.Dexterity = party_state.PartyMapActive, 30
	for y := references.Coordinate(98); y <= 103; y++ {
		for x := references.Coordinate(98); x <= 103; x++ {
			position := references.Position{X: x, Y: y}
			state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &position, indexes.Grass)
		}
	}
	destination := references.Position{X: 101, Y: 100}
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &destination, indexes.Forest3)
	source := addOutdoorRetreatSceneEnemy(t, state, references.MonsterSeaSerpent, references.Position{X: 102, Y: 100}, 31)
	addOutdoorRetreatSceneEnemy(t, state, references.MonsterSandtrap, references.Position{X: 101, Y: 101}, 20)
	before := state.DateTime

	// Real movement enters the forest and suspends check_slow in its first
	// extra phase. No synthetic EndCombat or completion state is installed.
	scene.keyboard.SimulateKeyPress(ebiten.KeyRight)
	require.NoError(t, scene.Update(nil))
	require.NotNil(t, state.CombatScene)
	assert.Equal(t, destination, state.CombatScene.PreCombatPlayerLocation.Position)
	assert.Same(t, source, state.CombatScene.OverworldOrigin.SourceEnemy)
	assert.Equal(t, before, state.DateTime)
	require.NoError(t, scene.Update(nil)) // Production first DEX selection.
	require.True(t, state.CombatScene.CombatTurnsStarted())
	require.Equal(t, game_state.CombatTurnIdentity{IsPartyMember: true, EntityIndex: 0}, state.CombatScene.GetCurrentTurnIdentity())
	require.False(t, state.CombatScene.HasScheduledCombatStep())
	// Place the sole PC at a legal map edge to isolate the physical-exit path.
	state.CombatScene.SetPartyMemberPosition(0, references.Position{X: 0, Y: 5})
	state.ActiveDurationEffect = game_state.DurationEffect{EffectType: game_state.DurationEffectProtection, TurnsRemaining: 5}
	state.SetRandomSeed(123)
	oracle := beradrng.New(123)
	scene.keyboard.SimulateKeyPress(ebiten.KeyLeft)
	require.NoError(t, scene.Update(nil))
	require.Nil(t, state.CombatScene)
	assert.False(t, state.PartyState.IsAllPartyDead())
	assert.Equal(t, destination, state.MapState.PlayerLocation.Position)
	assert.NotContains(t, *state.CurrentNPCAIController.GetNpcs(), source)
	assert.True(t, state.HasPendingOverworldContact(), "living retreat retains the rest of the slow step")
	wantHP := uint16(100 - oracle.Intn(8) - 1)
	assert.Equal(t, wantHP, avatar.GetCurrentHp(), "lower actor resolves after the world is restored")
	assert.Equal(t, before, state.DateTime)

	require.NoError(t, scene.Update(nil)) // Summary owns input before the pending-contact drain.
	require.NotNil(t, scene.battleSummaryDialog)
	assert.Equal(t, game_state.CombatDefeat, scene.lastBattleSummary.Reason)
	require.NoError(t, scene.Update(nil)) // Release the summary's opening gate.
	scene.keyboard.SimulateKeyPress(ebiten.KeyEnter)
	require.NoError(t, scene.Update(nil))
	require.Nil(t, scene.battleSummaryDialog)
	assert.Equal(t, before, state.DateTime, "dismissal cannot consume the pending player tail")
	assert.Equal(t, 4, state.ActiveDurationEffect.TurnsRemaining)

	require.NoError(t, scene.Update(nil)) // The ordinary production drain resumes check_slow.
	want := before
	want.Advance(4)
	want.Advance(2)
	wantHP -= uint16(oracle.Intn(8) + 1)
	wantHP -= uint16(oracle.Intn(8) + 1)
	assert.Equal(t, want, state.DateTime)
	assert.Equal(t, wantHP, avatar.GetCurrentHp())
	assert.Equal(t, 3, state.ActiveDurationEffect.TurnsRemaining)
	assert.Equal(t, oracle.Calls(), state.GetRNG().Calls())
	assert.False(t, state.HasPendingOverworldContact())
	assert.False(t, state.ShouldDeferOverworldContact())
	assert.False(t, scene.inputBlocked)
	require.NoError(t, scene.Update(nil))
	assert.Equal(t, want, state.DateTime)
	assert.Equal(t, oracle.Calls(), state.GetRNG().Calls(), "later frames cannot duplicate the completed phase")
}

func addOutdoorRetreatSceneEnemy(
	t *testing.T, state *game_state.GameState, monster references.MonsterType, position references.Position, slot int,
) *map_units.NPCEnemy {
	t.Helper()
	enemy := map_units.NewEnemyNPC((*state.GameReferences.EnemyReferences)[monster], slot)
	enemy.SetPos(position)
	enemy.SetFloor(0)
	enemy.SetVisible(true)
	require.True(t, state.CurrentNPCAIController.GetNpcs().AddEnemy(enemy))
	stored := state.CurrentNPCAIController.GetNpcs().GetVisibleEnemyAtPositionAndFloorOrNil(position, 0)
	require.NotNil(t, stored)
	stored.MapUnitDetails().NPCNum = slot

	return stored
}
