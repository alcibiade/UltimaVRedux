package main

import (
	"bytes"
	"fmt"
	"image"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

func TestCombatAttackBracketsProductionActorPass(t *testing.T) {
	manager := display.GetManager()
	saved := *manager
	t.Cleanup(func() { *manager = saved })
	manager.PublishLogicalSize(1280, 720)
	scene, actor := setupCombatAutoAttackInputTest(t)
	state := scene.gameState
	scene.gameConfig = state.GameConfig
	scene.mapImage = ebiten.NewImage(304, 208)
	t.Cleanup(scene.mapImage.Dispose)
	scene.spriteSheet = sharedPresentationTestSpriteSheet()
	state.CombatScene.MonsterManager.Clear()
	state.CombatScene.TurnQueue.RemoveAllMonsters()
	state.CombatScene.SetPartyMemberPosition(actor.EntityIndex, references.Position{X: 1, Y: 1})
	character := state.PartyState.GetCharacter(actor.EntityIndex)
	character.SetEquippedItem(references.SlotWeapon, references.MagicAxe)
	character.SetEquippedItem(references.SlotShield, references.NoEquipment)
	monsterRef := (*state.GameReferences.EnemyReferences)[references.MonsterOrc]
	for index := range 9 {
		position := references.Position{X: references.Coordinate(index%3 + 3), Y: references.Coordinate(index/3 + 3)}
		monster := game_state.NewCombatMonster(references.MonsterOrc, &monsterRef, position, state.AsRandomProvider())
		state.CombatScene.AddMonsterToQueue(monster)
	}
	preview := scene.combatAttackPreview()
	require.Len(t, preview.Targets, 9)
	var sample image.Rectangle
	for _, target := range preview.Targets {
		sample = sample.Union(combatFixtureRasterBounds(scene.combatMonsterDrawOptions(target.Position, target.MonsterIndex)))
	}
	require.LessOrEqual(t, sample.Dx()*sample.Dy(), 256*256, "nine nearby actors must use a local raster target: %v", sample)
	screen := ebiten.NewImageWithOptions(sample, &ebiten.NewImageOptions{Unmanaged: true})
	reference := ebiten.NewImageWithOptions(sample, &ebiten.NewImageOptions{Unmanaged: true})
	t.Cleanup(screen.Dispose)
	t.Cleanup(reference.Dispose)
	background := ucolor.EGALightCyan.ToRGBA()
	screen.Fill(background)
	// This calls the real outer composition, guarding against moving the
	// eligibility pass back beneath a later actor draw in drawAvatarOverlay.
	scene.drawAvatarOverlay(screen)
	actual := make([]byte, sample.Dx()*sample.Dy()*4)
	wanted := make([]byte, len(actual))
	screen.ReadPixels(actual)
	for _, target := range preview.Targets {
		reference.Fill(background)
		drawCombatAttackBrackets(reference, scene.combatMonsterDrawOptions(target.Position, target.MonsterIndex))
		reference.ReadPixels(wanted)
		markedPixels := 0
		for i := 0; i < len(wanted); i += 4 {
			if wanted[i] == background.R && wanted[i+1] == background.G && wanted[i+2] == background.B {
				continue
			}
			markedPixels++
			if !bytes.Equal(wanted[i:i+4], actual[i:i+4]) {
				t.Fatalf("production actor pass erased target %d marker at (%d,%d): want %v got %v", target.MonsterIndex, sample.Min.X+i/4%sample.Dx(), sample.Min.Y+i/4/sample.Dx(), wanted[i:i+4], actual[i:i+4])
			}
		}
		require.Positive(t, markedPixels)
	}
}

func TestCombatAttackBracketsDenseComposition(t *testing.T) {
	sheet := sharedPresentationTestSpriteSheet()
	for _, face := range combatAttackBracketFaces() {
		assert.True(t, face.Inset(-1).In(image.Rect(1, 1, 17, 17)), "full backing must fit the 16-pixel cell")
		assert.False(t, face.Inset(-1).Overlaps(combatHealthFrameBounds()), "reserve the entire exterior health frame, not just its colored fill")
	}
	for scaleIndex, scale := range []struct{ x, y float64 }{{1, 1}, {1280 * 0.76 / 304, 720.0 / 208}, {1366 * 0.76 / 304, 768.0 / 208}} {
		for originIndex, origin := range []struct{ x, y float64 }{{30.35, 30.65}, {30.65, 30.35}, {30.5, 30.5}} {
			for layoutIndex, layout := range []string{"all marked", "center only", "checkerboard"} {
				for _, aimCenter := range []bool{false, true} {
					// Check registered geometry throughout the full 54-case matrix.
					// Rasterize one origin per layout/aim role, distributing samples
					// over native and both fractional scales instead of their product.
					for _, face := range combatAttackBracketFaces() {
						backing := face.Inset(-1)
						left, top := origin.x+float64(backing.Min.X)*scale.x, origin.y+float64(backing.Min.Y)*scale.y
						right, bottom := origin.x+float64(backing.Max.X)*scale.x, origin.y+float64(backing.Max.Y)*scale.y
						assert.GreaterOrEqual(t, left, origin.x+scale.x)
						assert.GreaterOrEqual(t, top, origin.y+scale.y)
						assert.LessOrEqual(t, right, origin.x+17*scale.x)
						assert.LessOrEqual(t, bottom, origin.y+12*scale.y, "all scales reserve the exterior health frame")
					}
					rasterOrigin := layoutIndex
					if aimCenter {
						rasterOrigin = (rasterOrigin + 1) % 3
					}
					if scaleIndex != layoutIndex || originIndex != rasterOrigin {
						continue
					}
					t.Run(fmt.Sprintf("%.3fx%.3f/origin%.2f,%.2f/%s/aim%t", scale.x, scale.y, origin.x, origin.y, layout, aimCenter), func(t *testing.T) {
						const width = 256
						screen := ebiten.NewImage(width, width)
						protected := ebiten.NewImage(width, width)
						reference := ebiten.NewImage(width, width)
						t.Cleanup(screen.Dispose)
						t.Cleanup(protected.Dispose)
						t.Cleanup(reference.Dispose)
						background := ucolor.EGALightCyan.ToRGBA()
						faceColor := ucolor.CombatAttackEligibleFace.ToRGBA()
						screen.Fill(background)
						var options [9]*ebiten.DrawImageOptions
						var marked [9]bool
						for index := range 9 {
							x, y := index%3, index/3
							options[index] = &ebiten.DrawImageOptions{}
							options[index].GeoM.Scale(scale.x, scale.y)
							options[index].GeoM.Translate(origin.x+float64(x*16)*scale.x, origin.y+float64(y*16)*scale.y)
							marked[index] = layout == "all marked" || layout == "center only" && index == 4 || layout == "checkerboard" && (x+y)%2 == 0
							monster := sheet.GetTileEffectProcessor().ApplyDynamicSimple(sheet.GetSprite(indexes.Orc_KeyIndex+indexes.SpriteIndex(index%4)), sprites.BlackOutline)
							screen.DrawImage(monster, options[index])
							// Include real status and HP renderers for marked AND unmarked
							// neighbors. Their complete raster footprint is protected.
							for _, target := range []*ebiten.Image{screen, protected} {
								drawCombatFleeChevrons(target, options[index])
								loyalty := game_state.CombatMonsterLoyaltyMonster
								if !marked[index] {
									loyalty = game_state.CombatMonsterLoyaltyParty
								}
								drawCombatHealthFrameOutline(target, options[index], loyalty)
								if index == 4 {
									drawCombatPossessedMarker(target, options[index])
								}
								drawCombatHealthBar(target, options[index], 25+index%3*25, 100)
							}
						}
						before := make([]byte, width*width*4)
						after := make([]byte, len(before))
						wanted := make([]byte, len(before))
						protectedPixels := make([]byte, len(before))
						allowed := make([]bool, width*width)
						screen.ReadPixels(before)
						protected.ReadPixels(protectedPixels)
						// Production order: all actor artwork/HP/status, foreground
						// eligibility, then the dominant aiming reticle.
						for index := range 9 {
							if marked[index] && !(aimCenter && index == 4) {
								drawCombatAttackBrackets(screen, options[index])
							}
						}
						screen.ReadPixels(after)
						for i := 0; i < len(before); i += 4 {
							if protectedPixels[i+3] != 0 && !bytes.Equal(before[i:i+4], after[i:i+4]) {
								t.Fatalf("marker altered HP/status at (%d,%d): %v -> %v", i/4%width, i/4/width, before[i:i+4], after[i:i+4])
							}
						}
						for index := range 9 {
							reference.Fill(background)
							drawCombatAttackBrackets(reference, options[index])
							reference.ReadPixels(wanted)
							middle, _ := options[index].GeoM.Apply(9, 0)
							var faces, borders [2]int
							visible := marked[index] && !(aimCenter && index == 4)
							for i := 0; i < len(wanted); i += 4 {
								if wanted[i] == background.R && wanted[i+1] == background.G && wanted[i+2] == background.B {
									continue
								}
								if !visible {
									if !bytes.Equal(before[i:i+4], after[i:i+4]) {
										t.Fatalf("unmarked/aimed creature %d gained marker pixels", index)
									}
									continue
								}
								allowed[i/4] = true
								if !bytes.Equal(wanted[i:i+4], after[i:i+4]) {
									t.Fatalf("creature (%d,%d) lost marker contrast at (%d,%d): want %v got %v", index%3, index/3, i/4%width, i/4/width, wanted[i:i+4], after[i:i+4])
								}
								side := 0
								if float64(i/4%width) >= middle {
									side = 1
								}
								if wanted[i] == faceColor.R && wanted[i+1] == faceColor.G && wanted[i+2] == faceColor.B {
									faces[side]++
								} else {
									require.Equal(t, []byte{0, 0, 0}, wanted[i:i+3])
									borders[side]++
								}
								require.Equal(t, byte(255), wanted[i+3])
							}
							if visible {
								for side := range 2 {
									require.Positive(t, faces[side], "creature %d side %d blue arm", index, side)
									require.Positive(t, borders[side], "creature %d side %d full black border", index, side)
								}
							}
						}
						for i := 0; i < len(before); i += 4 {
							if !allowed[i/4] && !bytes.Equal(before[i:i+4], after[i:i+4]) {
								t.Fatalf("marker changed unrelated pixels at (%d,%d)", i/4%width, i/4/width)
							}
						}
						if aimCenter {
							left, top, right, bottom := combatReticleBounds(options[4])
							reference.Clear()
							for _, target := range []*ebiten.Image{screen, reference} {
								drawCombatReticle(target, left, top, right, bottom, ucolor.CombatAimEnemyBright)
							}
							reference.ReadPixels(wanted)
							screen.ReadPixels(after)
							for i := 0; i < len(wanted); i += 4 {
								if wanted[i+3] != 0 && !bytes.Equal(wanted[i:i+4], after[i:i+4]) {
									t.Fatalf("reticle lost foreground priority at (%d,%d)", i/4%width, i/4/width)
								}
							}
						}
					})
				}
			}
		}
	}
}

// Keep production transforms in native screen coordinates while allocating only
// the actor footprints plus four source pixels of guard for outlines and cues.
func combatFixtureRasterBounds(options *ebiten.DrawImageOptions) image.Rectangle {
	var bounds image.Rectangle
	for _, point := range []image.Point{{-4, -4}, {22, -4}, {-4, 22}, {22, 22}} {
		x, y := options.GeoM.Apply(float64(point.X), float64(point.Y))
		bounds = bounds.Union(image.Rect(int(math.Floor(x)), int(math.Floor(y)), int(math.Ceil(x))+1, int(math.Ceil(y))+1))
	}
	return bounds
}
