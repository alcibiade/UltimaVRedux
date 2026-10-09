package sprites

import (
	"context"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/graphics/legacy"
	"github.com/bradhannah/UltimaVRedux/internal/originaldata/dungeonarchive"
	"github.com/bradhannah/UltimaVRedux/internal/originaldata/graphicsarchive"
)

// Synthetic archive fixtures and the real-data image comparison share the
// production archive naming and decoding authorities.
func dungeonArchiveName(wallSet int) string {
	return dungeonarchive.WallArchiveName(wallSet)
}

func dungeonMonsterArchiveName(monster int) string {
	return dungeonarchive.MonsterArchiveName(monster)
}

func loadDungeonArchive(dataDir, name string) (legacy.Archive, error) {
	archive, err := graphicsarchive.Load(context.Background(), dataDir, name)
	if err != nil {
		return legacy.Archive{}, fmt.Errorf("loading dungeon archive %s: %w", name, err)
	}

	return archive, nil
}
