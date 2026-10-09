package references

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasementChestInstalledSourceCoordinates(t *testing.T) {
	filePath := filepath.Join(originaldata.Directory(t, "CASTLE.NPC"), "CASTLE.NPC")
	raw, err := os.ReadFile(filePath)
	require.NoError(t, err)
	parsed, err := getNPCsFromFile(filePath, int(LordBritishsCastle)-1)
	require.NoError(t, err)
	// NPC blocks contain 512 schedule bytes followed by 32 shape bytes.
	// Each schedule packs three action, X, Y, Z bytes and four times; ref:NPC-0001.
	require.Equal(t, 16, sizeOfNPCSchedule)
	for index, source := range LordBritishBasementChests() {
		offset := source.NPCNumber * 16
		record := raw[offset : offset+16]
		shape := raw[512+source.NPCNumber]
		t.Logf("%s record %d schedule[0x%x]=% x shape[0x%x]=%02x",
			filePath, source.NPCNumber, offset, record, 512+source.NPCNumber, shape)
		reference := parsed[source.NPCNumber]
		for slot := range 3 {
			assert.Equal(t, record[3+slot], reference.Schedule.X[slot])
			assert.Equal(t, record[6+slot], reference.Schedule.Y[slot])
			assert.Equal(t, record[9+slot], reference.Schedule.Floor[slot])
			assert.Equal(t, source.Position.X, Coordinate(record[3+slot]), "record %d X slot %d", source.NPCNumber, slot)
			assert.Equal(t, source.Position.Y, Coordinate(record[6+slot]), "record %d Y slot %d", source.NPCNumber, slot)
			assert.Equal(t, Basement, FloorNumber(record[9+slot]))
		}
		assert.Equal(t, indexes.SpriteIndex(indexes.Chest), reference.GetSpriteIndex())
		got, valid := BasementChestIndex(reference, source.NPCNumber)
		assert.True(t, valid, "raw source and bounded identity table must agree")
		assert.Equal(t, index, got)
	}
}

func TestBasementChestIdentityRejectsIncorrectThirdChestCoordinates(t *testing.T) {
	source := LordBritishBasementChests()[2]
	require.Equal(t, Position{X: 13, Y: 23}, source.Position)
	reference := NewNPCReferenceForSurfaceChest(LordBritishsCastle, source.Position, Basement)
	index, valid := BasementChestIndex(*reference, source.NPCNumber)
	require.True(t, valid)
	assert.Equal(t, 2, index)
	for slot := range reference.Schedule.X {
		changed := *reference
		changed.Schedule.X[slot] = 10
		_, valid = BasementChestIndex(changed, source.NPCNumber)
		assert.False(t, valid, "source identity must not accept a mismatched schedule slot")
	}
}
