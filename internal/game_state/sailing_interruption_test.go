package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func TestFurlStopsAutomaticSailingBeforeFeedback(t *testing.T) {
	state := newSourceHitFixture(t, references.FrigateVehicle, 100)
	details := state.PartyVehicle.GetVehicleDetails()
	details.SetSailState(map_units.SailRaised)
	require.True(t, state.StartAutomaticSailing(references.Left))
	feedback := 0
	state.SystemCallbacks.UI.ShowActionFeedback = func(message string, _ bool) {
		feedback++
		assert.Equal(t, "FURL!", message)
		assert.False(t, state.IsSailing())
		assert.Equal(t, map_units.SailFurled, details.SailState)
		assert.Equal(t, references.Left, details.CurrentDirection())
	}
	beforeTime, beforeRNG := state.DateTime, state.GetRNG().Calls()
	require.True(t, state.ActionYellLargeMap())
	for range 5 {
		state.CompleteAutomaticSailingWait()
	}
	assert.Equal(t, 1, feedback)
	assert.Equal(t, beforeTime, state.DateTime)
	assert.Equal(t, beforeRNG, state.GetRNG().Calls())
	assert.EqualValues(t, 1, state.AutomaticSailingStopEpoch())
	state.StopAutomaticSailing()
	assert.EqualValues(t, 1, state.AutomaticSailingStopEpoch(), "cleanup is idempotent")
}

func TestAutomaticSailingInvalidationClearsStateWithoutPhaseWork(t *testing.T) {
	for _, test := range []struct {
		name       string
		invalidate func(*GameState)
	}{
		{name: "furled", invalidate: func(state *GameState) {
			state.PartyVehicle.GetVehicleDetails().SetSailState(map_units.SailFurled)
		}},
		{name: "skiff", invalidate: func(state *GameState) {
			state.PartyVehicle = *createTestSkiff(50, 50)
		}},
		{name: "town", invalidate: func(state *GameState) {
			state.MapState.PlayerLocation.Location = references.Britain
		}},
		{name: "combat prepared", invalidate: func(state *GameState) {
			state.CombatScene = &CombatState{}
		}},
		{name: "party dead", invalidate: func(state *GameState) {
			state.PartyState.GetCharacter(0).SubtractHp(100)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := newSourceHitFixture(t, references.FrigateVehicle, 100)
			state.PartyVehicle.GetVehicleDetails().SetSailState(map_units.SailRaised)
			require.True(t, state.StartAutomaticSailing(references.Up))
			test.invalidate(state)
			require.True(t, state.IsSailing(), "query must expose, not conceal, stale authority")
			beforeTime, beforeRNG := state.DateTime, state.GetRNG().Calls()
			beforePosition := state.MapState.PlayerLocation
			state.CompleteAutomaticSailingWait()
			assert.False(t, state.IsSailing())
			assert.False(t, state.ContinueAutomaticSailing())
			assert.Equal(t, beforeTime, state.DateTime)
			assert.Equal(t, beforeRNG, state.GetRNG().Calls())
			assert.Equal(t, beforePosition, state.MapState.PlayerLocation)
		})
	}
}

func TestEnemyRangedHitInterruptsOnlyActiveSailingAndPreservesSails(t *testing.T) {
	for _, sails := range []map_units.SailState{map_units.SailRaised, map_units.SailFurled} {
		t.Run(sails.String(), func(t *testing.T) {
			state, callbacks := newOutdoorFireballFixture(t, 0, references.FrigateVehicle)
			details := state.PartyVehicle.GetVehicleDetails()
			details.SetSailState(sails)
			details.SetPartyVehicleDirection(references.Left)
			if sails == map_units.SailRaised {
				require.True(t, state.StartAutomaticSailing(references.Left))
			}
			pirate := addPirateTestEnemy(t, state, references.Position{X: 100, Y: 98}, indexes.PirateShipLeft)
			state.FirePirateCannon(pirate, state.MapState.PlayerLocation.Position)
			assert.False(t, state.IsSailing())
			assert.Equal(t, sails, details.SailState)
			assert.Equal(t, references.Left, details.CurrentDirection())
			assert.Positive(t, details.GetHullHP())
			wantNotices := 0
			if sails == map_units.SailRaised {
				wantNotices = 1
			}
			assert.Equal(t, wantNotices, countMessage(callbacks.Messages, "Under attack! Sailing interrupted."))
			state.FirePirateCannon(pirate, state.MapState.PlayerLocation.Position)
			assert.Equal(t, wantNotices, countMessage(callbacks.Messages, "Under attack! Sailing interrupted."))
			drainGameplayWorkflows(t, state)
			assert.Len(t, callbacks.ProjectilesStarted, 2)
		})
	}
}

func TestPirateWarningAndBlockedShotKeepAutomaticSailing(t *testing.T) {
	for _, blocked := range []bool{false, true} {
		state, callbacks := newOutdoorFireballFixture(t, 0, references.FrigateVehicle)
		state.PartyVehicle.GetVehicleDetails().SetSailState(map_units.SailRaised)
		require.True(t, state.StartAutomaticSailing(references.Up))
		sprite := indexes.SpriteIndex(indexes.PirateShipDown)
		if blocked {
			sprite = indexes.PirateShipLeft
			wall := references.Position{X: 100, Y: 99}
			state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &wall, indexes.StoneBrickWall)
		}
		pirate := addPirateTestEnemy(t, state, references.Position{X: 100, Y: 98}, sprite)
		require.True(t, state.checkPirateActorAttack(pirate, state.MapState.PlayerLocation.Position))
		assert.True(t, state.IsSailing())
		assert.Zero(t, state.AutomaticSailingStopEpoch())
		assert.NotContains(t, callbacks.Messages, "Under attack! Sailing interrupted.")
		assert.Zero(t, state.GetRNG().Calls())
	}
}

func TestOutdoorFireballSailingInterruptionRequiresSuccessfulHit(t *testing.T) {
	for _, monster := range []references.MonsterType{references.MonsterDragon, references.MonsterSeaSerpent} {
		for _, outcome := range []string{"hit", "blocked", "no shot"} {
			t.Run(monster.Name()+"/"+outcome, func(t *testing.T) {
				state, callbacks := newOutdoorFireballFixture(t, 0, references.FrigateVehicle)
				details := state.PartyVehicle.GetVehicleDetails()
				details.SetSailState(map_units.SailRaised)
				require.True(t, state.StartAutomaticSailing(references.Right))
				enemy := addOverworldTestEnemy(t, state, monster, references.Position{X: 102, Y: 100})
				roll := 0
				if outcome == "no shot" {
					roll = 1
				}
				state.SetRandomSeed(sourceHitSeedForFirstIntn(t, outdoorFireballOdds, roll))
				if outcome == "blocked" {
					wall := references.Position{X: 101, Y: 100}
					state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &wall, indexes.StoneBrickWall)
				}
				state.checkOutdoorFireball(enemy, state.MapState.PlayerLocation.Position)
				assert.Equal(t, outcome != "hit", state.IsSailing())
				assert.Equal(t, map_units.SailRaised, details.SailState)
				assert.Equal(t, references.Right, details.CurrentDirection())
				assert.Equal(t, outcome == "hit", countMessage(callbacks.Messages, "Under attack! Sailing interrupted.") == 1)
			})
		}
	}
}

func TestSinkingStopsBeforeFallbackOrDrowningCallbacks(t *testing.T) {
	for _, fallback := range []references.VehicleType{references.SkiffVehicle, references.CarpetVehicle, references.NoPartyVehicle} {
		t.Run(fallback.GetLowerCaseName(), func(t *testing.T) {
			state := newSourceHitFixture(t, references.FrigateVehicle, 1)
			details := state.PartyVehicle.GetVehicleDetails()
			details.SetSailState(map_units.SailRaised)
			details.SetHullHP(1)
			if fallback == references.SkiffVehicle {
				details.SetSkiffQuantity(1)
			} else if fallback == references.CarpetVehicle {
				state.PartyState.Inventory.QuestItems.Set(references.Carpet, 1)
			}
			require.True(t, state.StartAutomaticSailing(references.Down))
			hit := state.ResolveSourceHit()
			assert.False(t, state.IsSailing())
			assert.Equal(t, fallback, hit.VehicleAfter)
			beforeTime, beforeRNG := state.DateTime, state.GetRNG().Calls()
			state.CompleteAutomaticSailingWait()
			assert.Equal(t, beforeTime, state.DateTime)
			assert.Equal(t, beforeRNG, state.GetRNG().Calls())
		})
	}
}

func TestOutdoorFireballSinkingStopsForEveryFallback(t *testing.T) {
	for _, monster := range []references.MonsterType{references.MonsterDragon, references.MonsterSeaSerpent} {
		for _, fallback := range []references.VehicleType{references.SkiffVehicle, references.CarpetVehicle, references.NoPartyVehicle} {
			t.Run(monster.Name()+"/"+fallback.GetLowerCaseName(), func(t *testing.T) {
				state, callbacks := newOutdoorFireballFixture(t, 0, references.FrigateVehicle)
				details := state.PartyVehicle.GetVehicleDetails()
				details.SetSailState(map_units.SailRaised)
				details.SetHullHP(1)
				details.SetSkiffQuantity(0)
				state.PartyState.GetCharacter(0).SetHp(1)
				state.PartyState.Inventory.QuestItems.Set(references.Carpet, 0)
				if fallback == references.SkiffVehicle {
					details.SetSkiffQuantity(1)
				} else if fallback == references.CarpetVehicle {
					state.PartyState.Inventory.QuestItems.Set(references.Carpet, 1)
				}
				require.True(t, state.StartAutomaticSailing(references.Up))
				enemy := addOverworldTestEnemy(t, state, monster, references.Position{X: 102, Y: 101})
				state.SetRandomSeed(sourceHitSeedForFirstIntn(t, outdoorFireballOdds, 0))
				require.True(t, state.checkOutdoorFireball(enemy, state.MapState.PlayerLocation.Position))
				assert.False(t, state.IsSailing())
				assert.Equal(t, fallback, state.PartyVehicle.GetVehicleDetails().VehicleType)
				assert.Equal(t, fallback == references.NoPartyVehicle, state.PartyState.IsAllPartyDead())
				assert.Equal(t, 1, countMessage(callbacks.Messages, "Under attack! Sailing interrupted."))
				rng := state.GetRNG().Calls()
				drainGameplayWorkflows(t, state)
				assert.Equal(t, rng, state.GetRNG().Calls())
				assert.Len(t, callbacks.ProjectilesStarted, 1)
			})
		}
	}
}

func TestInterruptedOutdoorPhaseRetainsEveryLowerSlotHit(t *testing.T) {
	for _, lethal := range []bool{false, true} {
		state, callbacks := newOutdoorFireballFixture(t, 0, references.FrigateVehicle)
		details := state.PartyVehicle.GetVehicleDetails()
		details.SetSailState(map_units.SailRaised)
		details.SetSkiffQuantity(1)
		require.True(t, state.StartAutomaticSailing(references.Left))
		positions := []references.Position{{X: 100, Y: 98}, {X: 98, Y: 100}, {X: 100, Y: 102}}
		for index, position := range positions {
			sprite := indexes.SpriteIndex(indexes.PirateShipLeft)
			if index == 1 {
				sprite = indexes.PirateShipDown
			}
			pirate := addPirateTestEnemy(t, state, position, sprite)
			pirate.MapUnitDetails().NPCNum = 30 - index
		}
		const seed = uint64(464)
		want := beradrng.New(seed)
		firstDamage := want.Intn(30) + 1
		want.Intn(30)
		partyDamage := want.Intn(8) + 1
		details.SetHullHP(firstDamage + 1)
		member := state.PartyState.GetCharacter(0)
		if lethal {
			member.SetHp(1)
		}
		hpBefore := member.GetCurrentHp()
		state.SetRandomSeed(seed)
		state.largeMapProcessTurnEffects()
		require.Len(t, state.pirateAttackQueue, 3)
		for index, position := range positions {
			assert.Equal(t, position, state.pirateAttackQueue[index].result.Source)
		}
		assert.Equal(t, SourceHitFrigateDamaged, state.pirateAttackQueue[0].result.Hit.Outcome)
		assert.Equal(t, SourceHitEscapedBySkiff, state.pirateAttackQueue[1].result.Hit.Outcome)
		assert.Equal(t, SourceHitPartyDamaged, state.pirateAttackQueue[2].result.Hit.Outcome)
		assert.Equal(t, uint16(max(0, int(hpBefore)-partyDamage)), member.GetCurrentHp())
		assert.Equal(t, want.Calls(), state.GetRNG().Calls())
		assert.False(t, state.IsSailing())
		assert.Equal(t, 1, countMessage(callbacks.Messages, "Under attack! Sailing interrupted."))
		drainGameplayWorkflows(t, state)
		assert.Len(t, callbacks.ProjectilesStarted, 3)
		assert.Equal(t, want.Calls(), state.GetRNG().Calls(), "presentation must not replay the actor phase")
	}
}

func TestTacticalEntryAndNormalReturnFurlStationaryAndSailingFrigates(t *testing.T) {
	for _, route := range []string{"enemy contact", "player attack", "direct transition"} {
		for _, sailing := range []bool{false, true} {
			t.Run(route+"/"+map[bool]string{false: "stationary", true: "sailing"}[sailing], func(t *testing.T) {
				state, _ := newOutdoorFireballFixture(t, 0, references.FrigateVehicle)
				details := state.PartyVehicle.GetVehicleDetails()
				details.SetSailState(map_units.SailRaised)
				details.SetPartyVehicleDirection(references.Left)
				if sailing {
					require.True(t, state.StartAutomaticSailing(references.Left))
				}
				enemy := addOverworldTestEnemy(t, state, references.MonsterSeaSerpent, references.Position{X: 101, Y: 100})
				switch route {
				case "enemy contact":
					require.NoError(t, state.StartOverworldCombatEncounter(enemy))
				case "player attack":
					require.True(t, state.ActionAttackLargeMap(references.Right))
				case "direct transition":
					require.NoError(t, state.TransitionToCombatMap(int(references.ShipSeaCombatMap), references.BritanniaCombatTerritory, CombatEntrySouth))
				}
				require.NotNil(t, state.CombatScene)
				assert.False(t, state.IsSailing())
				assert.Equal(t, map_units.SailFurled, details.SailState)
				assert.Equal(t, references.Left, details.CurrentDirection())
				state.EndCombat(CombatDefeat)
				assert.False(t, state.IsSailing())
				assert.Equal(t, map_units.SailFurled, state.PartyVehicle.GetVehicleDetails().SailState)
			})
		}
	}
}

func TestRejectedTacticalEntryDoesNotFurlOrStopSailing(t *testing.T) {
	state := newSourceHitFixture(t, references.FrigateVehicle, 100)
	state.PartyVehicle.GetVehicleDetails().SetSailState(map_units.SailRaised)
	require.True(t, state.StartAutomaticSailing(references.Right))
	require.Error(t, state.StartOverworldCombatEncounter(nil))
	assert.True(t, state.IsSailing())
	assert.Equal(t, map_units.SailRaised, state.PartyVehicle.GetVehicleDetails().SailState)
	assert.Equal(t, references.Right, state.PartyVehicle.GetVehicleDetails().CurrentDirection())
}
