package widgets

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScrollableListState_NoScrollNeeded(t *testing.T) {
	// 3 items, 5 visible slots - all items fit
	s := NewScrollableListState(3, 5)

	assert.False(t, s.IsScrollingNeeded(), "should not need scrolling")
	assert.Equal(t, 0, s.GetScrollOffset(), "scroll offset should be 0")
	assert.Equal(t, 0, s.GetCurrentIndex(), "should start at index 0")
	assert.Equal(t, 0, s.GetCursorScreenPosition(), "cursor should be at position 0")

	// Navigate down
	assert.True(t, s.MoveDown())
	assert.Equal(t, 1, s.GetCurrentIndex())
	assert.Equal(t, 1, s.GetCursorScreenPosition())
	assert.Equal(t, 0, s.GetScrollOffset(), "no scrolling when all fit")

	// Navigate to last item
	assert.True(t, s.MoveDown())
	assert.Equal(t, 2, s.GetCurrentIndex())
	assert.Equal(t, 2, s.GetCursorScreenPosition())

	// Can't go past end
	assert.False(t, s.MoveDown())
	assert.Equal(t, 2, s.GetCurrentIndex())

	// Navigate back up
	assert.True(t, s.MoveUp())
	assert.Equal(t, 1, s.GetCurrentIndex())
}

func TestScrollableListState_CenterLockedScrolling(t *testing.T) {
	// 10 items, 5 visible slots - scrolling needed
	s := NewScrollableListState(10, 5)

	assert.True(t, s.IsScrollingNeeded(), "should need scrolling")

	// Midpoint is 5/2 = 2 (0-indexed positions 0,1,2,3,4)
	// At start: cursor at 0, scroll at 0
	assert.Equal(t, 0, s.GetCurrentIndex())
	assert.Equal(t, 0, s.GetScrollOffset())
	assert.Equal(t, 0, s.GetCursorScreenPosition())

	// Move down to index 1 - cursor moves, no scroll yet
	s.MoveDown()
	assert.Equal(t, 1, s.GetCurrentIndex())
	assert.Equal(t, 0, s.GetScrollOffset())
	assert.Equal(t, 1, s.GetCursorScreenPosition())

	// Move to index 2 (midpoint) - still no scroll
	s.MoveDown()
	assert.Equal(t, 2, s.GetCurrentIndex())
	assert.Equal(t, 0, s.GetScrollOffset())
	assert.Equal(t, 2, s.GetCursorScreenPosition())

	// Move to index 3 - NOW scrolling starts, cursor stays at midpoint
	s.MoveDown()
	assert.Equal(t, 3, s.GetCurrentIndex())
	assert.Equal(t, 1, s.GetScrollOffset(), "scroll should start")
	assert.Equal(t, 2, s.GetCursorScreenPosition(), "cursor stays at midpoint")

	// Move to index 4
	s.MoveDown()
	assert.Equal(t, 4, s.GetCurrentIndex())
	assert.Equal(t, 2, s.GetScrollOffset())
	assert.Equal(t, 2, s.GetCursorScreenPosition())

	// Move to index 5
	s.MoveDown()
	assert.Equal(t, 5, s.GetCurrentIndex())
	assert.Equal(t, 3, s.GetScrollOffset())
	assert.Equal(t, 2, s.GetCursorScreenPosition())

	// Move to index 6
	s.MoveDown()
	assert.Equal(t, 6, s.GetCurrentIndex())
	assert.Equal(t, 4, s.GetScrollOffset())
	assert.Equal(t, 2, s.GetCursorScreenPosition())

	// Move to index 7 - scroll maxes out at 5 (10-5=5)
	s.MoveDown()
	assert.Equal(t, 7, s.GetCurrentIndex())
	assert.Equal(t, 5, s.GetScrollOffset(), "scroll at max")
	assert.Equal(t, 2, s.GetCursorScreenPosition())

	// Move to index 8 - scroll stays at max, cursor moves
	s.MoveDown()
	assert.Equal(t, 8, s.GetCurrentIndex())
	assert.Equal(t, 5, s.GetScrollOffset())
	assert.Equal(t, 3, s.GetCursorScreenPosition(), "cursor moves past midpoint")

	// Move to index 9 (last item)
	s.MoveDown()
	assert.Equal(t, 9, s.GetCurrentIndex())
	assert.Equal(t, 5, s.GetScrollOffset())
	assert.Equal(t, 4, s.GetCursorScreenPosition(), "cursor at last visible slot")

	// Can't go further
	assert.False(t, s.MoveDown())
}

func TestScrollableListState_VisibleRange(t *testing.T) {
	s := NewScrollableListState(10, 5)

	// At start
	start, end := s.GetVisibleRange()
	assert.Equal(t, 0, start)
	assert.Equal(t, 5, end)

	// Move to middle
	s.SetIndex(5)
	start, end = s.GetVisibleRange()
	assert.Equal(t, 3, start, "scroll offset should be 3")
	assert.Equal(t, 8, end)

	// Move to end
	s.SetIndex(9)
	start, end = s.GetVisibleRange()
	assert.Equal(t, 5, start)
	assert.Equal(t, 10, end)
}

func TestScrollableListState_HasItemsAboveBelow(t *testing.T) {
	s := NewScrollableListState(10, 5)

	// At start
	assert.False(t, s.HasItemsAbove())
	assert.True(t, s.HasItemsBelow())

	// In middle
	s.SetIndex(5)
	assert.True(t, s.HasItemsAbove())
	assert.True(t, s.HasItemsBelow())

	// At end
	s.SetIndex(9)
	assert.True(t, s.HasItemsAbove())
	assert.False(t, s.HasItemsBelow())
}

func TestScrollableListState_EmptyList(t *testing.T) {
	s := NewScrollableListState(0, 5)

	assert.False(t, s.IsScrollingNeeded())
	assert.Equal(t, 0, s.GetCurrentIndex())
	assert.Equal(t, 0, s.GetScrollOffset())

	// Navigation does nothing
	assert.False(t, s.MoveDown())
	assert.False(t, s.MoveUp())

	start, end := s.GetVisibleRange()
	assert.Equal(t, 0, start)
	assert.Equal(t, 0, end)
}

func TestScrollableListState_SingleItem(t *testing.T) {
	s := NewScrollableListState(1, 5)

	assert.False(t, s.IsScrollingNeeded())
	assert.Equal(t, 0, s.GetCurrentIndex())

	// Can't move
	assert.False(t, s.MoveDown())
	assert.False(t, s.MoveUp())

	start, end := s.GetVisibleRange()
	assert.Equal(t, 0, start)
	assert.Equal(t, 1, end)
}

func TestScrollableListState_Reset(t *testing.T) {
	s := NewScrollableListState(10, 5)

	// Navigate to middle
	s.SetIndex(5)
	assert.Equal(t, 5, s.GetCurrentIndex())
	assert.True(t, s.GetScrollOffset() > 0)

	// Reset with new count
	s.Reset(3)
	assert.Equal(t, 0, s.GetCurrentIndex())
	assert.Equal(t, 0, s.GetScrollOffset())
	assert.False(t, s.IsScrollingNeeded())

	// Reset with larger count
	s.Reset(20)
	assert.Equal(t, 0, s.GetCurrentIndex())
	assert.Equal(t, 0, s.GetScrollOffset())
	assert.True(t, s.IsScrollingNeeded())
}

func TestScrollableListState_SetIndex(t *testing.T) {
	s := NewScrollableListState(10, 5)

	// Set to middle
	s.SetIndex(5)
	assert.Equal(t, 5, s.GetCurrentIndex())

	// Set beyond bounds (clamped)
	s.SetIndex(100)
	assert.Equal(t, 9, s.GetCurrentIndex())

	// Set negative (clamped)
	s.SetIndex(-5)
	assert.Equal(t, 0, s.GetCurrentIndex())
}

func TestScrollableListState_ExactFit(t *testing.T) {
	// Exactly 5 items, 5 slots - no scrolling needed
	s := NewScrollableListState(5, 5)

	assert.False(t, s.IsScrollingNeeded())

	// Navigate through all items
	for i := 0; i < 4; i++ {
		assert.True(t, s.MoveDown())
		assert.Equal(t, i+1, s.GetCurrentIndex())
		assert.Equal(t, 0, s.GetScrollOffset())
		assert.Equal(t, i+1, s.GetCursorScreenPosition())
	}

	// Can't go past
	assert.False(t, s.MoveDown())
}

func TestScrollableListState_ScrollUpFromMiddle(t *testing.T) {
	s := NewScrollableListState(10, 5)

	// Go to end
	s.SetIndex(9)
	assert.Equal(t, 5, s.GetScrollOffset())

	// Scroll back up
	s.MoveUp() // index 8
	assert.Equal(t, 5, s.GetScrollOffset())
	assert.Equal(t, 3, s.GetCursorScreenPosition())

	s.MoveUp() // index 7
	assert.Equal(t, 5, s.GetScrollOffset())
	assert.Equal(t, 2, s.GetCursorScreenPosition())

	s.MoveUp() // index 6 - now scroll starts moving down
	assert.Equal(t, 4, s.GetScrollOffset())
	assert.Equal(t, 2, s.GetCursorScreenPosition())

	s.MoveUp() // index 5
	assert.Equal(t, 3, s.GetScrollOffset())
	assert.Equal(t, 2, s.GetCursorScreenPosition())

	// Continue to start
	s.SetIndex(0)
	assert.Equal(t, 0, s.GetScrollOffset())
	assert.Equal(t, 0, s.GetCursorScreenPosition())
}
