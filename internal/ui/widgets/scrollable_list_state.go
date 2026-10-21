package widgets

// ScrollableListState manages scroll state for list-based UI components.
// It implements center-locked scrolling where the cursor stays at the midpoint
// while items scroll, except at list edges where the cursor moves.
//
// Usage:
//  1. Create with NewScrollableListState(totalItems, visibleSlots)
//  2. Call MoveUp()/MoveDown() on navigation
//  3. Use GetVisibleRange() to slice your items for rendering
//  4. Use GetCursorScreenPosition() to know where to draw the cursor
type ScrollableListState struct {
	totalItems   int // Total item count
	visibleSlots int // How many items fit on screen
	currentIndex int // Selected item (0 to totalItems-1)
	scrollOffset int // First visible item index
}

// NewScrollableListState creates a scroll state manager.
// totalItems: number of items in the list
// visibleSlots: how many items can be displayed at once
func NewScrollableListState(totalItems, visibleSlots int) *ScrollableListState {
	s := &ScrollableListState{
		totalItems:   totalItems,
		visibleSlots: visibleSlots,
		currentIndex: 0,
		scrollOffset: 0,
	}

	return s
}

// MoveUp moves selection up by one. Returns true if selection changed.
func (s *ScrollableListState) MoveUp() bool {
	if s.totalItems == 0 {
		return false
	}

	if s.currentIndex > 0 {
		s.currentIndex--
		s.calculateScrollOffset()

		return true
	}

	return false
}

// MoveDown moves selection down by one. Returns true if selection changed.
func (s *ScrollableListState) MoveDown() bool {
	if s.totalItems == 0 {
		return false
	}

	if s.currentIndex < s.totalItems-1 {
		s.currentIndex++
		s.calculateScrollOffset()

		return true
	}

	return false
}

// MovePageUp moves selection up by half a page. Returns true if selection changed.
func (s *ScrollableListState) MovePageUp() bool {
	if s.totalItems == 0 || s.currentIndex == 0 {
		return false
	}

	// Move by half a page (minimum 1)
	pageSize := max(1, s.visibleSlots/2)
	s.currentIndex = max(0, s.currentIndex-pageSize)
	s.calculateScrollOffset()

	return true
}

// MovePageDown moves selection down by half a page. Returns true if selection changed.
func (s *ScrollableListState) MovePageDown() bool {
	if s.totalItems == 0 || s.currentIndex >= s.totalItems-1 {
		return false
	}

	// Move by half a page (minimum 1)
	pageSize := max(1, s.visibleSlots/2)
	s.currentIndex = min(s.totalItems-1, s.currentIndex+pageSize)
	s.calculateScrollOffset()

	return true
}

// SetIndex sets the current selection to a specific index.
// Index is clamped to valid range.
func (s *ScrollableListState) SetIndex(index int) {
	if s.totalItems == 0 {
		s.currentIndex = 0
		s.scrollOffset = 0

		return
	}

	s.currentIndex = max(0, min(index, s.totalItems-1))
	s.calculateScrollOffset()
}

// Reset resets to first item with new total count.
// Use when category changes or list content changes.
func (s *ScrollableListState) Reset(totalItems int) {
	s.totalItems = totalItems
	s.currentIndex = 0
	s.scrollOffset = 0
}

// GetCurrentIndex returns the currently selected item index (0 to totalItems-1).
func (s *ScrollableListState) GetCurrentIndex() int {
	return s.currentIndex
}

// GetCursorScreenPosition returns the cursor position on screen (0 to visibleSlots-1).
// This is where to draw the selection indicator relative to the visible items.
func (s *ScrollableListState) GetCursorScreenPosition() int {
	return s.currentIndex - s.scrollOffset
}

// GetVisibleRange returns the start (inclusive) and end (exclusive) indices
// for slicing the item list to get only visible items.
// Usage: visibleItems := items[start:end]
func (s *ScrollableListState) GetVisibleRange() (start, end int) {
	if s.totalItems == 0 {
		return 0, 0
	}

	start = s.scrollOffset
	end = min(s.scrollOffset+s.visibleSlots, s.totalItems)

	return start, end
}

// HasItemsAbove returns true if there are items scrolled above the visible area.
func (s *ScrollableListState) HasItemsAbove() bool {
	return s.scrollOffset > 0
}

// HasItemsBelow returns true if there are items scrolled below the visible area.
func (s *ScrollableListState) HasItemsBelow() bool {
	return s.scrollOffset+s.visibleSlots < s.totalItems
}

// calculateScrollOffset implements center-locked scrolling:
// - Cursor stays at midpoint while items scroll
// - At list edges, cursor moves off-center to reach first/last items
func (s *ScrollableListState) calculateScrollOffset() {
	// No scrolling needed if all items fit
	if s.totalItems <= s.visibleSlots {
		s.scrollOffset = 0

		return
	}

	midpoint := s.visibleSlots / 2
	maxScroll := s.totalItems - s.visibleSlots

	// Center cursor when possible, clamp at edges
	s.scrollOffset = s.currentIndex - midpoint
	s.scrollOffset = max(0, min(s.scrollOffset, maxScroll))
}
