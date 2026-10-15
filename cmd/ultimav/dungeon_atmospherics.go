//nolint:mnd // Coordinates, character cells, phase counts, and random range are verified original presentation facts.
package main

import (
	"image"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	gametext "github.com/bradhannah/UltimaVRedux/internal/text"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

const (
	dungeonAtmosphereFrameMs       = 100
	dungeonGraffitiCharacterSize   = 8
	dungeonGraffitiMaxLines        = 3
	dungeonGraffitiMaxChars        = 10
	dungeonStalactiteVisiblePhases = 5
	dungeonStalactitePhaseCount    = 6
	dungeonEyeRandomRange          = 65
	dungeonEyeVisibleSamples       = 4
	dungeonEyeSampleStride         = 37
)

type dungeonGraffitiSign struct {
	message string
	column  int
	row     int
}

//nolint:gochecknoglobals // ref:DNG-0009: immutable dungeon graffiti layout table.
var dungeonGraffitiSigns = [...]dungeonGraffitiSign{
	{message: "BOTTOMLESS\n   PIT    ", column: 7, row: 10},
	{message: "[E MAZE\n OF LO_\n SOULS ", column: 8, row: 9},
	{message: "[E PRISON\n   WRO]  ", column: 8, row: 10},
	{message: "[E CRYPT", column: 8, row: 11},
	{message: "UPPER \nCRYPTS", column: 9, row: 10},
	{message: "LOWER \nCRYPTS", column: 9, row: 10},
	{message: "DEBTERS\n ALLY  ", column: 9, row: 10},
	{message: "D\\P", column: 10, row: 11},
	{message: "D\\PER", column: 9, row: 11},
	{message: "D\\PE_", column: 9, row: 11},
	{message: "MO[ER LODE\n   MAZE   ", column: 7, row: 10},
}

//nolint:gochecknoglobals // Indexed by original dungeon IDs 0x21..0x28.
var (
	dungeonFirstGraffiti = [...]byte{0, 1, 2, 2, 3, 7, 10, 11}
	dungeonGraffitiCount = [...]byte{1, 1, 0, 1, 4, 3, 1, 0}
)

func newDungeonGraffitiOutput(font *gametext.UltimaFont) *gametext.Output {
	if font == nil {
		panic("dungeon graffiti output requires a font")
	}

	return gametext.NewOutput(
		font,
		dungeonGraffitiCharacterSize,
		dungeonGraffitiMaxLines,
		dungeonGraffitiMaxChars,
	)
}

func appendDungeonCenterAtmospherics(
	plan *dungeonRenderPlan,
	cell references.DungeonCell,
	position references.Position,
	depth int,
) {
	if !plan.trueLight {
		return
	}
	appendDungeonCenterStalactite(plan, cell, position, depth)
	if depth == 1 {
		appendDungeonDepthOneWallAtmospherics(plan, cell, position)
	}
}

func appendDungeonCenterStalactite(
	plan *dungeonRenderPlan,
	cell references.DungeonCell,
	position references.Position,
	depth int,
) {
	if dungeonHasStalactites(plan.location) && cell.Kind() == references.DungeonSpecialWall &&
		(depth == 1 || depth == 2) {
		appendDungeonStalactite(plan, cell, position, depth, dungeonRenderCenter, false)
	}
}

func appendDungeonDepthOneWallAtmospherics(
	plan *dungeonRenderPlan,
	cell references.DungeonCell,
	position references.Position,
) {
	if cell.Kind() == references.DungeonWall {
		if signIndex, ok := dungeonGraffitiSignIndex(plan.location, cell.Payload()); ok {
			sign := dungeonGraffitiSigns[signIndex]
			plan.append(dungeonRenderOp{
				kind: dungeonRenderGraffiti, role: dungeonRenderCenter, position: position, depth: 1,
				spriteIndex: signIndex,
				x:           sign.column*dungeonGraffitiCharacterSize - 16,
				y:           sign.row*dungeonGraffitiCharacterSize - 14,
				mirror:      false, top: false, fieldColor: 0, phase: 0,
				dissolve: nil, fieldTimeMs: 0,
			})
		}

		return
	}
	if dungeonHasSkeletonEyes(plan.location) && cell.Kind() == references.DungeonSpecialWall &&
		dungeonEyeSample(*plan, cell, position) < dungeonEyeVisibleSamples {
		plan.append(dungeonRenderOp{
			kind: dungeonRenderSkeletonEyes, role: dungeonRenderCenter, position: position, depth: 1,
			spriteIndex: 0, x: 75, y: 73, mirror: false, top: false, fieldColor: 0, phase: 0,
			dissolve: nil, fieldTimeMs: 0,
		})
	}
}

func appendDungeonSideAtmospherics(
	plan *dungeonRenderPlan,
	cell references.DungeonCell,
	position references.Position,
	depth int,
	right bool,
) {
	if !plan.trueLight || !dungeonHasStalactites(plan.location) ||
		cell.Kind() != references.DungeonSpecialWall || depth > 1 {
		return
	}
	role := dungeonRenderLeft
	if right {
		role = dungeonRenderRight
	}
	appendDungeonStalactite(plan, cell, position, depth, role, right)
}

func appendDungeonStalactite(
	plan *dungeonRenderPlan,
	cell references.DungeonCell,
	position references.Position,
	depth int,
	role dungeonRenderRole,
	right bool,
) {
	phase := dungeonStalactitePhase(*plan, cell, position)
	xPosition, yPosition := 0, -1
	if phase < dungeonStalactiteVisiblePhases {
		xPosition, yPosition = dungeonStalactitePosition(role, depth, right, int(phase))
	}
	plan.append(dungeonRenderOp{
		kind: dungeonRenderStalactite, role: role, position: position, depth: depth,
		spriteIndex: 0, x: xPosition, y: yPosition, mirror: right, top: false, fieldColor: 0, phase: phase,
		dissolve: nil, fieldTimeMs: 0,
	})
}

func dungeonGraffitiSignIndex(location references.Location, payload byte) (int, bool) {
	locationIndex := int(location - references.Deceit)
	if locationIndex < 0 || locationIndex >= len(dungeonFirstGraffiti) ||
		payload == 0 || payload > dungeonGraffitiCount[locationIndex] {
		return 0, false
	}

	return int(dungeonFirstGraffiti[locationIndex] + payload - 1), true
}

func dungeonHasSkeletonEyes(location references.Location) bool {
	return location == references.Deceit || location == references.Wrong || location == references.Covetous
}

func dungeonHasStalactites(location references.Location) bool {
	return location == references.Despise || location == references.Destard || location == references.Doom
}

func dungeonAtmosphereCellHash(
	plan dungeonRenderPlan,
	cell references.DungeonCell,
	position references.Position,
) uint64 {
	floor := uint64(uint8(plan.floor))      //nolint:gosec // Dungeon render validation bounds floors to 0..7.
	positionX := uint64(uint16(position.X)) //nolint:gosec // Sampled dungeon coordinates are wrapped to 0..7.
	positionY := uint64(uint16(position.Y)) //nolint:gosec // Sampled dungeon coordinates are wrapped to 0..7.
	value := uint64(plan.location) |
		floor<<8 |
		positionX<<16 |
		positionY<<24 |
		uint64(cell.Raw())<<32

	return presentationMix64(value)
}

func presentationMix64(value uint64) uint64 {
	value += 0x9e3779b97f4a7c15
	value = (value ^ value>>30) * 0xbf58476d1ce4e5b9
	value = (value ^ value>>27) * 0x94d049bb133111eb

	return value ^ value>>31
}

func dungeonEyeSample(
	plan dungeonRenderPlan,
	cell references.DungeonCell,
	position references.Position,
) uint64 {
	block := plan.atmosphereFrame / dungeonEyeRandomRange
	frameInBlock := plan.atmosphereFrame % dungeonEyeRandomRange
	offset := presentationMix64(dungeonAtmosphereCellHash(plan, cell, position)^block) % dungeonEyeRandomRange

	// REDUX DEVIATION: DNG-0009: original wall-panel redraw samples shared RNG with inclusive
	// draws from 0..64, accepting values below 4. Redux permutes the same 65 outcomes per timed block so the cosmetic
	// 4/65 cadence is deterministic and never advances gameplay RNG.
	return (frameInBlock*dungeonEyeSampleStride + offset) % dungeonEyeRandomRange
}

func dungeonStalactitePhase(
	plan dungeonRenderPlan,
	cell references.DungeonCell,
	position references.Position,
) byte {
	// REDUX DEVIATION: DNG-0009: original stalactite animation advances the packed cell's low bits on
	// redraw, with phase 5 reading past drip_coord before resetting. Redux cycles an
	// immutable six-phase cosmetic value from elapsed time and cell identity; phase 5
	// is intentionally hidden, avoiding authoritative mutation and invalid indexing.
	offset := dungeonAtmosphereCellHash(plan, cell, position) % dungeonStalactitePhaseCount

	return byte((plan.atmosphereFrame + offset) % dungeonStalactitePhaseCount)
}

func dungeonStalactitePosition(
	role dungeonRenderRole,
	depth int,
	right bool,
	phase int,
) (int, int) {
	if phase < 0 || phase >= dungeonStalactiteVisiblePhases {
		panic("invalid visible dungeon stalactite phase")
	}

	if role == dungeonRenderCenter {
		coordinates := [2][dungeonStalactiteVisiblePhases]int{
			{40, 47, 66, 100, 146},
			{46, 50, 62, 82, 109},
		}
		if depth < 1 || depth > 2 {
			panic("invalid center dungeon stalactite depth")
		}

		return 79, coordinates[depth-1][phase]
	}
	if role != dungeonRenderLeft && role != dungeonRenderRight {
		panic("invalid dungeon stalactite role")
	}

	return dungeonSideStalactitePosition(depth, right, phase)
}

func dungeonSideStalactitePosition(depth int, right bool, phase int) (int, int) {
	coordinates := [2][dungeonStalactiteVisiblePhases]int{
		{14, 23, 50, 98, 159},
		{40, 45, 60, 84, 119},
	}
	if depth < 0 || depth > 1 {
		panic("invalid side dungeon stalactite depth")
	}
	xPosition := [2]int{17, 51}[depth]
	if right {
		xPosition = [2]int{141, 107}[depth]
	}

	return xPosition, coordinates[depth][phase]
}

func drawDungeonGraffiti(canvas *ebiten.Image, output *gametext.Output, command dungeonRenderOp) {
	if command.spriteIndex < 0 || command.spriteIndex >= len(dungeonGraffitiSigns) {
		panic("invalid dungeon graffiti sign")
	}
	lines := strings.Split(dungeonGraffitiSigns[command.spriteIndex].message, "\n")
	output.Reset()
	for lineIndex, line := range lines {
		vector.FillRect(
			canvas,
			float32(command.x),
			float32(command.y+lineIndex*dungeonGraffitiCharacterSize),
			float32(len(line)*dungeonGraffitiCharacterSize),
			dungeonGraffitiCharacterSize,
			ucolor.EGAWhite,
			false,
		)
		output.AppendLine().RunicWithColor(line, ucolor.EGABlack).Done()
	}
	output.DrawPanelContent(
		canvas,
		image.Pt(command.x, command.y),
		dungeonGraffitiMaxChars*dungeonGraffitiCharacterSize,
		etext.AlignStart,
	)
}

func drawDungeonSkeletonEyes(canvas *ebiten.Image, command dungeonRenderOp) {
	color := ucolor.EGALightRed
	vector.FillRect(canvas, float32(command.x+1), float32(command.y), 2, 1, color, false)
	vector.FillRect(canvas, float32(command.x), float32(command.y+1), 3, 1, color, false)
	vector.FillRect(canvas, float32(command.x+6), float32(command.y), 2, 1, color, false)
	vector.FillRect(canvas, float32(command.x+6), float32(command.y+1), 3, 1, color, false)
}

func drawDungeonStalactite(canvas *ebiten.Image, command dungeonRenderOp) {
	if command.phase >= dungeonStalactiteVisiblePhases {
		return
	}
	baseColor := ucolor.EGABlue
	if command.phase == dungeonStalactiteVisiblePhases-1 {
		baseColor = ucolor.EGALightCyan
	}
	vector.FillRect(canvas, float32(command.x-1), float32(command.y), 3, 1, baseColor, false)
	vector.FillRect(canvas, float32(command.x), float32(command.y-1), 1, 3, baseColor, false)
	if command.phase < dungeonStalactiteVisiblePhases-1 {
		canvas.Set(command.x, command.y, ucolor.EGALightBlue)
	}
}
