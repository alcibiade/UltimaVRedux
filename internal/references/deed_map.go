package references

import (
	"fmt"
	"slices"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// REDUX DEVIATION: CRAFT-0003: the original has no generated maps. A deed map is generated
// deterministically from its terrain, seed and generator version, so a save stores only those
// and the squares that changed since (ADR-014).

// DeedResourceNode is one gatherable square of a generated deed map. Reagent is meaningful only
// for reagent patches.
type DeedResourceNode struct {
	Position Position
	Kind     DeedResourceKind
	Sprite   indexes.SpriteIndex
	Reagent  Reagent
}

// DeedLayout is the output of the deed map generator. Tiles are indexed [x][y]; Nodes are
// sorted by row, then column.
type DeedLayout struct {
	Terrain DeedTerrain
	Version int
	Seed    uint32
	Tiles   [XSmallMapTiles][YSmallMapTiles]indexes.SpriteIndex
	Nodes   []DeedResourceNode
}

// Tile returns the generated tile at a square.
func (l *DeedLayout) Tile(position Position) indexes.SpriteIndex {
	return l.Tiles[position.X][position.Y]
}

// NodeAt returns the generated resource node at a square, if any.
func (l *DeedLayout) NodeAt(position Position) (DeedResourceNode, bool) {
	for _, node := range l.Nodes {
		if node.Position == position {
			return node, true
		}
	}

	return DeedResourceNode{}, false
}

// DeedSeed derives the generator seed from a deed's identity and overworld square. It uses no
// game RNG, so staking a deed never shifts the original random stream.
func DeedSeed(id int, square Position) uint32 {
	value := uint32(2166136261) //nolint:mnd // FNV-1a offset basis.
	for _, part := range []uint32{uint32(id), uint32(square.X), uint32(square.Y)} {
		for shift := 0; shift < 32; shift += 8 {
			value ^= (part >> shift) & 0xff
			value *= 16777619 //nolint:mnd // FNV-1a prime.
		}
	}

	return value
}

// DeedSquareDepth is how far a square lies inside the nearest map edge (0 on the edge).
func DeedSquareDepth(position Position) int {
	last := int(XSmallMapTiles) - 1

	return min(int(position.X), int(position.Y), last-int(position.X), last-int(position.Y))
}

// DeedInteriorDepth is the smallest depth of the interior that is guaranteed open: inside the
// solid ring and its ragged fringe.
const DeedInteriorDepth = DeedBorderDepth + 1

// IsDeedEntryPath reports whether a square is on the entry lane, from the south edge to the yard.
func IsDeedEntryPath(position Position) bool {
	return position.X == DeedMapEntry.X && position.Y > DeedMapCentre.Y+DeedYardRadius
}

// isDeedYard reports whether a square lies in the clear yard around the centre.
func isDeedYard(position Position) bool {
	return chebyshev(position, DeedMapCentre) <= DeedYardRadius
}

func chebyshev(a, b Position) int {
	dx, dy := int(a.X)-int(b.X), int(a.Y)-int(b.Y)

	return max(dx, -dx, dy, -dy)
}

// GenerateDeedMap builds the deed map of one terrain, seed and generator version. The output
// for a given version never changes; an unknown version is an error.
func GenerateDeedMap(terrain DeedTerrain, seed uint32, version int) (DeedLayout, error) {
	if terrain.Key() == "" {
		return DeedLayout{}, fmt.Errorf("deed map: unknown terrain %d", terrain)
	}
	switch version {
	case 1:
		return generateDeedMapV1(terrain, seed), nil
	default:
		return DeedLayout{}, fmt.Errorf("deed map: unsupported generator version %d", version)
	}
}

// generateDeedMapV1 lays out a clearing: a solid decorative ring with a ragged inner fringe, an
// open interior with scattered accents, a clear central yard, an entry lane from the south edge,
// and resource nodes in loose clusters. No two nodes touch, even diagonally, and every node keeps
// DeedNodeMargin squares from the ring, so blocking nodes can never cut the interior apart.
func generateDeedMapV1(terrain DeedTerrain, seed uint32) DeedLayout {
	profile := DeedProfile(terrain)
	random := newDeedRandom(seed ^ uint32(terrain+1)*0x9e3779b9) //nolint:mnd // Golden-ratio salt per terrain.
	layout := DeedLayout{Terrain: terrain, Version: 1, Seed: seed}
	for x := range XSmallMapTiles {
		for y := range YSmallMapTiles {
			position := Position{X: x, Y: y}
			depth := DeedSquareDepth(position)
			switch {
			case IsDeedEntryPath(position):
				layout.Tiles[x][y] = profile.Path
			case depth < DeedBorderDepth:
				layout.Tiles[x][y] = random.pick(profile.Border)
			case depth == DeedBorderDepth && random.percent(DeedFringePercent):
				layout.Tiles[x][y] = random.pick(profile.Fringe)
			case !isDeedYard(position) && random.percent(profile.AccentPercent):
				layout.Tiles[x][y] = random.pick(profile.Accents)
			default:
				layout.Tiles[x][y] = profile.Ground
			}
		}
	}
	layout.Nodes = placeDeedNodes(profile, random)
	for _, node := range layout.Nodes {
		layout.Tiles[node.Position.X][node.Position.Y] = node.Sprite
	}
	slices.SortFunc(layout.Nodes, func(a, b DeedResourceNode) int {
		if a.Position.Y != b.Position.Y {
			return int(a.Position.Y) - int(b.Position.Y)
		}

		return int(a.Position.X) - int(b.Position.X)
	})

	return layout
}

// deedNodesPerAnchor sets how many nodes of one quota gather around one cluster anchor.
const deedNodesPerAnchor = 5

// deedNodeJitter spreads a cluster so it does not read as a solid block.
const deedNodeJitter = 10

func placeDeedNodes(profile DeedTerrainProfile, random *deedRandom) []DeedResourceNode {
	candidates := make([]Position, 0, int(XSmallMapTiles*YSmallMapTiles))
	for y := range YSmallMapTiles {
		for x := range XSmallMapTiles {
			position := Position{X: x, Y: y}
			if DeedSquareDepth(position) < DeedBorderDepth+DeedNodeMargin || isDeedYard(position) ||
				nearDeedEntryPath(position) {
				continue
			}
			candidates = append(candidates, position)
		}
	}
	occupied := make(map[Position]bool)
	nodes := make([]DeedResourceNode, 0)
	for _, quota := range profile.Nodes {
		anchors := make([]Position, 0, quota.Count/deedNodesPerAnchor+1)
		for range quota.Count/deedNodesPerAnchor + 1 {
			anchors = append(anchors, candidates[random.intn(len(candidates))])
		}
		for index := range quota.Count {
			anchor := anchors[index%len(anchors)]
			best, bestScore := Position{}, -1
			for _, candidate := range candidates {
				if touchesDeedNode(candidate, occupied) {
					continue
				}
				dx, dy := int(candidate.X)-int(anchor.X), int(candidate.Y)-int(anchor.Y)
				score := dx*dx + dy*dy + random.intn(deedNodeJitter)
				if bestScore < 0 || score < bestScore {
					best, bestScore = candidate, score
				}
			}
			if bestScore < 0 {
				break
			}
			occupied[best] = true
			node := DeedResourceNode{Position: best, Kind: quota.Kind, Sprite: quota.Sprite}
			if len(quota.Reagents) > 0 {
				node.Reagent = quota.Reagents[random.intn(len(quota.Reagents))]
			}
			nodes = append(nodes, node)
		}
	}

	return nodes
}

func nearDeedEntryPath(position Position) bool {
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			neighbour := Position{X: position.X + Coordinate(dx), Y: position.Y + Coordinate(dy)}
			if IsDeedEntryPath(neighbour) {
				return true
			}
		}
	}

	return false
}

func touchesDeedNode(position Position, occupied map[Position]bool) bool {
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			if occupied[Position{X: position.X + Coordinate(dx), Y: position.Y + Coordinate(dy)}] {
				return true
			}
		}
	}

	return false
}

// deedRandom is the generator's own versioned PRNG (xorshift32). It never touches the game RNG,
// and its sequence is part of generator version 1.
type deedRandom struct {
	state uint32
}

func newDeedRandom(seed uint32) *deedRandom {
	if seed == 0 {
		seed = 0x6d2b79f5 //nolint:mnd // Any non-zero state; xorshift cannot leave zero.
	}

	return &deedRandom{state: seed}
}

func (r *deedRandom) next() uint32 {
	r.state ^= r.state << 13 //nolint:mnd // xorshift32 shifts.
	r.state ^= r.state >> 17 //nolint:mnd // xorshift32 shifts.
	r.state ^= r.state << 5  //nolint:mnd // xorshift32 shifts.

	return r.state
}

func (r *deedRandom) intn(n int) int {
	if n <= 0 {
		panic("deed random range must be positive")
	}

	return int(r.next() % uint32(n))
}

func (r *deedRandom) percent(chance int) bool {
	return r.intn(100) < chance //nolint:mnd // Percent scale.
}

func (r *deedRandom) pick(choices []DeedWeightedSprite) indexes.SpriteIndex {
	total := 0
	for _, choice := range choices {
		total += choice.Weight
	}
	roll := r.intn(total)
	for _, choice := range choices {
		if roll < choice.Weight {
			return choice.Sprite
		}
		roll -= choice.Weight
	}

	panic("weighted deed sprite pick fell through")
}
