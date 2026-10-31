package references

import (
	"strconv"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// REDUX DEVIATION: CRAFT-0001: the original has no gathering, crafting, building or land
// ownership. Redux adds an optional system behind Gameplay.Crafting.Enabled. Its tunables live
// here (materials, tools, deed price, generator and gathering parameters), in
// crafting_recipes.go (stations and recipes) and in crafting_pieces.go (build pieces) (ADR-014).

// Material is a crafting material carried by the party.
type Material int

// Materials, in save-data order. Raw materials come first, refined ones after them. The
// order is a save identity: append new materials, never reorder.
const (
	Wood     Material = 0
	Stone    Material = 1
	Ore      Material = 2
	Forage   Material = 3
	Planks   Material = 4
	CutStone Material = 5
	Ingots   Material = 6
)

// NumberOfMaterials is the number of material identities.
const NumberOfMaterials = 7

// Name returns the player-facing material name.
func (m Material) Name() string {
	switch m {
	case Wood:
		return "Wood"
	case Stone:
		return "Stone"
	case Ore:
		return "Ore"
	case Forage:
		return "Forage"
	case Planks:
		return "Planks"
	case CutStone:
		return "Cut stone"
	case Ingots:
		return "Ingots"
	default:
		return "Unknown"
	}
}

// IsRefined reports whether the material is made at a station rather than gathered.
func (m Material) IsRefined() bool {
	return m == Planks || m == CutStone || m == Ingots
}

// Description is the Ztats explanation of the material.
func (m Material) Description() string {
	switch m {
	case Wood:
		return "Timber from trees and stumps on your own land."
	case Stone:
		return "Stone from rock piles on your own land."
	case Ore:
		return "Metal ore broken from rocky outcrops on your own land."
	case Forage:
		return "Fruit and wild food gathered on your own land."
	case Planks:
		return "Sawn wood, made at a sawmill."
	case CutStone:
		return "Dressed stone, made at a workbench."
	case Ingots:
		return "Smelted metal, made at a forge."
	default:
		return ""
	}
}

// MaterialCatalog returns every material identity in save-data order.
func MaterialCatalog() []Material {
	return []Material{Wood, Stone, Ore, Forage, Planks, CutStone, Ingots}
}

// Tool is a crafted implement used for gathering.
type Tool int

// Tools, in save-data order. The order is a save identity: append, never reorder.
const (
	Axe  Tool = 0
	Pick Tool = 1
)

// NumberOfTools is the number of tool identities.
const NumberOfTools = 2

// Name returns the player-facing tool name.
func (t Tool) Name() string {
	switch t {
	case Axe:
		return "Axe"
	case Pick:
		return "Pick"
	default:
		return "Unknown"
	}
}

// Description is the Ztats explanation of the tool.
func (t Tool) Description() string {
	switch t {
	case Axe:
		return "Fells trees on your own land for full wood yields."
	case Pick:
		return "Breaks rocks and outcrops on your own land for stone and ore."
	default:
		return ""
	}
}

// ToolCatalog returns every tool identity in save-data order.
func ToolCatalog() []Tool {
	return []Tool{Axe, Pick}
}

// MaterialCost is an amount of each material, indexed by Material.
type MaterialCost [NumberOfMaterials]uint16

// Describe renders a cost such as "3 Wood, 2 Stone".
func (c MaterialCost) Describe() string {
	parts := make([]string, 0, NumberOfMaterials)
	for _, material := range MaterialCatalog() {
		if c[material] > 0 {
			parts = append(parts, strconv.Itoa(int(c[material]))+" "+material.Name())
		}
	}
	if len(parts) == 0 {
		return "nothing"
	}

	return strings.Join(parts, ", ")
}

// Deed rules (ref:CRAFT-0002).
const (
	// FirstDeedGoldCost is the price of the party's first deed, so a new party can afford one.
	FirstDeedGoldCost = 100
	// DeedGoldCost is the flat price of every further deed. There is no material cost and no
	// deed limit.
	DeedGoldCost = 200
	// MaxDeedNameLength bounds a deed's name; it fits the Atlas label rule.
	MaxDeedNameLength = 24
)

// IsValidDeedName reports whether a deed name is 1 to 24 printable game-font characters
// (' ' through '~') with no surrounding spaces. The naming prompt, the save validator and
// the Atlas label rule all accept exactly these names.
func IsValidDeedName(name string) bool {
	if name == "" || len(name) > MaxDeedNameLength || strings.TrimSpace(name) != name {
		return false
	}
	for index := range len(name) {
		if name[index] < ' ' || name[index] > '~' {
			return false
		}
	}

	return true
}

// DeedTerrain is the kind of land a deed was staked on. It selects the generated scenery
// and the resource mix of the deed map. Its Key is a save identity.
type DeedTerrain int

// Deed terrains.
const (
	DeedTerrainGrassland DeedTerrain = iota
	DeedTerrainForest
	DeedTerrainHills
	DeedTerrainDesert
)

// DeedTerrains returns every terrain in a stable order.
func DeedTerrains() []DeedTerrain {
	return []DeedTerrain{DeedTerrainGrassland, DeedTerrainForest, DeedTerrainHills, DeedTerrainDesert}
}

// Key returns the stable save identity of the terrain.
func (t DeedTerrain) Key() string {
	switch t {
	case DeedTerrainGrassland:
		return "grassland"
	case DeedTerrainForest:
		return "forest"
	case DeedTerrainHills:
		return "hills"
	case DeedTerrainDesert:
		return "desert"
	default:
		return ""
	}
}

// Name returns the player-facing terrain name.
func (t DeedTerrain) Name() string {
	switch t {
	case DeedTerrainGrassland:
		return "grassland"
	case DeedTerrainForest:
		return "woodland"
	case DeedTerrainHills:
		return "foothills"
	case DeedTerrainDesert:
		return "desert"
	default:
		return "land"
	}
}

// DeedTerrainFromKey parses a save identity.
func DeedTerrainFromKey(key string) (DeedTerrain, bool) {
	for _, terrain := range DeedTerrains() {
		if terrain.Key() == key {
			return terrain, true
		}
	}

	return 0, false
}

// DeedTerrainForSprite classifies an overworld terrain tile. Only natural, open ground can be
// deeded: water, swamp, mountains, roads, bridges, settlements and every structure cannot.
func DeedTerrainForSprite(sprite indexes.SpriteIndex) (DeedTerrain, bool) {
	//nolint:exhaustive // Every other tile is not buildable ground.
	switch sprite {
	case indexes.Grass, indexes.Forest1, indexes.PlowedField, indexes.WheatInField,
		indexes.OutsidePath1, indexes.OutsidePath2, indexes.OutsidePath3, indexes.OutsidePath4:
		return DeedTerrainGrassland, true
	case indexes.Forest2, indexes.Forest3, indexes.Forest4:
		return DeedTerrainForest, true
	case indexes.Hills1, indexes.LeftHills2, indexes.RightHills2:
		return DeedTerrainHills, true
	case indexes.Desert1, indexes.LeftDesert2, indexes.RightDesert2, indexes.Desert3:
		return DeedTerrainDesert, true
	default:
		return 0, false
	}
}

// DeedResourceKind is what a resource node on a deed map yields.
type DeedResourceKind int

// Resource node kinds.
const (
	DeedResourceWood DeedResourceKind = iota
	DeedResourceStone
	DeedResourceOre
	DeedResourceForage
	DeedResourceReagent
)

// Name returns a player-facing name for the node kind.
func (k DeedResourceKind) Name() string {
	switch k {
	case DeedResourceWood:
		return "timber"
	case DeedResourceStone:
		return "rocks"
	case DeedResourceOre:
		return "ore outcrop"
	case DeedResourceForage:
		return "forage"
	case DeedResourceReagent:
		return "reagent patch"
	default:
		return "resource"
	}
}

// Material returns the material a node yields; reagent patches yield a reagent instead.
func (k DeedResourceKind) Material() (Material, bool) {
	switch k {
	case DeedResourceWood:
		return Wood, true
	case DeedResourceStone:
		return Stone, true
	case DeedResourceOre:
		return Ore, true
	case DeedResourceForage:
		return Forage, true
	case DeedResourceReagent:
		return 0, false
	default:
		return 0, false
	}
}

// DeedNodeQuota asks the generator for Count nodes of one kind and sprite. Reagents lists the
// reagents a patch may hold; the generator picks one per patch.
type DeedNodeQuota struct {
	Kind     DeedResourceKind
	Sprite   indexes.SpriteIndex
	Count    int
	Reagents []Reagent
}

// DeedWeightedSprite is a sprite with a relative weight for random picks.
type DeedWeightedSprite struct {
	Sprite indexes.SpriteIndex
	Weight int
}

// DeedTerrainProfile is the generator recipe for one terrain.
type DeedTerrainProfile struct {
	Terrain DeedTerrain
	// Ground is the dominant open tile of the interior; Accents are scattered over it.
	Ground  indexes.SpriteIndex
	Accents []DeedWeightedSprite
	// AccentPercent is the share of interior squares that get an accent.
	AccentPercent int
	// Border is picked for the decorative ring; Fringe for its ragged inner edge.
	Border []DeedWeightedSprite
	Fringe []DeedWeightedSprite
	// Path is the entry lane from the south edge.
	Path indexes.SpriteIndex
	// Outer is drawn beyond the map edge.
	Outer indexes.SpriteIndex
	Nodes []DeedNodeQuota
}

// Deed map generator parameters (ref:CRAFT-0003). Changing the output of the generator for an
// existing seed requires a new DeedGeneratorVersion; old versions must keep their layout.
const (
	// DeedGeneratorVersion is the version stamped on new deeds.
	DeedGeneratorVersion = 1
	// DeedBorderDepth is the depth of the solid decorative ring at every edge.
	DeedBorderDepth = 3
	// DeedFringePercent is the chance that a square just inside the ring is also border.
	DeedFringePercent = 40
	// DeedNodeMargin keeps nodes this many squares away from the ring, so every node can
	// be reached and the ring never closes off part of the interior.
	DeedNodeMargin = 2
	// DeedYardRadius keeps a clear yard around the centre, free of nodes and accents.
	DeedYardRadius = 4
)

// DeedMapEntry is where the party arrives on a deed map: the bottom-centre square used by
// every original small map (action_enter.go), on the entry path.
//
//nolint:gochecknoglobals // Read-only tunable.
var DeedMapEntry = Position{X: 15, Y: 30}

// DeedMapCentre is the centre of the clear yard.
//
//nolint:gochecknoglobals // Read-only tunable.
var DeedMapCentre = Position{X: 15, Y: 15}

// Wild reagent patches hold only common reagents; the rare ones come from refining (ADR-014).
//
//nolint:gochecknoglobals // Read-only tunable.
var (
	meadowReagents = []Reagent{Ginseng, Garlic}
	forestReagents = []Reagent{SpiderSilk, BloodMoss, Ginseng}
	hillsReagents  = []Reagent{SulfurAsh, BloodMoss}
	desertReagents = []Reagent{SulfurAsh}
)

// DeedProfile returns the generator recipe for a terrain.
func DeedProfile(terrain DeedTerrain) DeedTerrainProfile {
	switch terrain {
	case DeedTerrainForest:
		return DeedTerrainProfile{
			Terrain: terrain, Ground: indexes.Grass, AccentPercent: 24,
			Accents: []DeedWeightedSprite{{indexes.Forest1, 2}, {indexes.Forest2, 3}},
			Border:  []DeedWeightedSprite{{indexes.Forest4, 3}, {indexes.Forest3, 2}},
			Fringe:  []DeedWeightedSprite{{indexes.Forest3, 2}, {indexes.Forest2, 2}},
			Path:    indexes.Path1, Outer: indexes.Forest4,
			Nodes: []DeedNodeQuota{
				{Kind: DeedResourceWood, Sprite: indexes.Forest3, Count: 14},
				{Kind: DeedResourceWood, Sprite: indexes.DeadTree, Count: 5},
				{Kind: DeedResourceStone, Sprite: indexes.SmallRockWall, Count: 3},
				{Kind: DeedResourceForage, Sprite: indexes.Tree, Count: 5},
				{Kind: DeedResourceReagent, Sprite: indexes.WheatInField, Count: 4, Reagents: forestReagents},
			},
		}
	case DeedTerrainHills:
		return DeedTerrainProfile{
			Terrain: terrain, Ground: indexes.Grass, AccentPercent: 12,
			Accents: []DeedWeightedSprite{{indexes.Hills1, 3}, {indexes.Forest1, 1}},
			Border:  []DeedWeightedSprite{{indexes.TallMountains, 2}, {indexes.SmallMountains, 3}},
			Fringe:  []DeedWeightedSprite{{indexes.Hills1, 3}, {indexes.LeftHills2, 1}, {indexes.RightHills2, 1}},
			Path:    indexes.Path1, Outer: indexes.Hills1,
			Nodes: []DeedNodeQuota{
				{Kind: DeedResourceWood, Sprite: indexes.Forest3, Count: 4},
				{Kind: DeedResourceStone, Sprite: indexes.SmallRockWall, Count: 9},
				{Kind: DeedResourceOre, Sprite: indexes.SmallMountains, Count: 7},
				{Kind: DeedResourceForage, Sprite: indexes.Tree, Count: 3},
				{Kind: DeedResourceReagent, Sprite: indexes.WheatInField, Count: 3, Reagents: hillsReagents},
			},
		}
	case DeedTerrainDesert:
		return DeedTerrainProfile{
			Terrain: terrain, Ground: indexes.Desert1, AccentPercent: 6,
			Accents: []DeedWeightedSprite{{indexes.LeftDesert2, 1}, {indexes.RightDesert2, 1}},
			Border:  []DeedWeightedSprite{{indexes.SmallMountains, 3}, {indexes.TallMountains, 1}},
			Fringe:  []DeedWeightedSprite{{indexes.Hills1, 2}, {indexes.SmallMountains, 1}},
			Path:    indexes.Path1, Outer: indexes.Desert1,
			Nodes: []DeedNodeQuota{
				{Kind: DeedResourceWood, Sprite: indexes.DeadTree, Count: 4},
				{Kind: DeedResourceStone, Sprite: indexes.SmallRockWall, Count: 7},
				{Kind: DeedResourceOre, Sprite: indexes.SmallMountains, Count: 4},
				{Kind: DeedResourceForage, Sprite: indexes.Cactus, Count: 6},
				{Kind: DeedResourceReagent, Sprite: indexes.WheatInField, Count: 2, Reagents: desertReagents},
			},
		}
	default:
		return DeedTerrainProfile{
			Terrain: DeedTerrainGrassland, Ground: indexes.Grass, AccentPercent: 8,
			Accents: []DeedWeightedSprite{{indexes.Forest1, 3}, {indexes.Forest2, 1}},
			Border:  []DeedWeightedSprite{{indexes.Forest3, 3}, {indexes.Forest4, 1}},
			Fringe:  []DeedWeightedSprite{{indexes.Forest2, 2}, {indexes.Forest1, 2}},
			Path:    indexes.Path1, Outer: indexes.Grass,
			Nodes: []DeedNodeQuota{
				{Kind: DeedResourceWood, Sprite: indexes.Forest3, Count: 7},
				{Kind: DeedResourceWood, Sprite: indexes.DeadTree, Count: 2},
				{Kind: DeedResourceStone, Sprite: indexes.SmallRockWall, Count: 5},
				{Kind: DeedResourceOre, Sprite: indexes.SmallMountains, Count: 1},
				{Kind: DeedResourceForage, Sprite: indexes.Tree, Count: 7},
				{Kind: DeedResourceReagent, Sprite: indexes.WheatInField, Count: 5, Reagents: meadowReagents},
			},
		}
	}
}

// IsDeedBuildableGround reports whether a deed-map tile is open ground that later steps may
// build on: grass, brush, foothills, desert, fields, paths and placed floors.
func IsDeedBuildableGround(sprite indexes.SpriteIndex) bool {
	//nolint:exhaustive // Every other tile is not buildable ground.
	switch sprite {
	case indexes.Grass, indexes.Forest1, indexes.Forest2, indexes.Desert1, indexes.LeftDesert2,
		indexes.RightDesert2, indexes.Hills1, indexes.LeftHills2, indexes.RightHills2,
		indexes.PlowedField, indexes.Path1, indexes.Path2, indexes.Path3, indexes.Path4, indexes.Path5,
		indexes.Path6, indexes.Path7, indexes.WoodFloor1, indexes.BrickFloor:
		return true
	default:
		return false
	}
}

// Gathering rules (ref:CRAFT-0005).

// DeedNodeRule is how one kind of resource node is gathered. A node yields until it has given
// Capacity successful harvests; it is then used up until a Trammel phase change restores it
// (ref:CRAFT-0006).
type DeedNodeRule struct {
	Kind DeedResourceKind
	// Capacity is the number of successful harvests a fresh node gives.
	Capacity int
	// NeedsTool means the kind is worked with Tool; otherwise it is gathered by hand.
	NeedsTool bool
	Tool      Tool
	// MinYield and MaxYield bound one harvest with the tool, or by hand for hand-gathered kinds.
	MinYield int
	MaxYield int
	// BareHandsOneIn, for a kind that needs a tool, is the chance (one in N) that an attempt
	// without the tool still yields BareHandsYield. Zero means the tool is required.
	BareHandsOneIn int
	BareHandsYield int
	// Minutes is the whole time of one attempt, the ordinary turn included.
	Minutes int
}

// DeedNodeRuleFor returns the gathering rule of a node kind.
func DeedNodeRuleFor(kind DeedResourceKind) DeedNodeRule {
	switch kind {
	case DeedResourceWood:
		return DeedNodeRule{Kind: kind, Capacity: 4, NeedsTool: true, Tool: Axe, MinYield: 2, MaxYield: 4,
			BareHandsOneIn: 3, BareHandsYield: 1, Minutes: 10}
	case DeedResourceStone:
		return DeedNodeRule{Kind: kind, Capacity: 4, NeedsTool: true, Tool: Pick, MinYield: 2, MaxYield: 3,
			BareHandsOneIn: 3, BareHandsYield: 1, Minutes: 10}
	case DeedResourceOre:
		return DeedNodeRule{Kind: kind, Capacity: 3, NeedsTool: true, Tool: Pick, MinYield: 1, MaxYield: 2, Minutes: 15}
	case DeedResourceForage:
		return DeedNodeRule{Kind: kind, Capacity: 3, MinYield: 1, MaxYield: 3, Minutes: 5}
	case DeedResourceReagent:
		return DeedNodeRule{Kind: kind, Capacity: 2, MinYield: 1, MaxYield: 2, Minutes: 10}
	default:
		panic("unknown deed resource kind")
	}
}

// DeedResourceKinds returns every node kind in a stable order.
func DeedResourceKinds() []DeedResourceKind {
	return []DeedResourceKind{DeedResourceWood, DeedResourceStone, DeedResourceOre, DeedResourceForage, DeedResourceReagent}
}

// DeedDepletedSprite is the tile a used-up node shows until it is restored: a felled tree
// leaves brush, a grubbed-out stump, quarried rocks and a picked cactus leave the terrain's
// ground, a worked-out ore vein leaves foothills, a picked fruit tree leaves brush and a picked
// reagent patch leaves a plowed patch.
func DeedDepletedSprite(terrain DeedTerrain, node DeedResourceNode) indexes.SpriteIndex {
	//nolint:exhaustive // Only generated node sprites deplete; anything else keeps the terrain's ground.
	switch node.Sprite {
	case indexes.Forest3:
		return indexes.Forest1
	case indexes.SmallMountains:
		return indexes.Hills1
	case indexes.Tree:
		return indexes.Forest2
	case indexes.WheatInField:
		return indexes.PlowedField
	default:
		return DeedProfile(terrain).Ground
	}
}

// DeedNodeLookName is how Look names a node, for example "an ore vein" or "a patch of Garlic".
// Spent names the node once used up; Look marks it "(depleted)".
func DeedNodeLookName(node DeedResourceNode) (fresh, spent string) {
	switch {
	case node.Kind == DeedResourceReagent:
		name := node.Reagent.Name()

		return "a patch of " + name, "a patch of " + name
	case node.Sprite == indexes.Forest3:
		return "a stand of timber", "a felled tree"
	case node.Sprite == indexes.DeadTree:
		return "a dry stump", "a grubbed-out stump"
	case node.Sprite == indexes.SmallRockWall:
		return "a pile of rocks", "a quarried rock pile"
	case node.Sprite == indexes.SmallMountains:
		return "an ore vein", "a worked-out ore vein"
	case node.Sprite == indexes.Tree:
		return "a fruit tree", "a fruit tree"
	case node.Sprite == indexes.Cactus:
		return "a fruiting cactus", "a cactus"
	default:
		return "something to gather", "something picked clean"
	}
}

// Respawn rules (ref:CRAFT-0006).
const (
	// DeedRespawnPercent is the share of used-up nodes on every deed that each Trammel phase
	// change restores, rounded up.
	DeedRespawnPercent = 25
)

// Raids (ref:CRAFT-0020, ref:CRAFT-0021). While the party is on its own deed map with the option
// on, every DeedRaidCheckTurns actor turns it rolls one in DeedRaidOdds for a raid, unless raiders
// are already on the map.
const (
	// DeedRaidCheckTurns is the number of deed-map actor turns between raid rolls.
	DeedRaidCheckTurns = 30
	// DeedRaidOdds is the one-in-N chance of a raid at each roll.
	DeedRaidOdds = 10
	// DeedRaidMinRaiders and DeedRaidMaxRaiders bound the size of a raid.
	DeedRaidMinRaiders = 1
	DeedRaidMaxRaiders = 3
	// DeedRaidSpawnDepth is the deepest a raider appears inside the map edge: the decorative
	// ring and its fringe, never the buildable interior.
	DeedRaidSpawnDepth = DeedBorderDepth
	// DeedRaidMinSpawnDistance is the smallest Manhattan distance between the party and a
	// square where a raider appears.
	DeedRaidMinSpawnDistance = 8
)

// DeedRaider is one monster that can raid a deed of a terrain, with its relative weight.
type DeedRaider struct {
	Monster MonsterType
	Weight  int
}

// DeedRaiders returns the weak-to-middling original monsters that raid a deed of a terrain.
func DeedRaiders(terrain DeedTerrain) []DeedRaider {
	switch terrain {
	case DeedTerrainGrassland:
		return []DeedRaider{{MonsterRat, 40}, {MonsterFighter, 30}, {MonsterOrc, 30}}
	case DeedTerrainForest:
		return []DeedRaider{{MonsterSpider, 35}, {MonsterOrc, 30}, {MonsterInsect, 20}, {MonsterEttin, 15}}
	case DeedTerrainHills:
		return []DeedRaider{{MonsterOrc, 45}, {MonsterHeadless, 30}, {MonsterTroll, 25}}
	case DeedTerrainDesert:
		return []DeedRaider{{MonsterSnake, 45}, {MonsterInsect, 35}, {MonsterOrc, 20}}
	default:
		return nil
	}
}

// DeedRaidCombatMap is the battlefield of a raid on a deed of a terrain, fought outdoors.
func DeedRaidCombatMap(terrain DeedTerrain) BritanniaCombatMap {
	switch terrain {
	case DeedTerrainForest:
		return TreedCombatMap
	case DeedTerrainHills:
		return MountainsCombatMap
	case DeedTerrainDesert:
		return DesertCombatMap
	case DeedTerrainGrassland:
		return GladeCombatMap
	default:
		return GladeCombatMap
	}
}
