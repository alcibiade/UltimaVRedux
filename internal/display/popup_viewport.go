package display

import "image"

// SetMapPopupViewportProvider replaces the presentation-only map popup viewport
// provider and returns the previous provider for restoration. Nil disables the
// override. Like other display configuration, it belongs to the game-loop thread.
func (dm *DisplayManager) SetMapPopupViewportProvider(
	provider func(image.Rectangle) image.Rectangle,
) func(image.Rectangle) image.Rectangle {
	previous := dm.mapPopupViewportProvider
	dm.mapPopupViewportProvider = provider

	return previous
}

// MapPopupViewport returns the provider's viewport clipped to screenBounds.
// An empty result means callers should retain their legacy popup placement.
func (dm *DisplayManager) MapPopupViewport(screenBounds image.Rectangle) image.Rectangle {
	if dm.mapPopupViewportProvider == nil || screenBounds.Empty() {
		return image.Rect(0, 0, 0, 0)
	}

	return dm.mapPopupViewportProvider(screenBounds).Intersect(screenBounds)
}
