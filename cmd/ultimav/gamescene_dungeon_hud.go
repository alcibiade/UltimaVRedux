package main

import (
	"fmt"
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

const (
	dungeonStatusMaxChars          = 16
	dungeonDebugMaxChars           = 64
	dungeonHUDRightRailStart       = 0.75
	dungeonHUDBorderHeightPercent  = 0.022
	dungeonHUDHorizontalInset      = 0.01
	dungeonHUDVerticalInset        = 0.01
	dungeonHUDDebugWidthProportion = 0.5
	dungeonDirectionNorth          = "North"
	dungeonDirectionEast           = "East"
	dungeonDirectionSouth          = "South"
	dungeonDirectionWest           = "West"
	// The debug coordinate overlay is not player-facing, so it stays small.
	dungeonHUDDebugFontSize widgets.StandardFontSize = 12
)

type dungeonHUDOutputs struct {
	level          *text.Output
	direction      *text.Output
	debug          *text.Output
	levelLabel     dungeonBorderLabel
	directionLabel dungeonBorderLabel
}

type dungeonHUDStatus struct {
	position references.Position
	floor    references.FloorNumber
	facing   references.DungeonFacing
}

type dungeonHUDLayout struct {
	frameBounds      image.Rectangle
	safePaneBounds   image.Rectangle
	borderBandHeight int
	levelPoint       image.Point
	directionPoint   image.Point
	debugBounds      image.Rectangle
}

func (g *GameScene) initializeDungeonHUDOutputs() {
	g.releaseDungeonHUD()
	_, g.dungeonHUD.level = widgets.GetScaledFontWithOutput(widgets.FontConfig{
		Size:     widgets.FontSizeHUD,
		MaxLines: 1,
		MaxChars: dungeonStatusMaxChars,
	})
	_, g.dungeonHUD.direction = widgets.GetScaledFontWithOutput(widgets.FontConfig{
		Size:     widgets.FontSizeHUD,
		MaxLines: 1,
		MaxChars: dungeonStatusMaxChars,
	})
	_, g.dungeonHUD.debug = widgets.GetScaledFontWithOutput(widgets.FontConfig{
		Size:     dungeonHUDDebugFontSize,
		MaxLines: 1,
		MaxChars: dungeonDebugMaxChars,
	})
}

// Called on resize before replacing fonts, and when the scene is disposed.
func (g *GameScene) releaseDungeonHUD() {
	g.dungeonHUD.levelLabel.release()
	g.dungeonHUD.directionLabel.release()
	g.dungeonHUD = dungeonHUDOutputs{}
}

func newDungeonHUDStatus(
	player references.PlayerLocation,
	facing references.DungeonFacing,
) dungeonHUDStatus {
	return dungeonHUDStatus{
		position: player.Position,
		floor:    player.Floor,
		facing:   facing,
	}
}

func (s dungeonHUDStatus) levelText() string {
	return fmt.Sprintf("L %d", int(s.floor)+1)
}

func (s dungeonHUDStatus) directionText() string {
	return "Dir: " + dungeonFacingCardinal(s.facing)
}

func (s dungeonHUDStatus) debugText() string {
	return fmt.Sprintf(
		"X: %d Y: %d Floor: %d Level: %d Facing: %s",
		s.position.X,
		s.position.Y,
		s.floor,
		int(s.floor)+1,
		dungeonFacingCardinal(s.facing),
	)
}

func dungeonFacingCardinal(facing references.DungeonFacing) string {
	switch facing {
	case references.DungeonFacingNorth:
		return dungeonDirectionNorth
	case references.DungeonFacingEast:
		return dungeonDirectionEast
	case references.DungeonFacingSouth:
		return dungeonDirectionSouth
	case references.DungeonFacingWest:
		return dungeonDirectionWest
	default:
		panic("invalid dungeon facing")
	}
}

func calculateDungeonHUDLayout(
	screenBounds image.Rectangle,
	levelHeight, directionHeight, debugHeight int,
) dungeonHUDLayout {
	frameRight := screenBounds.Min.X + int(math.Round(
		float64(screenBounds.Dx())*gameScreenPercents.EndPercentX,
	))
	safeRight := screenBounds.Min.X + int(math.Floor(
		float64(screenBounds.Dx())*dungeonHUDRightRailStart,
	))
	frameBounds := image.Rect(screenBounds.Min.X, screenBounds.Min.Y, frameRight, screenBounds.Max.Y)
	safePaneBounds := image.Rect(screenBounds.Min.X, screenBounds.Min.Y, safeRight, screenBounds.Max.Y)
	borderBandHeight := max(
		int(math.Round(float64(screenBounds.Dy())*dungeonHUDBorderHeightPercent)),
		levelHeight,
		directionHeight,
	)

	horizontalInset := max(1, int(math.Round(float64(screenBounds.Dx())*dungeonHUDHorizontalInset)))
	verticalInset := max(1, int(math.Round(float64(screenBounds.Dy())*dungeonHUDVerticalInset)))
	debugRight := safePaneBounds.Max.X - horizontalInset
	debugWidth := int(math.Round(float64(safePaneBounds.Dx()) * dungeonHUDDebugWidthProportion))
	debugLeft := max(safePaneBounds.Min.X+horizontalInset, debugRight-debugWidth)
	debugTop := min(
		safePaneBounds.Max.Y-debugHeight,
		safePaneBounds.Min.Y+borderBandHeight+verticalInset,
	)

	return dungeonHUDLayout{
		frameBounds:      frameBounds,
		safePaneBounds:   safePaneBounds,
		borderBandHeight: borderBandHeight,
		levelPoint: image.Pt(
			frameBounds.Min.X,
			frameBounds.Min.Y+(borderBandHeight-levelHeight)/2,
		),
		directionPoint: image.Pt(
			frameBounds.Min.X,
			frameBounds.Max.Y-borderBandHeight+(borderBandHeight-directionHeight)/2,
		),
		debugBounds: image.Rect(debugLeft, debugTop, debugRight, debugTop+debugHeight),
	}
}

func (g *GameScene) drawDungeonHUD(screen *ebiten.Image, status dungeonHUDStatus) {
	if g.dungeonHUD.level == nil || g.dungeonHUD.direction == nil || g.dungeonHUD.debug == nil {
		panic("dungeon HUD outputs are not initialized")
	}

	levelHeight := int(math.Ceil(g.dungeonHUD.level.GetTotalHeightForLines(1)))
	directionHeight := int(math.Ceil(g.dungeonHUD.direction.GetTotalHeightForLines(1)))
	debugHeight := int(math.Ceil(g.dungeonHUD.debug.GetTotalHeightForLines(1)))
	layout := calculateDungeonHUDLayout(screen.Bounds(), levelHeight, directionHeight, debugHeight)

	g.drawDungeonLevelLabel(screen, status)
	g.drawDungeonDirectionLabel(screen, status)

	g.dungeonHUD.debug.Reset()
	if !g.showRendererDiagnostics() {
		return
	}
	g.dungeonHUD.debug.AppendLine().White(status.debugText()).Done()
	g.dungeonHUD.debug.DrawPanelContent(
		screen,
		layout.debugBounds.Min,
		layout.debugBounds.Dx(),
		etext.AlignEnd,
	)
}

func (g *GameScene) showRendererDiagnostics() bool {
	options := g.gameState.DebugOptions

	return options.LightOverlay || options.SearchOverlay || options.DialogBoundingBoxes
}
