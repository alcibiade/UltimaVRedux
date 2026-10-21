package references

import (
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// Tiles is the tile table by sprite index.
type Tiles map[indexes.SpriteIndex]*Tile

// GetTile returns the tile for the sprite index, or nil when it is unknown.
func (t *Tiles) GetTile(tileNum indexes.SpriteIndex) *Tile {
	tile, exists := (*t)[tileNum]
	if !exists {
		return nil // Return nil if the tileNum doesn't exist
	}

	return tile
}

// NewTileReferences returns the tile table.
func NewTileReferences() *Tiles {
	tiles := newTileDefinitions()
	for index, tile := range tiles {
		tile.Index = index
	}

	return &tiles
}

// GetChairByPushDirection returns the chair tile that results from pushing a chair in the direction.
func (t *Tiles) GetChairByPushDirection(direction Direction) *Tile {
	switch direction {
	case Down:
		return t.GetTile(indexes.ChairBackBack)
	case Up:
		return t.GetTile(indexes.ChairBackForward)
	case Left:
		return t.GetTile(indexes.ChairBackLeft)
	case Right:
		return t.GetTile(indexes.ChairBackRight)
	case NoneDirection:
	}

	return t.GetTile(indexes.ChairBackBack)
}

// GetCannonByPushDirection returns the cannon tile that results from pushing a cannon in the direction.
func (t *Tiles) GetCannonByPushDirection(direction Direction) *Tile {
	switch direction {
	case Down:
		return t.GetTile(indexes.CannonDown)
	case Up:
		return t.GetTile(indexes.CannonUp)
	case Left:
		return t.GetTile(indexes.CannonLeft)
	case Right:
		return t.GetTile(indexes.CannonRight)
	case NoneDirection:
	}

	return t.GetTile(indexes.CannonLeft)
}
