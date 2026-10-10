package minimap

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// GridToScreenTile converts an actor's grid position to a screen tile position
// relative to the camera center.
//
// Parameters:
//   - actorPos: The actor's position in grid coordinates
//   - cameraCenter: The camera's center position in grid coordinates
//   - viewportSize: The number of tiles visible (e.g., 11 for 11x11 viewport)
//
// Returns:
//   - screenX, screenY: The screen tile coordinates (0-based)
//
// The camera center will appear at the center of the viewport.
// For an 11x11 viewport, the center is at screen tile (5,5).
func GridToScreenTile(actorPos, cameraCenter references.Position, viewportSize int) (int, int) {
	center := viewportSize / 2

	// Calculate offset from camera center
	offsetX := int(actorPos.X) - int(cameraCenter.X)
	offsetY := int(actorPos.Y) - int(cameraCenter.Y)

	// Add offset to screen center
	screenX := center + offsetX
	screenY := center + offsetY

	return screenX, screenY
}

// IsOnScreen returns true if the actor's grid position is visible
// within the viewport centered on the camera.
//
// Parameters:
//   - actorPos: The actor's position in grid coordinates
//   - cameraCenter: The camera's center position in grid coordinates
//   - viewportSize: The number of tiles visible (e.g., 11 for 11x11 viewport)
func IsOnScreen(actorPos, cameraCenter references.Position, viewportSize int) bool {
	screenX, screenY := GridToScreenTile(actorPos, cameraCenter, viewportSize)

	return screenX >= 0 && screenX < viewportSize &&
		screenY >= 0 && screenY < viewportSize
}
