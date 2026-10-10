package ai

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

const (
	encounterPressurePhaseLimit  = 256
	encounterPressureSampleCount = 256
	encounterPressureShortWindow = 16
	minimumPressureMarginPercent = 10
)

type encounterPressureScenario struct {
	name   string
	start  references.Position
	player references.Position
	tiles  map[references.Position]indexes.SpriteIndex
}

type encounterPressureSummary struct {
	contacts   int
	shortCount int
	total      int
	maximum    int
}

// TestLargeMapPursuitEncounterPressure is a deterministic characterization
// harness. A negative actual-minus-oracle delta means A* creates contact sooner
// than the independent original-source pursuit oracle and therefore adds pressure.
func TestLargeMapPursuitEncounterPressure(t *testing.T) {
	seeds := make([]int64, encounterPressureSampleCount)
	for index := range seeds {
		seeds[index] = int64(index + 1)
	}
	scenarios := encounterPressureScenarios()

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			oraclePhases := make([]int, 0, len(seeds))
			actualPhases := make([]int, 0, len(seeds))
			for _, seed := range seeds {
				oraclePhase := runOriginalPursuitOracle(scenario, seed, encounterPressurePhaseLimit)
				actualPhase := runLargeMapPursuitController(t, scenario, seed, encounterPressurePhaseLimit)
				require.NotZero(t, oraclePhase, "oracle did not contact by phase %d for seed %d", encounterPressurePhaseLimit, seed)
				require.NotZero(t, actualPhase, "controller did not contact by phase %d for seed %d", encounterPressurePhaseLimit, seed)
				oraclePhases = append(oraclePhases, oraclePhase)
				actualPhases = append(actualPhases, actualPhase)
			}

			oracleSummary := summarizeEncounterPressure(oraclePhases)
			actualSummary := summarizeEncounterPressure(actualPhases)
			assert.LessOrEqual(t, actualSummary.shortCount, oracleSummary.shortCount,
				"A* must not produce more short-window contacts")
			assert.GreaterOrEqual(t,
				actualSummary.total*100,
				oracleSummary.total*(100+minimumPressureMarginPercent),
				"A* mean contact time must retain the conservative pressure margin",
			)

			delta := float64(actualSummary.total-oracleSummary.total) / float64(len(seeds))
			t.Logf(
				"samples=%d oracle=%s actual=%s actual-minus-oracle=%+.2f phases (%s)",
				len(seeds), formatEncounterPressure(oracleSummary), formatEncounterPressure(actualSummary), delta,
				encounterPressureInterpretation(delta),
			)
		})
	}
}

func encounterPressureScenarios() []encounterPressureScenario {
	const (
		minimum = references.Coordinate(84)
		maximum = references.Coordinate(116)
	)
	player := references.Position{X: 100, Y: 100}
	start := references.Position{X: 92, Y: 100}
	open := filledEncounterPressureTerrain(minimum, maximum, indexes.Grass)
	slow := filledEncounterPressureTerrain(minimum, maximum, indexes.Grass)
	for x := start.X + 1; x < player.X; x++ {
		tile := indexes.SpriteIndex(indexes.Swamp)
		if x%2 == 0 {
			tile = indexes.Forest3
		}
		for y := minimum; y <= maximum; y++ {
			slow[references.Position{X: x, Y: y}] = tile
		}
	}
	obstacle := filledEncounterPressureTerrain(minimum, maximum, indexes.Grass)
	obstacle[references.Position{X: 96, Y: 100}] = indexes.Water1

	return []encounterPressureScenario{
		{name: "open terrain", start: start, player: player, tiles: open},
		{name: "slow terrain route", start: start, player: player, tiles: slow},
		{name: "obstacle navigation", start: start, player: player, tiles: obstacle},
	}
}

func filledEncounterPressureTerrain(
	minimum, maximum references.Coordinate,
	tile indexes.SpriteIndex,
) map[references.Position]indexes.SpriteIndex {
	terrain := make(map[references.Position]indexes.SpriteIndex, int(maximum-minimum+1)*int(maximum-minimum+1))
	for x := minimum; x <= maximum; x++ {
		for y := minimum; y <= maximum; y++ {
			terrain[references.Position{X: x, Y: y}] = tile
		}
	}

	return terrain
}

func runOriginalPursuitOracle(scenario encounterPressureScenario, seed int64, phaseLimit int) int {
	rng := rand.New(rand.NewSource(seed)) //nolint:gosec // Fixed-seed gameplay characterization.
	position := scenario.start
	for phase := 1; phase <= phaseLimit; phase++ {
		if cardinallyAdjacent(position, scenario.player) {
			return phase
		}
		position = originalPursuitOracleStep(position, scenario.player, scenario.tiles, rng)
	}

	return 0
}

func originalPursuitOracleStep(
	position references.Position,
	player references.Position,
	terrain map[references.Position]indexes.SpriteIndex,
	rng *rand.Rand,
) references.Position {
	xStep := references.Position{X: position.X + signCoordinate(player.X-position.X), Y: position.Y}
	yStep := references.Position{X: position.X, Y: position.Y + signCoordinate(player.Y-position.Y)}
	playerward := [2]references.Position{xStep, yStep}
	if rng.Intn(2) != 0 { //nolint:gosec // Fixed-seed gameplay characterization.
		playerward[0], playerward[1] = playerward[1], playerward[0]
	}
	for _, destination := range playerward {
		if destination == position || !encounterPressureTileLegal(terrain, destination) {
			continue
		}
		return originalPursuitMoveTo(position, destination, terrain[destination], rng)
	}

	directions := position.GetFourDirectionsWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
	for range 3 {
		destination := directions[rng.Intn(len(directions))] //nolint:gosec // Fixed-seed gameplay characterization.
		if encounterPressureTileLegal(terrain, destination) {
			return originalPursuitMoveTo(position, destination, terrain[destination], rng)
		}
	}

	return position
}

func originalPursuitMoveTo(
	position references.Position,
	destination references.Position,
	tile indexes.SpriteIndex,
	rng *rand.Rand,
) references.Position {
	switch tile {
	case 4, 6, 7, 8, 30, 31:
		if rng.Intn(2) != 0 { //nolint:gosec // Fixed-seed gameplay characterization.
			return position
		}
	case 9, 10, 11, 12, 13, 14, 15:
		if rng.Intn(3) != 2 { //nolint:gosec // Fixed-seed gameplay characterization.
			return position
		}
	}

	return destination
}

func encounterPressureTileLegal(
	terrain map[references.Position]indexes.SpriteIndex,
	position references.Position,
) bool {
	_, exists := terrain[position]
	return exists && terrain[position] != indexes.Water1
}

func runLargeMapPursuitController(
	t *testing.T,
	scenario encounterPressureScenario,
	seed int64,
	phaseLimit int,
) int {
	t.Helper()
	rng := rand.New(rand.NewSource(seed)) //nolint:gosec // Fixed-seed gameplay characterization.
	controller, layeredMap := newLargeMapGenerationTestController(
		t, references.FloorNumber(references.OVERWORLD), 12, rng,
	)
	controller.debugOptions.MonsterGen = false
	controller.mapState.PlayerLocation.Position = scenario.player
	for position, tile := range scenario.tiles {
		position := position
		layeredMap.SetTileByLayer(map_state.MapLayer, &position, tile)
	}

	enemyReference := encounterPressureEnemyReference(t, controller.enemyReferences)
	enemy := map_units.NewEnemyNPC(*enemyReference, 0)
	enemy.SetPos(scenario.start)
	enemy.SetFloor(references.FloorNumber(references.OVERWORLD))
	require.True(t, controller.mapUnits.AddEnemy(enemy))
	activeEnemy, ok := controller.mapUnits[0].(*map_units.NPCEnemy)
	require.True(t, ok)
	controller.FreshenExistingNPCsOnMap()

	for phase := 1; phase <= phaseLimit; phase++ {
		contact := false
		controller.AdvanceNextTurnWithActorCheck(func(*map_units.NPCEnemy) OutdoorActorAction {
			contact = cardinallyAdjacent(activeEnemy.Pos(), scenario.player)
			if contact {
				return OutdoorActorAttacked
			}
			return OutdoorActorNoAttack
		})
		if contact {
			return phase
		}
	}

	return 0
}

func encounterPressureEnemyReference(
	t *testing.T,
	enemies *references.EnemyReferences,
) *references.EnemyReference {
	t.Helper()
	for index := range *enemies {
		if (*enemies)[index].MonsterType == references.MonsterOrc {
			return &(*enemies)[index]
		}
	}
	t.Fatal("missing land enemy reference")

	return nil
}

func cardinallyAdjacent(first, second references.Position) bool {
	return first.HeuristicTileDistance(second) == 1
}

func signCoordinate(value references.Coordinate) references.Coordinate {
	switch {
	case value < 0:
		return -1
	case value > 0:
		return 1
	default:
		return 0
	}
}

func summarizeEncounterPressure(phases []int) encounterPressureSummary {
	summary := encounterPressureSummary{contacts: len(phases)}
	for _, phase := range phases {
		summary.total += phase
		summary.maximum = max(summary.maximum, phase)
		if phase <= encounterPressureShortWindow {
			summary.shortCount++
		}
	}

	return summary
}

func formatEncounterPressure(summary encounterPressureSummary) string {
	return fmt.Sprintf(
		"contacts=%d within-%d=%d mean=%.2f max=%d",
		summary.contacts,
		encounterPressureShortWindow,
		summary.shortCount,
		float64(summary.total)/float64(summary.contacts),
		summary.maximum,
	)
}

func encounterPressureInterpretation(delta float64) string {
	switch {
	case delta < 0:
		return "A* is higher pressure"
	case delta > 0:
		return "A* is lower pressure"
	default:
		return "equal pressure"
	}
}
