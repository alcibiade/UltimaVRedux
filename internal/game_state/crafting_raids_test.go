package game_state

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/ai"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// passDeedTurns completes n ordinary deed-map turns.
func passDeedTurns(state *GameState, n int) {
	for range n {
		state.FinishTurn()
	}
}

func raidersOf(state *GameState) []*map_units.NPCEnemy {
	return state.deedRaiders()
}

func raiderPositions(state *GameState) []references.Position {
	positions := make([]references.Position, 0)
	for _, raider := range raidersOf(state) {
		positions = append(positions, raider.Pos())
	}

	return positions
}

// deedFixtureOfTerrain stakes and enters a deed on an overworld square of the given sprite.
func deedFixtureOfTerrain(t *testing.T, sprite indexes.SpriteIndex) (*GameState, *MockSystemCallbacks, Deed) {
	t.Helper()
	state, callbacks := newDeedFixture(t, true)
	setOverworldTile(state, deedFixtureSquare, sprite)
	state.DateTime.Hour, state.DateTime.Minute = 10, 0
	deed := stakeFixtureDeed(t, state, "Home")
	enterFixtureDeed(t, state)
	state.DateTime.Hour, state.DateTime.Minute = 10, 0
	state.MapState.PlayerLocation.Position = yard
	callbacks.Reset()

	return state, callbacks, deed
}

func TestRaidsRollOnlyOnTheOwnDeedMapWithTheOptionOn(t *testing.T) {
	state, callbacks, _ := newBuildFixture(t)

	raided := false
	for range references.DeedRaidCheckTurns * 60 {
		state.FinishTurn()
		if state.DeedRaiderCount() > 0 {
			raided = true

			break
		}
	}
	require.True(t, raided, "a raid comes within sixty rolls with a fixed seed")
	assert.True(t, callbacks.MessagesContain("Raiders approach!"))
	assert.Equal(t, 0, state.DeedRaidTurnsForTest(), "a roll restarts the count")
	count := state.DeedRaiderCount()
	assert.GreaterOrEqual(t, count, references.DeedRaidMinRaiders)
	assert.LessOrEqual(t, count, references.DeedRaidMaxRaiders)

	// While raiders are on the map, no new raid is counted.
	state.SuppressDeedRaidsForTest(false)
	state.MapState.PlayerLocation.Position = yard
	turns := state.DeedRaidTurnsForTest()
	state.processDeedRaidTurn()
	if state.DeedRaiderCount() > 0 {
		assert.Equal(t, turns, state.DeedRaidTurnsForTest())
	}
}

func TestRaidsNeverRollOffTheDeedMapOrWithTheOptionOff(t *testing.T) {
	t.Run("overworld", func(t *testing.T) {
		state, _ := newDeedFixture(t, true)
		calls := state.GetRNG().Calls()
		for range references.DeedRaidCheckTurns * 3 {
			state.processDeedRaidTurn()
		}
		assert.Equal(t, calls, state.GetRNG().Calls())
		assert.Equal(t, 0, state.DeedRaidTurnsForTest())
	})
	t.Run("deed map, option off", func(t *testing.T) {
		state, _, _ := newBuildFixture(t)
		setCraftingEnabled(state, false)
		calls := state.GetRNG().Calls()
		for range references.DeedRaidCheckTurns * 3 {
			state.processDeedRaidTurn()
		}
		assert.Equal(t, calls, state.GetRNG().Calls())
		assert.Equal(t, 0, state.DeedRaiderCount())
	})
	t.Run("the option off ends a raid in progress", func(t *testing.T) {
		state, _, _ := newBuildFixture(t)
		require.Positive(t, state.StartDeedRaidForTest())
		setCraftingEnabled(state, false)
		state.processDeedRaidTurn()
		assert.Equal(t, 0, state.DeedRaiderCount())
	})
}

// A town's turns draw exactly the same RNG with the option on and off.
func TestRaidHookLeavesTownRNGDrawsUnchanged(t *testing.T) {
	draws := func(enabled bool) (uint64, int) {
		state, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).WithPlayerAt(15, 15).
			WithSystemCallbacks().WithRandomSeed(1234).Build()
		setCraftingEnabled(state, enabled)
		for range references.DeedRaidCheckTurns * 2 {
			state.FinishTurn()
		}

		return state.GetRNG().Calls(), state.RandomIntInRange(0, 1<<30)
	}
	offCalls, offNext := draws(false)
	onCalls, onNext := draws(true)
	assert.Equal(t, offCalls, onCalls)
	assert.Equal(t, offNext, onNext)
}

func TestRaidSpawnsAreDeterministicWithAFixedSeed(t *testing.T) {
	spawn := func() ([]references.Position, references.MonsterType) {
		state, _, _ := newBuildFixture(t)
		state.SetRandomSeed(4242)
		require.Positive(t, state.StartDeedRaidForTest())

		return raiderPositions(state), raidersOf(state)[0].EnemyReference.MonsterType
	}
	firstPositions, firstMonster := spawn()
	secondPositions, secondMonster := spawn()
	assert.Equal(t, firstPositions, secondPositions)
	assert.Equal(t, firstMonster, secondMonster)
}

func TestRaidSpawnSquaresAreValidOnEveryTerrain(t *testing.T) {
	for _, terrain := range []struct {
		name    string
		sprite  indexes.SpriteIndex
		terrain references.DeedTerrain
	}{
		{"grassland", indexes.Grass, references.DeedTerrainGrassland},
		{"forest", indexes.Forest3, references.DeedTerrainForest},
		{"hills", indexes.Hills1, references.DeedTerrainHills},
		{"desert", indexes.Desert1, references.DeedTerrainDesert},
	} {
		t.Run(terrain.name, func(t *testing.T) {
			state, _, deed := deedFixtureOfTerrain(t, terrain.sprite)
			require.Equal(t, terrain.terrain, deed.Terrain)
			// A small closed room just inside the buildable interior, west of the yard.
			for _, square := range []references.Position{{X: 4, Y: 14}, {X: 5, Y: 14}, {X: 6, Y: 14},
				{X: 4, Y: 16}, {X: 5, Y: 16}, {X: 6, Y: 16}, {X: 4, Y: 15}, {X: 6, Y: 15}} {
				if _, node := state.crafting.activeLayout.NodeAt(square); !node {
					buildAt(t, state, references.PieceStoneWall, square)
				}
			}
			party := state.MapState.PlayerLocation.Position
			squares := state.DeedRaidSpawnSquares()
			require.NotEmpty(t, squares)
			layered := state.GetLayeredMapByCurrentLocation()
			active := state.deedByID(deed.ID)
			for _, square := range squares {
				assert.LessOrEqual(t, references.DeedSquareDepth(square), references.DeedRaidSpawnDepth, "%v", square)
				_, node := state.crafting.activeLayout.NodeAt(square)
				assert.False(t, node, "%v is a node", square)
				_, built := active.Placement(square)
				assert.False(t, built, "%v is built", square)
				assert.NotEqual(t, party, square)
				assert.GreaterOrEqual(t, absInt(int(square.X-party.X))+absInt(int(square.Y-party.Y)),
					references.DeedRaidMinSpawnDistance)
				assert.True(t, ai.IsDeedRaiderTerrainPassable(layered.GetSmallMapPersistentTerrainTile(&square, 10)), "%v", square)
				assert.NotEqual(t, references.Position{X: 5, Y: 15}, square, "never inside a built room")
			}

			placed := state.StartDeedRaidForTest()
			require.Positive(t, placed)
			allowed := references.DeedRaiders(terrain.terrain)
			for _, raider := range raidersOf(state) {
				assert.True(t, slices.ContainsFunc(allowed, func(r references.DeedRaider) bool {
					return r.Monster == raider.EnemyReference.MonsterType
				}), "monster %d suits %s", raider.EnemyReference.MonsterType, terrain.name)
				assert.Contains(t, squares, raider.Pos())
			}
		})
	}
}

// buildHutAroundYard walls in the yard square with a closed door to the south and returns the
// built squares.
func buildHutAroundYard(t *testing.T, state *GameState) []references.Position {
	t.Helper()
	var built []references.Position
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			square := references.Position{X: yard.X + references.Coordinate(dx), Y: yard.Y + references.Coordinate(dy)}
			piece := references.PieceStoneWall
			if dx == 0 && dy == 1 {
				piece = references.PieceDoor
			}
			state.PartyState.Inventory.Pieces.IncrementBy(piece, 1)
			// Build from inside the hut where the square is orthogonal, else from just outside it.
			state.MapState.PlayerLocation.Position = yard
			direction, orthogonal := directionTo(yard, square)
			if !orthogonal {
				state.MapState.PlayerLocation.Position = references.Position{X: square.X, Y: square.Y + references.Coordinate(dy)}
				direction, _ = directionTo(state.MapState.PlayerLocation.Position, square)
			}
			require.True(t, state.BuildDeedPiece(piece, direction), "build at %v", square)
			built = append(built, square)
		}
	}
	state.MapState.PlayerLocation.Position = yard

	return built
}

func directionTo(from, to references.Position) (references.Direction, bool) {
	switch {
	case to.X == from.X && to.Y == from.Y-1:
		return references.Up, true
	case to.X == from.X && to.Y == from.Y+1:
		return references.Down, true
	case to.Y == from.Y && to.X == from.X-1:
		return references.Left, true
	case to.Y == from.Y && to.X == from.X+1:
		return references.Right, true
	default:
		return references.NoneDirection, false
	}
}

func TestRaidersNeverAlterPlacementsAndCannotPassAClosedDoor(t *testing.T) {
	state, _, deed := newBuildFixture(t)
	built := buildHutAroundYard(t, state)
	before := state.deedByID(deed.ID)
	placements := make(map[references.Position]DeedPlacement)
	for _, square := range before.PlacementPositions() {
		placement, _ := before.Placement(square)
		placements[square] = placement
	}
	overrides := before.TileOverridePositions()

	state.SpawnDeedRaidersForTest(references.MonsterOrc, []references.Position{{X: 15, Y: 26}, {X: 5, Y: 15}})
	passDeedTurns(state, 80)

	require.Nil(t, state.CombatScene, "a closed hut keeps the raid out")
	require.Equal(t, 2, state.DeedRaiderCount())
	after := state.deedByID(deed.ID)
	for square, placement := range placements {
		got, ok := after.Placement(square)
		require.True(t, ok)
		assert.Equal(t, placement, got)
		tile, _ := state.DeedTileAt(deed.ID, square)
		assert.Equal(t, placement.Piece.Piece().Sprite, tile)
	}
	assert.Equal(t, overrides, after.TileOverridePositions())
	for _, raider := range raidersOf(state) {
		assert.NotContains(t, built, raider.Pos(), "raiders never stand on walls or doors")
		distance := absInt(int(raider.Pos().X-yard.X)) + absInt(int(raider.Pos().Y-yard.Y))
		assert.Equal(t, 2, distance, "raiders close in on the walls: %v", raider.Pos())
	}
}

func TestRaidersPathAroundWallsAndStartCombatOnContact(t *testing.T) {
	state, _, deed := newBuildFixture(t)
	// A wall across the lane north of the entry, with the raider south of it.
	for x := 13; x <= 17; x++ {
		buildAt(t, state, references.PieceStoneWall, references.Position{X: references.Coordinate(x), Y: 22})
	}
	state.MapState.PlayerLocation.Position = yard
	state.SpawnDeedRaidersForTest(references.MonsterOrc, []references.Position{{X: 15, Y: 26}, {X: 5, Y: 5}})

	for range 40 {
		state.FinishTurn()
		if state.CombatScene != nil {
			break
		}
		for _, raider := range raidersOf(state) {
			_, built := state.deedByID(deed.ID).Placement(raider.Pos())
			require.False(t, built, "raider on a wall at %v", raider.Pos())
		}
	}
	require.NotNil(t, state.CombatScene, "the raider walks around the wall and attacks")
	origin := state.CombatScene.SmallMapOrigin
	require.NotNil(t, origin)
	assert.Equal(t, SmallMapEncounterDeedRaid, origin.SourceKind)
	assert.Equal(t, references.GladeCombatMap, origin.CombatMap)
	assert.Equal(t, references.MonsterOrc, origin.MonsterType)
	assert.Equal(t, 2, origin.RaiderCount)
	assert.Equal(t, 2, state.CombatScene.MonsterManager.Count(), "the whole raid joins the battle")
	assert.Equal(t, 0, state.DeedRaiderCount())
}

func TestADeedRaidBattleReturnsToTheIntactDeedMap(t *testing.T) {
	for _, reason := range []CombatEndReason{CombatVictory, CombatEvacuated} {
		state, callbacks, deed := newBuildFixture(t)
		wall := references.Position{X: yard.X + 2, Y: yard.Y}
		buildAt(t, state, references.PieceStoneWall, wall)
		state.MapState.PlayerLocation.Position = yard
		state.SpawnDeedRaidersForTest(references.MonsterRat,
			[]references.Position{{X: yard.X, Y: yard.Y + 1}, {X: 4, Y: 4}, {X: 27, Y: 27}})

		state.FinishTurn()
		require.NotNil(t, state.CombatScene)
		assert.True(t, callbacks.MessagesContain("Attacked!"))
		state.EndCombat(reason)

		require.Nil(t, state.CombatScene)
		require.True(t, state.IsOnDeedMap())
		assert.Equal(t, yard, state.MapState.PlayerLocation.Position)
		assert.Equal(t, 0, state.DeedRaiderCount(), "the raid ends with its battle")
		assert.Equal(t, indexes.SpriteIndex(indexes.StoneBrickWall), liveTile(state, wall))
		_, built := state.deedByID(deed.ID).Placement(wall)
		assert.True(t, built)
		// The deed keeps rolling for the next raid after the battle.
		turns := state.DeedRaidTurnsForTest()
		state.FinishTurn()
		assert.Equal(t, turns+1, state.DeedRaidTurnsForTest())
	}
}

func TestAttackingARaiderStartsTheRaidBattle(t *testing.T) {
	state, _, _ := newBuildFixture(t)
	state.SpawnDeedRaidersForTest(references.MonsterRat, []references.Position{{X: yard.X + 1, Y: yard.Y}})

	require.True(t, state.ActionAttackSmallMap(references.Right))

	require.NotNil(t, state.CombatScene)
	assert.Equal(t, SmallMapEncounterDeedRaid, state.CombatScene.SmallMapOrigin.SourceKind)
	assert.Equal(t, SmallMapEncounterAttack, state.CombatScene.SmallMapOrigin.Cause)
}

func TestARaiderOnABuiltFloorFightsOnTheTownFloor(t *testing.T) {
	state, _, _ := newBuildFixture(t)
	floor := references.Position{X: yard.X + 1, Y: yard.Y}
	buildAt(t, state, references.PieceWoodFloor, floor)
	state.MapState.PlayerLocation.Position = yard
	state.SpawnDeedRaidersForTest(references.MonsterRat, []references.Position{floor})

	state.FinishTurn()

	require.NotNil(t, state.CombatScene)
	assert.Equal(t, references.BrickCombatMap, state.CombatScene.SmallMapOrigin.CombatMap)
}

func TestLeavingTheDeedMapDespawnsRaidersAndSavesDropThem(t *testing.T) {
	state, _, _ := newBuildFixture(t)
	require.Positive(t, state.StartDeedRaidForTest())

	reloaded := saveAndReload(t, state)
	require.True(t, reloaded.IsOnDeedMap())
	assert.Equal(t, 0, reloaded.DeedRaiderCount(), "raiders are not saved")

	state.MapState.PlayerLocation.Position = references.DeedMapEntry
	state.SuppressDeedRaidsForTest(true) // Leaving takes turns; keep them from rolling again.
	leaveDeedMap(t, state)
	require.False(t, state.IsOnDeedMap())
	assert.Equal(t, 0, state.DeedRaiderCount())
	enterFixtureDeed(t, state)
	assert.Equal(t, 0, state.DeedRaiderCount(), "re-entering starts without raiders")
	assert.Equal(t, 0, state.DeedRaidTurnsForTest(), "and the raid count starts again")
}
