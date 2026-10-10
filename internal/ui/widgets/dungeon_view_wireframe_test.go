//go:build dungeonviewwireframe

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

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
)

func TestGenerateDungeonViewWireframes(t *testing.T) {
	outputDirectory := os.Getenv("DUNGEON_VIEW_SAMPLE_DIR")
	if outputDirectory == "" {
		t.Skip("set DUNGEON_VIEW_SAMPLE_DIR to write representative PNG wireframes")
	}
	require.NoError(t, os.MkdirAll(outputDirectory, 0o700))
	model := representativeDungeonViewModel()
	placement := sprites.PercentBasedPlacement{
		StartPercentX: 0.03,
		EndPercentX:   0.97,
		StartPercentY: 0.04,
		EndPercentY:   0.96,
	}
	tests := []struct {
		name   string
		width  int
		height int
		scale  int
	}{
		{name: "dungeon-view-720p.png", width: 1280, height: 720, scale: 4},
		{name: "dungeon-view-1080p.png", width: 1920, height: 1080, scale: 5},
		{name: "dungeon-view-1440p.png", width: 2560, height: 1440, scale: 7},
		{name: "dungeon-view-2160p.png", width: 3840, height: 2160, scale: 11},
		{name: "dungeon-view-narrow.png", width: 320, height: 560, scale: 1},
	}
	for _, test := range tests {
		setResponsiveDialogTestResolution(t, test.width, test.height)
		dialog := NewDungeonViewDialog(DungeonViewDialogConfig{
			Placement: placement,
			Keyboard:  nil,
			Model:     model,
		})
		require.Equal(t, test.scale, dialog.layout.mapScale)
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

func representativeDungeonViewModel() game_state.DungeonViewModel {
	var sourceCells [references.YDungeonMapTiles][references.XDungeonMapTiles]references.DungeonCell
	for coordinate := range int(references.XDungeonMapTiles) {
		sourceCells[0][coordinate] = 0xb0
		sourceCells[int(references.YDungeonMapTiles)-1][coordinate] = 0xb0
		sourceCells[coordinate][0] = 0xb0
		sourceCells[coordinate][int(references.XDungeonMapTiles)-1] = 0xb0
	}
	sourceCells[0][4] = 0xe0
	sourceCells[4][0] = 0xd0
	sourceCells[7][4] = 0xc0
	sourceCells[1][1] = 0x10
	sourceCells[1][2] = 0x20
	sourceCells[1][3] = 0x30
	sourceCells[1][5] = 0x4f
	sourceCells[2][1] = 0x50
	sourceCells[2][2] = 0x83
	sourceCells[2][3] = 0xa0
	sourceCells[2][4] = 0xff
	sourceCells[2][5] = 0x08
	sourceCells[4][2] = 0x60
	sourceCells[4][3] = 0x61
	sourceCells[4][4] = 0x62
	sourceCells[4][5] = 0x68
	sourceCells[3][2] = 0xb0
	sourceCells[3][3] = 0xe0
	sourceCells[3][4] = 0xb0
	sourceCells[3][0] = 0x00
	sourceCells[3][7] = 0x00
	sourceCells[0][6] = 0x00
	sourceCells[7][6] = 0x00

	model := game_state.DungeonViewModel{
		Location: references.Deceit,
		Floor:    2,
		Facing:   references.DungeonFacingNorth,
	}
	for displayY := range game_state.DungeonViewSize {
		for displayX := range game_state.DungeonViewSize {
			sourceX := wrappedDungeonViewAxis(displayX - game_state.DungeonViewCenter + 3)
			sourceY := wrappedDungeonViewAxis(displayY - game_state.DungeonViewCenter + 3)
			model.Cells[displayY][displayX] = game_state.DungeonViewCell{
				Visible: displayX > 1 && displayX < game_state.DungeonViewSize-2 &&
					displayY > 1 && displayY < game_state.DungeonViewSize-2,
				Player: displayX == game_state.DungeonViewCenter && displayY == game_state.DungeonViewCenter,
				SourcePosition: references.Position{
					X: references.Coordinate(sourceX), Y: references.Coordinate(sourceY),
				},
				SourceCell: sourceCells[sourceY][sourceX],
			}
			if model.Cells[displayY][displayX].Visible {
				model.VisibleCount++
			}
		}
	}

	return model
}
