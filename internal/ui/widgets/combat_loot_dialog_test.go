package widgets

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

func TestCombatLootDialogOverflowAndAcknowledgementIsolation(t *testing.T) {
	entries := make([]game_state.CombatLootReveal, 31)
	for index := range entries {
		entries[index] = game_state.CombatLootReveal{Sprite: indexes.ItemWeapon, Description: "some armour!"}
	}
	pressed := false
	keyboard := input.NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool { return key == ebiten.KeyEnter && pressed })
	dialog := NewCombatLootDialog(entries, keyboard, &sprites.SpriteSheet{})
	entries[0].Description = "not detached"
	assert.Equal(t, "some armour!", dialog.entries[0].Description)
	pressed = true
	dialog.Update()
	assert.False(t, dialog.IsDismissed())
	dialog.Reveal(31)
	dialog.FinishReveal()
	dialog.Update()
	assert.False(t, dialog.IsDismissed(), "held early Enter cannot acknowledge")
	for _, size := range []image.Point{{320, 200}, {1280, 720}, {1920, 1080}} {
		setResponsiveDialogTestResolution(t, size.X, size.Y)
		bounds := image.Rectangle{Max: size}
		dialog.prepareDiscovery(bounds, image.Rect(0, 0, size.X*3/4, size.Y))
		assert.True(t, dialog.bounds.In(bounds))
		require.Positive(t, dialog.capacity)
		assert.Less(t, dialog.capacity, len(entries))
		assert.Contains(t, dialog.output.GetOutputStr(false), "Continue")
		before := dialog.offset
		keyboard.SimulateKeyPress(ebiten.KeyUp)
		dialog.Update()
		assert.LessOrEqual(t, dialog.offset, before)
	}
	pressed = false
	keyboard.ClearFrameConsumption()
	dialog.Update()
	pressed = true
	dialog.Update()
	assert.True(t, dialog.IsDismissed())
}

func TestLootRowsCenterIconsAndSingleOrWrappedDescriptions(t *testing.T) {
	for _, screenWidth := range []int{1280, 1920} {
		_, output := GetScaledFontWithOutputForWidth(FontConfig{Size: FontSizeBody, LineSpacingMultiplier: 1, MaxLines: 4, MaxChars: 40}, screenWidth)
		for _, lines := range []int{1, 2, 3} {
			iconHeight := output.GetTotalHeightForLines(2)
			textHeight := output.GetTotalHeightForLines(lines)
			height := max(iconHeight, textHeight)
			iconTop, textTop := lootRowContentTops(50, height, iconHeight, textHeight)
			assert.InDelta(t, iconTop+iconHeight/2, textTop+textHeight/2, 0.001)
			assert.GreaterOrEqual(t, iconTop, 50.0)
			assert.GreaterOrEqual(t, textTop, 50.0)
			if lines == 1 {
				assert.Greater(t, textTop, iconTop, "one-line text must not share the tall icon's top edge")
			}
		}
	}
}

func TestLootIconAlignmentIgnoresTransparentPaddingWithoutRescaling(t *testing.T) {
	source := ebiten.NewImage(16, 16)
	var pixels [16 * 16 * 4]byte
	for row := 8; row < 14; row++ {
		for column := 3; column < 11; column++ {
			offset := (row*16 + column) * 4
			pixels[offset], pixels[offset+1], pixels[offset+3] = 255, 255, 255
		}
	}
	source.WritePixels(pixels[:])
	trimmed := trimLootIcon(source)
	require.Equal(t, image.Rect(0, 0, 8, 6), trimmed.Bounds())
	iconTop, textTop := lootRowContentTops(0, 32, float64(trimmed.Bounds().Dy())*2, 16)
	assert.InDelta(t, iconTop+6, textTop+8, 0.001)
}
