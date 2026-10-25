// Integration tests for waterfall hazard system
// Tests DEX-based damage, dead character handling, and carpet behavior
package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
)

func TestWaterfallHazard_DexterityChecks_Integration(t *testing.T) {
	tests := []struct {
		name           string
		characterDex   int
		sourceValue    int
		expectedDamage bool
		description    string
	}{
		{
			name:           "Roll below high DEX passes",
			characterDex:   30,
			sourceValue:    59, // maps to 29
			expectedDamage: false,
			description:    "DEX 30: mapped roll 29 is below DEX",
		},
		{
			name:           "Roll equal to low DEX fails",
			characterDex:   5,
			sourceValue:    10, // maps to 5
			expectedDamage: true,
			description:    "DEX 5: mapped roll 5 equals DEX",
		},
		{
			name:           "Roll equal to mid DEX fails",
			characterDex:   15,
			sourceValue:    30, // maps to 15
			expectedDamage: true,
			description:    "DEX 15: mapped roll 15 equals DEX",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs, _ := NewIntegrationTestBuilder(t).
				WithLocation(references.BritanniaUnderworld). // Large map for waterfalls
				WithPlayerAt(15, 15).
				WithSystemCallbacks().
				Build()

			seed := seedForFirstIntnValue(t, gs, originalD30SourceValueCount, tt.sourceValue)
			gs.SetRandomSeed(seed)

			// Place waterfall tile BELOW player position (waterfall is below triggering tile)
			playerPos := references.Position{X: 15, Y: 15}
			waterfallPos := *playerPos.GetPositionDown() // Y=16
			gs.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &waterfallPos, indexes.Waterfall1)

			character := gs.PartyState.GetCharacter(0)
			character.Dexterity = uint8(tt.characterDex)
			initialHP := character.GetCurrentHp()

			// Trigger waterfall hazard (pass player position, function checks below it)
			gs.ProcessWaterfallHazard(&playerPos)

			// Process event queue to execute damage
			gs.EventQueueManager.Update() // Execute First()
			gs.ElapsedMs += 400           // Advance past fade effects (300ms + 100ms)
			for i := 0; i < 20; i++ {
				gs.EventQueueManager.Update()
				gs.ElapsedMs++
			}

			// Advance past glide duration (2500ms)
			gs.ElapsedMs += 2500
			for i := 0; i < 20; i++ {
				gs.EventQueueManager.Update()
				gs.ElapsedMs++
			}

			// Advance past fade in (300ms)
			gs.ElapsedMs += 300
			for i := 0; i < 20; i++ {
				gs.EventQueueManager.Update()
				gs.ElapsedMs++
			}

			if tt.expectedDamage {
				assert.Less(t, character.GetCurrentHp(), initialHP,
					"Should take 1 HP damage when DEX check fails")
			} else {
				assert.Equal(t, initialHP, character.GetCurrentHp(),
					"Should take no damage when DEX check passes")
			}
		})
	}
}

func TestWaterfallHazard_DeadCharactersSkipped_Integration(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld). // Large map for waterfalls
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	gs.SetRandomSeed(12345)

	// Place waterfall tile BELOW player position (waterfall is below triggering tile)
	playerPos := references.Position{X: 15, Y: 15}
	waterfallPos := *playerPos.GetPositionDown() // Y=16
	gs.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &waterfallPos, indexes.Waterfall1)

	character := gs.PartyState.GetCharacter(0)
	character.Dexterity = 5 // Low DEX
	initialHP := character.GetCurrentHp()
	character.SetStatus(party_state.Dead)

	runWaterfallHazard(gs, playerPos)

	assert.Equal(t, initialHP, character.GetCurrentHp(),
		"Dead character should not take waterfall damage")
}

func TestWaterfallHazard_CarpetNotProtected_Integration(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld). // Large map for waterfalls
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	gs.SetRandomSeed(3) // Seed that causes roll >= 5 for DEX 5

	// Board carpet manually (pattern from skiff_deep_water_test.go)
	carpet := map_units.NewNPCFriendlyVehiceNewRef(references.CarpetVehicle, references.Position{X: 15, Y: 15}, 0)
	gs.PartyVehicle = *carpet

	// Place waterfall tile BELOW player position (waterfall is below triggering tile)
	playerPos := references.Position{X: 15, Y: 15}
	waterfallPos := *playerPos.GetPositionDown() // Y=16
	gs.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &waterfallPos, indexes.Waterfall1)

	character := gs.PartyState.GetCharacter(0)
	character.Dexterity = 5 // Low DEX = high failure chance
	initialHP := character.GetCurrentHp()

	runWaterfallHazard(gs, playerPos)

	assert.Less(t, character.GetCurrentHp(), initialHP,
		"Carpet should NOT protect from waterfall damage")
}

func TestWaterfallHazard_NonWaterfallTile_NoEffect_Integration(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld). // Large map for waterfalls
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	gs.SetRandomSeed(12345)

	character := gs.PartyState.GetCharacter(0)
	character.Dexterity = 5 // Low DEX
	initialHP := character.GetCurrentHp()

	// Trigger waterfall hazard on non-waterfall tile
	gs.ProcessWaterfallHazard(&references.Position{X: 15, Y: 15})

	assert.Equal(t, initialHP, character.GetCurrentHp(),
		"Non-waterfall tile should not trigger hazard")
}

// runWaterfallHazard triggers the waterfall below playerPos and advances the
// queue through its message, glide and fade-in phases.
func runWaterfallHazard(gs *GameState, playerPos references.Position) {
	gs.ProcessWaterfallHazard(&playerPos)
	gs.EventQueueManager.Update()
	for _, advanceMs := range []int64{400, 2500, 300} {
		gs.ElapsedMs += advanceMs
		for range 20 {
			gs.EventQueueManager.Update()
			gs.ElapsedMs++
		}
	}
}
