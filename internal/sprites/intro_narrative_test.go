package sprites

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseIntroductionStoryBlocksUsesVerifiedOffsets(t *testing.T) {
	const requiredSize = 11678
	data := make([]byte, requiredSize)
	for index, offset := range introductionStoryOffsets {
		limit := requiredSize
		if index+1 < len(introductionStoryOffsets) {
			limit = introductionStoryOffsets[index+1]
		}
		for position := offset; position < limit-1; position++ {
			data[position] = byte('A' + index%26)
		}
	}

	blocks, err := parseIntroductionStoryBlocks(data)

	require.NoError(t, err)
	require.Len(t, blocks, introductionStoryBlockCount)
	assert.Len(t, blocks[0], 272)
	assert.Len(t, blocks[19], 774)
	assert.Equal(t, byte('A'), blocks[0][0])
	assert.Equal(t, byte('T'), blocks[19][0])
}

func TestParseIntroductionStoryBlocksRejectsTruncatedData(t *testing.T) {
	_, err := parseIntroductionStoryBlocks(make([]byte, 100))

	require.ErrorContains(t, err, "block 0 is truncated before 273")
}

func TestParseIntroductionStoryBlocksRejectsEarlyTerminator(t *testing.T) {
	data := make([]byte, 11678)
	data[1] = 0

	_, err := parseIntroductionStoryBlocks(data)

	require.ErrorContains(t, err, "terminator is not at expected offset 272")
}
