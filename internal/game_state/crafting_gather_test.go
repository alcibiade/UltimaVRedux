package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// newGatherFixture stakes a grassland deed (every node kind) and enters it, at 10:00.
func newGatherFixture(t *testing.T) (*GameState, *MockSystemCallbacks, Deed) {
	t.Helper()
	state, callbacks := newDeedFixture(t, true)
	state.DateTime.Hour, state.DateTime.Minute = 10, 0
	deed := stakeFixtureDeed(t, state, "Home")
	require.Equal(t, references.DeedTerrainGrassland, deed.Terrain)
	enterFixtureDeed(t, state)
	state.DateTime.Hour, state.DateTime.Minute = 10, 0
	callbacks.Reset()

	return state, callbacks, deed
}

// nodesOf returns the deed's nodes of one kind and sprite, in row-major order.
func nodesOf(state *GameState, deed Deed, kind references.DeedResourceKind, sprite indexes.SpriteIndex) []DeedResourceNodeState {
	var found []DeedResourceNodeState
	for _, node := range state.DeedResourceNodes(deed.ID) {
		if node.Kind == kind && (sprite == 0 || node.Sprite == sprite) {
			found = append(found, node)
		}
	}

	return found
}

// standSouthOf puts the party just south of a node; nodes never touch, so that square is no node.
func standSouthOf(state *GameState, node DeedResourceNodeState) {
	state.MapState.PlayerLocation.Position = references.Position{X: node.Position.X, Y: node.Position.Y + 1}
}

func gatherMinutes(callbacks *MockSystemCallbacks) int {
	total := 0
	for _, minutes := range callbacks.TimeAdvanced() {
		total += minutes
	}

	return total
}

func TestGatherIsRefusedOffTheDeedMap(t *testing.T) {
	state, callbacks := newDeedFixture(t, true)

	assert.False(t, state.GatherDeedNode(references.Up))

	assert.Equal(t, []string{"Not here!"}, callbacks.Messages)
	assert.Empty(t, callbacks.TimeAdvanced())
}

func TestGatherWoodWithTheAxeYieldsAndSpendsTheStatedMinutes(t *testing.T) {
	state, callbacks, deed := newGatherFixture(t)
	state.PartyState.Inventory.Tools.Set(references.Axe, 1)
	tree := nodesOf(state, deed, references.DeedResourceWood, indexes.Forest3)[0]
	standSouthOf(state, tree)

	require.True(t, state.GatherDeedNode(references.Up))

	wood := int(state.PartyState.Inventory.Materials.Get(references.Wood))
	assert.GreaterOrEqual(t, wood, 2)
	assert.LessOrEqual(t, wood, 4)
	assert.True(t, callbacks.MessagesContain("Gathered"))
	assert.True(t, callbacks.SoundEffectWasPlayed(SoundPushObject))
	assert.Equal(t, references.DeedNodeRuleFor(references.DeedResourceWood).Minutes-DefaultSmallMapMinutesPerTurn,
		gatherMinutes(callbacks), "the caller completes the ordinary one-minute turn")
	active, _ := state.ActiveDeed()
	assert.Equal(t, 1, active.NodeHarvests(tree.Position))
}

func TestGatherByBareHandsSometimesWinsOneWoodOrStone(t *testing.T) {
	for _, kind := range []references.DeedResourceKind{references.DeedResourceWood, references.DeedResourceStone} {
		state, callbacks, deed := newGatherFixture(t)
		material, _ := kind.Material()
		rule := references.DeedNodeRuleFor(kind)
		successes, failures := 0, 0
		for _, node := range nodesOf(state, deed, kind, 0) {
			standSouthOf(state, node)
			for range 6 {
				callbacks.Reset()
				before := state.PartyState.Inventory.Materials.Get(material)
				if !state.GatherDeedNode(references.Up) {
					break // Used up.
				}
				assert.Equal(t, rule.Minutes-DefaultSmallMapMinutesPerTurn, gatherMinutes(callbacks), "a failed try takes the time too")
				switch state.PartyState.Inventory.Materials.Get(material) - before {
				case 1:
					successes++
				case 0:
					failures++
					assert.Contains(t, callbacks.Messages, "Thy bare hands win no "+material.Name()+".")
				default:
					t.Fatalf("bare hands won more than one %s", material.Name())
				}
			}
		}
		assert.Positive(t, successes, kind.Name())
		assert.Positive(t, failures, kind.Name())
	}
}

func TestOreNeedsThePick(t *testing.T) {
	state, callbacks, deed := newGatherFixture(t)
	ore := nodesOf(state, deed, references.DeedResourceOre, 0)[0]
	standSouthOf(state, ore)

	assert.False(t, state.GatherDeedNode(references.Up))
	assert.Equal(t, []string{"Thou dost need a Pick!"}, callbacks.Messages)
	assert.Empty(t, callbacks.TimeAdvanced(), "a refusal is free")

	state.PartyState.Inventory.Tools.Set(references.Pick, 1)
	require.True(t, state.GatherDeedNode(references.Up))
	got := state.PartyState.Inventory.Materials.Get(references.Ore)
	assert.True(t, got >= 1 && got <= 2, "ore yield %d", got)
}

func TestForageAndReagentsAreGatheredByHand(t *testing.T) {
	state, _, deed := newGatherFixture(t)
	forage := nodesOf(state, deed, references.DeedResourceForage, 0)[0]
	standSouthOf(state, forage)
	require.True(t, state.GatherDeedNode(references.Up))
	assert.Positive(t, state.PartyState.Inventory.Materials.Get(references.Forage))

	patch := nodesOf(state, deed, references.DeedResourceReagent, 0)[0]
	standSouthOf(state, patch)
	require.True(t, state.GatherDeedNode(references.Up))
	got := state.PartyState.Inventory.Reagent.Get(patch.Reagent)
	assert.True(t, got >= 1 && got <= 2, "%s yield %d goes into the original reagents", patch.Reagent.Name(), got)
}

func TestReagentGatheringRespectsTheReagentCap(t *testing.T) {
	state, callbacks, deed := newGatherFixture(t)
	patch := nodesOf(state, deed, references.DeedResourceReagent, 0)[0]
	standSouthOf(state, patch)
	state.PartyState.Inventory.Reagent.Set(patch.Reagent, party_state.ItemQuantitySmallMax)

	assert.False(t, state.GatherDeedNode(references.Up))
	assert.Equal(t, []string{"Thou canst carry no more " + patch.Reagent.Name() + "!"}, callbacks.Messages)
	assert.Empty(t, callbacks.TimeAdvanced())
	active, _ := state.ActiveDeed()
	assert.Zero(t, active.NodeHarvests(patch.Position), "a refusal uses no capacity")

	state.PartyState.Inventory.Reagent.Set(patch.Reagent, party_state.ItemQuantitySmallMax-1)
	require.True(t, state.GatherDeedNode(references.Up))
	assert.Equal(t, party_state.ItemQuantitySmallMax, state.PartyState.Inventory.Reagent.Get(patch.Reagent), "a harvest saturates")
}

func TestANodeIsUsedUpAfterItsCapacityAndLookSaysSo(t *testing.T) {
	state, callbacks, deed := newGatherFixture(t)
	state.PartyState.Inventory.Tools.Set(references.Axe, 1)
	tree := nodesOf(state, deed, references.DeedResourceWood, indexes.Forest3)[0]
	standSouthOf(state, tree)
	require.True(t, state.ActionLookSmallMap(references.Up))
	assert.Contains(t, callbacks.Messages, "Look-Thou dost see a stand of timber")

	for range references.DeedNodeRuleFor(references.DeedResourceWood).Capacity {
		require.True(t, state.GatherDeedNode(references.Up))
	}

	assert.True(t, callbacks.MessagesContain("It is used up!"))
	tile, _ := state.DeedTileAt(deed.ID, tree.Position)
	assert.Equal(t, indexes.SpriteIndex(indexes.Forest1), tile, "a felled tree leaves brush")
	assert.Equal(t, indexes.SpriteIndex(indexes.Forest1),
		state.GetLayeredMapByCurrentLocation().SpriteIndexByLayer(map_state.MapLayer, tree.Position))
	node := nodesOf(state, deed, references.DeedResourceWood, indexes.Forest3)[0]
	assert.True(t, node.Depleted)
	assert.False(t, node.Covered)

	callbacks.Reset()
	assert.False(t, state.GatherDeedNode(references.Up))
	assert.Equal(t, []string{"Nothing left to gather!"}, callbacks.Messages)
	assert.Empty(t, callbacks.TimeAdvanced())
	require.True(t, state.ActionLookSmallMap(references.Up))
	assert.Contains(t, callbacks.Messages, "Look-Thou dost see a felled tree (depleted)")
}

func TestLookNamesEveryNodeKind(t *testing.T) {
	state, callbacks, deed := newGatherFixture(t)
	ore := nodesOf(state, deed, references.DeedResourceOre, 0)[0]
	patch := nodesOf(state, deed, references.DeedResourceReagent, 0)[0]
	for _, testCase := range []struct {
		node DeedResourceNodeState
		want string
	}{
		{ore, "an ore vein"},
		{patch, "a patch of " + patch.Reagent.Name()},
		{nodesOf(state, deed, references.DeedResourceForage, 0)[0], "a fruit tree"},
		{nodesOf(state, deed, references.DeedResourceStone, 0)[0], "a pile of rocks"},
		{nodesOf(state, deed, references.DeedResourceWood, indexes.DeadTree)[0], "a dry stump"},
	} {
		callbacks.Reset()
		standSouthOf(state, testCase.node)
		require.True(t, state.ActionLookSmallMap(references.Up))
		assert.Equal(t, []string{"Look-Thou dost see " + testCase.want}, callbacks.Messages)
	}
}

func TestGatherRefusesOccupiedAndPlainSquares(t *testing.T) {
	state, callbacks, deed := newGatherFixture(t)
	forage := nodesOf(state, deed, references.DeedResourceForage, 0)[0]
	standSouthOf(state, forage)
	require.NoError(t, state.ItemStacksMap.AddOrMerge(&forage.Position, references.ItemAndQuantity{Item: references.Food, Quantity: 1}))

	assert.False(t, state.GatherDeedNode(references.Up), "an item stack covers the node")
	state.MapState.PlayerLocation.Position = references.DeedMapCentre
	assert.False(t, state.GatherDeedNode(references.Up), "the yard is no node")

	assert.Equal(t, []string{"Nothing to gather!", "Nothing to gather!"}, callbacks.Messages)
	assert.Empty(t, callbacks.TimeAdvanced())
	assert.Zero(t, state.PartyState.Inventory.Materials.Get(references.Forage))
}

func TestGatherRefusesANodeCoveredBySomethingElse(t *testing.T) {
	state, callbacks, deed := newGatherFixture(t)
	forage := nodesOf(state, deed, references.DeedResourceForage, 0)[0]
	standSouthOf(state, forage)
	state.SetDeedTile(deed.ID, forage.Position, indexes.BrickFloor)

	assert.False(t, state.GatherDeedNode(references.Up))
	assert.Equal(t, []string{"Nothing to gather!"}, callbacks.Messages)
}

// depleteNode uses a node up as gathering would.
func depleteNode(state *GameState, deedID int, node DeedResourceNodeState) {
	deed := state.deedByID(deedID)
	if deed.harvests == nil {
		deed.harvests = map[references.Position]int{}
	}
	deed.harvests[node.Position] = node.Capacity
	state.SetDeedTile(deedID, node.Position, references.DeedDepletedSprite(deed.Terrain, node.DeedResourceNode))
}

func depletedCount(state *GameState, deedID int) int {
	count := 0
	for _, node := range state.DeedResourceNodes(deedID) {
		if node.Depleted {
			count++
		}
	}

	return count
}

// setPhaseChangeEve sets the date to 23:50 on a day whose next day changes Trammel's phase
// (change true) or keeps it (change false).
func setPhaseChangeEve(state *GameState, change bool) {
	state.DateTime = datetime.UltimaDate{Year: 139, Month: 5, Day: 1, Hour: 23, Minute: 50}
	for {
		today := datetime.TrammelPhaseChanges(state.DateTime.GetTotalDays())
		if (datetime.TrammelPhaseChanges(state.DateTime.GetTotalDays()+1) > today) == change {
			return
		}
		state.DateTime.Day++
	}
}

func TestOnePhaseChangeRestoresAQuarterOfTheUsedUpNodesRoundedUp(t *testing.T) {
	for _, testCase := range []struct{ depleted, restored int }{{4, 1}, {5, 2}, {1, 1}, {8, 2}} {
		state, _ := newDeedFixture(t, true)
		setPhaseChangeEve(state, true)
		deed := stakeFixtureDeed(t, state, "Home")
		nodes := state.DeedResourceNodes(deed.ID)
		for _, node := range nodes[:testCase.depleted] {
			depleteNode(state, deed.ID, node)
		}

		state.AdvanceGameplayMinutes(20)

		assert.Equal(t, testCase.depleted-testCase.restored, depletedCount(state, deed.ID), "%d used up", testCase.depleted)
		for _, node := range state.DeedResourceNodes(deed.ID)[:testCase.restored] {
			assert.False(t, node.Depleted, "row-major order restores the first nodes")
			assert.Equal(t, node.Sprite, node.Current, "the generated tile is back")
		}
	}
}

func TestADayWithoutAPhaseChangeRestoresNothing(t *testing.T) {
	state, _ := newDeedFixture(t, true)
	setPhaseChangeEve(state, false)
	deed := stakeFixtureDeed(t, state, "Home")
	for _, node := range state.DeedResourceNodes(deed.ID)[:4] {
		depleteNode(state, deed.ID, node)
	}

	state.AdvanceGameplayMinutes(20)

	assert.Equal(t, 4, depletedCount(state, deed.ID))
}

func TestManyPhaseChangesCatchUpOnAnUnloadedDeed(t *testing.T) {
	state, _ := newDeedFixture(t, true)
	setPhaseChangeEve(state, true)
	deed := stakeFixtureDeed(t, state, "Home")
	nodes := state.DeedResourceNodes(deed.ID)
	for _, node := range nodes[:8] {
		depleteNode(state, deed.ID, node)
	}
	_, loaded := state.ActiveDeed()
	require.False(t, loaded, "the deed map is not loaded")

	// Time jumps (a debug skip) to the eve of the day that brings the second phase change since
	// staking; nothing is processed until midnight passes, then both changes apply: 8 -> 6 -> 4.
	eve := state.DateTime
	for datetime.TrammelPhaseChanges(eve.GetTotalDays()+1) < deed.TrammelPhaseChanges()+2 {
		eve.Day++
	}
	state.DateTime = eve
	state.AdvanceGameplayMinutes(20)

	assert.Equal(t, 4, depletedCount(state, deed.ID), "8 -> 6 -> 4 over two changes")
	restored := state.deedByID(deed.ID)
	assert.Equal(t, datetime.TrammelPhaseChanges(state.DateTime.GetTotalDays()), restored.TrammelPhaseChanges())
	assert.Len(t, restored.TileOverridePositions(), 4, "restored tiles drop their overrides")

	// A month away restores everything, in one step at the next day.
	state.DateTime.Month++
	state.DateTime.Hour, state.DateTime.Minute = 23, 50
	state.AdvanceGameplayMinutes(20)
	assert.Zero(t, depletedCount(state, deed.ID))
	assert.Empty(t, restored.TileOverridePositions())
}

func TestPartlyGatheredNodesRefillAtAPhaseChange(t *testing.T) {
	state, _ := newDeedFixture(t, true)
	setPhaseChangeEve(state, true)
	deed := stakeFixtureDeed(t, state, "Home")
	node := state.DeedResourceNodes(deed.ID)[0]
	state.deedByID(deed.ID).harvests = map[references.Position]int{node.Position: 1}

	state.AdvanceGameplayMinutes(20)

	assert.Zero(t, state.deedByID(deed.ID).NodeHarvests(node.Position))
}

func TestRespawnSkipsANodeUnderTheParty(t *testing.T) {
	state, _, deed := newGatherFixture(t)
	nodes := state.DeedResourceNodes(deed.ID)
	depleteNode(state, deed.ID, nodes[0])
	depleteNode(state, deed.ID, nodes[1])
	state.MapState.PlayerLocation.Position = nodes[0].Position
	setPhaseChangeEve(state, true)
	state.deedByID(deed.ID).trammelPhaseChanges = datetime.TrammelPhaseChanges(state.DateTime.GetTotalDays())

	state.AdvanceGameplayMinutes(20)

	after := state.DeedResourceNodes(deed.ID)
	assert.True(t, after[0].Depleted, "the party stands there")
	assert.False(t, after[1].Depleted, "the next node in order is restored instead")
	assert.Equal(t, nodes[1].Sprite, state.GetLayeredMapByCurrentLocation().SpriteIndexByLayer(map_state.MapLayer, nodes[1].Position))
}

func TestANewDeedStartsAtTodaysPhaseCountAndTimeGoingBackOnlyResetsTheMarker(t *testing.T) {
	state, _ := newDeedFixture(t, true)
	setPhaseChangeEve(state, true)
	deed := stakeFixtureDeed(t, state, "Home")
	assert.Equal(t, datetime.TrammelPhaseChanges(state.DateTime.GetTotalDays()), deed.TrammelPhaseChanges())
	depleteNode(state, deed.ID, state.DeedResourceNodes(deed.ID)[0])

	state.DateTime.Year--
	state.processDeedRespawn()

	assert.Equal(t, 1, depletedCount(state, deed.ID))
	assert.Equal(t, datetime.TrammelPhaseChanges(state.DateTime.GetTotalDays()), state.deedByID(deed.ID).TrammelPhaseChanges())
}

func TestHarvestsAndTheRespawnMarkerSurviveASave(t *testing.T) {
	state, _, deed := newGatherFixture(t)
	nodes := state.DeedResourceNodes(deed.ID)
	depleteNode(state, deed.ID, nodes[0])
	state.deedByID(deed.ID).harvests[nodes[1].Position] = 1
	state.deedByID(deed.ID).trammelPhaseChanges = 1234

	restored := saveAndReload(t, state)

	active, ok := restored.ActiveDeed()
	require.True(t, ok)
	assert.Equal(t, nodes[0].Capacity, active.NodeHarvests(nodes[0].Position))
	assert.Equal(t, 1, active.NodeHarvests(nodes[1].Position))
	assert.Equal(t, 1234, active.TrammelPhaseChanges())
	after := restored.DeedResourceNodes(deed.ID)
	assert.True(t, after[0].Depleted)
	assert.False(t, after[0].Covered)
	assert.Equal(t, references.DeedDepletedSprite(deed.Terrain, nodes[0].DeedResourceNode),
		restored.GetLayeredMapByCurrentLocation().SpriteIndexByLayer(map_state.MapLayer, nodes[0].Position))
}
