package game_state

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// deedFixtureSquare lies far from every settlement and moongate.
var deedFixtureSquare = references.Position{X: 130, Y: 100}

// newDeedFixture puts the party on foot on open grass of the overworld, with 1000 gold.
func newDeedFixture(t *testing.T, enabled bool) (*GameState, *MockSystemCallbacks) {
	t.Helper()
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(int(deedFixtureSquare.X), int(deedFixtureSquare.Y)).
		WithSystemCallbacks().
		WithRandomSeed(77).
		Build()
	setCraftingEnabled(state, enabled)
	state.DebugOptions.MonsterGen = false
	state.CurrentNPCAIController.GetNpcs().ClearAllMapUnits()
	state.ItemStacksMap = *references.NewItemStacksMap()
	for dy := -3; dy <= 3; dy++ {
		for dx := -3; dx <= 3; dx++ {
			setOverworldTile(state, offsetSquare(deedFixtureSquare, dx, dy), indexes.Grass)
		}
	}
	state.PartyState.Inventory.Gold.Set(1000)
	callbacks.Reset()

	return state, callbacks
}

func setCraftingEnabled(state *GameState, enabled bool) {
	configuration := *state.GameConfig
	gameplay := *configuration.GameplayConfiguration
	gameplay.CraftingEnabled = enabled
	configuration.GameplayConfiguration = &gameplay
	state.GameConfig = &configuration
}

func offsetSquare(position references.Position, dx, dy int) references.Position {
	return *(&references.Position{
		X: position.X + references.Coordinate(dx), Y: position.Y + references.Coordinate(dy),
	}).GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
}

func setOverworldTile(state *GameState, position references.Position, tile indexes.SpriteIndex) {
	state.overworldMap().UnSetTileByLayer(map_state.MapOverrideLayer, &position)
	state.overworldMap().SetTileByLayer(map_state.MapLayer, &position, tile)
}

func overworldTile(state *GameState, position references.Position) indexes.SpriteIndex {
	return state.overworldMap().SpriteIndexByLayer(map_state.MapLayer, position)
}

func stakeFixtureDeed(t *testing.T, state *GameState, name string) Deed {
	t.Helper()
	deed, err := state.StakeDeed(name)
	require.NoError(t, err)

	return deed
}

func runNamedWorkflow(t *testing.T, state *GameState, name string) {
	t.Helper()
	for range 200 {
		if !state.EventQueueManager.HasWorkflow(name) {
			return
		}
		state.EventQueueManager.Update()
		state.ElapsedMs += 1000
		state.ProcessWindThroughElapsed(state.ElapsedMs)
	}
	require.False(t, state.EventQueueManager.HasWorkflow(name))
}

func enterFixtureDeed(t *testing.T, state *GameState) {
	t.Helper()
	require.Equal(t, EnterLargeMapTransition, state.ResolveEnterLargeMap())
	runNamedWorkflow(t, state, "enter_location")
}

func leaveDeedMap(t *testing.T, state *GameState) {
	t.Helper()
	state.EventQueueManager.Workflow("test_exit").Priority(event_queue.PriorityLow).Sequential().First(func() {}).
		ThenSequence(state.ThenExitLocationTransition()).Start()
	runNamedWorkflow(t, state, "test_exit")
}

func TestProposeDeedRefusesEveryUnsuitableSquare(t *testing.T) {
	probe, _ := newDeedFixture(t, true)
	britain, _ := probe.GameReferences.LocationReferences.WorldLocations.GetWorldLocation(references.Britain)
	for _, testCase := range []struct {
		name    string
		prepare func(*GameState)
		want    error
	}{
		{"option off", func(s *GameState) { setCraftingEnabled(s, false) }, ErrCraftingDisabled},
		{"water", func(s *GameState) { setOverworldTile(s, deedFixtureSquare, indexes.Water1) }, ErrDeedNotBuildable},
		{"swamp", func(s *GameState) { setOverworldTile(s, deedFixtureSquare, indexes.Swamp) }, ErrDeedNotBuildable},
		{"mountains", func(s *GameState) { setOverworldTile(s, deedFixtureSquare, indexes.SmallMountains) }, ErrDeedNotBuildable},
		{"road", func(s *GameState) { setOverworldTile(s, deedFixtureSquare, indexes.Path1) }, ErrDeedNotBuildable},
		{"settlement sprite", func(s *GameState) { setOverworldTile(s, deedFixtureSquare, indexes.Village) }, ErrDeedNotBuildable},
		{"world location", func(s *GameState) { s.MapState.PlayerLocation.Position = britain }, ErrDeedSettlement},
		{"charted moongate", func(s *GameState) {
			s.MapState.PlayerLocation.Position = references.ChartedMoonstoneOrigins()[1].Location.Position
			setOverworldTile(s, s.MapState.PlayerLocation.Position, indexes.Grass)
		}, ErrDeedMoongate},
		{"underworld", func(s *GameState) { s.MapState.PlayerLocation.Floor = references.FloorNumber(references.UNDERWORLD) }, ErrDeedNotOnSurface},
		{"town", func(s *GameState) { s.MapState.PlayerLocation.Location = references.Britain }, ErrDeedNotOnSurface},
		{"aboard a horse", func(s *GameState) {
			s.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(references.HorseVehicle, deedFixtureSquare, 0)
		}, ErrDeedNotOnFoot},
		{"not enough gold", func(s *GameState) { s.PartyState.Inventory.Gold.Set(references.FirstDeedGoldCost - 1) }, ErrDeedNotEnoughGold},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			state, _ := newDeedFixture(t, true)
			testCase.prepare(state)

			require.ErrorIs(t, state.ProposeDeed().Err, testCase.want)
			_, err := state.StakeDeed("Home")
			require.ErrorIs(t, err, testCase.want)
			assert.Empty(t, state.Deeds())
		})
	}
}

func TestProposeDeedRefusesOwnAndAdjacentSquaresButNotDiagonals(t *testing.T) {
	state, _ := newDeedFixture(t, true)
	stakeFixtureDeed(t, state, "Home")

	assert.ErrorIs(t, state.ProposeDeed().Err, ErrDeedAlreadyOwned)
	for _, offset := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		state.MapState.PlayerLocation.Position = offsetSquare(deedFixtureSquare, offset[0], offset[1])
		assert.ErrorIs(t, state.ProposeDeed().Err, ErrDeedTooClose, "offset %v", offset)
	}
	state.MapState.PlayerLocation.Position = offsetSquare(deedFixtureSquare, 1, 1)
	require.NoError(t, state.ProposeDeed().Err, "diagonal neighbours are allowed")
	stakeFixtureDeed(t, state, "Second")
	assert.Len(t, state.Deeds(), 2, "there is no deed limit")
}

func TestFirstDeedCostsLessThanLaterDeeds(t *testing.T) {
	state, callbacks := newDeedFixture(t, true)
	state.PartyState.Inventory.Gold.Set(references.FirstDeedGoldCost)

	require.Equal(t, uint16(references.FirstDeedGoldCost), state.DeedPrice())
	require.NoError(t, state.ProposeDeed().Err, "a new party affords its first deed")
	stakeFixtureDeed(t, state, "First")
	assert.Zero(t, state.PartyState.Inventory.Gold.Get())

	assert.Equal(t, uint16(references.DeedGoldCost), state.DeedPrice(), "every later deed pays the flat price")
	state.MapState.PlayerLocation.Position = offsetSquare(deedFixtureSquare, 2, 0)
	state.PartyState.Inventory.Gold.Set(references.DeedGoldCost - 1)
	proposal := state.ProposeDeed()
	require.ErrorIs(t, proposal.Err, ErrDeedNotEnoughGold)
	callbacks.Reset()
	assert.Equal(t, "A deed costs 200 gold; thou hast 199.", state.DeedRejection(proposal.Err))
	state.PartyState.Inventory.Gold.Set(references.DeedGoldCost)
	stakeFixtureDeed(t, state, "Second")
	assert.Zero(t, state.PartyState.Inventory.Gold.Get())
}

func TestTouchesDeedWrapsAcrossTheWorldSeam(t *testing.T) {
	state, _ := newDeedFixture(t, true)
	state.crafting.deeds = []*Deed{{ID: 0, Name: "Seam", Square: references.Position{X: 255, Y: 40}}}

	assert.True(t, state.touchesDeed(references.Position{X: 0, Y: 40}))
	assert.False(t, state.touchesDeed(references.Position{X: 1, Y: 40}))
}

func TestStakeDeedPaysMarksTheHutAndPinsTheAtlas(t *testing.T) {
	state, callbacks := newDeedFixture(t, true)
	setOverworldTile(state, deedFixtureSquare, indexes.Forest3)

	deed := stakeFixtureDeed(t, state, "  Oakhollow  ")

	assert.Equal(t, "Oakhollow", deed.Name)
	assert.Equal(t, references.DeedTerrainForest, deed.Terrain)
	assert.Equal(t, references.DeedGeneratorVersion, deed.GeneratorVersion)
	assert.Equal(t, references.DeedSeed(0, deedFixtureSquare), deed.Seed)
	assert.Equal(t, uint16(1000-references.FirstDeedGoldCost), state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, indexes.SpriteIndex(indexes.Hut), overworldTile(state, deedFixtureSquare))
	assert.Equal(t, []string{"Oakhollow is yours!"}, callbacks.Messages)
	assert.Empty(t, callbacks.TimeAdvanced(), "the scene charges the ordinary turn, never the action")
	notes := state.CaptureAtlasNotes()
	require.Len(t, notes.Records, 1)
	assert.Equal(t, "Oakhollow", notes.Records[0].Label)
	assert.Equal(t, journal.SiteLocation{Map: references.BritanniaUnderworld, Floor: 0, X: 130, Y: 100}, notes.Records[0].Position)
	require.NoError(t, journal.ValidateAtlasNotes(notes))
}

func TestStakeDeedRejectsNamesOutsideTheAtlasRepertoire(t *testing.T) {
	state, _ := newDeedFixture(t, true)
	for _, name := range []string{"", "   ", "Café", "abcdefghijklmnopqrstuvwxy"} {
		_, err := state.StakeDeed(name)
		assert.ErrorIs(t, err, ErrInvalidDeedName, "name %q", name)
	}
	// Every accepted deed name is also an accepted Atlas label.
	for _, name := range []string{"A", "Lord's Rest", "~ x ~", "abcdefghijklmnopqrstuvwx"} {
		require.True(t, references.IsValidDeedName(name), name)
		assert.NoError(t, journal.ValidateAtlasNoteText(name, ""), name)
	}
	assert.Empty(t, state.Deeds())
	assert.Equal(t, uint16(1000), state.PartyState.Inventory.Gold.Get())
}

func TestDeedTerrainFollowsTheStakedSquare(t *testing.T) {
	for sprite, want := range map[indexes.SpriteIndex]references.DeedTerrain{
		indexes.Grass: references.DeedTerrainGrassland, indexes.Forest1: references.DeedTerrainGrassland,
		indexes.Forest4: references.DeedTerrainForest, indexes.Hills1: references.DeedTerrainHills,
		indexes.RightDesert2: references.DeedTerrainDesert,
	} {
		state, _ := newDeedFixture(t, true)
		setOverworldTile(state, deedFixtureSquare, sprite)
		assert.Equal(t, want, stakeFixtureDeed(t, state, "Plot").Terrain, "sprite %d", sprite)
	}
}

func TestEnterDeedHutOpensTheDeedMapAndLeavingReturnsToTheHut(t *testing.T) {
	state, _ := newDeedFixture(t, true)
	deed := stakeFixtureDeed(t, state, "Home")

	enterFixtureDeed(t, state)

	location := state.MapState.PlayerLocation
	assert.Equal(t, references.DeedLocation, location.Location)
	assert.Equal(t, references.DeedMapEntry, location.Position)
	assert.Equal(t, references.FloorNumber(0), location.Floor)
	assert.True(t, state.IsOnDeedMap())
	reference := state.GetCurrentSmallLocationReference()
	require.NotNil(t, reference)
	assert.Equal(t, "Home", reference.FriendlyLocationName)
	assert.Equal(t, "Enter your land\n\nHome", reference.EnteringText)
	assert.Equal(t, references.DeedProfile(deed.Terrain).Outer, reference.GetOuterTile())
	assert.Nil(t, state.GameReferences.LocationReferences.GetLocationReference(references.DeedLocation),
		"deed maps never enter the shared reference tables")
	assert.Empty(t, state.atlasLandmarks, "a deed is not a public landmark")
	layout := deed.Layout()
	live := state.GetLayeredMapByCurrentLocation()
	assert.Equal(t, layout.Tile(references.DeedMapCentre), live.SpriteIndexByLayer(map_state.MapLayer, references.DeedMapCentre))

	leaveDeedMap(t, state)

	location = state.MapState.PlayerLocation
	assert.Equal(t, references.BritanniaUnderworld, location.Location)
	assert.Equal(t, deedFixtureSquare, location.Position)
	assert.Equal(t, references.FloorNumber(0), location.Floor)
	assert.False(t, state.IsOnDeedMap())
	_, active := state.ActiveDeed()
	assert.False(t, active, "leaving releases the deed map")
}

func TestEnterOnAHutAnswersWhatWhileTheOptionIsOff(t *testing.T) {
	state, callbacks := newDeedFixture(t, true)
	stakeFixtureDeed(t, state, "Home")
	setCraftingEnabled(state, false)
	callbacks.Reset()

	assert.Equal(t, EnterLargeMapRejected, state.ResolveEnterLargeMap())
	assert.Equal(t, references.BritanniaUnderworld, state.MapState.PlayerLocation.Location)
}

func TestSetDeedTileKeepsOnlyDifferencesAndUpdatesTheLoadedMap(t *testing.T) {
	state, _ := newDeedFixture(t, true)
	deed := stakeFixtureDeed(t, state, "Home")
	enterFixtureDeed(t, state)
	square := references.Position{X: 10, Y: 10}
	generated, ok := state.DeedTileAt(deed.ID, square)
	require.True(t, ok)

	state.SetDeedTile(deed.ID, square, indexes.BrickFloor)

	current, _ := state.DeedTileAt(deed.ID, square)
	assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), current)
	assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), state.GetLayeredMapByCurrentLocation().SpriteIndexByLayer(map_state.MapLayer, square))
	assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), state.GetCurrentSmallLocationReference().GetTileNumber(square, 0))
	active, _ := state.ActiveDeed()
	assert.Equal(t, []references.Position{square}, active.TileOverridePositions())

	state.SetDeedTile(deed.ID, square, generated)
	active, _ = state.ActiveDeed()
	assert.Empty(t, active.TileOverridePositions(), "restoring the generated tile drops the override")
}

func TestDeedResourceNodesComeFromTheGeneratorAndReportCoveredSquares(t *testing.T) {
	state, _ := newDeedFixture(t, true)
	deed := stakeFixtureDeed(t, state, "Home")
	layout := deed.Layout()
	nodes := state.DeedResourceNodes(deed.ID)
	require.Len(t, nodes, len(layout.Nodes))
	for _, node := range nodes {
		assert.False(t, node.Covered)
		assert.Equal(t, node.Sprite, node.Current)
	}

	state.SetDeedTile(deed.ID, nodes[0].Position, indexes.Grass)

	assert.True(t, state.DeedResourceNodes(deed.ID)[0].Covered)
}

func saveAndReload(t *testing.T, state *GameState) *GameState {
	t.Helper()
	snapshot, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)
	data, err := save_snapshot.EncodeCanonical(save_snapshot.Envelope{
		Format: save_snapshot.EnvelopeFormat, EnvelopeVersion: save_snapshot.CurrentEnvelopeVersion,
		Metadata: save_snapshot.Metadata{Name: "Deeds", CreatedAt: time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)},
		Snapshot: snapshot,
	})
	require.NoError(t, err)
	decoded, err := save_snapshot.Decode(data)
	require.NoError(t, err)
	restored, err := NewGameStateFromSaveSnapshot(decoded.Snapshot, state.GameConfig, state.GameReferences,
		state.MapState.XTilesVisibleOnGameScreen, state.MapState.YTilesVisibleOnGameScreen)
	require.NoError(t, err)
	callbacks := NewMockSystemCallbacks(t)
	restored.SystemCallbacks = callbacks.ToSystemCallbacks()
	restored.SetupEnvironmentalSystemCallbacks()
	callbacks.BindGameState(restored)

	return restored
}

func TestDeedsSurviveASaveOnTheOverworld(t *testing.T) {
	state, _ := newDeedFixture(t, true)
	deed := stakeFixtureDeed(t, state, "Home")
	state.SetDeedTile(deed.ID, references.Position{X: 9, Y: 9}, indexes.WoodFloor1)
	state.PartyState.Inventory.Materials.Set(references.Ore, 12)
	state.PartyState.Inventory.Tools.Set(references.Pick, 1)

	restored := saveAndReload(t, state)

	require.Equal(t, state.Deeds(), restored.Deeds())
	assert.Equal(t, indexes.SpriteIndex(indexes.Hut), overworldTile(restored, deedFixtureSquare), "the hut is a map mutation")
	assert.Equal(t, uint16(12), restored.PartyState.Inventory.Materials.Get(references.Ore))
	assert.Equal(t, uint16(1), restored.PartyState.Inventory.Tools.Get(references.Pick))
	tile, _ := restored.DeedTileAt(deed.ID, references.Position{X: 9, Y: 9})
	assert.Equal(t, indexes.SpriteIndex(indexes.WoodFloor1), tile)
	assert.Len(t, restored.CaptureAtlasNotes().Records, 1)
}

func TestDeedMapSurvivesASaveMadeOnIt(t *testing.T) {
	state, _ := newDeedFixture(t, true)
	deed := stakeFixtureDeed(t, state, "Home")
	enterFixtureDeed(t, state)
	state.SetDeedTile(deed.ID, references.Position{X: 12, Y: 12}, indexes.BrickFloor)
	state.MapState.PlayerLocation.Position = references.Position{X: 15, Y: 20}

	snapshot, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)
	require.NotNil(t, snapshot.Crafting.ActiveDeedID)
	assert.Equal(t, deed.ID, *snapshot.Crafting.ActiveDeedID)
	assert.Equal(t, int(references.DeedLocation), snapshot.World.Position.Location)
	restored := saveAndReload(t, state)

	assert.Equal(t, references.DeedLocation, restored.MapState.PlayerLocation.Location)
	assert.Equal(t, references.Position{X: 15, Y: 20}, restored.MapState.PlayerLocation.Position)
	assert.True(t, restored.IsOnDeedMap())
	live := restored.GetLayeredMapByCurrentLocation()
	assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), live.SpriteIndexByLayer(map_state.MapLayer, references.Position{X: 12, Y: 12}))

	leaveDeedMap(t, restored)
	assert.Equal(t, deedFixtureSquare, restored.MapState.PlayerLocation.Position)
	assert.Equal(t, references.BritanniaUnderworld, restored.MapState.PlayerLocation.Location)
}

func TestDeedsOfAnEarlierGameNeverLeakIntoTheNext(t *testing.T) {
	first, _ := newDeedFixture(t, true)
	stakeFixtureDeed(t, first, "Home")
	enterFixtureDeed(t, first)
	second, _ := newDeedFixture(t, true)
	second.GameReferences = first.GameReferences

	assert.Empty(t, second.Deeds())
	assert.Nil(t, second.locationReference(references.DeedLocation))
	restored := saveAndReload(t, second)
	assert.Empty(t, restored.Deeds())
	assert.False(t, restored.IsOnDeedMap())
	assert.Nil(t, first.GameReferences.LocationReferences.GetLocationReference(references.DeedLocation))
}

func TestSaveFromBeforeVersion26LoadsWithNoCraftingState(t *testing.T) {
	state, _ := newDeedFixture(t, true)
	snapshot, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)
	snapshot.Crafting = save_snapshot.CraftingSnapshot{}

	restored, err := NewGameStateFromSaveSnapshot(snapshot, state.GameConfig, state.GameReferences,
		state.MapState.XTilesVisibleOnGameScreen, state.MapState.YTilesVisibleOnGameScreen)
	require.NoError(t, err)

	assert.Empty(t, restored.Deeds())
	for _, material := range references.MaterialCatalog() {
		assert.Zero(t, restored.PartyState.Inventory.Materials.Get(material))
	}
}

func TestDeedMapSaveIsRejectedWithoutItsDeed(t *testing.T) {
	state, _ := newDeedFixture(t, true)
	stakeFixtureDeed(t, state, "Home")
	enterFixtureDeed(t, state)
	snapshot, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)
	snapshot.Crafting.ActiveDeedID = nil

	_, err = NewGameStateFromSaveSnapshot(snapshot, state.GameConfig, state.GameReferences,
		state.MapState.XTilesVisibleOnGameScreen, state.MapState.YTilesVisibleOnGameScreen)
	assert.Error(t, err)
}

func TestCraftingCommandsApplyOnlyWhereDocumented(t *testing.T) {
	state, _ := newDeedFixture(t, true)
	assert.False(t, state.CraftingCommandApplies(CraftingGather), "gathering is deed-map only")
	assert.True(t, state.CraftingCommandApplies(CraftingCraft))
	assert.True(t, state.CraftingCommandApplies(CraftingBuild))

	stakeFixtureDeed(t, state, "Home")
	enterFixtureDeed(t, state)
	for _, command := range []CraftingCommand{CraftingGather, CraftingCraft, CraftingBuild} {
		assert.True(t, state.CraftingCommandApplies(command), "command %d on the deed map", command)
	}

	setCraftingEnabled(state, false)
	assert.False(t, state.CraftingCommandApplies(CraftingBuild))
}

func TestCraftingCommandsDoNotApplyInTowns(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.Britain).WithPlayerAt(15, 15).
		WithSystemCallbacks().Build()
	setCraftingEnabled(state, true)
	for _, command := range []CraftingCommand{CraftingGather, CraftingCraft, CraftingBuild} {
		assert.False(t, state.CraftingCommandApplies(command))
	}
}

func TestDeedMapSextantFixIsTheHutSquare(t *testing.T) {
	state, _ := newDeedFixture(t, true)
	stakeFixtureDeed(t, state, "Home")
	enterFixtureDeed(t, state)
	state.GameConfig.GameplayConfiguration.FlexibleSextantUse = true
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)

	fix, ok := state.CurrentSextantFix()

	require.True(t, ok)
	assert.Equal(t, references.PlayerLocation{Location: references.BritanniaUnderworld, Floor: 0, Position: deedFixtureSquare}, fix)
}
