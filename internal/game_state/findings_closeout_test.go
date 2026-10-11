package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestFindingsLargeAdvanceUsesOneAcceptedInterval(t *testing.T) {
	for _, requested := range []int{540, 541, 600} {
		state, _ := NewIntegrationTestBuilder(t).WithLocation(references.Britain).WithSystemCallbacks().Build()
		state.DateTime.Hour, state.DateTime.Minute = 0, 59
		state.LastCamped = 20
		state.MapState.Lighting.RestoreTorchMinutes(1000)
		before := state.MapState.Lighting.TorchMinutesRemaining()
		elapsed := state.AdvanceGameplayMinutes(requested)
		assert.Equal(t, 540, elapsed)
		assert.Equal(t, byte(9), state.DateTime.Hour)
		assert.Equal(t, byte(59), state.DateTime.Minute)
		assert.Equal(t, 11, state.LastCamped)
		assert.Equal(t, before-540, state.MapState.Lighting.TorchMinutesRemaining())
		assert.Equal(t, []int{9}, state.pendingProvisionHours)
	}
}

func TestFindingsProvisionObservationReplacedByRestAndTimeStop(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).WithLocation(references.Britain).WithSystemCallbacks().Build()
	state.PartyState.Inventory.Provisions.Food.Set(0)
	state.DateTime.Hour, state.DateTime.Minute = 5, 59
	state.AdvanceGameplayMinutes(1)
	require.Equal(t, []int{6}, state.pendingProvisionHours)
	state.AdvanceBedRestMinutes(2)
	state.checkHourlyFoodConsumption()
	assert.NotContains(t, callbacks.Messages, "Starving!")
	state.DateTime.Hour, state.DateTime.Minute = 5, 59
	state.AdvanceGameplayMinutes(1)
	require.True(t, state.ActivateTimeStopEffect(10))
	assert.Zero(t, state.AdvanceGameplayMinutes(2))
	state.checkHourlyFoodConsumption()
	assert.NotContains(t, callbacks.Messages, "Starving!")
}

func TestFindingsTelescopeSelectionAndFallback(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.Britain).WithSystemCallbacks().Build()
	state.PartyState.ForEachCharacter(func(_ int, character *party_state.PlayerCharacter) {
		character.SetStatus(party_state.Sleep)
	})
	state.activePlayerIndex = -1
	assert.Equal(t, -1, state.TelescopeSunDamageTarget())
	state.PartyState.GetCharacter(1).SetStatus(party_state.Poisoned)
	assert.Equal(t, 1, state.TelescopeSunDamageTarget())
	state.activePlayerIndex = 0
	assert.Equal(t, 0, state.TelescopeSunDamageTarget(), "explicit designation does not invoke fallback")
	assert.Equal(t, 0, state.activePlayerIndex, "selection must not mutate active-player state")
}

func TestFindingsStoppedSurfaceCompletionReplacesPriorHourObservation(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).WithLocation(references.Britain).WithSystemCallbacks().Build()
	state.PartyState.Inventory.Provisions.Food.Set(0)
	state.DateTime.Hour, state.DateTime.Minute = 5, 59
	state.AdvanceGameplayMinutes(1)
	require.True(t, state.ActivateTimeStopEffect(2))
	state.completeSurfaceCycleWithActorPhase(references.SmallMapType, 1, false)
	assert.Equal(t, byte(0), state.DateTime.Minute)
	assert.Empty(t, state.pendingProvisionHours)
	assert.Equal(t, 6, state.lastProcessedHour)
	assert.NotContains(t, callbacks.Messages, "Starving!")
	assert.Equal(t, 1, state.ActiveDurationEffect.TurnsRemaining)
}

func TestFindingsSkiffTeleportRejectionPreservesResources(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).WithSystemCallbacks().Build()
	caster := state.PartyState.GetCharacter(0)
	caster.Class, caster.Level = party_state.Wizard, 8
	caster.SetMp(8)
	state.PartyState.Inventory.Spells.Set(references.VasRelPor, 1)
	state.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(references.SkiffVehicle,
		state.MapState.PlayerLocation.Position, state.MapState.PlayerLocation.Floor)
	result := state.ValidateVasRelPorPrerequisites(CastRequest{Spell: references.VasRelPor, CasterIdx: 0})
	assert.False(t, result.Valid)
	assert.Equal(t, VasRelPorShipMessage, result.ErrorMsg)
	assert.Equal(t, byte(8), caster.GetCurrentMp())
	assert.Equal(t, uint16(1), state.PartyState.Inventory.Spells.Get(references.VasRelPor))
}

func TestFindingsPurpleStandingPoisonAndDirectContact(t *testing.T) {
	for _, status := range []party_state.CharacterStatus{party_state.Good, party_state.Poisoned} {
		for _, swamp := range []bool{false, true} {
			state, callbacks, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
			require.NotNil(t, state)
			identity := state.CombatScene.GetCurrentTurnIdentity()
			character := state.PartyState.GetCharacter(identity.EntityIndex)
			character.SetStatus(status)
			character.SetHp(100)
			position := state.CombatScene.GetCombatantPosition(identity)
			state.CombatScene.CombatMapRef.SetTileNumber(position, indexes.BrickFloor)
			if swamp {
				state.CombatScene.CombatMapRef.SetTileNumber(position, indexes.Swamp)
			} else {
				require.True(t, state.CombatScene.AddField(NewCombatField(CombatFieldPoison, position)))
			}
			require.True(t, state.TransformActivePartyMemberIntoRat(identity.EntityIndex))
			calls := state.GetRNG().Calls()
			callbacks.Reset()
			state.resolveCombatPCStandingEffects(identity)
			assert.Equal(t, status, character.GetStatus())
			assert.Equal(t, uint16(100), character.GetCurrentHp())
			assert.Equal(t, calls, state.GetRNG().Calls())
			assert.Empty(t, callbacks.CombatEvents)
			state.applyCombatFieldToPC(NewCombatField(CombatFieldPoison, position), identity, true)
			assert.True(t, character.IsPoisoned())
			if status == party_state.Poisoned {
				assert.Greater(t, state.GetRNG().Calls(), calls)
			}
		}
	}
}

func TestFindingsReturnedBowOverflowIsDiscarded(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.Britain).WithSystemCallbacks().Build()
	state.PartyState.Inventory.Equipment.Set(references.Crossbow, 99)
	character := state.PartyState.GetCharacter(0)
	character.SetEquippedItem(references.SlotWeapon, references.Crossbow)
	state.unreadyAllCombatWeapons(references.Crossbow)
	assert.Equal(t, uint16(99), state.PartyState.Inventory.Equipment.Get(references.Crossbow))
	assert.Equal(t, references.NoEquipment, character.GetEquippedItem(references.SlotWeapon))
}

func TestFindingsTownCannonStopsAtWallAndProtectsShadowlords(t *testing.T) {
	for _, test := range []struct {
		name string
		wall bool
		lord bool
	}{
		{name: "clear ordinary target"},
		{name: "wall protects ordinary target", wall: true},
		{name: "Shadowlord immune", lord: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, _ := newShadowlordSpawnTestState(t)
			cannon := references.Position{X: 10, Y: 10}
			blocker := references.Position{X: 11, Y: 10}
			target := references.Position{X: 12, Y: 10}
			for x := references.Coordinate(11); x <= 14; x++ {
				position := references.Position{X: x, Y: 10}
				state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, &position, indexes.BrickFloor)
			}
			if test.wall {
				state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, &blocker, indexes.StoneBrickWall)
			}
			if test.lord {
				addPhysicalShadowlord(t, state, references.Faulinei, target)
			} else {
				addSpawnBlocker(t, state, target)
			}
			endpoint, door, victim := state.findSmallMapCannonTarget(cannon, references.Right)
			assert.False(t, door)
			if test.wall {
				assert.Equal(t, blocker, endpoint)
			} else {
				assert.Equal(t, target, endpoint)
			}
			if test.wall || test.lord {
				assert.Nil(t, victim)
			} else {
				require.NotNil(t, victim)
			}
			state.StartSmallMapCannonFireWorkflow(cannon, endpoint, references.Right, door, victim)
			drainGameplayWorkflows(t, state)
			remaining := state.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionAndFloorOrNil(target, state.MapState.PlayerLocation.Floor)
			if test.wall || test.lord {
				require.NotNil(t, remaining)
			} else {
				assert.Nil(t, remaining)
			}
		})
	}
}

func TestFindingsPurpleCloneProtectionIsEntryLocal(t *testing.T) {
	state, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	require.NotNil(t, state)
	source := state.CombatScene.GetCurrentTurnIdentity()
	character := state.PartyState.GetCharacter(source.EntityIndex)
	character.SetStatus(party_state.Good)
	clone := state.CombatScene.addPCCloneFromSource(source, source.EntityIndex,
		references.Position{X: 1, Y: 1}, NewCombatPCProvider(character))
	state.CombatScene.CurrentTurnIdentity = clone
	state.CombatScene.currentTurnIdentitySet = true
	require.True(t, state.TransformActivePartyMemberIntoRat(source.EntityIndex))
	state.applyCombatFieldToPC(NewCombatField(CombatFieldPoison, references.Position{X: 1, Y: 1}), clone, false)
	assert.True(t, character.IsGood())
	state.applyCombatFieldToPC(NewCombatField(CombatFieldPoison, state.CombatScene.GetCombatantPosition(source)), source, false)
	assert.True(t, character.IsPoisoned(), "the untransformed source shares roster status but not Purple protection")
}

func TestFindingsBlackPotionReplacesPurpleButInvisibilitySpellDoesNot(t *testing.T) {
	state, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	require.NotNil(t, state)
	identity := state.CombatScene.GetCurrentTurnIdentity()
	character := state.PartyState.GetCharacter(identity.EntityIndex)
	character.SetStatus(party_state.Good)
	require.True(t, state.TransformActivePartyMemberIntoRat(identity.EntityIndex))
	require.True(t, state.CastCombatSanctLor(identity.EntityIndex))
	assert.True(t, state.CombatScene.IsCombatPCRat(identity))
	state.PartyState.Inventory.Potions.Set(references.Black, 1)
	commit, _ := state.BeginPotionUse(references.Black, UsageCombat)
	require.NotNil(t, commit)
	outcome, ok := state.CommitPotionUse(commit)
	require.True(t, ok)
	state.SetRandomSeed(42)
	require.Equal(t, PotionEffectInvisible, state.CommitPotionEffect(outcome, identity.EntityIndex))
	assert.False(t, state.CombatScene.IsCombatPCRat(identity))
	state.applyCombatFieldToPC(NewCombatField(CombatFieldPoison, state.CombatScene.GetCombatantPosition(identity)), identity, false)
	assert.True(t, character.IsPoisoned())
}
