package references

import (
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

type Tiles map[indexes.SpriteIndex]*Tile

func (t *Tiles) GetTile(tileNum indexes.SpriteIndex) *Tile {
	tile, exists := (*t)[tileNum]
	if !exists {
		return nil // Return nil if the tileNum doesn't exist
	}

	return tile
}

func NewTileReferences() *Tiles {
	tiles := newTileDefinitions()
	for index, tile := range tiles {
		tile.Index = index
	}

	return &tiles
}

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
	default:
		return t.GetTile(indexes.ChairBackBack)
	}
}

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
	default:
		return t.GetTile(indexes.CannonLeft)
	}
}
