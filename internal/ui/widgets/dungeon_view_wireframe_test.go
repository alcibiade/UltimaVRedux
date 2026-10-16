//go:build dungeonviewwireframe

package widgets

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// Opt-in synthetic sample generator, not a golden test or native-play acceptance.
func TestGenerateDungeonViewWireframes(t *testing.T) {
	outputDirectory := os.Getenv("DUNGEON_VIEW_SAMPLE_DIR")
	if outputDirectory == "" {
		t.Skip("set DUNGEON_VIEW_SAMPLE_DIR to write representative PNG samples")
	}
	require.NoError(t, os.MkdirAll(outputDirectory, 0o700))
	for _, size := range []image.Point{image.Pt(1280, 720), image.Pt(1920, 1080)} {
		setResponsiveDialogTestResolution(t, size.X, size.Y)
		for _, name := range dungeonViewSampleNames() {
			writeDungeonViewSample(t, outputDirectory, name, size, dungeonViewSampleModel(name))
		}
		for _, facing := range dungeonViewTestFacings() {
			for _, feature := range []struct {
				name string
				cell references.DungeonCell
			}{{"ladder", 0x30}, {"pit-ceiling", 0x69}} {
				model := dungeonViewSampleModel("dense")
				model.Facing = facing
				for row := range model.Cells {
					for column := range model.Cells[row] {
						cell := &model.Cells[row][column]
						if cell.SourcePosition == (references.Position{X: 3, Y: 3}) {
							cell.SourceCell = feature.cell
						}
					}
				}
				name := "party-" + feature.name + "-" + strings.ToLower(facing.Direction().GetDirectionCompassName())
				writeDungeonViewSample(t, outputDirectory, name, size, model)
			}
		}
	}
}

func writeDungeonViewSample(t *testing.T, directory, name string, size image.Point, model game_state.DungeonViewModel) {
	t.Helper()
	dialog := NewDungeonViewDialog(DungeonViewDialogConfig{
		Placement: dungeonViewSamplePlacement(), Keyboard: nil, Model: model,
	})
	screen := ebiten.NewImage(size.X, size.Y)
	dialog.Draw(screen)
	pixels := make([]byte, size.X*size.Y*4)
	screen.ReadPixels(pixels)
	screen.Deallocate()
	dialog.buffer.Deallocate()
	dialog.border.Dispose()
	raster := &image.RGBA{Pix: pixels, Stride: size.X * 4, Rect: image.Rectangle{Max: size}}
	for _, reduced := range []bool{false, true} {
		suffix := "colour"
		if reduced {
			dungeonViewReduceColour(pixels)
			suffix = "grey4"
		}
		var encoded bytes.Buffer
		require.NoError(t, png.Encode(&encoded, raster))
		filename := fmt.Sprintf("dungeon-view-%s-%dx%d-%s.png", name, size.X, size.Y, suffix)
		require.NoError(t, os.WriteFile(filepath.Join(directory, filename), encoded.Bytes(), 0o600))
	}
}
