package astar

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

type Node struct {
	Position references.Position
	GScore   int // Cost from start to current node
	FScore   int // GScore + heuristic cost to the goal
	Parent   *Node
}
