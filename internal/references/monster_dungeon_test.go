package references

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDungeonWanderingMonsterDefinitionsMatchSourceTables(t *testing.T) {
	want := [...]DungeonWanderingMonsterDefinition{
		{Monster: MonsterRat, Animation: 0x60},
		{Monster: MonsterBat, Animation: 0xa0},
		{Monster: MonsterSpider, Animation: 0x00},
		{Monster: MonsterGhost, Animation: 0x90},
		{Monster: MonsterSlime, Animation: 0x80},
		{Monster: MonsterGremlin, Animation: 0x60},
		{Monster: MonsterGazer, Animation: 0x00},
		{Monster: MonsterReaper, Animation: 0x00},
	}
	require.Equal(t, len(want), DungeonWanderingMonsterCount)

	for index, expected := range want {
		definition, ok := DungeonWanderingMonsterDefinitionAt(index)
		require.True(t, ok)
		assert.Equal(t, expected, definition)

		resolved, archive, ok := DungeonWanderingMonsterDefinitionFor(expected.Monster)
		require.True(t, ok)
		assert.Equal(t, index, archive)
		assert.Equal(t, expected, resolved)
		assert.True(t, IsDungeonWanderingMonster(expected.Monster))
	}

	_, ok := DungeonWanderingMonsterDefinitionAt(-1)
	assert.False(t, ok)
	_, ok = DungeonWanderingMonsterDefinitionAt(DungeonWanderingMonsterCount)
	assert.False(t, ok)
	_, _, ok = DungeonWanderingMonsterDefinitionFor(MonsterDragon)
	assert.False(t, ok)
	assert.False(t, IsDungeonWanderingMonster(MonsterDragon))
}
