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

func TestSourceMissilePassageMatchesEveryOriginalTerrainBit(t *testing.T) {
	expected := [32]byte{
		0xff, 0xf3, 0xc3, 0x8f, 0xff, 0xff, 0xff, 0xc0,
		0xdd, 0xf8, 0x03, 0xdf, 0xff, 0xff, 0x00, 0x00,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x3f,
		0xff, 0xff, 0xff, 0xfe, 0xff, 0xff, 0xff, 0xff,
	}
	for tile := range 256 {
		want := expected[tile>>3]&(0x80>>uint(tile&7)) != 0
		assert.Equal(t, want, sourceMissilePasses(indexes.SpriteIndex(tile)), "terrain %#02x", tile)
	}
}

func TestResolveShipCannonFireRemovesAtAndBelowZeroExactlyOnce(t *testing.T) {
	const seed = uint64(464)
	wantDamage := beradrng.New(seed).Intn(shipCannonDamageMax) + 1

	for _, test := range []struct {
		name        string
		hp          int
		wantOutcome ShipCannonOutcome
		wantHP      int
		wantRemoved bool
	}{
		{name: "below", hp: max(wantDamage-1, 0), wantOutcome: ShipCannonDestroyed, wantRemoved: true},
		{name: "equal", hp: wantDamage, wantOutcome: ShipCannonDestroyed, wantRemoved: true},
		{name: "above", hp: wantDamage + 1, wantOutcome: ShipCannonHit, wantHP: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, _ := newShipCannonAuthorityState(t)
			targetPos := *references.Right.GetNewPositionInDirection(&state.MapState.PlayerLocation.Position)
			target := addOverworldTestEnemy(t, state, references.MonsterOrc, targetPos)
			target.SetCurrentHP(test.hp)
			state.SetRandomSeed(seed)

			result := state.ResolveShipCannonFire(references.Right)

			assert.Equal(t, test.wantOutcome, result.Outcome)
			assert.Equal(t, wantDamage, result.Damage)
			assert.Equal(t, test.wantHP, result.HPAfter)
			assert.Equal(t, uint64(1), state.GetRNG().Calls())
			stored := state.GetCurrentLargeMapNPCAIController().GetNpcs().
				GetVisibleEnemyAtPositionAndFloorOrNil(targetPos, state.MapState.PlayerLocation.Floor)
			assert.Equal(t, test.wantRemoved, stored == nil)

			if test.wantRemoved {
				second := state.ResolveShipCannonFire(references.Right)
				assert.Equal(t, ShipCannonMissed, second.Outcome)
				assert.Equal(t, uint64(1), state.GetRNG().Calls(), "removed target cannot be damaged twice")
			}
		})
	}
}

func TestResolveShipCannonFireUsesVisibleCurrentFloorTarget(t *testing.T) {
	state, _ := newShipCannonAuthorityState(t)
	player := state.MapState.PlayerLocation.Position
	first := *references.Right.GetNewPositionInDirection(&player)
	second := *references.Right.GetNewPositionInDirection(&first)
	third := *references.Right.GetNewPositionInDirection(&second)
	hidden := addOverworldTestEnemy(t, state, references.MonsterOrc, first)
	hidden.SetVisible(false)
	otherFloor := addOverworldTestEnemy(t, state, references.MonsterTroll, second)
	otherFloor.SetFloor(state.MapState.PlayerLocation.Floor + 1)
	target := addOverworldTestEnemy(t, state, references.MonsterDragon, third)
	target.SetCurrentHP(100)
	hiddenHP := hidden.CurrentHP()
	otherFloorHP := otherFloor.CurrentHP()
	state.SetRandomSeed(464)

	result := state.ResolveShipCannonFire(references.Right)

	assert.Equal(t, ShipCannonHit, result.Outcome)
	assert.Equal(t, third, result.TargetPosition)
	assert.Equal(t, hiddenHP, hidden.CurrentHP())
	assert.Equal(t, otherFloorHP, otherFloor.CurrentHP())
	assert.Less(t, target.CurrentHP(), 100)
	assert.Equal(t, uint64(1), state.GetRNG().Calls())
}

func TestResolveShipCannonFireStopsAtSourceTerrainBlocker(t *testing.T) {
	state, _ := newShipCannonAuthorityState(t)
	player := state.MapState.PlayerLocation.Position
	blocker := *references.Right.GetNewPositionInDirection(&player)
	targetPos := *references.Right.GetNewPositionInDirection(&blocker)
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &blocker, indexes.StoneBrickWall)
	target := addOverworldTestEnemy(t, state, references.MonsterOrc, targetPos)
	target.SetCurrentHP(50)

	result := state.ResolveShipCannonFire(references.Right)

	assert.Equal(t, ShipCannonBlocked, result.Outcome)
	assert.Equal(t, blocker, result.ProjectileEnd)
	assert.Equal(t, 50, target.CurrentHP())
	assert.Equal(t, uint64(0), state.GetRNG().Calls())
}

func TestShipCannonWorkflowPresentsResolvedHitAndCleansEffects(t *testing.T) {
	state, callbacks := newShipCannonAuthorityState(t)
	targetPos := state.MapState.PlayerLocation.Position
	for range shipCannonRange {
		targetPos = *references.Right.GetNewPositionInDirection(&targetPos)
	}
	target := addOverworldTestEnemy(t, state, references.MonsterOrc, targetPos)
	target.SetCurrentHP(100)
	hpBefore := target.CurrentHP()
	impactHP := -1
	state.SystemCallbacks.Visual.PlayKapowAtPosition = func(position references.Position) {
		assert.Equal(t, targetPos, position)
		assert.Equal(t, 1, callbacks.ProjectilesClearedCount)
		impactHP = target.CurrentHP()
	}

	require.True(t, state.ActionFireLargeMap(references.Right))
	assert.Less(t, target.CurrentHP(), hpBefore)
	drainGameplayWorkflows(t, state)

	assert.Equal(t, target.CurrentHP(), impactHP)
	require.NotEmpty(t, callbacks.ScreenFlashIntensities)
	assert.Zero(t, callbacks.ScreenFlashIntensities[len(callbacks.ScreenFlashIntensities)-1])
	require.NotEmpty(t, callbacks.ScreenShakeOffsets)
	lastShake := callbacks.ScreenShakeOffsets[len(callbacks.ScreenShakeOffsets)-1]
	assert.Zero(t, lastShake.X)
	assert.Zero(t, lastShake.Y)
}

func TestShipFireOwnsOneCycleAndAllowsPirateRetaliation(t *testing.T) {
	state, callbacks := newShipCannonAuthorityState(t)
	player := state.MapState.PlayerLocation.Position
	targetPos := player
	for range shipCannonRange {
		targetPos = *references.Right.GetNewPositionInDirection(&targetPos)
	}
	target := addOverworldTestEnemy(t, state, references.MonsterOrc, targetPos)
	target.SetCurrentHP(100)
	piratePos := *references.Left.GetNewPositionInDirection(&player)
	piratePos = *references.Left.GetNewPositionInDirection(&piratePos)
	addPirateTestEnemy(t, state, piratePos, indexes.PirateShipUp)
	before := state.DateTime
	hullBefore := state.PartyVehicle.GetVehicleDetails().GetHullHP()

	require.True(t, state.ActionFireLargeMap(references.Right))
	drainGameplayWorkflows(t, state)

	wantTime := before
	wantTime.Advance(DefaultLargeMapMinutesPerTurn)
	assert.Equal(t, wantTime, state.DateTime)
	assert.Less(t, state.PartyVehicle.GetVehicleDetails().GetHullHP(), hullBefore)
	assert.Equal(t, 1, countEqual(callbacks.Messages, "* BOOOM! *"))
}

func newShipCannonAuthorityState(t *testing.T) (*GameState, *MockSystemCallbacks) {
	t.Helper()
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		Build()
	frigate := map_units.NewNPCFriendlyVehiceNewRef(
		references.FrigateVehicle,
		state.MapState.PlayerLocation.Position,
		state.MapState.PlayerLocation.Floor,
	)
	frigate.GetVehicleDetails().SetPartyVehicleDirection(references.Up)
	frigate.GetVehicleDetails().SetHullHP(map_units.MaxHullHP)
	state.PartyVehicle = *frigate
	current := state.MapState.PlayerLocation.Position
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &current, indexes.Water1)
	for range shipCannonRange {
		current = *references.Right.GetNewPositionInDirection(&current)
		state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &current, indexes.Water1)
	}
	current = state.MapState.PlayerLocation.Position
	for range shipCannonRange {
		current = *references.Left.GetNewPositionInDirection(&current)
		state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &current, indexes.Water1)
	}
	callbacks.Reset()

	return state, callbacks
}
