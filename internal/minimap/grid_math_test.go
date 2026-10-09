package minimap

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestGridToScreenTile_ActorAtCameraCenter(t *testing.T) {
	// Actor at camera center (5,5) should render at screen tile center (5,5)
	cameraCenter := references.Position{X: 5, Y: 5}
	actorPos := references.Position{X: 5, Y: 5}

	screenX, screenY := GridToScreenTile(actorPos, cameraCenter, 11)

	assert.Equal(t, 5, screenX)
	assert.Equal(t, 5, screenY)
}

func TestGridToScreenTile_ActorBelowCamera(t *testing.T) {
	// Actor at (5,10) with camera at (5,5) should render 5 tiles below center
	cameraCenter := references.Position{X: 5, Y: 5}
	actorPos := references.Position{X: 5, Y: 10}

	screenX, screenY := GridToScreenTile(actorPos, cameraCenter, 11)

	assert.Equal(t, 5, screenX)
	assert.Equal(t, 10, screenY) // 5 + (10-5) = 10
}

func TestGridToScreenTile_ActorAboveCamera(t *testing.T) {
	// Actor at (5,2) with camera at (5,5) should render 3 tiles above center
	cameraCenter := references.Position{X: 5, Y: 5}
	actorPos := references.Position{X: 5, Y: 2}

	screenX, screenY := GridToScreenTile(actorPos, cameraCenter, 11)

	assert.Equal(t, 5, screenX)
	assert.Equal(t, 2, screenY) // 5 + (2-5) = 2
}

func TestGridToScreenTile_ActorRightOfCamera(t *testing.T) {
	// Actor at (8,5) with camera at (5,5) should render 3 tiles to the right
	cameraCenter := references.Position{X: 5, Y: 5}
	actorPos := references.Position{X: 8, Y: 5}

	screenX, screenY := GridToScreenTile(actorPos, cameraCenter, 11)

	assert.Equal(t, 8, screenX) // 5 + (8-5) = 8
	assert.Equal(t, 5, screenY)
}

func TestGridToScreenTile_ActorOffscreen(t *testing.T) {
	// Actor way outside viewport - should still calculate position
	// (even if it won't be rendered)
	cameraCenter := references.Position{X: 5, Y: 5}
	actorPos := references.Position{X: 20, Y: 20}

	screenX, screenY := GridToScreenTile(actorPos, cameraCenter, 11)

	assert.Equal(t, 20, screenX) // 5 + (20-5) = 20
	assert.Equal(t, 20, screenY)
}

func TestIsOnScreen_CenterActor(t *testing.T) {
	cameraCenter := references.Position{X: 5, Y: 5}
	actorPos := references.Position{X: 5, Y: 5}

	assert.True(t, IsOnScreen(actorPos, cameraCenter, 11))
}

func TestIsOnScreen_EdgeActors(t *testing.T) {
	cameraCenter := references.Position{X: 5, Y: 5}

	// All corners of 11x11 grid should be on screen
	assert.True(t, IsOnScreen(references.Position{X: 0, Y: 0}, cameraCenter, 11))
	assert.True(t, IsOnScreen(references.Position{X: 10, Y: 0}, cameraCenter, 11))
	assert.True(t, IsOnScreen(references.Position{X: 0, Y: 10}, cameraCenter, 11))
	assert.True(t, IsOnScreen(references.Position{X: 10, Y: 10}, cameraCenter, 11))
}

func TestIsOnScreen_OffscreenActors(t *testing.T) {
	cameraCenter := references.Position{X: 5, Y: 5}

	// Just outside viewport
	assert.False(t, IsOnScreen(references.Position{X: -1, Y: 5}, cameraCenter, 11))
	assert.False(t, IsOnScreen(references.Position{X: 11, Y: 5}, cameraCenter, 11))
	assert.False(t, IsOnScreen(references.Position{X: 5, Y: -1}, cameraCenter, 11))
	assert.False(t, IsOnScreen(references.Position{X: 5, Y: 11}, cameraCenter, 11))
}

func TestGridToScreenTile_DifferentViewportSize(t *testing.T) {
	// Test with 9x9 viewport (center at 4,4)
	cameraCenter := references.Position{X: 4, Y: 4}
	actorPos := references.Position{X: 4, Y: 4}

	screenX, screenY := GridToScreenTile(actorPos, cameraCenter, 9)

	assert.Equal(t, 4, screenX)
	assert.Equal(t, 4, screenY)
}
