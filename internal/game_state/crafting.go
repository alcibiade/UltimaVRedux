package game_state

import (
	"errors"
	"fmt"
	"slices"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// REDUX DEVIATION: CRAFT-0001: the original has no gathering, crafting, building or land
// ownership. Redux adds them behind Gameplay.Crafting.Enabled; see ref:CRAFT-0001 and ADR-014.

// Crafting rejection reasons.
var (
	ErrCraftingDisabled  = errors.New("crafting is disabled")
	ErrDeedNotOnSurface  = errors.New("deeds are staked only on the overworld surface")
	ErrDeedNotOnFoot     = errors.New("deeds are staked only on foot")
	ErrDeedNotBuildable  = errors.New("the ground cannot be deeded")
	ErrDeedSettlement    = errors.New("a settlement or landmark stands here")
	ErrDeedMoongate      = errors.New("a moongate square cannot be deeded")
	ErrDeedAlreadyOwned  = errors.New("the square is already deeded")
	ErrDeedTooClose      = errors.New("the square touches another deed")
	ErrDeedNotEnoughGold = errors.New("not enough gold for a deed")
	ErrInvalidDeedName   = errors.New("invalid deed name")
)

// Deed is one parcel of land the party owns: a hut square on the overworld and the generated
// deed map behind it (ref:CRAFT-0002, ref:CRAFT-0003). Deeds are permanent.
//
// Later steps add per-deed state (placements, chests) as fields here and in
// save_snapshot.DeedSnapshot.
type Deed struct {
	ID               int
	Name             string
	Square           references.Position
	Terrain          references.DeedTerrain
	GeneratorVersion int
	Seed             uint32
	// overrides holds every deed-map square whose tile differs from the generated one.
	overrides map[references.Position]indexes.SpriteIndex
	// harvests holds, for every resource node that has been gathered since it was last fresh,
	// the number of successful harvests taken (ref:CRAFT-0005). A node absent from the map is
	// fresh; one whose count reaches its kind's capacity is used up.
	harvests map[references.Position]int
	// trammelPhaseChanges is the deed's respawn marker: the Trammel phase-change count
	// (datetime.TrammelPhaseChanges) its nodes have been brought up to (ref:CRAFT-0006).
	trammelPhaseChanges int
}

// TrammelPhaseChanges returns the deed's respawn marker (ref:CRAFT-0006).
func (d *Deed) TrammelPhaseChanges() int {
	return d.trammelPhaseChanges
}

// NodeHarvests returns the harvests taken from the node at a deed-map square since it was last
// fresh; zero for a fresh node or a square that is no node.
func (d *Deed) NodeHarvests(position references.Position) int {
	return d.harvests[position]
}

// HarvestedNodePositions returns every node with harvests taken, in row-major order.
func (d *Deed) HarvestedNodePositions() []references.Position {
	positions := make([]references.Position, 0, len(d.harvests))
	for position := range d.harvests {
		positions = append(positions, position)
	}
	slices.SortFunc(positions, compareRowMajor)

	return positions
}

// Layout regenerates the deed map as first generated, without overrides.
func (d *Deed) Layout() references.DeedLayout {
	layout, err := references.GenerateDeedMap(d.Terrain, d.Seed, d.GeneratorVersion)
	if err != nil {
		panic(fmt.Errorf("validated deed %d cannot regenerate: %w", d.ID, err))
	}

	return layout
}

// TileOverride returns the override at a deed-map square, if any.
func (d *Deed) TileOverride(position references.Position) (indexes.SpriteIndex, bool) {
	sprite, ok := d.overrides[position]

	return sprite, ok
}

// TileOverridePositions returns every overridden square in row-major order.
func (d *Deed) TileOverridePositions() []references.Position {
	positions := make([]references.Position, 0, len(d.overrides))
	for position := range d.overrides {
		positions = append(positions, position)
	}
	slices.SortFunc(positions, compareRowMajor)

	return positions
}

func compareRowMajor(a, b references.Position) int {
	if a.Y != b.Y {
		return int(a.Y) - int(b.Y)
	}

	return int(a.X) - int(b.X)
}

// DeedResourceNodeState is one generated resource node of a deed with its current square.
// Harvests counts the harvests taken since the node was last fresh, out of Capacity; Depleted
// means it is used up. Covered is true when the square shows neither the node's sprite nor,
// for a used-up node, its depleted sprite: something else, such as a built piece, stands there.
type DeedResourceNodeState struct {
	references.DeedResourceNode
	Current  indexes.SpriteIndex
	Harvests int
	Capacity int
	Depleted bool
	Covered  bool
}

// craftingState is the persistent crafting state of one game. It lives on GameState, never in
// the shared references, so deeds of an earlier game can never leak into the next one.
type craftingState struct {
	deeds []*Deed
	// active is the deed whose map is loaded under references.DeedLocation, if any.
	active       *Deed
	activeMap    *references.SmallLocationReference
	activeLayout *references.DeedLayout
}

// IsCraftingEnabled reports whether the optional crafting rules are on.
func (g *GameState) IsCraftingEnabled() bool {
	return g.GameConfig != nil && g.GameConfig.GameplayConfiguration != nil &&
		g.GameConfig.GameplayConfiguration.CraftingEnabled
}

// isOnOverworldSurface reports whether the party stands on the overworld (floor 0).
func (g *GameState) isOnOverworldSurface() bool {
	return g.MapState.PlayerLocation.Location == references.BritanniaUnderworld &&
		g.MapState.PlayerLocation.Floor == references.FloorNumber(references.OVERWORLD)
}

func (g *GameState) isPartyOnFoot() bool {
	return g.PartyVehicle.GetVehicleDetails().VehicleType == references.NoPartyVehicle
}

// IsOnDeedMap reports whether the party stands on one of its loaded deed maps.
func (g *GameState) IsOnDeedMap() bool {
	return g.MapState.PlayerLocation.Location.IsDeedLocation() && g.crafting.active != nil
}

func (g *GameState) overworldMap() *map_state.LayeredMap {
	return g.MapState.LayeredMaps.GetLayeredMap(references.LargeMapType, references.FloorNumber(references.OVERWORLD))
}

// Deeds returns copies of the party's deeds in creation order.
func (g *GameState) Deeds() []Deed {
	deeds := make([]Deed, 0, len(g.crafting.deeds))
	for _, deed := range g.crafting.deeds {
		deeds = append(deeds, *deed)
	}

	return deeds
}

// DeedAt returns the deed whose hut stands on the overworld square, if any.
func (g *GameState) DeedAt(square references.Position) (Deed, bool) {
	if deed := g.deedAt(square); deed != nil {
		return *deed, true
	}

	return Deed{}, false
}

func (g *GameState) deedAt(square references.Position) *Deed {
	for _, deed := range g.crafting.deeds {
		if deed.Square == square {
			return deed
		}
	}

	return nil
}

func (g *GameState) deedByID(id int) *Deed {
	return g.crafting.deedByID(id)
}

func (c *craftingState) deedByID(id int) *Deed {
	for _, deed := range c.deeds {
		if deed.ID == id {
			return deed
		}
	}

	return nil
}

// ActiveDeed returns the deed whose map the party is on, if any.
func (g *GameState) ActiveDeed() (Deed, bool) {
	if g.crafting.active == nil {
		return Deed{}, false
	}

	return *g.crafting.active, true
}

// locationReference resolves a location to its source map. The shared deed location resolves
// to the active deed's generated map; every other location comes from the original data.
func (g *GameState) locationReference(location references.Location) *references.SmallLocationReference {
	if location.IsDeedLocation() {
		return g.crafting.activeMap
	}

	return g.GameReferences.LocationReferences.GetLocationReference(location)
}

// DeedTileAt returns a deed-map square's current tile: its override, or the generated tile.
func (g *GameState) DeedTileAt(deedID int, position references.Position) (indexes.SpriteIndex, bool) {
	deed := g.deedByID(deedID)
	if deed == nil || !isDeedMapSquare(position) {
		return 0, false
	}
	if sprite, ok := deed.overrides[position]; ok {
		return sprite, true
	}
	if deed == g.crafting.active {
		return g.crafting.activeLayout.Tile(position), true
	}
	layout := deed.Layout()

	return layout.Tile(position), true
}

// DeedResourceNodes lists every generated resource node of a deed, in row-major order, with
// the square's current tile. Step 2 (gathering) reads node positions and kinds from here and
// changes node squares through SetDeedTile.
func (g *GameState) DeedResourceNodes(deedID int) []DeedResourceNodeState {
	deed := g.deedByID(deedID)
	if deed == nil {
		return nil
	}
	layout := g.deedLayout(deed)
	nodes := make([]DeedResourceNodeState, 0, len(layout.Nodes))
	for _, node := range layout.Nodes {
		nodes = append(nodes, deed.nodeState(layout, node))
	}

	return nodes
}

// deedLayout returns a deed's generated layout, shared with the loaded map when it is active.
func (g *GameState) deedLayout(deed *Deed) *references.DeedLayout {
	if deed == g.crafting.active {
		return g.crafting.activeLayout
	}
	layout := deed.Layout()

	return &layout
}

func (d *Deed) nodeState(layout *references.DeedLayout, node references.DeedResourceNode) DeedResourceNodeState {
	current := node.Sprite
	if sprite, ok := d.overrides[node.Position]; ok {
		current = sprite
	}
	state := DeedResourceNodeState{
		DeedResourceNode: node, Current: current, Harvests: d.harvests[node.Position],
		Capacity: references.DeedNodeRuleFor(node.Kind).Capacity,
	}
	state.Depleted = state.Harvests >= state.Capacity
	switch {
	case current == node.Sprite:
	case state.Depleted && current == references.DeedDepletedSprite(layout.Terrain, node):
	default:
		state.Covered = true
	}

	return state
}

// SetDeedTile changes one square of a deed map. The change is kept as an override (or dropped
// when it restores the generated tile) and, when the deed map is loaded, written to its source
// map and the live layered map, so it survives map rebuilds, combat returns and saves.
func (g *GameState) SetDeedTile(deedID int, position references.Position, sprite indexes.SpriteIndex) {
	deed := g.deedByID(deedID)
	if deed == nil || !isDeedMapSquare(position) {
		panic(fmt.Sprintf("SetDeedTile: deed %d square %v does not exist", deedID, position))
	}
	generated := g.deedLayout(deed).Tile(position)
	if sprite == generated {
		delete(deed.overrides, position)
	} else {
		if deed.overrides == nil {
			deed.overrides = make(map[references.Position]indexes.SpriteIndex)
		}
		deed.overrides[position] = sprite
	}
	if deed != g.crafting.active {
		return
	}
	g.crafting.activeMap.SetTileNumber(position, 0, sprite)
	if live := g.MapState.LayeredMaps.GetLayeredMap(references.SmallMapType, 0); live != nil &&
		g.MapState.PlayerLocation.Location.IsDeedLocation() {
		live.SetTileByLayer(map_state.MapLayer, &position, sprite)
		live.UnSetTileByLayer(map_state.MapOverrideLayer, &position)
	}
}

func isDeedMapSquare(position references.Position) bool {
	return position.X >= 0 && position.Y >= 0 &&
		position.X < references.XSmallMapTiles && position.Y < references.YSmallMapTiles
}

// activateDeed loads a deed's map under the shared deed location: the generated layout with
// every override applied. Any previously active deed map is released first.
func (g *GameState) activateDeed(deed *Deed) *references.SmallLocationReference {
	return g.crafting.activate(deed)
}

func (c *craftingState) activate(deed *Deed) *references.SmallLocationReference {
	layout := deed.Layout()
	tiles := layout.Tiles
	for position, sprite := range deed.overrides {
		tiles[position.X][position.Y] = sprite
	}
	c.active = deed
	c.activeLayout = &layout
	c.activeMap = references.NewGeneratedSmallLocationReference(
		references.DeedLocation, deed.Name, &tiles, references.DeedProfile(deed.Terrain).Outer)

	return c.activeMap
}

// releaseDeedMap forgets the loaded deed map once the party has left it for good.
func (g *GameState) releaseDeedMap() {
	g.crafting.active = nil
	g.crafting.activeMap = nil
	g.crafting.activeLayout = nil
}

// releaseDeedMapAfterTransition drops the deed map when a full transition reaches any physical
// map other than the deed map. Combat and camping suspend the deed map and keep it.
func (g *GameState) releaseDeedMapAfterTransition(destination references.Location) {
	if g.crafting.active == nil || destination.IsDeedLocation() {
		return
	}
	switch destination.GetMapType() {
	case references.LargeMapType, references.SmallMapType, references.DungeonMapType:
		g.releaseDeedMap()
	case references.CombatMapType, references.CampingMapType:
		// Virtual scenes return to the suspended deed map.
	}
}

// deedExit returns the overworld square a deed map leads back to.
func (g *GameState) deedExit() (references.Position, references.FloorNumber) {
	if g.crafting.active == nil {
		panic("leaving a deed map with no active deed")
	}

	return g.crafting.active.Square, references.FloorNumber(references.OVERWORLD)
}

// locationReferenceFrom resolves a location against an explicit reference table, with the same
// deed-location rule as locationReference.
func (g *GameState) locationReferenceFrom(
	locations *references.LocationReferences, location references.Location,
) *references.SmallLocationReference {
	if location.IsDeedLocation() {
		return g.crafting.activeMap
	}

	return locations.GetLocationReference(location)
}

// CraftingCommand is one of the three optional crafting chords (ref:CRAFT-0001).
type CraftingCommand int

// Crafting commands.
const (
	CraftingGather CraftingCommand = iota
	CraftingCraft
	CraftingBuild
)

// CraftingCommandApplies reports whether a crafting command can act where the party stands,
// with the option on. Gather works only on a deed map; Craft and Build work on the overworld
// surface and on deed maps. All three need the party on foot.
func (g *GameState) CraftingCommandApplies(command CraftingCommand) bool {
	if !g.IsCraftingEnabled() || !g.isPartyOnFoot() {
		return false
	}
	switch command {
	case CraftingGather:
		return g.IsOnDeedMap()
	case CraftingCraft, CraftingBuild:
		return g.IsOnDeedMap() || g.isOnOverworldSurface()
	default:
		return false
	}
}
