package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

func TestCombatAttackBracketsFullOutlinedPixelGeometry(t *testing.T) {
	bluePixel := func(x, y int) bool {
		return (x == 5 || x == 11) && y >= 9 && y <= 10 ||
			y == 10 && (x >= 5 && x <= 6 || x >= 10 && x <= 11)
	}
	assert.Equal(t, ucolor.EGALightBlue, ucolor.CombatAttackEligibleFace)
	assert.Equal(t, ucolor.EGABlack, ucolor.CombatAttackEligibleBorder)
	for _, scale := range []int{1, 2, 4} {
		t.Run(fmt.Sprint(scale), func(t *testing.T) {
			screen := ebiten.NewImage(24*scale, 24*scale)
			t.Cleanup(screen.Dispose)
			background := ucolor.EGALightGreen.ToRGBA()
			screen.Fill(background)
			options := &ebiten.DrawImageOptions{}
			options.GeoM.Scale(float64(scale), float64(scale))
			options.GeoM.Translate(float64(3*scale), float64(2*scale))
			drawCombatAttackBrackets(screen, options)
			pixels := make([]byte, 24*scale*24*scale*4)
			screen.ReadPixels(pixels)
			for y := range 24 * scale {
				for x := range 24 * scale {
					sx, sy := x/scale-3, y/scale-2
					want := background
					for dy := -1; dy <= 1; dy++ {
						for dx := -1; dx <= 1; dx++ {
							if bluePixel(sx+dx, sy+dy) {
								want = ucolor.EGABlack.ToRGBA()
							}
						}
					}
					if bluePixel(sx, sy) {
						want = ucolor.EGALightBlue.ToRGBA()
					}
					i := (y*24*scale + x) * 4
					got := color.RGBA{R: pixels[i], G: pixels[i+1], B: pixels[i+2], A: pixels[i+3]}
					require.Equal(t, want, got, "pixel (%d,%d), scale %d", sx, sy, scale)
				}
			}
		})
	}
}

func TestCombatAttackBracketsAimedTargetSuppressionAndExit(t *testing.T) {
	scene, _ := setupCombatAutoAttackInputTest(t)
	scene.mapImage = ebiten.NewImage(304, 208)
	screen := newPresentationRaster(1280, 720)
	t.Cleanup(screen.Dispose)
	t.Cleanup(scene.mapImage.Dispose)
	pixels := make([]byte, 1280*720*4)
	facePixels := func() int {
		screen.Fill(ucolor.EGABlack)
		scene.drawCombatAttackEligibility(screen)
		screen.ReadPixels(pixels)
		face := ucolor.CombatAttackEligibleFace.ToRGBA()
		count := 0
		for i := 0; i < len(pixels); i += 4 {
			if pixels[i] == face.R && pixels[i+1] == face.G && pixels[i+2] == face.B && pixels[i+3] == face.A {
				count++
			}
		}
		return count
	}
	before := facePixels()
	require.Positive(t, before)
	scene.gameState.ElapsedMs += 500
	assert.Equal(t, before, facePixels(), "marker does not pulse with game time")
	require.True(t, scene.gameState.BeginCombatAttack())
	scene.secondaryKeyState = CombatAttackTargetInput
	scene.gameState.CombatScene.AttackTargeting.Cursor = scene.gameState.CombatScene.MonsterManager.GetMonster(0).Position()
	assert.Zero(t, facePixels(), "selected reticle owns the aimed target's square")
	scene.finishCombatAttackTargetAction(scene.gameState.CancelInitialCombatAttack())
	assert.Equal(t, before, facePixels(), "cancelling Aim restores normal any-source eligibility")
	scene.gameState.CombatScene = nil
	assert.Zero(t, facePixels(), "combat exit has no cached markers")
}

func TestCombatAttackPreviewPresentationRefreshAndCleanup(t *testing.T) {
	scene, actor := setupCombatAutoAttackInputTest(t)
	state := scene.gameState
	character := state.PartyState.GetCharacter(actor.EntityIndex)
	character.SetEquippedItem(references.SlotWeapon, references.Bow)
	character.SetEquippedItem(references.SlotShield, references.NoEquipment)
	state.PartyState.Inventory.Equipment.Set(references.Arrows, 2)
	rngCalls := state.GetRNG().Calls()
	preview := scene.combatAttackPreview()
	require.Len(t, preview.Targets, 1)
	assert.Nil(t, preview.Aim)
	require.True(t, state.BeginCombatAttack())
	scene.secondaryKeyState = CombatAttackTargetInput
	state.CombatScene.AttackTargeting.Cursor = preview.Targets[0].Position
	assert.NotNil(t, scene.combatAttackPreview().Aim)
	state.PartyState.Inventory.Equipment.Set(references.Arrows, 0)
	assert.Empty(t, scene.combatAttackPreview().Targets)
	assert.Equal(t, game_state.CombatShotNoAmmo, scene.combatAttackPreview().Aim.Status)
	scene.finishCombatAttackTargetAction(state.CancelInitialCombatAttack())
	assert.Nil(t, scene.combatAttackPreview().Aim)
	state.PartyState.Inventory.Equipment.Set(references.Arrows, 2)
	assert.Len(t, scene.combatAttackPreview().Targets, 1)
	scene.inputBlocked = true
	assert.Empty(t, scene.combatAttackPreview())
	scene.inputBlocked = false
	scene.secondaryKeyState = CombatSpellTargetInput
	assert.Empty(t, scene.combatAttackPreview())
	scene.secondaryKeyState = PrimaryInput
	state.CombatScene.MonsterManager.GetMonster(0).SetInvisible(true)
	assert.Empty(t, scene.combatAttackPreview().Targets)
	state.CombatScene = nil
	assert.Empty(t, scene.combatAttackPreview())
	assert.Equal(t, rngCalls, state.GetRNG().Calls())
}

func TestCombatAttackReadoutNamesRangeAndUnavailableSquare(t *testing.T) {
	output := text.NewOutput(nil, 1, combatHUDOutputLines, 80)
	appendCombatAttackReadout(output, game_state.CombatAttackPreview{
		Sources: []game_state.CombatAttackSource{
			{Equipment: references.SpikedHelm, MaxRange: 1},
			{Equipment: references.Bow, MaxRange: 7},
			{Equipment: references.MainGauche, MaxRange: 1},
		},
	})
	readout := output.GetOutputStr(false)
	assert.Contains(t, readout, "Spiked Helm (R1)")
	assert.Contains(t, readout, "Bow (R7)")
	assert.Contains(t, readout, "Main Gauche (R1)")
	output.Reset()
	appendCombatAttackReadout(output, game_state.CombatAttackPreview{
		Sources: []game_state.CombatAttackSource{{Equipment: references.NoEquipment, MaxRange: 1}},
		Aim:     &game_state.CombatAttackTargetPreview{Distance: 3, Status: game_state.CombatShotOutOfRange},
	})
	readout = output.GetOutputStr(false)
	assert.Contains(t, readout, "Bare Hands (R1)")
	assert.Contains(t, readout, "No one seen (D3)")
	assert.Contains(t, readout, "Attack: Too far")
	assert.NotContains(t, readout, "Orc")
}

func TestCombatAttackReadoutRetainsCompleteLabelsAtSupportedResolutions(t *testing.T) {
	statuses := []game_state.CombatShotStatus{
		game_state.CombatShotClear,
		game_state.CombatShotTerrainBypass,
		game_state.CombatShotOutOfRange,
		game_state.CombatShotBlocked,
		game_state.CombatShotNoAmmo,
		game_state.CombatShotInterference,
		game_state.CombatShotUnready,
		game_state.CombatShotChooseTarget,
		game_state.CombatShotFriendly,
	}
	for _, resolution := range []image.Point{{1280, 720}, {1366, 768}, {1920, 1080}, {2560, 1440}, {3840, 2160}} {
		t.Run(fmt.Sprintf("%dx%d", resolution.X, resolution.Y), func(t *testing.T) {
			screen := image.Rectangle{Max: resolution}
			turns, _ := combatHUDStackedRectangles(screen, combatHUDRectangle(screen))
			bounds := combatHUDContentBounds(screen, turns)
			scale := float64(resolution.X) / 1920
			font := text.NewUltimaFont(float64(combatHUDFontSize) * scale)
			output := text.NewOutput(font, float64(combatHUDFontSize)*scale, combatHUDOutputLines, 80)

			equipment := []references.Equipment{references.NoEquipment}
			for item := references.LeatherHelm; item <= references.MysticSword; item++ {
				if item.GetAttackValue() > 0 {
					equipment = append(equipment, item)
				}
			}
			for _, item := range equipment {
				name := item.String()
				if item == references.NoEquipment {
					name = "Bare Hands"
				}
				height := prepareCombatAttackReadout(output, game_state.CombatAttackPreview{
					Sources: []game_state.CombatAttackSource{{Equipment: item, MaxRange: 15}},
				}, bounds.Dx())
				assert.Equal(t, fmt.Sprintf("%s (R15)", name), strings.Join(strings.Fields(output.GetOutputStr(false)), " "))
				assertCombatAttackReadoutFits(t, output, bounds, height)
				if measuredTextWidth(font, fmt.Sprintf("%s (R15)", name)) > float64(bounds.Dx()) {
					assert.Greater(t, output.GetPanelContentLineCount(), 1, "%s needs complete wrapped rows", name)
				}
			}

			for monster := references.MonsterType(0); monster < references.TotalMonsterTypes; monster++ {
				aimText := fmt.Sprintf("%s (D15)", monster.Name())
				height := prepareCombatAttackReadout(output, game_state.CombatAttackPreview{
					Sources: []game_state.CombatAttackSource{{Equipment: references.MagicBow, MaxRange: 15}},
					Aim: &game_state.CombatAttackTargetPreview{
						Name: monster.Name(), Distance: 15, Status: game_state.CombatShotTerrainBypass,
					},
				}, bounds.Dx())
				assert.Equal(t, "Magic Bow (R15) "+aimText+" Attack: Bypass",
					strings.Join(strings.Fields(output.GetOutputStr(false)), " "))
				assertCombatAttackReadoutFits(t, output, bounds, height)
			}

			for _, status := range statuses {
				height := prepareCombatAttackReadout(output, game_state.CombatAttackPreview{
					Aim: &game_state.CombatAttackTargetPreview{Status: status},
				}, bounds.Dx())
				target := "No one seen"
				if status == game_state.CombatShotChooseTarget {
					target = "Choose target"
				}
				assert.Equal(t, target+" (D0) Attack: "+combatShotStatusText(status),
					strings.Join(strings.Fields(output.GetOutputStr(false)), " "))
				assertCombatAttackReadoutFits(t, output, bounds, height)
			}
			assert.Same(t, font, output.Font)
			assert.Equal(t, float64(combatHUDFontSize)*scale, output.Font.TextFace.Size)
		})
	}
	assert.Equal(t, "Bypass", combatShotStatusText(game_state.CombatShotTerrainBypass))
	assert.Equal(t, "Bypasses terrain on hit", string(game_state.CombatShotTerrainBypass),
		"presentation wording must not change the authoritative status")
}

func assertCombatAttackReadoutFits(t *testing.T, output *text.Output, bounds image.Rectangle, height int) {
	t.Helper()
	for _, row := range strings.Split(output.GetOutputStr(false), "\n") {
		assert.LessOrEqual(t, measuredTextWidth(output.Font, strings.TrimSpace(row)), float64(bounds.Dx()), row)
	}
	assert.GreaterOrEqual(t, height, int(math.Ceil(output.GetTotalHeightForLines(output.GetPanelContentLineCount()))))
	assert.LessOrEqual(t, height, bounds.Dy())
}

func TestCombatAttackHUDMeasuredReadoutFitsSupportedResolutions(t *testing.T) {
	for _, resolution := range []image.Point{{1280, 720}, {1366, 768}, {1920, 1080}, {2560, 1440}, {3840, 2160}} {
		t.Run(fmt.Sprintf("%dx%d", resolution.X, resolution.Y), func(t *testing.T) {
			screen := image.Rectangle{Max: resolution}
			turns, events := combatHUDStackedRectangles(screen, combatHUDRectangle(screen))
			bounds := combatHUDContentBounds(screen, turns)
			scale := float64(resolution.X) / 1920
			font := text.NewUltimaFont(float64(combatHUDFontSize) * scale)
			output := text.NewOutput(font, float64(combatHUDFontSize)*scale, combatHUDOutputLines, 80)
			lineHeight := int(math.Ceil(output.GetLineSpacing()))
			gap := lineHeight / combatHUDLineGapDivisor
			spriteHeight := int(math.Round(18 * float64(resolution.Y) / 208))
			spriteWidth := int(math.Round(18 * float64(resolution.X) * 0.76 / 304))
			output.SetMaxCharsPerLine(max(1, font.CharsForWidth(bounds.Dx()-spriteWidth-gap)))
			output.AppendLine().White("BLACKTHORN CLONE").Done()
			output.AppendLine().LightCyan("Enemy / Possessed").Done()
			output.AppendLine().LightGray("D30 C06").Done()
			actorHeight := max(spriteHeight, int(math.Ceil(output.GetTotalHeightForLines(output.GetPanelContentLineCount()))))
			for _, preview := range []game_state.CombatAttackPreview{
				{Sources: []game_state.CombatAttackSource{
					{Equipment: references.SpikedHelm, MaxRange: 1},
					{Equipment: references.MorningStar, MaxRange: 2},
					{Equipment: references.SpikedShield, MaxRange: 1},
				}},
				{Sources: []game_state.CombatAttackSource{
					{Equipment: references.SpikedHelm, MaxRange: 1},
					{Equipment: references.TwoHHammer, MaxRange: 1},
				}},
				{Sources: []game_state.CombatAttackSource{{Equipment: references.TwoHHammer, MaxRange: 1}},
					Aim: &game_state.CombatAttackTargetPreview{Name: "Lord British", Distance: 15, Status: game_state.CombatShotOutOfRange}},
				{Sources: []game_state.CombatAttackSource{{Equipment: references.MagicBow, MaxRange: 15}},
					Aim: &game_state.CombatAttackTargetPreview{Name: "Lord British", Distance: 15, Status: game_state.CombatShotTerrainBypass}},
			} {
				readoutHeight := prepareCombatAttackReadout(output, preview, bounds.Dx())
				readout := strings.Join(strings.Fields(output.GetOutputStr(false)), " ")
				for _, source := range preview.Sources {
					assert.Contains(t, readout, fmt.Sprintf("%s (R%d)", source.Equipment, source.MaxRange))
				}
				assertCombatAttackReadoutFits(t, output, bounds, readoutHeight)
				requiredHeight := lineHeight + gap + actorHeight + gap + readoutHeight + 2*lineHeight + spriteHeight + int(math.Ceil(output.GetTotalHeightForLines(1)))
				assert.LessOrEqual(t, requiredHeight, bounds.Dy(), "full actor, wrapped readout and NEXT must fit: %s", readout)
				assert.GreaterOrEqual(t, output.GetLineCapacity(combatHUDContentBounds(screen, events).Dy()), 10, "keep a useful event tape")
			}
		})
	}
}

func measuredTextWidth(font *text.UltimaFont, value string) float64 {
	width, _ := font.MeasureString(value)

	return width
}

func TestCombatCurrentNameFullAndNextAbbreviated(t *testing.T) {
	scene, actor := setupCombatAutoAttackInputTest(t)
	character := scene.gameState.PartyState.GetCharacter(actor.EntityIndex)
	clear(character.Name[:])
	copy(character.Name[:], "Shamino")
	scene.spriteSheet = &sprites.SpriteSheet{SpriteImage: ebiten.NewImage(512, 256)}
	scene.mapImage = ebiten.NewImage(304, 208)
	scene.combatHUD = newCombatHUD()
	font := text.NewUltimaFont(float64(combatHUDFontSize) * 1280 / 1920)
	scene.combatHUD.turnOutput = text.NewOutput(font, float64(combatHUDFontSize)*1280/1920, combatHUDOutputLines, 80)
	screen := newPresentationRaster(1280, 720)
	t.Cleanup(screen.Dispose)
	for _, clone := range []bool{false, true} {
		identity := actor
		want := "SHAMINO"
		if clone {
			scene.gameState.CombatScene.CurrentTurnIdentity = actor
			require.True(t, scene.gameState.CastCombatInQuasXen(actor.EntityIndex, actor))
			scene.gameState.CombatScene.ForEachPCClone(func(exact game_state.CombatTurnIdentity, _ references.Position) { identity = exact })
			want += " CLONE"
		}
		scene.gameState.CombatScene.CurrentTurnIdentity = identity
		for _, detail := range []bool{false, true} {
			scene.combatHUD.detail = detail
			for _, autoPass := range []bool{false, true} {
				scene.gameState.CombatScene.SoloMoveMode = autoPass
				scene.gameState.CombatScene.SoloMoveCharacterIndex = (actor.EntityIndex + 1) % 6
				lineHeight := int(math.Ceil(scene.combatHUD.turnOutput.GetLineSpacing()))
				bottom := scene.drawCombatCurrentTurn(screen, image.Rect(622, 22, 947, 338), lineHeight, lineHeight/3)
				content := scene.combatHUD.turnOutput.GetOutputStr(false)
				assert.Contains(t, strings.ReplaceAll(content, "\n", " "), want)
				assert.GreaterOrEqual(t, bottom, 22+lineHeight+lineHeight/3+int(math.Ceil(scene.combatHUD.turnOutput.GetTotalHeightForLines(scene.combatHUD.turnOutput.GetPanelContentLineCount()))))
				scene.drawCombatQueueLabel(screen, image.Rect(622, 200, 704, 230), identity, autoPass)
				wantNext := "SHAM"
				if clone {
					wantNext = "CLN"
				}
				assert.Equal(t, wantNext, strings.TrimSpace(scene.combatHUD.turnOutput.GetOutputStr(false)))
			}
		}
	}
}
