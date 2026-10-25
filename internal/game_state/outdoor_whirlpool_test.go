package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func addOutdoorWhirlpoolTestAttacker(t *testing.T, gs *GameState, pirate bool, pos references.Position, slot int) *map_units.NPCEnemy {
	t.Helper()
	var enemy *map_units.NPCEnemy
	if pirate {
		enemy = addPirateTestEnemy(t, gs, pos, indexes.PirateShipLeft)
	} else {
		enemy = addOverworldTestEnemy(t, gs, references.MonsterSeaSerpent, pos)
	}
	enemy.MapUnitDetails().NPCNum = slot

	return enemy
}

func TestOutdoorWhirlpoolFootHitPrecedesEveryLowerSlotAttack(t *testing.T) {
	for _, pirate := range []bool{false, true} {
		t.Run(fmt.Sprintf("pirate%v", pirate), func(t *testing.T) {
			gs, mock := newOutdoorFireballFixture(t, 0, references.NoPartyVehicle)
			origin := gs.MapState.PlayerLocation
			addOutdoorWhirlpoolTestAttacker(t, gs, pirate, references.Position{X: 100, Y: 98}, 1)
			// Two contacts must both commit, even though the first already owns a
			// presentation workflow. Neither may coalesce authoritative damage.
			for i, pos := range []references.Position{{X: 101, Y: 100}, {X: 99, Y: 100}} {
				require.True(t, gs.spawnWhirlpoolAt(pos))
				gs.GetWhirlpoolAt(pos).MapUnitDetails().NPCNum = 31 - i
			}
			seed := outdoorFireballSeed(t, func(rng *beradrng.Source) bool {
				rng.Intn(8)
				rng.Intn(8)
				return pirate || rng.Intn(8) == 0
			})
			gs.SetRandomSeed(seed)
			want := beradrng.New(seed)
			hpBeforeRanged := 100 - (want.Intn(8) + 1) - (want.Intn(8) + 1)
			if !pirate {
				want.Intn(8)
			}
			wantHP := hpBeforeRanged - (want.Intn(8) + 1)

			gs.largeMapProcessTurnEffects()

			assert.Equal(t, uint16(wantHP), gs.PartyState.GetCharacter(0).GetCurrentHp(), "all three hits commit before presentation")
			assert.Equal(t, want.Calls(), gs.GetRNG().Calls())
			require.Len(t, gs.pirateAttackQueue, 1)
			require.Len(t, gs.pirateAttackQueue[0].result.Hit.MemberDamage, 1)
			assert.Equal(t, uint16(hpBeforeRanged), gs.pirateAttackQueue[0].result.Hit.MemberDamage[0].HPBefore)
			assert.True(t, gs.EventQueueManager.HasWorkflow(outdoorWhirlpoolWorkflowName(31)))
			assert.True(t, gs.EventQueueManager.HasWorkflow(outdoorWhirlpoolWorkflowName(30)))
			assert.Equal(t, origin, gs.MapState.PlayerLocation)
			assert.Empty(t, mock.ProjectilesStarted)

			drainGameplayWorkflows(t, gs)

			assert.Equal(t, uint16(wantHP), gs.PartyState.GetCharacter(0).GetCurrentHp(), "captured whirlpool feedback cannot hit again")
			assert.Equal(t, want.Calls(), gs.GetRNG().Calls())
			assert.Len(t, mock.ProjectilesStarted, 1)
			assert.Equal(t, origin, gs.MapState.PlayerLocation)
			assert.False(t, mock.InputBlocked)
		})
	}
}

func TestOutdoorWhirlpoolFatalFootHitCannotBeReorderedAfterPirate(t *testing.T) {
	gs, mock := newOutdoorFireballFixture(t, 0, references.NoPartyVehicle)
	gs.PartyState.GetCharacter(0).SetHp(1)
	addOutdoorWhirlpoolTestAttacker(t, gs, true, references.Position{X: 100, Y: 98}, 1)
	pos := references.Position{X: 101, Y: 100}
	require.True(t, gs.spawnWhirlpoolAt(pos))
	gs.GetWhirlpoolAt(pos).MapUnitDetails().NPCNum = 31
	deaths := 0
	gs.SystemCallbacks.Flow.Consequences.TriggerDeathSequence = func() { deaths++ }

	gs.largeMapProcessTurnEffects()

	assert.True(t, gs.PartyState.IsAllPartyDead())
	assert.Equal(t, uint64(1), gs.GetRNG().Calls(), "pirate must see a dead party, not consume its damage roll first")
	require.Len(t, gs.pirateAttackQueue, 1)
	assert.Empty(t, gs.pirateAttackQueue[0].result.Hit.MemberDamage)
	assert.False(t, gs.pirateAttackQueue[0].result.Hit.PartyWiped)
	drainGameplayWorkflows(t, gs)
	assert.Equal(t, 1, deaths)
	assert.Len(t, mock.ProjectilesStarted, 1)
}

// Seed a separate destination controller without running a transition or turn.
func prepareOutdoorWhirlpoolDestination(t *testing.T, gs *GameState, pirate bool) (*map_units.NPCEnemy, *map_units.NPCEnemy) {
	t.Helper()
	before, controller := gs.MapState.PlayerLocation, gs.CurrentNPCAIController
	gs.MapState.PlayerLocation.Floor = -1
	gs.MapState.PlayerLocation.Position = references.Position{X: 34, Y: 18}
	gs.CurrentNPCAIController = gs.GetCurrentLargeMapNPCAIController()
	gs.CurrentNPCAIController.GetNpcs().ReplaceWith(nil)
	fillOutdoorFireballTerrain(gs, indexes.Water2)
	lower := addOutdoorWhirlpoolTestAttacker(t, gs, pirate, references.Position{X: 34, Y: 16}, 1)
	higher := addOutdoorWhirlpoolTestAttacker(t, gs, pirate, references.Position{X: 34, Y: 15}, 31)
	gs.MapState.PlayerLocation, gs.CurrentNPCAIController = before, controller

	return lower, higher
}

func TestOutdoorWhirlpoolTransportResumesLowerSlotsOnCurrentWorld(t *testing.T) {
	for _, pirate := range []bool{false, true} {
		for _, sinks := range []bool{false, true} {
			t.Run(fmt.Sprintf("pirate%v/sinks%v", pirate, sinks), func(t *testing.T) {
				gs, mock := newOutdoorFireballFixture(t, 0, references.FrigateVehicle)
				hull := 99
				if sinks {
					hull = 1
				}
				gs.PartyVehicle.GetVehicleDetails().SetHullHP(hull)
				gs.PartyVehicle.GetVehicleDetails().SetSkiffQuantity(1)
				lower, higher := prepareOutdoorWhirlpoolDestination(t, gs, pirate)
				oldController := gs.CurrentNPCAIController
				oldEnemy := addOutdoorWhirlpoolTestAttacker(t, gs, pirate, references.Position{X: 100, Y: 98}, 1)
				// An old-floor actor near the DESTINATION must not be carried across either.
				oldAtDestination := addOutdoorWhirlpoolTestAttacker(t, gs, pirate, references.Position{X: 34, Y: 17}, 2)
				pos := references.Position{X: 101, Y: 100}
				require.True(t, gs.spawnWhirlpoolAt(pos))
				gs.GetWhirlpoolAt(pos).MapUnitDetails().NPCNum = 20
				seed := outdoorFireballSeed(t, func(rng *beradrng.Source) bool {
					rng.Intn(30)
					return pirate || rng.Intn(8) == 0
				})
				gs.SetRandomSeed(seed)
				want := beradrng.New(seed)
				whirlpoolDamage := want.Intn(30) + 1
				if !pirate {
					want.Intn(8)
				}
				wantHP, wantHull := 100, hull-whirlpoolDamage
				if sinks {
					wantHP -= want.Intn(8) + 1
				} else {
					wantHull -= want.Intn(30) + 1
				}
				gs.largeMapProcessTurnEffects()
				assert.Zero(t, gs.GetRNG().Calls())
				assert.Empty(t, gs.pirateAttackQueue)
				assert.Equal(t, uint16(100), gs.PartyState.GetCharacter(0).GetCurrentHp())
				assert.Equal(t, hull, gs.PartyVehicle.GetVehicleDetails().GetHullHP())
				higherBefore, lowerBefore := higher.Pos(), lower.Pos()
				drainGameplayWorkflows(t, gs)

				assert.Equal(t, references.FloorNumber(-1), gs.MapState.PlayerLocation.Floor)
				assert.Equal(t, references.Position{X: 34, Y: 18}, gs.MapState.PlayerLocation.Position)
				assert.NotSame(t, oldController, gs.CurrentNPCAIController)
				assert.Equal(t, uint16(wantHP), gs.PartyState.GetCharacter(0).GetCurrentHp())
				if sinks {
					assert.Equal(t, references.SkiffVehicle, gs.PartyVehicle.GetVehicleDetails().VehicleType)
				} else {
					assert.Equal(t, wantHull, gs.PartyVehicle.GetVehicleDetails().GetHullHP())
				}
				assert.Equal(t, want.Calls(), gs.GetRNG().Calls(), "resume must not repeat generation, gates, or higher slots")
				require.Len(t, mock.ProjectilesStarted, 1)
				assert.Equal(t, lowerBefore, mock.ProjectilesStarted[0].FromPos)
				assert.Equal(t, gs.MapState.PlayerLocation.Position, mock.ProjectilesStarted[0].ToPos)
				assert.Equal(t, higherBefore, higher.Pos())
				assert.Equal(t, lowerBefore, lower.Pos(), "prior attack still suppresses movement")
				assert.Equal(t, references.FloorNumber(0), oldEnemy.Floor())
				assert.Equal(t, references.FloorNumber(0), oldAtDestination.Floor())
				assert.False(t, mock.InputBlocked)
			})
		}
	}
}

func TestOutdoorWhirlpoolTransportCancellationDoesNotResumeActors(t *testing.T) {
	for _, committed := range []bool{false, true} {
		t.Run(fmt.Sprintf("committed%v", committed), func(t *testing.T) {
			gs, mock := newOutdoorFireballFixture(t, 0, references.FrigateVehicle)
			gs.PartyVehicle.GetVehicleDetails().SetHullHP(99)
			prepareOutdoorWhirlpoolDestination(t, gs, true)
			pos := references.Position{X: 101, Y: 100}
			require.True(t, gs.spawnWhirlpoolAt(pos))
			gs.GetWhirlpoolAt(pos).MapUnitDetails().NPCNum = 20
			gs.largeMapProcessTurnEffects()
			if committed {
				for updates := 0; gs.MapState.PlayerLocation.Floor == 0; updates++ {
					require.Less(t, updates, 100)
					gs.ElapsedMs += 100
					gs.EventQueueManager.Update()
				}
				assert.Equal(t, uint64(1), gs.GetRNG().Calls())
			} else {
				assert.Zero(t, gs.GetRNG().Calls())
			}
			calls, hull := gs.GetRNG().Calls(), gs.PartyVehicle.GetVehicleDetails().GetHullHP()
			gs.EventQueueManager.Shutdown()
			assert.Equal(t, calls, gs.GetRNG().Calls())
			assert.Equal(t, hull, gs.PartyVehicle.GetVehicleDetails().GetHullHP())
			assert.Empty(t, gs.pirateAttackQueue)
			assert.Empty(t, mock.ProjectilesStarted)
			assert.False(t, mock.InputBlocked)
		})
	}
}

func TestOutdoorWhirlpoolSameWorldTransportRetainsDescendingContinuation(t *testing.T) {
	gs, mock := newOutdoorFireballFixture(t, -1, references.FrigateVehicle)
	gs.PartyVehicle.GetVehicleDetails().SetHullHP(99)
	controller := gs.CurrentNPCAIController
	origin := gs.MapState.PlayerLocation.Position
	gs.MapState.PlayerLocation.Position = references.Position{X: 34, Y: 18}
	fillOutdoorFireballTerrain(gs, indexes.Water2)
	attacker := addOutdoorWhirlpoolTestAttacker(t, gs, true, references.Position{X: 34, Y: 16}, 1)
	// A second, lower-slot whirlpool at the destination must suspend/resume the
	// same phase again, even though this transport does not change controllers.
	second := references.Position{X: 35, Y: 18}
	require.True(t, gs.spawnWhirlpoolAt(second))
	gs.GetWhirlpoolAt(second).MapUnitDetails().NPCNum = 10
	gs.MapState.PlayerLocation.Position = origin
	first := references.Position{X: 101, Y: 100}
	require.True(t, gs.spawnWhirlpoolAt(first))
	gs.GetWhirlpoolAt(first).MapUnitDetails().NPCNum = 20
	gs.SetRandomSeed(123)
	want := beradrng.New(123)
	wantHull := 99 - (want.Intn(30) + 1) - (want.Intn(30) + 1) - (want.Intn(30) + 1)

	gs.largeMapProcessTurnEffects()

	assert.Zero(t, gs.GetRNG().Calls())
	assert.True(t, gs.EventQueueManager.HasWorkflow(outdoorWhirlpoolWorkflowName(20)))
	drainGameplayWorkflows(t, gs)
	assert.Same(t, controller, gs.CurrentNPCAIController)
	assert.Equal(t, references.Position{X: 34, Y: 18}, gs.MapState.PlayerLocation.Position)
	assert.Equal(t, wantHull, gs.PartyVehicle.GetVehicleDetails().GetHullHP())
	assert.Equal(t, want.Calls(), gs.GetRNG().Calls())
	assert.Nil(t, gs.GetWhirlpoolAt(first))
	assert.Nil(t, gs.GetWhirlpoolAt(second))
	require.Len(t, mock.ProjectilesStarted, 1)
	assert.Equal(t, attacker.Pos(), mock.ProjectilesStarted[0].FromPos)
	assert.Equal(t, 2, countEqual(mock.Messages, "WHIRLPOOL!"))
	assert.False(t, mock.InputBlocked)
}
