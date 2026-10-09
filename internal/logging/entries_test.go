package logging

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoggerRetainsOnlyAcceptedStructuredEntries(t *testing.T) {
	logger := newLoggerWithCapacity(4)
	logger.SetEnabled(true)
	logger.SetCategoryEnabled(CategoryCombat, true)
	logger.SetMinLevel(INFO)
	logger.Debug(CategoryCombat, "filtered")
	logger.Info(CategoryCombat, "hit %d", 7)
	logger.Info(CategoryAI, "disabled category")

	batch := logger.EntriesAfter(0, 10)
	require.Len(t, batch.Entries, 1)
	assert.Equal(t, uint64(1), batch.Entries[0].Sequence)
	assert.Equal(t, "INFO", batch.Entries[0].Level)
	assert.Equal(t, CategoryCombat, batch.Entries[0].Category)
	assert.Equal(t, "hit 7", batch.Entries[0].Message)
}

func TestLoggerEntryCursorReportsOverwriteAndPagination(t *testing.T) {
	logger := newLoggerWithCapacity(3)
	logger.SetEnabled(true)
	logger.SetCategoryEnabled(CategorySystem, true)
	for index := range 5 {
		logger.Info(CategorySystem, "entry %d", index+1)
	}

	first := logger.EntriesAfter(0, 2)
	require.Len(t, first.Entries, 2)
	assert.Equal(t, uint64(2), first.Dropped)
	assert.Equal(t, uint64(3), first.Entries[0].Sequence)
	assert.True(t, first.HasMore)
	second := logger.EntriesAfter(first.NextCursor, 2)
	require.Len(t, second.Entries, 1)
	assert.Equal(t, uint64(5), second.NextCursor)
	assert.False(t, second.HasMore)
}

func TestLoggerBoundsRetainedMessage(t *testing.T) {
	logger := newLoggerWithCapacity(1)
	logger.SetEnabled(true)
	logger.SetCategoryEnabled(CategorySystem, true)
	logger.Info(CategorySystem, "%s", strings.Repeat("x", maxEntryMessageBytes+100))

	batch := logger.EntriesAfter(0, 1)
	require.Len(t, batch.Entries, 1)
	assert.LessOrEqual(t, len(batch.Entries[0].Message), maxEntryMessageBytes)
}
