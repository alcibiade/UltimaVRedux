//go:build signwireframe

package widgets

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/sprites"
)

func TestGenerateSignDialogWireframes(t *testing.T) {
	outputDirectory := os.Getenv("SIGN_DIALOG_SAMPLE_DIR")
	if outputDirectory == "" {
		t.Skip("set SIGN_DIALOG_SAMPLE_DIR to write representative PNG wireframes")
	}
	require.NoError(t, os.MkdirAll(outputDirectory, 0o700))
	placement := sprites.PercentBasedPlacement{
		StartPercentX: 0.10,
		EndPercentX:   0.65,
		StartPercentY: 0.20,
		EndPercentY:   0.80,
	}
	rawText := []byte(
		"8lllllmlllll9\n" +
			"g           g\n" +
			"g  FRAME    g\n" +
			"g  CHECK    g\n" +
			"g           g\n" +
			":lllllnlllll;\n",
	)
	tests := []struct {
		name   string
		width  int
		height int
		scale  int
	}{
		{name: "sign-dialog-720p.png", width: 1280, height: 720, scale: 3},
		{name: "sign-dialog-1080p.png", width: 1920, height: 1080, scale: 5},
		{name: "sign-dialog-1440p.png", width: 2560, height: 1440, scale: 6},
	}
	for _, test := range tests {
		setResponsiveDialogTestResolution(t, test.width, test.height)
		dialog := NewSignDialog(SignDialogConfig{Placement: placement, RawText: rawText})
		require.Equal(t, test.scale, fittingSignScale(dialog.border.GetInteriorBounds()))
		screen := ebiten.NewImage(test.width, test.height)
		dialog.Draw(screen)
		pixels := make([]byte, test.width*test.height*4)
		screen.ReadPixels(pixels)
		wireframe := &image.RGBA{
			Pix: pixels, Stride: test.width * 4, Rect: image.Rect(0, 0, test.width, test.height),
		}
		var encoded bytes.Buffer
		require.NoError(t, png.Encode(&encoded, wireframe))
		require.NoError(t, os.WriteFile(filepath.Join(outputDirectory, test.name), encoded.Bytes(), 0o600))
	}
}
