package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/ai"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func TestDefenselessMurderAppliesCompleteConsequences(t *testing.T) {
	tests := []struct {
		name string
		tile indexes.SpriteIndex
	}{
		{name: "stocks", tile: indexes.Stocks},
		{name: "manacles", tile: indexes.Manacles},
		{name: "bed", tile: indexes.LeftBed},
		{name: "broken mirror", tile: indexes.MirrorBroken},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newDefenselessAttackFixture(t, test.tile, indexes.TownsPerson_KeyIndex, 1)

			result := fixture.state.ActionAttackSmallMap(references.Up)

			require.True(t, result)
			require.NotEmpty(t, fixture.callbacks.Messages)
			assert.Equal(t, "Murdered!", fixture.callbacks.Messages[0])
			assert.Equal(t, 40, fixture.state.PartyState.Karma.Value,
				"the initial attack and murder penalties must both apply")
			assert.True(t, fixture.state.IsCriminal())
			assert.Equal(t, []references.Position{fixture.target}, fixture.kapows)
			assert.Contains(t, fixture.callbacks.SoundEffectsPlayed, SoundWhiteNoiseDamage)
			assert.True(t, fixture.victimLoadedAtKapow)
			assert.False(t, fixture.victimDeadAtKapow,
				"persistent death follows original kapow feedback")
			assert.True(t, fixture.state.PartyState.IsNPCDead(references.Britain, fixture.victimSlot))
			assert.False(t, mapUnitsContain(fixture.controller.GetNpcs(), fixture.victim))
			assert.Equal(t, references.AiType(references.DrudgeWorthThing),
				fixture.guard.MapUnitDetails().GetOverriddenAiType())
			assert.Equal(t, uint8(1), fixture.state.DateTime.Minute,
				"murder must complete the normal town turn")
			assert.Equal(t, []int{1}, fixture.callbacks.TimeAdvanced(),
				"the authoritative turn must advance one minute")
		})
	}
}

func TestNewSystemCallbacksRejectsMissingKapowCallback(t *testing.T) {
	valid := NewMockSystemCallbacks(t).ToSystemCallbacks()
	visualCallbacks := valid.Visual
	visualCallbacks.PlayKapowAtPosition = nil

	callbacks, err := NewSystemCallbacks(
		valid.Message,
		visualCallbacks,
		valid.Audio,
		valid.Screen,
		valid.Flow,
		valid.Talk,
		valid.UI,
		valid.Combat,
	)

	require.Nil(t, callbacks)
	require.ErrorIs(t, err, errKapowCallbackRequired)
}

func TestDefenselessMurderUsesActorTypeForBlackthornImmunity(t *testing.T) {
	t.Run("dialogue 14 civilian is not immune", func(t *testing.T) {
		fixture := newDefenselessAttackFixture(t, indexes.Stocks, indexes.TownsPerson_KeyIndex, 14)

		require.True(t, fixture.state.ActionAttackSmallMap(references.Up))

		assert.Equal(t, "Murdered!", fixture.callbacks.Messages[0])
		assert.Equal(t, 40, fixture.state.PartyState.Karma.Value)
		assert.True(t, fixture.state.PartyState.IsNPCDead(references.Britain, fixture.victimSlot))
		assert.False(t, mapUnitsContain(fixture.controller.GetNpcs(), fixture.victim))
	})

	t.Run("Blackthorn actor is immune regardless of dialogue", func(t *testing.T) {
		fixture := newDefenselessAttackFixture(t, indexes.Stocks, indexes.Blackthorn1, 1)

		require.True(t, fixture.state.ActionAttackSmallMap(references.Up))

		assert.Equal(t, "Missed!", fixture.callbacks.Messages[0])
		assert.Equal(t, 45, fixture.state.PartyState.Karma.Value,
			"Blackthorn still receives the initial hostile-attack penalty")
		assert.True(t, fixture.state.IsCriminal())
		assert.Empty(t, fixture.kapows)
		assert.NotContains(t, fixture.callbacks.SoundEffectsPlayed, SoundWhiteNoiseDamage)
		assert.False(t, fixture.state.PartyState.IsNPCDead(references.Britain, fixture.victimSlot))
		assert.True(t, mapUnitsContain(fixture.controller.GetNpcs(), fixture.victim))
		assert.Equal(t, references.AiType(references.DrudgeWorthThing),
			fixture.guard.MapUnitDetails().GetOverriddenAiType())
		assert.Equal(t, uint8(1), fixture.state.DateTime.Minute)
	})
}

func TestDefenselessGuardDeathIsNotPersistent(t *testing.T) {
	fixture := newDefenselessAttackFixture(t, indexes.Manacles, indexes.Guard_KeyIndex, 0)

	require.True(t, fixture.state.ActionAttackSmallMap(references.Up))

	assert.Equal(t, "Murdered!", fixture.callbacks.Messages[0])
	assert.False(t, fixture.state.PartyState.IsNPCDead(references.Britain, fixture.victimSlot))
	assert.False(t, mapUnitsContain(fixture.controller.GetNpcs(), fixture.victim))
}

func TestDefenselessMurderActivatesGuardsInSlotOrderBeforeImpact(t *testing.T) {
	fixture := newDefenselessAttackFixture(t, indexes.LeftBed, indexes.TownsPerson_KeyIndex, 1)
	fixture.victim.MapUnitDetails().NPCNum = 2
	firstCivilian := defenselessAttackNPC(1, indexes.TownsPerson_KeyIndex, 1, true, references.Britain)
	firstCivilian.SetPos(references.Position{X: 8, Y: 8})
	lastCivilian := defenselessAttackNPC(3, indexes.MerchantKeyIndex, 1, true, references.Britain)
	lastCivilian.SetPos(references.Position{X: 9, Y: 9})
	fixture.controller.GetNpcs().ReplaceWith([]map_units.MapUnit{
		lastCivilian, fixture.victim, firstCivilian, fixture.guard,
	})

	const seed = uint64(1)
	expectedRNG := beradrng.New(seed)
	expectedFlee := [3]bool{}
	for index := range expectedFlee {
		expectedFlee[index] = expectedRNG.Intn(guardActivationFleeRollMaximum+1) < guardActivationFleeThreshold
	}
	fixture.state.SetRandomSeed(seed)
	var callsAtImpact uint64
	fixture.state.SystemCallbacks.Visual.PlayKapowAtPosition = func(_ references.Position) {
		callsAtImpact = fixture.state.GetRNG().Calls()
	}

	require.True(t, fixture.state.ActionAttackSmallMap(references.Up))

	assert.Equal(t, uint64(3), callsAtImpact,
		"all civilian rolls, including the loaded victim, precede impact")
	assert.Equal(t, expectedFlee[0], isFleeing(firstCivilian))
	assert.Equal(t, expectedFlee[1], isFleeing(fixture.victim))
	assert.Equal(t, expectedFlee[2], isFleeing(lastCivilian))
	assert.Equal(t, references.AiType(references.DrudgeWorthThing),
		fixture.guard.MapUnitDetails().GetOverriddenAiType())
}

func TestDefenselessMurderDeathSurvivesNativeHydration(t *testing.T) {
	fixture := newDefenselessAttackFixture(t, indexes.Stocks, indexes.TownsPerson_KeyIndex, 1)
	require.True(t, fixture.state.ActionAttackSmallMap(references.Up))
	fixture.state.TransitionToLocation(
		references.Yew,
		0,
		references.Position{X: 10, Y: 10},
		TransitionFullLocation,
		false,
	)
	fixture.state.TransitionToLocation(
		references.Britain,
		0,
		references.Position{X: 5, Y: 5},
		TransitionFullLocation,
		false,
	)
	for _, unit := range *fixture.state.CurrentNPCAIController.GetNpcs() {
		assert.NotEqual(t, fixture.victimSlot, unit.MapUnitDetails().NPCNum,
			"map rebuild must not restore a persistently murdered NPC")
	}
	fixture.state.TransitionToLocation(
		references.Yew,
		0,
		references.Position{X: 10, Y: 10},
		TransitionFullLocation,
		false,
	)

	snapshot, err := fixture.state.CaptureSaveSnapshot()
	require.NoError(t, err)
	hydrated, err := NewGameStateFromSaveSnapshot(
		snapshot,
		fixture.state.GameConfig,
		fixture.state.GameReferences,
		xTilesVisibleOnGameScreen,
		yTilesVisibleOnGameScreen,
	)
	require.NoError(t, err)

	assert.True(t, hydrated.PartyState.IsNPCDead(references.Britain, fixture.victimSlot))
}

func TestAttackEmptyDefenselessTerrainHasNoCrimeConsequences(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(5, 5).
		WithSystemCallbacks().
		Build()
	target := references.Position{X: 5, Y: 4}
	state.GetLayeredMapByCurrentLocation().SetTileByLayer(map_state.MapLayer, &target, indexes.Stocks)
	state.CurrentNPCAIController.GetNpcs().ReplaceWith(nil)
	state.PartyState.Karma.Value = 50

	result := state.ActionAttackSmallMap(references.Up)

	assert.False(t, result)
	assert.Equal(t, 50, state.PartyState.Karma.Value)
	assert.False(t, state.IsCriminal())
	assert.Equal(t, "Nothing to attack!", callbacks.Messages[0])
}

type defenselessAttackFixture struct {
	state               *GameState
	callbacks           *MockSystemCallbacks
	controller          *ai.NPCAIControllerSmallMap
	victim              *map_units.NPCFriendly
	guard               *map_units.NPCFriendly
	target              references.Position
	victimSlot          int
	kapows              []references.Position
	victimLoadedAtKapow bool
	victimDeadAtKapow   bool
}

func newDefenselessAttackFixture(
	t *testing.T,
	baseTile indexes.SpriteIndex,
	victimSprite indexes.SpriteIndex,
	dialogNumber byte,
) *defenselessAttackFixture {
	t.Helper()

	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(5, 5).
		WithSystemCallbacks().
		WithRandomSeed(1).
		Build()
	controller, ok := state.CurrentNPCAIController.(*ai.NPCAIControllerSmallMap)
	require.True(t, ok)
	state.DateTime.Hour = 12
	state.DateTime.Minute = 0
	state.PartyState.Karma.Value = 50
	target := references.Position{X: 5, Y: 4}
	currentMap := state.GetLayeredMapByCurrentLocation()
	currentMap.SetTileByLayer(map_state.MapLayer, &target, baseTile)
	currentMap.UnSetTileByLayer(map_state.MapOverrideLayer, &target)

	victimSlot := -1
	for _, unit := range *controller.GetNpcs() {
		if _, friendly := unit.(*map_units.NPCFriendly); friendly && unit.MapUnitDetails().NPCNum != 0 {
			victimSlot = unit.MapUnitDetails().NPCNum

			break
		}
	}
	require.NotEqual(t, -1, victimSlot, "Britain fixture requires a persistent nonzero NPC slot")
	victim := defenselessAttackNPC(victimSlot, victimSprite, dialogNumber, true, references.Britain)
	victim.SetPos(target)
	victim.SetFloor(0)
	guard := defenselessAttackNPC(0, indexes.Guard_KeyIndex, 0, false, references.Britain)
	guard.SetPos(references.Position{X: 8, Y: 8})
	guard.SetFloor(0)
	controller.GetNpcs().ReplaceWith([]map_units.MapUnit{victim, guard})

	fixture := &defenselessAttackFixture{
		state: state, callbacks: callbacks, controller: controller,
		victim: victim, guard: guard, target: target, victimSlot: victimSlot,
		kapows: make([]references.Position, 0, 1),
	}
	state.SystemCallbacks.Visual.PlayKapowAtPosition = func(position references.Position) {
		fixture.kapows = append(fixture.kapows, position)
		live := controller.GetNpcs().GetMapUnitAtPositionAndFloorOrNil(target, 0)
		fixture.victimLoadedAtKapow = live != nil && *live == victim
		fixture.victimDeadAtKapow = state.PartyState.IsNPCDead(references.Britain, victimSlot)
	}

	return fixture
}

func defenselessAttackNPC(
	slot int,
	sprite indexes.SpriteIndex,
	dialogNumber byte,
	movable bool,
	location references.Location,
) *map_units.NPCFriendly {
	npc := guardActivationTestNPC(slot, sprite, dialogNumber, movable)
	npc.NPCReference.Location = location
	npc.SetVisible(true)

	return npc
}

func mapUnitsContain(units *map_units.MapUnits, wanted map_units.MapUnit) bool {
	for _, unit := range *units {
		if unit == wanted {
			return true
		}
	}

	return false
}

func isFleeing(npc *map_units.NPCFriendly) bool {
	return npc.MapUnitDetails().GetOverriddenAiType() == references.FleeFromAvatar
}
