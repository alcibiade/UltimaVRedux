package widgets

import (
	"github.com/bradhannah/UltimaVRedux/pkg/input"
	"github.com/hajimehoshi/ebiten/v2"
)

// Embedding navigation opts a list widget into DialogStack focus notifications.
type navigation struct {
	input.ListNavigation
	keyState func(ebiten.Key) bool
}

func newNavigation(keyState func(ebiten.Key) bool) navigation {
	var state input.ListNavigation

	return navigation{ListNavigation: state, keyState: keyState}
}

func (n *navigation) onDialogFocusChanged(_ bool) {
	n.BlockUntilRelease(n.keyState)
}

// listNavigationDelta preserves each list's measured page size. A zero page
// size explicitly disables paging without falling through to row movement.
func listNavigationDelta(key ebiten.Key, pageSize int) int {
	switch key { //nolint:exhaustive // Other navigation intents have no row delta.
	case ebiten.KeyUp:
		return -1
	case ebiten.KeyDown:
		return 1
	case ebiten.KeyPageUp:
		return -pageSize
	case ebiten.KeyPageDown:
		return pageSize
	default:
		return 0
	}
}

func navigateScrollableList(state *ScrollableListState, key ebiten.Key) bool {
	switch key { //nolint:exhaustive // Only list movement intents change selection.
	case ebiten.KeyUp:
		return state.MoveUp()
	case ebiten.KeyDown:
		return state.MoveDown()
	case ebiten.KeyPageUp:
		return state.MovePageUp()
	case ebiten.KeyPageDown:
		return state.MovePageDown()
	}

	return false
}
