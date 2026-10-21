package sprites

import (
	"github.com/bradhannah/UltimaVRedux/internal/originaldata/dungeonarchive"
)

// Synthetic archive fixtures and the real-data image comparison share the
// production archive naming and decoding authorities.
func dungeonArchiveName(wallSet int) string {
	return dungeonarchive.WallArchiveName(wallSet)
}

func dungeonMonsterArchiveName(monster int) string {
	return dungeonarchive.MonsterArchiveName(monster)
}
