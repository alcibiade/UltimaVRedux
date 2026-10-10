package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func newPersistentEffectsTestState(t *testing.T) *GameState {
	t.Helper()
	state := createMinimalGameStateForToggleTests(t)
	state.ShadowlordState = references.NewShadowlordState()
	state.GameReferences = &references.GameReferences{InventoryItemReferences: references.NewInventoryItemsReferences()}
	state.PartyState.Inventory.Provisions.Food.Set(100)
	return state
}

func TestPersistentEffectsUseDeterministicCategoryOrder(t *testing.T) {
	state := newPersistentEffectsTestState(t)
	state.MapState.Lighting.RestoreTorchMinutes(20)
	state.MagicLightMinutesRemaining = 100
	state.MapState.Lighting.SetMagicLight(true)
	state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectProtection, TurnsRemaining: 4}
	state.smashed = 25
	state.ShadowlordState.AutomaticPresence = references.Astaroth
	state.PartyState.Inventory.Provisions.Food.Set(0)
	frigate := map_units.NewNPCFriendlyVehiceNewRef(references.FrigateVehicle, references.Position{}, 0)
	state.PartyVehicle = *frigate
	state.SetHMSCapeActivated(true)

	effects := state.GetPersistentEffectStatuses()
	ids := make([]PersistentEffectID, len(effects))
	for i := range effects {
		ids[i] = effects[i].ID
	}

	assert.Equal(t, []PersistentEffectID{
		PersistentEffectTorch,
		PersistentEffectMagicLight,
		PersistentEffectProtection,
		PersistentEffectIntoxication,
		PersistentEffectShadowlord,
		PersistentEffectStarvation,
		PersistentEffectHMSCape,
	}, ids)
	assert.Equal(t, EffectAtmosphereFading, effects[0].Atmosphere)
	assert.Equal(t, EffectAtmosphereFading, effects[2].Atmosphere)
	assert.Equal(t, EffectAtmosphereDanger, effects[3].Atmosphere)
	assert.Equal(t, EffectAtmosphereOminous, effects[4].Atmosphere)

	state.ToggleCrown()
	effects = state.GetPersistentEffectStatuses()
	require.Len(t, effects, 7)
	assert.Equal(t, PersistentEffectCrownSuppression, effects[2].ID)
	assert.Equal(t, EffectAtmosphereSteady, effects[2].Atmosphere)
}

func TestPersistentCrownSuppressionIsPermanentDetachedAndReadOnly(t *testing.T) {
	state := newPersistentEffectsTestState(t)
	state.ToggleAmulet()
	state.ToggleBadge()
	assert.Empty(t, state.GetPersistentEffectStatuses(), "no speculative Amulet or Badge benefits")
	state.ToggleCrown()
	state.SetRandomSeed(536)
	state.ElapsedMs = 9876
	state.MagicLightMinutesRemaining = 100
	state.MapState.Lighting.RestoreTorchMinutes(100)
	want := state.GetPersistentEffectStatuses()
	require.Len(t, want, 3)
	assert.Equal(t, PersistentEffectCrownSuppression, want[2].ID)
	assert.Equal(t, EffectAtmosphereSteady, want[2].Atmosphere)
	for range 300 {
		state.processToggleItems()
		assert.Equal(t, want, state.GetPersistentEffectStatuses())
	}
	assert.Equal(t, ToggleItemDuration, state.CrownState.TurnsRemaining)
	assert.Equal(t, int64(9876), state.ElapsedMs)
	assert.Zero(t, state.GetRNG().Calls())
	want[2].ID = PersistentEffectNegateMagic
	assert.Equal(t, PersistentEffectCrownSuppression, state.GetPersistentEffectStatuses()[2].ID)
	state.ToggleCrown()
	assert.Len(t, state.GetPersistentEffectStatuses(), 2)
}

func TestPersistentIntoxicationTracksSmashedButNotDrunk(t *testing.T) {
	state := newPersistentEffectsTestState(t)
	state.drunk = 4
	assert.Empty(t, state.GetPersistentEffectStatuses())

	state.smashed = 25
	effects := state.GetPersistentEffectStatuses()
	require.Len(t, effects, 1)
	assert.Equal(t, PersistentEffectIntoxication, effects[0].ID)
	assert.Equal(t, EffectAtmosphereDanger, effects[0].Atmosphere)

	state.smashed = 1
	assert.Equal(t, effects, state.GetPersistentEffectStatuses(), "decrements retain one detached status")

	state.smashed = 0
	assert.Empty(t, state.GetPersistentEffectStatuses())
}

func TestPersistentEffectsDeriveCurrentStateWithoutDuplicateRegistration(t *testing.T) {
	state := newPersistentEffectsTestState(t)
	state.MapState.Lighting.LightTorch()
	state.MapState.Lighting.LightTorch()

	effects := state.GetPersistentEffectStatuses()
	require.Len(t, effects, 1)
	assert.Equal(t, PersistentEffectTorch, effects[0].ID)

	state.MapState.Lighting.ExtinguishTorch()
	assert.Empty(t, state.GetPersistentEffectStatuses())
}

func TestPersistentEffectsReplaceCrownWithInAn(t *testing.T) {
	state := newPersistentEffectsTestState(t)
	state.ToggleCrown()
	state.ActivateNegateMagicEffect(10)

	effects := state.GetPersistentEffectStatuses()
	require.Len(t, effects, 1)
	assert.Equal(t, PersistentEffectNegateMagic, effects[0].ID)
	assert.False(t, state.IsCrownActive())
}

func TestPersistentStarvationTracksZeroFoodContinuously(t *testing.T) {
	state := newPersistentEffectsTestState(t)
	state.PartyState.Inventory.Provisions.Food.Set(0)

	effects := state.GetPersistentEffectStatuses()
	require.Len(t, effects, 1)
	assert.Equal(t, PersistentEffectStarvation, effects[0].ID)

	state.PartyState.Inventory.Provisions.Food.Set(1)
	assert.Empty(t, state.GetPersistentEffectStatuses())
}

func TestPersistentShadowlordUsesOnlyAutomaticTownPresence(t *testing.T) {
	state := newPersistentEffectsTestState(t)
	require.True(t, state.ShadowlordState.Summon(references.Faulinei))
	assert.Empty(t, state.GetPersistentEffectStatuses(), "yell-summoned Shadowlords are encounter state, not town influence")

	state.ShadowlordState.AutomaticPresence = references.Nosfentor
	effects := state.GetPersistentEffectStatuses()
	require.Len(t, effects, 1)
	assert.Equal(t, PersistentEffectShadowlord, effects[0].ID)
	assert.Equal(t, EffectAtmosphereOminous, effects[0].Atmosphere)

	state.ShadowlordState.ClearAutomaticPresence()
	assert.Empty(t, state.GetPersistentEffectStatuses())
}

func TestPersistentHMSCapeRequiresActiveFrigateContext(t *testing.T) {
	state := newPersistentEffectsTestState(t)
	frigate := map_units.NewNPCFriendlyVehiceNewRef(references.FrigateVehicle, references.Position{}, 0)
	state.PartyVehicle = *frigate
	state.SetHMSCapeActivated(true)

	effects := state.GetPersistentEffectStatuses()
	require.Len(t, effects, 1)
	assert.Equal(t, PersistentEffectHMSCape, effects[0].ID)

	state.PartyVehicle.GetVehicleDetails().VehicleType = references.NoPartyVehicle
	assert.Empty(t, state.GetPersistentEffectStatuses())
}

func TestPersistentAndWornNativeContinuationNeedsNoPresentationHydration(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()
	state.ToggleCrown()
	if !state.IsCrownActive() {
		state.ToggleCrown()
	}
	state.AmuletState = state.CrownState
	state.BadgeState = state.CrownState
	for _, item := range []references.QuestItem{references.Crown, references.Amulet, references.BlackBadge} {
		state.PartyState.Inventory.QuestItems.Set(item, 1)
	}
	state.MagicLightMinutesRemaining = 100
	state.MapState.Lighting.SetMagicLight(true)
	for _, crown := range []bool{true, false} {
		if !crown {
			state.ActivateNegateMagicEffect(10)
		}
		snapshot, err := state.CaptureSaveSnapshot()
		require.NoError(t, err)
		restored, err := NewGameStateFromSaveSnapshot(snapshot, state.GameConfig, state.GameReferences,
			state.MapState.XTilesVisibleOnGameScreen, state.MapState.YTilesVisibleOnGameScreen)
		require.NoError(t, err)
		restored.SystemCallbacks = nil
		assert.Equal(t, state.GetWornArtifactStatuses(), restored.GetWornArtifactStatuses())
		assert.Equal(t, state.GetPersistentEffectStatuses(), restored.GetPersistentEffectStatuses())
		after, err := restored.CaptureSaveSnapshot()
		require.NoError(t, err)
		assert.Equal(t, snapshot, after, "projection cannot replay feedback or mutate native continuation")
	}
}
