// Test helpers for integration testing with real Ultima V data
// Following patterns established in TESTING.md
package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/test/helpers"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"
)

// IntegrationTestBuilder provides a fluent API for creating integration tests
// with real game data, comprehensive state setup, and proper cleanup.
type IntegrationTestBuilder struct {
	t               *testing.T
	gameConfig      *config.UltimaVConfiguration
	gameRefs        *references.GameReferences
	gameState       *GameState
	location        references.Location
	floor           references.FloorNumber
	playerX         int
	playerY         int
	systemCallbacks *MockSystemCallbacks
	skipIfNoData    bool
	randomSeed      *uint64 // Optional random seed for deterministic testing
}

// NewIntegrationTestBuilder creates a new builder for integration tests
func NewIntegrationTestBuilder(t *testing.T) *IntegrationTestBuilder {
	return &IntegrationTestBuilder{
		t:            t,
		location:     references.BritanniaUnderworld,
		floor:        references.FloorNumber(0),
		playerX:      50,
		playerY:      50,
		skipIfNoData: true,
	}
}

// WithLocation sets the initial game location
func (b *IntegrationTestBuilder) WithLocation(location references.Location) *IntegrationTestBuilder {
	b.location = location

	return b
}

// WithFloor sets the floor number within the location
func (b *IntegrationTestBuilder) WithFloor(floor references.FloorNumber) *IntegrationTestBuilder {
	b.floor = floor

	return b
}

// WithPlayerAt sets the initial player position
func (b *IntegrationTestBuilder) WithPlayerAt(x, y int) *IntegrationTestBuilder {
	b.playerX = x
	b.playerY = y

	return b
}

// WithSystemCallbacks sets up mock system callbacks for testing
func (b *IntegrationTestBuilder) WithSystemCallbacks() *IntegrationTestBuilder {
	b.systemCallbacks = NewMockSystemCallbacks(b.t)

	return b
}

// FailIfNoGameData changes the default behavior to fail instead of skip when game data isn't found
func (b *IntegrationTestBuilder) FailIfNoGameData() *IntegrationTestBuilder {
	b.skipIfNoData = false

	return b
}

// WithRandomSeed sets a deterministic random seed for testing
func (b *IntegrationTestBuilder) WithRandomSeed(seed uint64) *IntegrationTestBuilder {
	b.randomSeed = &seed

	return b
}

// Build creates the fully configured GameState for testing
func (b *IntegrationTestBuilder) Build() (*GameState, *MockSystemCallbacks) {
	b.t.Helper()

	// Load game configuration and references (cached)
	if err := b.loadGameData(); err != nil {
		if !b.skipIfNoData {
			b.t.Fatalf("Failed to load Ultima V game data: %v", err)
		}
		originaldata.Check(b.t, err)

		return nil, nil
	}

	// Use the checked-in bootstrap save rather than mutable player progress.
	rawSave := helpers.LoadBootstrapSaveData(b.t)
	var err error
	b.gameState, err = NewGameStateFromLegacyTemplateBytes(
		rawSave,
		b.gameConfig,
		b.gameRefs,
		19, 13) // Standard screen dimensions for testing
	if err != nil {
		b.t.Fatalf("Failed to load legacy test save: %v", err)
	}
	// Reset campaign progress that changes command behavior while retaining map
	// and character data needed by integration fixtures.
	b.gameState.SetHMSCapeActivated(false)
	b.gameState.ShadowlordState = references.NewShadowlordState()
	b.gameState.DungeonStates = [8]bool{}
	b.gameState.PartyState.RestoreNPCFlags(nil, nil)
	b.gameState.PartyState.Karma = party_state.NewKarma(50)
	b.gameState.PartyState.Inventory = *party_state.NewInventory()
	b.gameState.PartyState.Inventory.Provisions.Food.Set(999)
	b.gameState.PartyState.Inventory.Provisions.Keys.Set(1)
	b.gameState.SearchManager.GetAuthenticLocationManager().ResetAllFoundStatus()
	b.gameState.SearchManager.reagentCachesFoundDay = [3]byte{}
	b.gameState.questJournal = newQuestJournal()

	// Set up initial position and location
	b.gameState.MapState.PlayerLocation.Location = b.location
	b.gameState.MapState.PlayerLocation.Floor = b.floor
	b.gameState.MapState.PlayerLocation.Position = references.Position{
		X: references.Coordinate(b.playerX),
		Y: references.Coordinate(b.playerY),
	}

	// Set up mock system callbacks if requested
	if b.systemCallbacks != nil {
		b.gameState.SystemCallbacks = b.systemCallbacks.ToSystemCallbacks()
		// Update environmental systems with proper callbacks
		b.gameState.SetupEnvironmentalSystemCallbacks()
	}

	// Set deterministic random seed if specified
	if b.randomSeed != nil {
		b.gameState.SetRandomSeed(*b.randomSeed)
	}

	// Update to specified map type and load NPCs/objects
	switch b.location.GetMapType() {
	case references.DungeonMapType:
		b.gameState.MapState.PlayerLocation.Position = *b.gameState.MapState.PlayerLocation.Position.GetWrapped(
			references.XDungeonMapTiles,
			references.YDungeonMapTiles,
		)
		dungeon, err := b.gameState.PrepareDungeonState(
			b.location,
			references.DungeonFacingEast,
			references.DungeonArrivedFromAbove,
		)
		if err != nil {
			b.t.Fatalf("Failed to prepare dungeon integration state: %v", err)
		}
		b.gameState.installDungeonState(dungeon)
	case references.SmallMapType:
		b.gameState.UpdateSmallMap(b.gameRefs.TileReferences, b.gameRefs.LocationReferences)
	case references.LargeMapType:
		// For large maps, ensure quest items are properly placed
		b.gameState.RefreshQuestItemsOnMap()
	case references.CombatMapType, references.CampingMapType:
		// Focused combat/camping builders materialize their own scene state.
	}
	// A neutral daylight hour prevents the saved campaign's transient moongate
	// presentation from leaking into unrelated integration tests.
	b.gameState.DateTime.Hour = 12
	b.gameState.DateTime.Minute = 0
	b.gameState.lastProcessedHour = int(b.gameState.DateTime.Hour)
	b.gameState.initializeMoongateStage()
	if b.systemCallbacks != nil {
		b.systemCallbacks.BindGameState(b.gameState)
	}

	b.t.Logf("Integration test initialized: location=%v, floor=%v, player=(%d,%d)",
		b.location, b.floor, b.playerX, b.playerY)

	return b.gameState, b.systemCallbacks
}

// loadGameData loads game configuration and references with caching
func (b *IntegrationTestBuilder) loadGameData() error {
	var err error
	b.gameConfig, b.gameRefs, err = helpers.LoadGameData()

	return err
}
