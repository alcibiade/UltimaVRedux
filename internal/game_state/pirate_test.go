package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/deterministic_random"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// TestPirateInCannonRangeDistinguishesFireFromContact documents that one
// blank tile still permits original-U5 cannon fire but is not contact combat.
func TestPirateInCannonRangeDistinguishesFireFromContact(t *testing.T) {
	assert.True(t, pirateInCannonRange(0, 1))
	assert.True(t, pirateInCannonRange(0, 2), "one blank cardinal tile remains in cannon range")
	assert.True(t, pirateInCannonRange(3, 0))
	assert.False(t, pirateInCannonRange(0, 4))
	assert.False(t, pirateInCannonRange(2, 1), "pirates do not fire diagonally")
}

func TestPirateCannonResolvesBeforeOrderedProjectileAndImpact(t *testing.T) {
	gs, mock := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		Build()
	if gs == nil {
		return
	}
	target := gs.MapState.PlayerLocation.Position
	gs.PartyVehicle = map_units.NewNPCFriendlyVehiceNoVehicle()
	pirate := addPirateTestEnemy(t, gs, references.Position{X: 100, Y: 98}, indexes.PirateShipDown)
	avatar := gs.PartyState.GetCharacter(0)
	hpBefore := avatar.GetCurrentHp()
	kapows := make([]references.Position, 0, 1)
	gs.SystemCallbacks.Visual.PlayKapowAtPosition = func(position references.Position) {
		assert.Less(t, avatar.GetCurrentHp(), hpBefore, "authoritative damage must precede presentation")
		assert.Equal(t, 1, mock.ProjectilesClearedCount, "projectile must clear before impact feedback")
		kapows = append(kapows, position)
	}

	gs.FirePirateCannon(pirate, target)
	assert.Less(t, avatar.GetCurrentHp(), hpBefore)
	assert.Equal(t, 1, mock.StatsDisplayUpdated)
	for range 10 {
		gs.ElapsedMs += 1000
		gs.EventQueueManager.Update()
		if !gs.EventQueueManager.HasActiveWorkflows() {
			break
		}
	}

	require.Len(t, mock.ProjectilesStarted, 1)
	assert.Equal(t, references.MissileCannonBall, mock.ProjectilesStarted[0].ProjectileType)
	assert.Equal(t, []references.Position{target}, kapows)
	assert.Less(t, avatar.GetCurrentHp(), hpBefore)
	assert.Positive(t, mock.ProjectilesClearedCount)
	assert.False(t, gs.EventQueueManager.HasActiveWorkflows())
}

func TestPirateCannonSerializesEveryQueuedAttack(t *testing.T) {
	const seed = uint64(330)
	gs, mock := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		Build()
	for index := range gs.PartyState.GetCharacterCount() {
		gs.PartyState.GetCharacter(index).PartyMap = party_state.PartyMapAtInn(references.Britain)
	}
	member := gs.PartyState.GetCharacter(0)
	member.PartyMap = party_state.PartyMapActive
	member.SetMaxHp(100)
	member.SetHp(100)
	gs.PartyVehicle = map_units.NewNPCFriendlyVehiceNoVehicle()
	target := gs.MapState.PlayerLocation.Position
	pirate := addPirateTestEnemy(t, gs, references.Position{X: 100, Y: 98}, indexes.PirateShipDown)
	gs.SetRandomSeed(seed)
	mock.Reset()
	wantRNG := deterministic_random.New(seed)
	wantHP := 100
	for range 3 {
		wantHP -= wantRNG.Intn(8) + 1
	}

	require.NotPanics(t, func() {
		gs.FirePirateCannon(pirate, target)
		gs.FirePirateCannon(pirate, target)
		gs.FirePirateCannon(pirate, target)
	})
	assert.Len(t, gs.pirateAttackQueue, 3)
	assert.Equal(t, 0, countMessage(mock.Messages, "* BOOOM! *"), "queued attacks must not present early")

	drainGameplayWorkflows(t, gs)
	assert.Empty(t, gs.pirateAttackQueue)
	assert.Equal(t, 3, countMessage(mock.Messages, "* BOOOM! *"))
	assert.Len(t, mock.ProjectilesStarted, 3)
	assert.Equal(t, uint16(wantHP), member.GetCurrentHp())
	assert.Equal(t, uint64(3), gs.GetRNG().Calls())
}

func TestPirateCannonStartsPartyWipeOnceAfterSinkingPresentation(t *testing.T) {
	gs, mock := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		Build()
	for index := range gs.PartyState.GetCharacterCount() {
		gs.PartyState.GetCharacter(index).PartyMap = party_state.PartyMapAtInn(references.Britain)
	}
	member := gs.PartyState.GetCharacter(0)
	member.PartyMap = party_state.PartyMapActive
	member.SetStatus(party_state.Good)
	member.SetMaxHp(1)
	member.SetHp(1)
	frigate := map_units.NewNPCFriendlyVehiceNewRef(
		references.FrigateVehicle,
		gs.MapState.PlayerLocation.Position,
		gs.MapState.PlayerLocation.Floor,
	)
	frigate.GetVehicleDetails().SetHullHP(1)
	frigate.GetVehicleDetails().SetSkiffQuantity(0)
	gs.PartyVehicle = *frigate
	gs.PartyState.Inventory.QuestItems.Set(references.Carpet, 0)
	deathSequences := 0
	gs.SystemCallbacks.Flow.Consequences.TriggerDeathSequence = func() {
		deathSequences++
		assert.Contains(t, mock.Messages, "DROWNING!!!")
	}
	pirate := addPirateTestEnemy(t, gs, references.Position{X: 100, Y: 98}, indexes.PirateShipDown)
	gs.SetRandomSeed(461)
	mock.Reset()

	gs.FirePirateCannon(pirate, gs.MapState.PlayerLocation.Position)
	gs.FirePirateCannon(pirate, gs.MapState.PlayerLocation.Position)
	drainGameplayWorkflows(t, gs)

	assert.Equal(t, 1, deathSequences)
	assert.True(t, gs.PartyState.IsAllPartyDead())
	assert.Equal(t, references.NoPartyVehicle, gs.PartyVehicle.GetVehicleDetails().VehicleType)
}

func TestPirateAttackPhaseWarnsAndFiresAllAlignedInSourceOrder(t *testing.T) {
	const seed = uint64(464)
	gs, mock := newPirateAttackPhaseState(t)
	member := gs.PartyState.GetCharacter(0)
	first := addPirateTestEnemy(t, gs, references.Position{X: 100, Y: 97}, indexes.PirateShipRight)
	warning := addPirateTestEnemy(t, gs, references.Position{X: 97, Y: 100}, indexes.PirateShipLeft)
	last := addPirateTestEnemy(t, gs, references.Position{X: 100, Y: 98}, indexes.PirateShipLeft)
	setPirateTestWaterLine(gs, first.Pos(), gs.MapState.PlayerLocation.Position)
	setPirateTestWaterLine(gs, warning.Pos(), gs.MapState.PlayerLocation.Position)
	setPirateTestWaterLine(gs, last.Pos(), gs.MapState.PlayerLocation.Position)
	gs.SetRandomSeed(seed)
	wantRNG := deterministic_random.New(seed)
	wantHP := 100 - (wantRNG.Intn(sourceHitPartyDamageSides) + 1) -
		(wantRNG.Intn(sourceHitPartyDamageSides) + 1)

	require.True(t, gs.CheckPirateAttack())

	require.Len(t, gs.pirateAttackQueue, 3)
	assert.Equal(t, last.Pos(), gs.pirateAttackQueue[0].result.Source)
	assert.Equal(t, PirateAttackHit, gs.pirateAttackQueue[0].result.Outcome)
	assert.Equal(t, warning.Pos(), gs.pirateAttackQueue[1].result.Source)
	assert.Equal(t, PirateAttackWarning, gs.pirateAttackQueue[1].result.Outcome)
	assert.Equal(t, first.Pos(), gs.pirateAttackQueue[2].result.Source)
	assert.Equal(t, PirateAttackHit, gs.pirateAttackQueue[2].result.Outcome)
	assert.Equal(t, references.Up, warning.Facing())
	assert.Equal(t, uint16(wantHP), member.GetCurrentHp())
	assert.Equal(t, uint64(2), gs.GetRNG().Calls())
	assert.Equal(t, 2, mock.StatsDisplayUpdated)

	drainGameplayWorkflows(t, gs)
	assert.Equal(t, []string{"* BOOOM! *", "* Pirate ship turns *", "* BOOOM! *"}, mock.Messages)
}

func TestMisalignedPirateWarnsOnePhaseThenFires(t *testing.T) {
	gs, mock := newPirateAttackPhaseState(t)
	pirate := addPirateTestEnemy(t, gs, references.Position{X: 98, Y: 100}, indexes.PirateShipLeft)
	setPirateTestWaterLine(gs, pirate.Pos(), gs.MapState.PlayerLocation.Position)

	require.True(t, gs.CheckPirateAttack())
	require.Len(t, gs.pirateAttackQueue, 1)
	assert.Equal(t, PirateAttackWarning, gs.pirateAttackQueue[0].result.Outcome)
	assert.Equal(t, uint64(0), gs.GetRNG().Calls())
	drainGameplayWorkflows(t, gs)
	assert.Equal(t, []string{"* Pirate ship turns *"}, mock.Messages)

	mock.Reset()
	require.True(t, gs.CheckPirateAttack())
	require.Len(t, gs.pirateAttackQueue, 1)
	assert.Equal(t, PirateAttackHit, gs.pirateAttackQueue[0].result.Outcome)
	assert.Equal(t, uint64(1), gs.GetRNG().Calls())
	drainGameplayWorkflows(t, gs)
	assert.Equal(t, []string{"* BOOOM! *"}, mock.Messages)
}

func TestPirateAttackUsesSourceHitForFrigateAndBlockedLOSUsesNoRNG(t *testing.T) {
	const seed = uint64(464)

	t.Run("frigate", func(t *testing.T) {
		gs, _ := newPirateAttackPhaseState(t)
		frigate := map_units.NewNPCFriendlyVehiceNewRef(
			references.FrigateVehicle,
			gs.MapState.PlayerLocation.Position,
			gs.MapState.PlayerLocation.Floor,
		)
		frigate.GetVehicleDetails().SetHullHP(map_units.MaxHullHP)
		gs.PartyVehicle = *frigate
		pirate := addPirateTestEnemy(t, gs, references.Position{X: 100, Y: 98}, indexes.PirateShipLeft)
		setPirateTestWaterLine(gs, pirate.Pos(), gs.MapState.PlayerLocation.Position)
		gs.SetRandomSeed(seed)
		wantDamage := deterministic_random.New(seed).Intn(sourceHitHullDamageSides) + 1

		require.True(t, gs.CheckPirateAttack())

		require.Len(t, gs.pirateAttackQueue, 1)
		result := gs.pirateAttackQueue[0].result
		assert.Equal(t, PirateAttackHit, result.Outcome)
		assert.Equal(t, SourceHitFrigateDamaged, result.Hit.Outcome)
		assert.Equal(t, wantDamage, result.Hit.HullDamage)
		assert.Equal(t, map_units.MaxHullHP-wantDamage, gs.PartyVehicle.GetVehicleDetails().GetHullHP())
		assert.Equal(t, uint64(1), gs.GetRNG().Calls())
	})

	t.Run("blocked", func(t *testing.T) {
		gs, _ := newPirateAttackPhaseState(t)
		addPirateTestEnemy(t, gs, references.Position{X: 100, Y: 98}, indexes.PirateShipLeft)
		blocker := references.Position{X: 100, Y: 99}
		gs.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &blocker, indexes.StoneBrickWall)

		require.True(t, gs.CheckPirateAttack())

		require.Len(t, gs.pirateAttackQueue, 1)
		result := gs.pirateAttackQueue[0].result
		assert.Equal(t, PirateAttackBlocked, result.Outcome)
		assert.Equal(t, blocker, result.ProjectileEnd)
		assert.Equal(t, uint64(0), gs.GetRNG().Calls())
	})
}

func newPirateAttackPhaseState(t *testing.T) (*GameState, *MockSystemCallbacks) {
	t.Helper()
	gs, mock := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		Build()
	for index := range gs.PartyState.GetCharacterCount() {
		gs.PartyState.GetCharacter(index).PartyMap = party_state.PartyMapAtInn(references.Britain)
	}
	member := gs.PartyState.GetCharacter(0)
	member.PartyMap = party_state.PartyMapActive
	member.SetStatus(party_state.Good)
	member.SetMaxHp(100)
	member.SetHp(100)
	gs.PartyVehicle = map_units.NewNPCFriendlyVehiceNoVehicle()
	mock.Reset()

	return gs, mock
}

func setPirateTestWaterLine(gs *GameState, from, to references.Position) {
	current := from
	for current != to {
		direction := gs.calculateDirectionToward(current, to)
		current = *direction.GetNewPositionInDirection(&current).
			GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
		gs.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &current, indexes.Water1)
	}
}
