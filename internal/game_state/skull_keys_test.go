package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestSkullKeysFoundOncePerDay(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Minoc).
		WithPlayerAt(2, 1).
		WithSystemCallbacks().
		Build()
	gs.DateTime.Day = 15

	require.True(t, gs.ActionSearchSmallMap(references.Down))
	require.Len(t, mockCallbacks.Messages, 1)
	assert.Equal(t, "Thou dost find\na ring of keys!", mockCallbacks.Messages[0])

	mockCallbacks.Reset()
	assert.False(t, gs.ActionSearchSmallMap(references.Down))
	require.Len(t, mockCallbacks.Messages, 1)
	assert.Contains(t, mockCallbacks.Messages[0], "Already found items")
}

func TestSkullKeysAvailableNextDay(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Minoc).
		WithPlayerAt(2, 1).
		WithSystemCallbacks().
		Build()
	gs.DateTime.Day = 15
	require.True(t, gs.ActionSearchSmallMap(references.Down))

	mockCallbacks.Reset()
	gs.DateTime.Day = 16
	require.True(t, gs.ActionSearchSmallMap(references.Down))
	require.Len(t, mockCallbacks.Messages, 1)
	assert.Equal(t, "Thou dost find\na ring of keys!", mockCallbacks.Messages[0])
	position := references.Position{X: 2, Y: 2}
	stack, ok := gs.ItemStacksMap.Snapshot(&position)
	require.True(t, ok)
	require.Len(t, stack.Items, 2)
	for _, item := range stack.Items {
		assert.Equal(t, references.SkullKeys, item.Item)
		assert.Equal(t, uint16(5), item.Quantity)
	}
}

func TestDailySearchMarkersResetAtMonthBoundary(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Minoc).
		WithSystemCallbacks().
		Build()
	gs.SearchManager.authenticLocationManager.skullKeysFoundDay = 1
	gs.SearchManager.reagentCachesFoundDay = [3]byte{1, 2, 3}
	gs.DateTime.Month = 1
	gs.DateTime.Day = 28
	gs.DateTime.Hour = 23
	gs.DateTime.Minute = 59

	assert.Equal(t, 1, gs.AdvanceGameplayMinutes(1))
	assert.Zero(t, gs.SearchManager.authenticLocationManager.skullKeysFoundDay)
	assert.Equal(t, [3]byte{}, gs.SearchManager.reagentCachesFoundDay)
}

func TestSkullKeySearchLocation(t *testing.T) {
	var skullKeyLocation *references.SearchLocationReference
	for _, location := range references.GetAllSearchLocations() {
		if location.Index == 14 {
			location := location
			skullKeyLocation = &location
			break
		}
	}

	require.NotNil(t, skullKeyLocation)
	assert.Equal(t, int(references.Minoc), skullKeyLocation.MapNumber)
	assert.Equal(t, 2, int(skullKeyLocation.X))
	assert.Equal(t, 2, int(skullKeyLocation.Y))
	assert.True(t, skullKeyLocation.Requirements.IsSkullKeys)
}

func TestSkullKeysNotFoundAtWrongLocation(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	assert.False(t, gs.ActionSearchSmallMap(references.Down))
	require.Len(t, mockCallbacks.Messages, 1)
	assert.NotContains(t, mockCallbacks.Messages[0], "skull keys")
}
