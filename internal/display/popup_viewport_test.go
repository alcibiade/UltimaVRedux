package display

import (
	"image"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapPopupViewportClipsToScreen(t *testing.T) {
	screen := image.Rect(10, 20, 1290, 740)
	for _, test := range []struct {
		name     string
		viewport image.Rectangle
		want     image.Rectangle
	}{
		{"empty", image.Rectangle{}, image.Rectangle{}},
		{"inside", image.Rect(30, 40, 600, 700), image.Rect(30, 40, 600, 700)},
		{"partial", image.Rect(-10, -20, 600, 700), image.Rect(10, 20, 600, 700)},
		{"oversized", image.Rect(-10, -20, 2000, 1000), screen},
		{"outside", image.Rect(2000, 2000, 2100, 2100), image.Rectangle{}},
		{"zero width", image.Rect(30, 40, 30, 700), image.Rectangle{}},
		{"inverted", image.Rectangle{Min: image.Pt(600, 700), Max: image.Pt(30, 40)}, image.Rectangle{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			manager := &DisplayManager{}
			manager.SetMapPopupViewportProvider(func(bounds image.Rectangle) image.Rectangle {
				assert.Equal(t, screen, bounds)

				return test.viewport
			})
			got := manager.MapPopupViewport(screen)
			assert.Equal(t, test.want, got)
			if !got.Empty() {
				assert.True(t, got.In(screen))
				assert.Positive(t, got.Dx())
				assert.Positive(t, got.Dy())
			}
		})
	}
}

func TestMapPopupViewportProviderRestore(t *testing.T) {
	manager := &DisplayManager{}
	screen := image.Rect(0, 0, 1280, 720)
	assert.Equal(t, image.Rectangle{}, manager.MapPopupViewport(screen))
	viewport := image.Rect(20, 30, 600, 700)
	previous := manager.SetMapPopupViewportProvider(func(image.Rectangle) image.Rectangle { return viewport })
	require.Nil(t, previous)
	assert.Equal(t, viewport, manager.MapPopupViewport(screen))
	installed := manager.SetMapPopupViewportProvider(previous)
	require.NotNil(t, installed)
	assert.Equal(t, image.Rectangle{}, manager.MapPopupViewport(screen))
	manager.SetMapPopupViewportProvider(installed)
	assert.Equal(t, viewport, manager.MapPopupViewport(screen))
}

func TestMapPopupViewportEmptyScreenSkipsProvider(t *testing.T) {
	manager := &DisplayManager{}
	manager.SetMapPopupViewportProvider(func(image.Rectangle) image.Rectangle {
		t.Fatal("empty screen must not invoke the provider")

		return image.Rectangle{}
	})
	assert.Equal(t, image.Rectangle{}, manager.MapPopupViewport(image.Rectangle{}))
}
