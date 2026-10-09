package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/deterministic_random"
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestResolveSourceHitUniformHullRollAndLethalEquality(t *testing.T) {
	tests := []struct {
		name        string
		roll        int
		hull        int
		wantOutcome SourceHitOutcome
		wantHull    int
	}{
		{name: "minimum roll", roll: 1, hull: 99, wantOutcome: SourceHitFrigateDamaged, wantHull: 98},
		{name: "maximum roll", roll: 30, hull: 99, wantOutcome: SourceHitFrigateDamaged, wantHull: 69},
		{name: "below hull survives positive", roll: 29, hull: 30, wantOutcome: SourceHitFrigateDamaged, wantHull: 1},
		{name: "equal hull is lethal", roll: 30, hull: 30, wantOutcome: SourceHitEscapedBySkiff},
		{name: "above hull is lethal", roll: 30, hull: 29, wantOutcome: SourceHitEscapedBySkiff},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := newSourceHitFixture(t, references.FrigateVehicle, 100)
			details := state.PartyVehicle.GetVehicleDetails()
			details.SetHullHP(test.hull)
			if test.wantOutcome == SourceHitEscapedBySkiff {
				details.SetSkiffQuantity(1)
			}
			state.SetRandomSeed(sourceHitSeedForFirstIntn(t, 30, test.roll-1))

			result := state.ResolveSourceHit()

			assert.Equal(t, test.roll, result.HullDamage)
			assert.Equal(t, test.wantOutcome, result.Outcome)
			assert.Equal(t, test.wantHull, result.HullAfter)
			assert.Equal(t, uint64(1), state.GetRNG().Calls())
			if result.Outcome == SourceHitFrigateDamaged {
				assert.Positive(t, state.PartyVehicle.GetVehicleDetails().GetHullHP())
			} else {
				assert.Equal(t, references.SkiffVehicle, state.PartyVehicle.GetVehicleDetails().VehicleType)
			}
		})
	}
}

func TestResolveSourceHitLethalFallbackOrderDirectionAndRNG(t *testing.T) {
	t.Run("carried skiff wins and preserves source direction", func(t *testing.T) {
		state := newSourceHitFixture(t, references.FrigateVehicle, 100)
		details := state.PartyVehicle.GetVehicleDetails()
		details.SetHullHP(1)
		details.SetSkiffQuantity(2)
		details.SetPartyVehicleDirection(references.Up)
		state.PartyState.Inventory.QuestItems.Set(references.Carpet, 1)
		state.SetRandomSeed(461)

		result := state.ResolveSourceHit()

		assert.Equal(t, SourceHitEscapedBySkiff, result.Outcome)
		assert.Equal(t, references.SkiffVehicle, result.VehicleAfter)
		assert.Equal(t, references.Up, result.DirectionAfter)
		assert.Equal(t, references.Up, state.PartyVehicle.GetVehicleDetails().CurrentDirection())
		assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.Carpet))
		assert.Equal(t, uint64(1), state.GetRNG().Calls(), "skiff fallback has no orientation roll")
		assert.Equal(t, state.MapState.PlayerLocation.Position, state.PartyVehicle.Pos())
		assert.Equal(t, state.MapState.PlayerLocation.Floor, state.PartyVehicle.Floor())
	})

	t.Run("carpet is consumed and uses the next source orientation roll", func(t *testing.T) {
		const seed = uint64(461)
		state := newSourceHitFixture(t, references.FrigateVehicle, 100)
		state.PartyVehicle.GetVehicleDetails().SetHullHP(1)
		state.PartyState.Inventory.QuestItems.Set(references.Carpet, 2)
		state.SetRandomSeed(seed)
		wantRNG := deterministic_random.New(seed)
		wantRNG.Intn(30)
		wantDirection := references.Right
		if wantRNG.Intn(2) == 1 {
			wantDirection = references.Left
		}

		result := state.ResolveSourceHit()

		assert.Equal(t, SourceHitEscapedByCarpet, result.Outcome)
		assert.Equal(t, references.CarpetVehicle, result.VehicleAfter)
		assert.Equal(t, wantDirection, result.DirectionAfter)
		assert.Equal(t, wantDirection, state.PartyVehicle.GetVehicleDetails().CurrentDirection())
		assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.Carpet))
		assert.Equal(t, uint64(2), state.GetRNG().Calls())
	})
}

func TestResolveSourceHitDrownsInIndependentPartyOrder(t *testing.T) {
	const seed = uint64(461)
	state := newSourceHitFixture(t, references.FrigateVehicle, 9, 13)
	state.PartyVehicle.GetVehicleDetails().SetHullHP(1)
	state.SetRandomSeed(seed)
	wantRNG := deterministic_random.New(seed)
	wantRNG.Intn(30)
	wantHP := []int{9, 13}
	wantRolls := make([]SourceHitMemberDamage, 0)
	for wantHP[0] > 0 || wantHP[1] > 0 {
		for index := range wantHP {
			if wantHP[index] == 0 {
				continue
			}
			before := wantHP[index]
			roll := wantRNG.Intn(8) + 1
			wantHP[index] = max(0, before-roll)
			wantRolls = append(wantRolls, SourceHitMemberDamage{
				MemberIndex: index, Damage: roll, HPBefore: uint16(before), HPAfter: uint16(wantHP[index]),
				Died: wantHP[index] == 0,
			})
		}
	}

	result := state.ResolveSourceHit()

	assert.Equal(t, SourceHitDrowned, result.Outcome)
	assert.Equal(t, references.NoPartyVehicle, result.VehicleAfter)
	assert.Equal(t, wantRolls, result.MemberDamage)
	assert.True(t, result.PartyWiped)
	assert.True(t, state.PartyState.IsAllPartyDead())
	assert.Equal(t, uint64(1+len(wantRolls)), state.GetRNG().Calls())
}

func TestResolveSourceHitNonFrigatesDamageEachLivingActiveMemberOnce(t *testing.T) {
	for _, vehicleType := range []references.VehicleType{
		references.NoPartyVehicle,
		references.SkiffVehicle,
		references.CarpetVehicle,
	} {
		t.Run(vehicleType.GetLowerCaseName(), func(t *testing.T) {
			const seed = uint64(330)
			state := newSourceHitFixture(t, vehicleType, 100, 100)
			state.SetRandomSeed(seed)
			wantRNG := deterministic_random.New(seed)
			wantDamage := []int{wantRNG.Intn(8) + 1, wantRNG.Intn(8) + 1}

			result := state.ResolveSourceHit()

			require.Len(t, result.MemberDamage, 2)
			assert.Equal(t, wantDamage[0], result.MemberDamage[0].Damage)
			assert.Equal(t, wantDamage[1], result.MemberDamage[1].Damage)
			assert.Equal(t, vehicleType, result.VehicleAfter)
			assert.Zero(t, result.HullBefore)
			assert.Zero(t, result.HullAfter)
			assert.Zero(t, result.HullDamage)
			assert.Equal(t, map_units.MaxHullHP, state.PartyVehicle.GetVehicleDetails().GetHullHP(),
				"non-frigate source hits must not mutate the synthetic hull field")
			assert.Equal(t, uint64(2), state.GetRNG().Calls())
			result.MemberDamage[0].HPAfter = 0
			assert.Equal(t, uint16(100-wantDamage[0]), state.PartyState.GetCharacter(0).GetCurrentHp(),
				"detached presentation data must not alias character state")
		})
	}
}

func TestResolveSourceHitSkipsInactiveAndDeadMembers(t *testing.T) {
	const seed = uint64(330)
	state := newSourceHitFixture(t, references.NoPartyVehicle, 100, 100, 100)
	state.PartyState.GetCharacter(1).PartyMap = party_state.PartyMapAtInn(references.Britain)
	state.PartyState.GetCharacter(2).SetStatus(party_state.Dead)
	state.SetRandomSeed(seed)
	wantDamage := deterministic_random.New(seed).Intn(8) + 1

	result := state.ResolveSourceHit()

	require.Len(t, result.MemberDamage, 1)
	assert.Equal(t, 0, result.MemberDamage[0].MemberIndex)
	assert.Equal(t, wantDamage, result.MemberDamage[0].Damage)
	assert.Equal(t, uint16(100), state.PartyState.GetCharacter(1).GetCurrentHp())
	assert.Equal(t, uint16(100), state.PartyState.GetCharacter(2).GetCurrentHp())
	assert.Equal(t, uint64(1), state.GetRNG().Calls())
}

func TestResolveWhirlpoolHitTeleportsAndSynchronizesResult(t *testing.T) {
	state := newSourceHitFixture(t, references.FrigateVehicle, 100)
	details := state.PartyVehicle.GetVehicleDetails()
	details.SetHullHP(1)
	details.SetSkiffQuantity(1)
	details.SetPartyVehicleDirection(references.Down)
	state.SetRandomSeed(461)

	result := state.ResolveWhirlpoolHit()

	assert.Equal(t, SourceHitEscapedBySkiff, result.Hit.Outcome)
	assert.True(t, result.Teleported)
	assert.Equal(t, references.BritanniaUnderworld, state.MapState.PlayerLocation.Location)
	assert.Equal(t, references.Position{X: 34, Y: 18}, state.MapState.PlayerLocation.Position)
	assert.Equal(t, references.FloorNumber(-1), state.MapState.PlayerLocation.Floor)
	assert.Equal(t, state.MapState.PlayerLocation.Position, state.PartyVehicle.Pos())
	assert.Equal(t, state.MapState.PlayerLocation.Floor, state.PartyVehicle.Floor())
	assert.Equal(t, references.Down, state.PartyVehicle.GetVehicleDetails().CurrentDirection())
}

func TestWhirlpoolWorkflowStartsPartyWipeOnceAfterPresentationAndTeleport(t *testing.T) {
	state := newSourceHitFixture(t, references.FrigateVehicle, 1)
	state.PartyVehicle.GetVehicleDetails().SetHullHP(1)
	deathSequences := 0
	state.SystemCallbacks.Flow.Consequences.TriggerDeathSequence = func() {
		deathSequences++
		assert.Equal(t, references.Position{X: 34, Y: 18}, state.MapState.PlayerLocation.Position)
		assert.Equal(t, references.FloorNumber(-1), state.MapState.PlayerLocation.Floor)
	}
	state.SetRandomSeed(461)
	q := state.EventQueueManager.Workflow("source_hit_whirlpool_test").
		Priority(event_queue.PriorityHigh).
		Sequential()
	state.BuildWhirlpoolTransportationQueue(q)
	q.Start()

	drainSourceHitWorkflows(t, state)
	state.EventQueueManager.Update()

	assert.Equal(t, 1, deathSequences)
	assert.True(t, state.PartyState.IsAllPartyDead())
}

func TestResolveWhirlpoolHitUsesPartyDamageForNonFrigates(t *testing.T) {
	for _, test := range []struct {
		name       string
		vehicle    references.VehicleType
		teleported bool
	}{
		{name: "foot remains in source world", vehicle: references.NoPartyVehicle},
		{name: "skiff teleports", vehicle: references.SkiffVehicle, teleported: true},
		{name: "carpet teleports", vehicle: references.CarpetVehicle, teleported: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			const seed = uint64(461)
			state := newSourceHitFixture(t, test.vehicle, 100)
			origin := state.MapState.PlayerLocation
			state.SetRandomSeed(seed)
			wantDamage := deterministic_random.New(seed).Intn(8) + 1

			result := state.ResolveWhirlpoolHit()

			require.Len(t, result.Hit.MemberDamage, 1)
			assert.Equal(t, wantDamage, result.Hit.MemberDamage[0].Damage)
			assert.Equal(t, test.teleported, result.Teleported)
			assert.Equal(t, uint16(100-wantDamage), state.PartyState.GetCharacter(0).GetCurrentHp())
			if test.teleported {
				assert.Equal(t, references.Position{X: 34, Y: 18}, state.MapState.PlayerLocation.Position)
				assert.Equal(t, state.MapState.PlayerLocation.Position, state.PartyVehicle.Pos())
			} else {
				assert.Equal(t, origin, state.MapState.PlayerLocation)
			}
		})
	}
}

func TestAdjacentWhirlpoolDamagesFootPartyWithoutTransportOrRemoval(t *testing.T) {
	const seed = uint64(461)
	state := newSourceHitFixture(t, references.NoPartyVehicle, 100)
	origin := state.MapState.PlayerLocation
	whirlpoolPosition := references.Position{X: origin.Position.X + 1, Y: origin.Position.Y}
	require.True(t, state.spawnWhirlpoolAt(whirlpoolPosition))
	state.SetRandomSeed(seed)
	wantDamage := deterministic_random.New(seed).Intn(8) + 1

	require.True(t, state.CheckWhirlpoolAdjacency())
	drainSourceHitWorkflows(t, state)

	assert.Equal(t, origin, state.MapState.PlayerLocation)
	assert.Equal(t, uint16(100-wantDamage), state.PartyState.GetCharacter(0).GetCurrentHp())
	assert.NotNil(t, state.GetWhirlpoolAt(whirlpoolPosition))
	assert.Equal(t, uint64(1), state.GetRNG().Calls())
}

func TestClassifySailingCollisionDistinguishesTerrainShoalPierAndObject(t *testing.T) {
	tests := []struct {
		name       string
		tile       indexes.SpriteIndex
		withObject bool
		want       SailingCollisionKind
	}{
		{name: "pier", tile: indexes.Dock, want: SailingCollisionPier},
		{name: "shoal", tile: indexes.WaterCoast, want: SailingCollisionShoal},
		{name: "blocking terrain", tile: indexes.TallMountains, want: SailingCollisionTerrain},
		{name: "object over navigable water", tile: indexes.Water1, withObject: true, want: SailingCollisionObject},
		{name: "navigable water", tile: indexes.Water1, want: SailingCollisionNone},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := newSourceHitFixture(t, references.FrigateVehicle, 100)
			target := references.Position{X: 51, Y: 50}
			state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, &target, test.tile)
			if test.withObject {
				addSourceHitTestEnemy(t, state, references.MonsterDragon, target)
			}

			assert.Equal(t, test.want, state.ClassifySailingCollision(target))
		})
	}
}

func TestClassifySailingCollisionGivesShoalsPrecedenceOverObjects(t *testing.T) {
	state := newSourceHitFixture(t, references.FrigateVehicle, 100)
	target := references.Position{X: 51, Y: 50}
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, &target, indexes.WaterCoast)
	addSourceHitTestEnemy(t, state, references.MonsterDragon, target)

	assert.Equal(t, SailingCollisionShoal, state.ClassifySailingCollision(target))
}

func TestResolveSailingCollisionKeepsPositionAndPierSafe(t *testing.T) {
	tests := []struct {
		name    string
		tile    indexes.SpriteIndex
		wantHit bool
	}{
		{name: "pier", tile: indexes.Dock},
		{name: "shoal", tile: indexes.WaterCoast, wantHit: true},
		{name: "terrain", tile: indexes.TallMountains, wantHit: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := newSourceHitFixture(t, references.FrigateVehicle, 100)
			state.PartyVehicle.GetVehicleDetails().SetSailState(map_units.SailRaised)
			origin := state.MapState.PlayerLocation.Position
			target := references.Position{X: origin.X + 1, Y: origin.Y}
			state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, &target, test.tile)
			require.True(t, state.StartAutomaticSailing(references.Right))
			state.SetRandomSeed(461)

			result := state.ResolveSailingCollision(target)

			assert.Equal(t, test.wantHit, result.HasHit)
			assert.Equal(t, origin, state.MapState.PlayerLocation.Position)
			assert.False(t, state.IsSailing())
			assert.Equal(t, map_units.SailFurled, state.PartyVehicle.GetVehicleDetails().SailState)
			if test.wantHit {
				assert.Equal(t, uint64(1), state.GetRNG().Calls())
				assert.Positive(t, state.PartyVehicle.GetVehicleDetails().GetHullHP())
			} else {
				assert.Zero(t, state.GetRNG().Calls())
				assert.Equal(t, 99, state.PartyVehicle.GetVehicleDetails().GetHullHP())
			}
		})
	}
}

func TestSourceHitFallbackPersistsWithoutDestroyedFrigate(t *testing.T) {
	state := newSourceHitFixture(t, references.FrigateVehicle, 100)
	details := state.PartyVehicle.GetVehicleDetails()
	details.SetHullHP(1)
	details.SetSkiffQuantity(1)
	details.SetPartyVehicleDirection(references.Left)
	state.SetRandomSeed(461)
	state.ResolveSourceHit()

	snapshot, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)
	require.NotNil(t, snapshot.PartyVehicle)
	assert.Equal(t, int(references.SkiffVehicle), snapshot.PartyVehicle.Type)
	assert.Equal(t, int(references.Left), snapshot.PartyVehicle.CurrentDirection)
	assert.Zero(t, snapshot.PartyVehicle.Skiffs)
	hydrated, err := NewGameStateFromSaveSnapshot(
		snapshot,
		state.GameConfig,
		state.GameReferences,
		19,
		13,
	)
	require.NoError(t, err)
	assert.Equal(t, references.SkiffVehicle, hydrated.PartyVehicle.GetVehicleDetails().VehicleType)
	assert.Equal(t, references.Left, hydrated.PartyVehicle.GetVehicleDetails().CurrentDirection())
	assert.Equal(t, hydrated.MapState.PlayerLocation.Position, hydrated.PartyVehicle.Pos())
	assert.Equal(t, hydrated.MapState.PlayerLocation.Floor, hydrated.PartyVehicle.Floor())
}

func newSourceHitFixture(t *testing.T, vehicleType references.VehicleType, hitPoints ...uint16) *GameState {
	t.Helper()
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(50, 50).
		WithSystemCallbacks().
		Build()
	for index := range party_state.MaxActiveParty {
		member := state.PartyState.GetCharacter(index)
		member.PartyMap = party_state.PartyMapAtInn(references.Britain)
		member.SetStatus(party_state.Good)
		member.SetMaxHp(100)
		member.SetHp(100)
	}
	for index, hp := range hitPoints {
		member := state.PartyState.GetCharacter(index)
		member.PartyMap = party_state.PartyMapActive
		member.SetMaxHp(max(100, hp))
		member.SetHp(hp)
	}
	state.PartyState.Inventory.QuestItems.Set(references.Carpet, 0)
	if vehicleType == references.NoPartyVehicle {
		state.PartyVehicle = map_units.NewNPCFriendlyVehiceNoVehicle()
	} else {
		state.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(
			vehicleType,
			state.MapState.PlayerLocation.Position,
			state.MapState.PlayerLocation.Floor,
		)
	}
	if vehicleType == references.FrigateVehicle {
		state.PartyVehicle.GetVehicleDetails().SetSkiffQuantity(0)
	}
	callbacks.Reset()

	return state
}

func sourceHitSeedForFirstIntn(t *testing.T, maximum, wanted int) uint64 {
	t.Helper()
	for seed := uint64(0); ; seed++ {
		if deterministic_random.New(seed).Intn(maximum) == wanted {
			return seed
		}
	}
}

func addSourceHitTestEnemy(
	t *testing.T,
	state *GameState,
	monsterType references.MonsterType,
	position references.Position,
) {
	t.Helper()
	reference := state.enemyReference(monsterType)
	require.NotNil(t, reference)
	enemy := map_units.NewEnemyNPC(*reference, 1)
	enemy.SetPos(position)
	enemy.SetFloor(state.MapState.PlayerLocation.Floor)
	require.True(t, state.GetCurrentLargeMapNPCAIController().GetNpcs().AddEnemy(enemy))
}

func drainSourceHitWorkflows(t *testing.T, state *GameState) {
	t.Helper()
	for updates := 0; state.EventQueueManager.HasActiveWorkflows(); updates++ {
		require.Less(t, updates, 100, "source hit workflow did not finish")
		state.ElapsedMs += 1000
		state.EventQueueManager.Update()
	}
}
