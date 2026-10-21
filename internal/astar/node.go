package astar

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// Node is one A* search state; Parent links back to the previous step on the best known path.
type Node struct {
	Position references.Position
	GScore   int // Cost from start to current node
	FScore   int // GScore + heuristic cost to the goal
	Parent   *Node
}
