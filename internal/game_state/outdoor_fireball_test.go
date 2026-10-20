package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func newOutdoorFireballFixture(t *testing.T, floor references.FloorNumber, vehicle references.VehicleType) (*GameState, *MockSystemCallbacks) {
	t.Helper()
	gs, mock := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).
		WithFloor(floor).WithPlayerAt(100, 100).WithSystemCallbacks().Build()
	gs.CurrentNPCAIController = gs.GetCurrentLargeMapNPCAIController()
	gs.CurrentNPCAIController.RemoveAllEnemies()
	gs.DebugOptions.MonsterGen = false
	gs.DateTime = datetime.UltimaDate{Year: 1, Month: 1, Day: 1, Hour: 12}
	gs.MapState.Lighting.ExtinguishTorch()
	gs.MapState.Lighting.SetMagicLight(false)
	gs.MagicLightMinutesRemaining = 0
	gs.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(vehicle, gs.MapState.PlayerLocation.Position, floor)
	for i := range gs.PartyState.GetCharacterCount() {
		member := gs.PartyState.GetCharacter(i)
		member.PartyMap = party_state.PartyMapAtInn(references.Britain)
		member.SetStatus(party_state.Good)
		member.SetMaxHp(100)
		member.SetHp(100)
		member.SetEquippedItem(references.SlotRing, references.NoEquipment)
	}
	gs.PartyState.GetCharacter(0).PartyMap = party_state.PartyMapActive
	fillOutdoorFireballTerrain(gs, indexes.Water2)
	gs.SetRandomSeed(1)
	mock.Reset()

	return gs, mock
}

func fillOutdoorFireballTerrain(gs *GameState, tile indexes.SpriteIndex) {
	center := gs.MapState.PlayerLocation.Position
	for dy := references.Coordinate(-9); dy <= 9; dy++ {
		for dx := references.Coordinate(-9); dx <= 9; dx++ {
			pos := references.Position{X: center.X + dx, Y: center.Y + dy}
			pos = *pos.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
			gs.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &pos, tile)
		}
	}
}

func outdoorFireballSeed(t *testing.T, matches func(*beradrng.Source) bool) uint64 {
	t.Helper()
	for seed := uint64(0); seed < 100000; seed++ {
		if matches(beradrng.New(seed)) {
			return seed
		}
	}
	t.Fatal("no matching deterministic sequence")

	return 0
}

func TestOutdoorFireballWrappedSquareRangeAndOdds(t *testing.T) {
	for _, monster := range []references.MonsterType{references.MonsterSeaSerpent, references.MonsterDragon} {
		for _, floor := range []references.FloorNumber{0, -1} {
			for _, offset := range []references.Position{{X: 1, Y: 1}, {X: 2, Y: 1}, {X: 3, Y: 3}, {X: -3, Y: -2}, {X: 4}, {Y: -4}} {
				t.Run(fmt.Sprintf("%v/floor%d/%v", monster, floor, offset), func(t *testing.T) {
					gs, _ := newOutdoorFireballFixture(t, floor, references.NoPartyVehicle)
					gs.MapState.PlayerLocation.Position = references.Position{}
					fillOutdoorFireballTerrain(gs, indexes.Grass)
					inRange := absInt(int(offset.X)) <= 3 && absInt(int(offset.Y)) <= 3
					pos := offset
					pos = *pos.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
					enemy := addOverworldTestEnemy(t, gs, monster, pos)
					enemy.EnemyReference.DoNotMove = true
					gs.SetRandomSeed(sourceHitSeedForFirstIntn(t, 8, 0))
					gs.largeMapProcessTurnEffects()
					if inRange {
						require.Len(t, gs.pirateAttackQueue, 1)
						assert.True(t, gs.pirateAttackQueue[0].fireball)
						assert.Equal(t, PirateAttackHit, gs.pirateAttackQueue[0].result.Outcome)
						assert.Equal(t, uint64(2), gs.GetRNG().Calls())
					} else {
						assert.Empty(t, gs.pirateAttackQueue)
						assert.Zero(t, gs.GetRNG().Calls(), "out of range consumes no firing roll")
					}
					assert.Equal(t, pos, enemy.Pos())
					assert.NotNil(t, gs.CurrentNPCAIController.GetNpcs().GetVisibleEnemyAtPositionAndFloorOrNil(pos, floor), "wrapped cleanup retains nearby actors")
				})
			}
		}
	}
	for roll := range 8 {
		t.Run(fmt.Sprintf("roll%d", roll), func(t *testing.T) {
			gs, _ := newOutdoorFireballFixture(t, 0, references.NoPartyVehicle)
			enemy := addOverworldTestEnemy(t, gs, references.MonsterSeaSerpent, references.Position{X: 102, Y: 101})
			enemy.EnemyReference.DoNotMove = true
			gs.SetRandomSeed(sourceHitSeedForFirstIntn(t, 8, roll))
			gs.largeMapProcessTurnEffects()
			if roll == 0 {
				assert.Len(t, gs.pirateAttackQueue, 1)
				assert.Equal(t, uint64(2), gs.GetRNG().Calls())
			} else {
				assert.Empty(t, gs.pirateAttackQueue)
				assert.Equal(t, uint64(1), gs.GetRNG().Calls())
			}
		})
	}
}

func TestOutdoorFireballPassUnderworldCarpetContactAndRanged(t *testing.T) {
	for _, terrain := range []indexes.SpriteIndex{indexes.Water1, indexes.Water2, indexes.WaterCoast} {
		for _, diagonal := range []bool{false, true} {
			t.Run(fmt.Sprintf("terrain%d/diagonal%v", terrain, diagonal), func(t *testing.T) {
				gs, mock := newOutdoorFireballFixture(t, -1, references.CarpetVehicle)
				fillOutdoorFireballTerrain(gs, terrain)
				pos := references.Position{X: 101, Y: 100}
				if diagonal {
					pos.Y++
				}
				enemy := addOverworldTestEnemy(t, gs, references.MonsterSeaSerpent, pos)
				gs.PartyState.GetCharacter(1).PartyMap = party_state.PartyMapActive
				seed := outdoorFireballSeed(t, func(rng *beradrng.Source) bool {
					return rng.Intn(256) != 0x69 && rng.Intn(256) != 0x69 && (!diagonal || rng.Intn(8) == 0)
				})
				gs.SetRandomSeed(seed)
				want := beradrng.New(seed)
				want.Intn(256) // Underworld earthquake, first player update.
				passOutdoorFireballTurn(gs)
				assert.Equal(t, uint16(100), gs.PartyState.GetCharacter(0).GetCurrentHp(), "first mounted phase skips actors")
				assert.Equal(t, want.Calls(), gs.GetRNG().Calls())
				want.Intn(256)
				if diagonal {
					want.Intn(8)
				}
				wantHP := []uint16{uint16(100 - want.Intn(8) - 1), uint16(100 - want.Intn(8) - 1)}
				passOutdoorFireballTurn(gs)
				assert.Equal(t, wantHP[0], gs.PartyState.GetCharacter(0).GetCurrentHp())
				assert.Equal(t, wantHP[1], gs.PartyState.GetCharacter(1).GetCurrentHp())
				assert.Equal(t, want.Calls(), gs.GetRNG().Calls())
				assert.Nil(t, gs.CombatScene)
				assert.Equal(t, pos, enemy.Pos())
				drainGameplayWorkflows(t, gs)
				assert.Equal(t, wantHP[0], gs.PartyState.GetCharacter(0).GetCurrentHp(), "presentation cannot repeat damage")
				assert.NotContains(t, mock.Messages, "Rough seas!", "Pass does not enter deep water")
				assert.NotContains(t, mock.Messages, "* BOOOM! *")
				if diagonal {
					require.Len(t, mock.ProjectilesStarted, 1)
					assert.Equal(t, references.MissileRed, mock.ProjectilesStarted[0].ProjectileType)
					assert.NotContains(t, mock.Messages, "Attacked!")
					assert.Contains(t, mock.SoundEffectsPlayed, SoundCannonFire)
					assert.Contains(t, mock.SoundEffectsPlayed, SoundBroadside)
				} else {
					assert.Empty(t, mock.ProjectilesStarted)
					assert.Equal(t, 1, countMessage(mock.Messages, "Attacked!"))
				}
			})
		}
	}
}

func passOutdoorFireballTurn(gs *GameState) {
	gs.BeginSurfaceCommandRouting()
	gs.ActionPass()
	gs.EndSurfaceCommandRouting()
	gs.CompleteSurfacePassCycle(references.LargeMapType)
}

func TestOutdoorFireballSourceSampledObstruction(t *testing.T) {
	for _, test := range []struct {
		name                  string
		from, blocker         references.Position
		lit, occupied, hidden bool
		blocked               bool
	}{
		{name: "arbitrary angle sampled tile", from: references.Position{X: 102, Y: 101}, blocker: references.Position{X: 101, Y: 100}, blocked: true},
		{name: "arbitrary angle skipped tile", from: references.Position{X: 102, Y: 101}, blocker: references.Position{X: 101, Y: 101}},
		{name: "dark buffer permits unseen blocker", from: references.Position{X: 103, Y: 103}, blocker: references.Position{X: 102, Y: 102}},
		{name: "torch reveals sampled blocker", from: references.Position{X: 103, Y: 103}, blocker: references.Position{X: 102, Y: 102}, lit: true, blocked: true},
		{name: "displayed object clears buff0", from: references.Position{X: 103, Y: 103}, blocker: references.Position{X: 102, Y: 102}, lit: true, occupied: true},
		{name: "hidden object does not clear buff0", from: references.Position{X: 103, Y: 103}, blocker: references.Position{X: 102, Y: 102}, lit: true, occupied: true, hidden: true, blocked: true},
		{name: "shooter exempt", from: references.Position{X: 102, Y: 101}, blocker: references.Position{X: 102, Y: 101}, lit: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			gs, _ := newOutdoorFireballFixture(t, -1, references.NoPartyVehicle)
			gs.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &test.blocker, indexes.StoneBrickWall)
			if test.lit {
				gs.MapState.Lighting.LightTorch()
			}
			if test.occupied {
				enemy := addOverworldTestEnemy(t, gs, references.MonsterOrc, test.blocker)
				enemy.SetVisible(!test.hidden)
			}
			end, reached := gs.traceOutdoorMissile(test.from, gs.MapState.PlayerLocation.Position, references.MissileRed)
			assert.Equal(t, !test.blocked, reached)
			if test.blocked {
				assert.Equal(t, test.blocker, end)
			} else {
				assert.Equal(t, gs.MapState.PlayerLocation.Position, end)
			}
			assert.Zero(t, gs.GetRNG().Calls())
		})
	}
}

func TestOutdoorFireballMultipleActorsSourceOrderAndBlockedConsumption(t *testing.T) {
	gs, mock := newOutdoorFireballFixture(t, 0, references.CarpetVehicle)
	mover := addOverworldTestEnemy(t, gs, references.MonsterDragon, references.Position{X: 95, Y: 100})
	mover.MapUnitDetails().NPCNum = 1
	serpent := addOverworldTestEnemy(t, gs, references.MonsterSeaSerpent, references.Position{X: 99, Y: 101})
	serpent.MapUnitDetails().NPCNum = 5
	pirate := addPirateTestEnemy(t, gs, references.Position{X: 100, Y: 97}, indexes.PirateShipLeft)
	pirate.MapUnitDetails().NPCNum = 10
	contact := addOverworldTestEnemy(t, gs, references.MonsterSeaSerpent, references.Position{X: 101, Y: 100})
	contact.MapUnitDetails().NPCNum = 20
	blocked := addOverworldTestEnemy(t, gs, references.MonsterDragon, references.Position{X: 103, Y: 100})
	blocked.MapUnitDetails().NPCNum = 31
	wall := references.Position{X: 102, Y: 100}
	gs.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &wall, indexes.StoneBrickWall)
	seed := outdoorFireballSeed(t, func(rng *beradrng.Source) bool {
		if rng.Intn(8) != 0 {
			return false
		}
		rng.Intn(8) // Cardinal contact damage before the pirate.
		rng.Intn(8) // Pirate hit, no firing probability roll.
		return rng.Intn(8) == 0
	})
	gs.SetRandomSeed(seed)
	want := beradrng.New(seed)
	want.Intn(8)
	hp := 100 - (want.Intn(8) + 1) - (want.Intn(8) + 1)
	want.Intn(8)
	hp -= want.Intn(8) + 1
	gs.largeMapProcessTurnEffects()
	require.Len(t, gs.pirateAttackQueue, 3)
	assert.Equal(t, []references.Position{blocked.Pos(), pirate.Pos(), serpent.Pos()}, []references.Position{
		gs.pirateAttackQueue[0].result.Source, gs.pirateAttackQueue[1].result.Source, gs.pirateAttackQueue[2].result.Source,
	})
	assert.Equal(t, PirateAttackBlocked, gs.pirateAttackQueue[0].result.Outcome)
	assert.Equal(t, wall, gs.pirateAttackQueue[0].result.ProjectileEnd)
	assert.Equal(t, uint16(hp), gs.PartyState.GetCharacter(0).GetCurrentHp())
	assert.Equal(t, want.Calls(), gs.GetRNG().Calls())
	assert.Equal(t, references.Position{X: 95, Y: 100}, mover.Pos(), "blocked shot suppresses subsequent movement")
	assert.Equal(t, references.Position{X: 103, Y: 100}, blocked.Pos())
	assert.Equal(t, 1, countMessage(mock.Messages, "Attacked!"))
	drainGameplayWorkflows(t, gs)
	require.Len(t, mock.ProjectilesStarted, 3)
	assert.Equal(t, references.MissileRed, mock.ProjectilesStarted[0].ProjectileType)
	assert.Equal(t, references.MissileCannonBall, mock.ProjectilesStarted[1].ProjectileType)
	assert.Equal(t, references.MissileRed, mock.ProjectilesStarted[2].ProjectileType)
	assert.Equal(t, 1, countMessage(mock.Messages, "* BOOOM! *"))
	assert.Equal(t, want.Calls(), gs.GetRNG().Calls(), "feedback has no gameplay draws")
}

func TestOutdoorFireballVehiclesAndHullBounds(t *testing.T) {
	for _, vehicle := range []references.VehicleType{references.NoPartyVehicle, references.HorseVehicle, references.CarpetVehicle, references.SkiffVehicle, references.FrigateVehicle} {
		for _, maximum := range []bool{false, true} {
			t.Run(fmt.Sprintf("vehicle%d/maximum%v", vehicle, maximum), func(t *testing.T) {
				gs, _ := newOutdoorFireballFixture(t, -1, vehicle)
				addOverworldTestEnemy(t, gs, references.MonsterDragon, references.Position{X: 103, Y: 102})
				sides, damage := 8, 1
				if vehicle == references.FrigateVehicle {
					sides = 30
					gs.PartyVehicle.GetVehicleDetails().SetHullHP(99)
				}
				if maximum {
					damage = sides
				}
				seed := outdoorFireballSeed(t, func(rng *beradrng.Source) bool {
					return rng.Intn(8) == 0 && rng.Intn(sides) == damage-1
				})
				gs.SetRandomSeed(seed)
				gs.largeMapProcessTurnEffects()
				require.Len(t, gs.pirateAttackQueue, 1)
				if vehicle == references.FrigateVehicle {
					assert.Equal(t, 99-damage, gs.PartyVehicle.GetVehicleDetails().GetHullHP())
					assert.Equal(t, uint16(100), gs.PartyState.GetCharacter(0).GetCurrentHp())
				} else {
					assert.Equal(t, uint16(100-damage), gs.PartyState.GetCharacter(0).GetCurrentHp())
				}
				assert.Equal(t, uint64(2), gs.GetRNG().Calls())
			})
		}
	}
}

func TestOutdoorFireballActorCadenceAndCleanup(t *testing.T) {
	for _, quick := range []bool{false, true} {
		t.Run(fmt.Sprintf("quick%v", quick), func(t *testing.T) {
			gs, mock := newOutdoorFireballFixture(t, -1, references.CarpetVehicle)
			addOverworldTestEnemy(t, gs, references.MonsterSeaSerpent, references.Position{X: 101, Y: 101})
			gs.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectTimeStop, TurnsRemaining: 20}
			gs.SetRandomSeed(sourceHitSeedForFirstIntn(t, 8, 0))
			for range 4 {
				gs.processNonCombatWorldActorPhaseForCommand(true)
			}
			assert.Zero(t, gs.GetRNG().Calls())
			assert.False(t, gs.outdoorMountedActorSkip)
			assert.False(t, gs.quicknessWorldActorSkip)
			gs.ActiveDurationEffect = DurationEffect{}
			stride := 2
			if quick {
				gs.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectQuickness, TurnsRemaining: 20}
				stride = 4
			}
			for phase := 1; phase < stride; phase++ {
				gs.processNonCombatWorldActorPhaseForCommand(true)
				assert.Zero(t, gs.GetRNG().Calls())
			}
			gs.processNonCombatWorldActorPhaseForCommand(true)
			require.Len(t, gs.pirateAttackQueue, 1)
			assert.Equal(t, uint64(2), gs.GetRNG().Calls())
			hp := gs.PartyState.GetCharacter(0).GetCurrentHp()
			gs.EventQueueManager.Update()
			gs.EventQueueManager.Shutdown()
			assert.Empty(t, gs.pirateAttackQueue)
			assert.False(t, mock.InputBlocked)
			assert.Positive(t, mock.ProjectilesClearedCount)
			assert.Equal(t, hp, gs.PartyState.GetCharacter(0).GetCurrentHp(), "cancellation cannot undo or replay damage")
		})
	}
}

func TestOutdoorFireballWrappedShipBlockedAndSinkingOrder(t *testing.T) {
	for _, blocked := range []bool{false, true} {
		t.Run(fmt.Sprintf("blocked%v", blocked), func(t *testing.T) {
			gs, mock := newOutdoorFireballFixture(t, 0, references.FrigateVehicle)
			startProjectile := gs.SystemCallbacks.Visual.StartProjectile
			gs.SystemCallbacks.Visual.StartProjectile = func(from, to references.Position, missile references.MissileType) uint64 {
				startProjectile(from, to, missile)
				return 1 // This fixture's generic mock duration does not wrap; the scene does.
			}
			gs.MapState.PlayerLocation.Position = references.Position{}
			fillOutdoorFireballTerrain(gs, indexes.Water2)
			gs.PartyVehicle.GetVehicleDetails().SetHullHP(1)
			gs.PartyVehicle.GetVehicleDetails().SetSkiffQuantity(1)
			later := addOverworldTestEnemy(t, gs, references.MonsterSeaSerpent, references.Position{X: 1, Y: 1})
			later.MapUnitDetails().NPCNum = 1
			first := addOverworldTestEnemy(t, gs, references.MonsterDragon, references.Position{X: 254, Y: 255})
			first.MapUnitDetails().NPCNum = 31
			wall := references.Position{X: 255, Y: 0}
			if blocked {
				gs.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &wall, indexes.StoneBrickWall)
			}
			seed := outdoorFireballSeed(t, func(rng *beradrng.Source) bool {
				if rng.Intn(8) != 0 {
					return false
				}
				if !blocked {
					rng.Intn(30)
				}
				return rng.Intn(8) == 0
			})
			gs.SetRandomSeed(seed)
			gs.largeMapProcessTurnEffects()
			require.Len(t, gs.pirateAttackQueue, 2)
			firstHit, lastHit := gs.pirateAttackQueue[0].result, gs.pirateAttackQueue[1].result
			if blocked {
				assert.Equal(t, PirateAttackBlocked, firstHit.Outcome)
				assert.Equal(t, wall, firstHit.ProjectileEnd)
				assert.Equal(t, SourceHitEscapedBySkiff, lastHit.Hit.Outcome)
				assert.Equal(t, uint16(100), gs.PartyState.GetCharacter(0).GetCurrentHp())
				assert.Equal(t, uint64(3), gs.GetRNG().Calls())
			} else {
				assert.Equal(t, SourceHitEscapedBySkiff, firstHit.Hit.Outcome)
				assert.Equal(t, SourceHitPartyDamaged, lastHit.Hit.Outcome, "later shot observes authoritative sinking fallback")
				assert.Less(t, gs.PartyState.GetCharacter(0).GetCurrentHp(), uint16(100))
				assert.Equal(t, uint64(4), gs.GetRNG().Calls())
			}
			assert.Equal(t, references.SkiffVehicle, gs.PartyVehicle.GetVehicleDetails().VehicleType)
			drainGameplayWorkflows(t, gs)
			assert.Equal(t, 1, countMessage(mock.Messages, "Ship sunk!"))
			assert.Equal(t, 1, countMessage(mock.Messages, "Abandon ship!"))
		})
	}
}

func TestOutdoorFireballRoughSeasAreIndependentOfMonsterContact(t *testing.T) {
	for _, terrain := range []indexes.SpriteIndex{indexes.Water1, indexes.Water2, indexes.WaterCoast} {
		t.Run(fmt.Sprintf("terrain%d", terrain), func(t *testing.T) {
			gs, mock := newOutdoorFireballFixture(t, -1, references.CarpetVehicle)
			fillOutdoorFireballTerrain(gs, terrain)
			addOverworldTestEnemy(t, gs, references.MonsterSeaSerpent, references.Position{X: 101, Y: 100})
			// The accepted movement route invokes this entry hazard separately
			// from ordinary completion. Its first mounted actor phase is skipped.
			gs.ProcessSkiffDeepWaterDamage(&gs.MapState.PlayerLocation.Position)
			gs.CompleteSurfaceMovementCycle(references.LargeMapType, true)
			drainGameplayWorkflows(t, gs)
			hpAfterEntry := gs.PartyState.GetCharacter(0).GetCurrentHp()
			if terrain == indexes.Water1 {
				assert.Less(t, hpAfterEntry, uint16(100))
				assert.Equal(t, 1, countMessage(mock.Messages, "Rough seas!"))
			} else {
				assert.Equal(t, uint16(100), hpAfterEntry)
				assert.NotContains(t, mock.Messages, "Rough seas!")
			}
			assert.NotContains(t, mock.Messages, "Attacked!")
			passOutdoorFireballTurn(gs)
			assert.Less(t, gs.PartyState.GetCharacter(0).GetCurrentHp(), hpAfterEntry)
			assert.Equal(t, 1, countMessage(mock.Messages, "Attacked!"))
			assert.Empty(t, mock.ProjectilesStarted)
		})
	}
}

func TestOutdoorFireballFiltersActorsAndKeepsTacticalContact(t *testing.T) {
	gs, _ := newOutdoorFireballFixture(t, 0, references.NoPartyVehicle)
	for _, monster := range []references.MonsterType{references.MonsterSeaSerpent, references.MonsterDragon, references.MonsterOrc} {
		enemy := addOverworldTestEnemy(t, gs, monster, references.Position{X: 102, Y: 101 + references.Coordinate(monster%3)})
		enemy.EnemyReference.DoNotMove = true
		switch monster {
		case references.MonsterSeaSerpent:
			enemy.SetVisible(false)
		case references.MonsterDragon:
			enemy.SetFloor(-1)
		}
	}
	gs.largeMapProcessTurnEffects()
	assert.Zero(t, gs.GetRNG().Calls())
	assert.Empty(t, gs.pirateAttackQueue)
	gs.CurrentNPCAIController.RemoveAllEnemies()
	contact := addOverworldTestEnemy(t, gs, references.MonsterDragon, references.Position{X: 101, Y: 100})
	gs.largeMapProcessTurnEffects()
	assert.Same(t, contact, gs.pendingOverworldEncounter)
	assert.Empty(t, gs.pirateAttackQueue)
	assert.Zero(t, gs.GetRNG().Calls(), "cardinal dragon goes to tactical contact without rolling fire")
}

func TestOutdoorFireballFixedSourceLightAndSight(t *testing.T) {
	gs, _ := newOutdoorFireballFixture(t, -1, references.NoPartyVehicle)
	wall := references.Position{X: 102, Y: 102}
	gs.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &wall, indexes.StoneBrickWall)
	light := references.Position{X: 103, Y: 101}
	gs.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &light, indexes.LampPost)
	end, reached := gs.traceOutdoorMissile(references.Position{X: 103, Y: 103}, gs.MapState.PlayerLocation.Position, references.MissileRed)
	assert.False(t, reached, "original fixed terrain light reveals the otherwise dark blocker")
	assert.Equal(t, wall, end)
	assert.Equal(t, 2, gs.outdoorMissileSight())
	gs.MapState.Lighting.LightTorch()
	assert.Equal(t, 10, gs.outdoorMissileSight())
	gs.MapState.Lighting.SetMagicLight(true)
	assert.Equal(t, 18, gs.outdoorMissileSight())
	gs.MapState.Lighting.SetMagicLight(false)
	gs.MapState.Lighting.ExtinguishTorch()
	gs.MapState.PlayerLocation.Floor = 0
	assert.Equal(t, 50, gs.outdoorMissileSight())
	gs.DateTime.Hour, gs.DateTime.Minute = 5, 20
	assert.Equal(t, 10, gs.outdoorMissileSight())
	gs.DateTime.Hour, gs.DateTime.Minute = 19, 39
	assert.Equal(t, 10, gs.outdoorMissileSight())
}

func TestOutdoorWhirlpoolTransportDoesNotCarryOldMapAttackers(t *testing.T) {
	for _, attacker := range []references.MonsterType{references.MonsterSeaSerpent, references.MonsterStocks} {
		for _, test := range []struct {
			name    string
			vehicle references.VehicleType
			hull    int
		}{
			{name: "frigate survives", vehicle: references.FrigateVehicle, hull: 99},
			{name: "frigate sinks to skiff", vehicle: references.FrigateVehicle, hull: 1},
			{name: "carpet party hit", vehicle: references.CarpetVehicle},
		} {
			t.Run(fmt.Sprintf("%v/%s", attacker, test.name), func(t *testing.T) {
				gs, mock := newOutdoorFireballFixture(t, 0, test.vehicle)
				origin := gs.MapState.PlayerLocation.Position
				if test.vehicle == references.FrigateVehicle {
					gs.PartyVehicle.GetVehicleDetails().SetHullHP(test.hull)
					gs.PartyVehicle.GetVehicleDetails().SetSkiffQuantity(1)
				}
				position := references.Position{X: 100, Y: 98}
				var enemy *map_units.NPCEnemy
				if attacker == references.MonsterStocks {
					enemy = addPirateTestEnemy(t, gs, position, indexes.PirateShipLeft)
				} else {
					enemy = addOverworldTestEnemy(t, gs, attacker, position)
				}
				enemy.MapUnitDetails().NPCNum = 1
				whirlpoolPosition := references.Position{X: 101, Y: 100}
				require.True(t, gs.spawnWhirlpoolAt(whirlpoolPosition))
				gs.GetWhirlpoolAt(whirlpoolPosition).MapUnitDetails().NPCNum = 31
				// The serpent would fire on its first draw; the pirate is already
				// aligned. Neither may act before the higher-slot whirlpool resolves.
				seed := sourceHitSeedForFirstIntn(t, 8, 0)
				gs.SetRandomSeed(seed)
				gs.largeMapProcessTurnEffects()

				require.True(t, gs.EventQueueManager.HasWorkflow(outdoorWhirlpoolWorkflowName(31)))
				assert.Equal(t, origin, gs.MapState.PlayerLocation.Position)
				assert.Equal(t, test.vehicle, gs.PartyVehicle.GetVehicleDetails().VehicleType)
				assert.Equal(t, uint16(100), gs.PartyState.GetCharacter(0).GetCurrentHp())
				if test.vehicle == references.FrigateVehicle {
					assert.Equal(t, test.hull, gs.PartyVehicle.GetVehicleDetails().GetHullHP())
				}
				assert.Zero(t, gs.GetRNG().Calls(), "no lower-slot firing or damage rolls before handoff")
				assert.Empty(t, gs.pirateAttackQueue)
				assert.False(t, gs.EventQueueManager.HasWorkflow(pirateAttackWorkflowName))
				assert.Equal(t, position, enemy.Pos(), "handoff also suppresses lower-slot movement")

				drainGameplayWorkflows(t, gs)

				wantRNG := beradrng.New(seed)
				if test.vehicle == references.FrigateVehicle {
					damage := wantRNG.Intn(sourceHitHullDamageSides) + 1
					assert.Equal(t, uint16(100), gs.PartyState.GetCharacter(0).GetCurrentHp())
					if test.hull == 1 {
						assert.Equal(t, references.SkiffVehicle, gs.PartyVehicle.GetVehicleDetails().VehicleType)
					} else {
						assert.Equal(t, test.hull-damage, gs.PartyVehicle.GetVehicleDetails().GetHullHP())
					}
				} else {
					assert.Equal(t, uint16(100-wantRNG.Intn(sourceHitPartyDamageSides)-1), gs.PartyState.GetCharacter(0).GetCurrentHp())
				}
				assert.Equal(t, wantRNG.Calls(), gs.GetRNG().Calls(), "only the whirlpool's source hit consumes RNG")
				if test.vehicle != references.NoPartyVehicle {
					assert.Equal(t, references.FloorNumber(-1), gs.MapState.PlayerLocation.Floor)
					assert.Equal(t, references.Position{X: 34, Y: 18}, gs.MapState.PlayerLocation.Position)
				} else {
					assert.Equal(t, origin, gs.MapState.PlayerLocation.Position)
				}
				assert.Empty(t, mock.ProjectilesStarted, "no stale projectile can play after transport")
				assert.Empty(t, gs.pirateAttackQueue)
				assert.NotContains(t, mock.Messages, "* BOOOM! *")
				assert.False(t, mock.InputBlocked)
			})
		}
	}
}
