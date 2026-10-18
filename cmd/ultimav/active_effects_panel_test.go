package main

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	gametext "github.com/bradhannah/UltimaVRedux/internal/text"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

func TestPersistentEffectPresentationUsesAgreedAtlasPlaceholders(t *testing.T) {
	expected := map[game_state.PersistentEffectID]indexes.SpriteIndex{
		game_state.PersistentEffectTorch:        indexes.ItemTorch,
		game_state.PersistentEffectMagicLight:   indexes.BlueFlame,
		game_state.PersistentEffectProtection:   indexes.ItemShield,
		game_state.PersistentEffectNegateMagic:  indexes.ItemAnkh,
		game_state.PersistentEffectTimeStop:     indexes.Hourglass1,
		game_state.PersistentEffectQuickness:    indexes.Hourglass2,
		game_state.PersistentEffectConfusion:    indexes.Whirpool1,
		game_state.PersistentEffectIntoxication: indexes.ItemPotion,
		game_state.PersistentEffectShadowlord:   indexes.ShadowLord1,
		game_state.PersistentEffectStarvation:   indexes.ItemFood,
		game_state.PersistentEffectHMSCape:      indexes.ShipSailsUp,
	}

	for id, sprite := range expected {
		presentation, ok := persistentEffectPresentationFor(id)
		require.True(t, ok)
		assert.Equal(t, sprite, presentation.sprite, id)
	}
	_, crownMapped := persistentEffectPresentationFor(game_state.PersistentEffectCrownSuppression)
	assert.False(t, crownMapped, "Crown suppression is represented by the worn Crown only")
	intoxication, ok := persistentEffectPresentationFor(game_state.PersistentEffectIntoxication)
	require.True(t, ok)
	assert.False(t, intoxication.animated)
	assert.Equal(t, fullTint, intoxication.tintR)
	assert.Equal(t, intoxicationGreenTint, intoxication.tintG)
	assert.Equal(t, fullTint, intoxication.tintB)
}

func TestActiveEffectIconLayoutInsetsRightAndCentersVertically(t *testing.T) {
	bounds := image.Rect(100, 40, 400, 90)
	icons := activeEffectIconLayout(bounds, 3)

	require.Len(t, icons, 3)
	assert.Equal(t, 12, bounds.Max.X-icons[len(icons)-1].Max.X)
	assert.Equal(t, bounds.Min.Y+(bounds.Dy()-icons[0].Dy())/2, icons[0].Min.Y)
	assert.Equal(t, icons[1].Min.X-icons[0].Max.X, icons[2].Min.X-icons[1].Max.X)
	assert.True(t, icons[0].Min.X < icons[1].Min.X && icons[1].Min.X < icons[2].Min.X)
}

func TestActiveEffectIconLayoutRejectsUndeclaredCategoryInsteadOfHidingIt(t *testing.T) {
	assert.Panics(t, func() { activeEffectIconLayout(image.Rect(0, 0, 120, 40), 8) })
}

func TestLoneShadowlordHasRightGutterAndNoAccentAtEveryAnimationTime(t *testing.T) {
	state := &game_state.GameState{ShadowlordState: references.NewShadowlordState()}
	state.ShadowlordState.AutomaticPresence = references.Astaroth
	state.PartyState.Inventory.Provisions.Food.Set(1)
	for _, size := range []image.Point{{1280, 720}, {1920, 1080}} {
		t.Run(size.String(), func(t *testing.T) {
			band := calculateRightRailLayout(image.Rectangle{Max: size}).activeBorder
			band = band.Sub(band.Min)
			panel := NewActiveEffectsPanel(state, sharedPresentationTestSpriteSheet(), band)
			screen := newPresentationRaster(band.Dx(), band.Dy())
			t.Cleanup(screen.Dispose)
			for _, elapsed := range []int64{0, 350, 1050} {
				state.ElapsedMs = elapsed
				screen.Clear()
				panel.Draw(screen)
				require.Len(t, panel.visibleEffects, 1)
				require.Equal(t, game_state.PersistentEffectShadowlord, panel.visibleEffects[0].ID)
				icon := panel.iconBounds[0]
				require.GreaterOrEqual(t, panel.layout.content.Max.X-icon.Max.X, 8)
				require.True(t, icon.In(panel.layout.content))
				assertStatusPixelsPresent(t, screen, icon.Inset(statusCellInset(icon)))
				assertStatusPixelsEmpty(t, screen, image.Rect(icon.Min.X, icon.Max.Y-statusCellInset(icon), icon.Max.X, icon.Max.Y))
				assertStatusPixelsEmpty(t, screen, image.Rect(icon.Max.X, 0, band.Max.X, band.Max.Y))
			}
		})
	}
}

func TestStatusCellInsetScalesInsideConstantIconBounds(t *testing.T) {
	assert.Equal(t, 1, statusCellInset(image.Rect(0, 0, 18, 18)))
	assert.Equal(t, 2, statusCellInset(image.Rect(0, 0, 36, 36)))
	assert.Equal(t, 3, statusCellInset(image.Rect(0, 0, 54, 54)))
}

func TestEffectAnimationPhaseUsesStableIdentity(t *testing.T) {
	first := persistentEffectAnimationHash(game_state.PersistentEffectShadowlord)
	second := persistentEffectAnimationHash(game_state.PersistentEffectShadowlord)
	assert.Equal(t, first, second)
	assert.NotEqual(t, first, persistentEffectAnimationHash(game_state.PersistentEffectConfusion))
}

func TestActiveEffectPanelRefreshPreservesStatusOrderAndCapacity(t *testing.T) {
	state := &game_state.GameState{ShadowlordState: references.NewShadowlordState()}
	state.MapState.Lighting.RestoreTorchMinutes(20)
	state.MagicLightMinutesRemaining = 100
	state.ActiveDurationEffect = game_state.DurationEffect{
		EffectType:     game_state.DurationEffectProtection,
		TurnsRemaining: 4,
	}
	state.ShadowlordState.AutomaticPresence = references.Astaroth
	state.PartyState.Inventory.Provisions.Food.Set(0)
	want := state.GetPersistentEffectStatuses()
	require.Len(t, want, 5)

	panel := NewActiveEffectsPanel(state, nil, image.Rect(0, 0, 120, 40))
	panel.refreshLayout()
	assert.Equal(t, want, panel.visibleEffects)
	assert.Len(t, panel.iconBounds, len(panel.visibleEffects))

	panel.SetBounds(image.Rect(0, 0, 400, 40))
	panel.refreshLayout()
	assert.Equal(t, want, panel.visibleEffects)
	assert.Len(t, panel.iconBounds, len(want))
}

func TestStatusBandSevenEffectsAndThreeWornFitEverySupportedSize(t *testing.T) {
	for _, size := range []image.Point{{1280, 720}, {1366, 768}, {1600, 900}, {1920, 1080}, {2560, 1440}, {3840, 2160}, {3440, 1440}, {1600, 1200}} {
		t.Run(size.String(), func(t *testing.T) {
			screen := logicalGameBounds(size.X, size.Y)
			rail := calculateRightRailLayout(screen)
			panel := NewActiveEffectsPanel(nil, nil, rail.activeBorder)
			layout := panel.layout
			assert.True(t, layout.content.In(rail.activeBorder))
			fullEffects := activeEffectIconLayout(layout.content, maximumPersistentEffects)
			fullWorn := statusIconRow(layout.content, maximumWornArtifacts, maximumWornArtifacts, false)
			require.Len(t, fullEffects, maximumPersistentEffects)
			require.Len(t, fullWorn, maximumWornArtifacts)
			assert.Equal(t, layout.content.Min.X, fullWorn[0].Min.X)
			assert.Less(t, fullEffects[len(fullEffects)-1].Max.X, layout.content.Max.X)
			assert.Equal(t, fullWorn[0].Size(), fullEffects[0].Size())
			assert.False(t, fullWorn[len(fullWorn)-1].Overlaps(fullEffects[0]))
			assert.Less(t, fullWorn[len(fullWorn)-1].Max.X, fullEffects[0].Min.X)
			for count := range 8 {
				icons := activeEffectIconLayout(layout.content, count)
				assert.Len(t, icons, count)
				for index, icon := range icons {
					assert.Equal(t, fullEffects[0].Size(), icon.Size(), "ordinary count changes never resize cells")
					assert.True(t, icon.In(layout.content))
					assert.Equal(t, fullEffects[0].Min.Y, icon.Min.Y, "one row")
					assert.GreaterOrEqual(t, icon.Inset(statusCellInset(icon)).Dx(), statusArtworkPixels)
					if index > 0 {
						assert.Less(t, icons[index-1].Max.X, icon.Min.X)
					}
				}
				if count > 0 {
					assert.Equal(t, fullEffects[len(fullEffects)-1].Max.X, icons[len(icons)-1].Max.X)
				}
			}
			for index, icon := range fullWorn {
				assert.True(t, icon.In(layout.content))
				assert.Equal(t, fullEffects[0].Size(), icon.Size())
				if index > 0 {
					assert.False(t, icon.Overlaps(fullWorn[index-1]))
				}
			}
		})
	}
}

func TestStatusBandTinyBoundsAreSafeWithoutDroppingProjectionEntries(t *testing.T) {
	// Deliberately test subminimum component geometry. Production render sizing
	// now promotes tiny output containers to a 720p canvas before rail layout.
	tinyRail := calculateRightRailLayout(image.Rect(0, 0, 16, 9)).activeBorder
	for _, bounds := range []image.Rectangle{{}, image.Rect(0, 0, 1, 1), tinyRail} {
		state := &game_state.GameState{CrownState: game_state.ToggleItemState{IsActive: true}}
		panel := NewActiveEffectsPanel(state, nil, bounds)
		assert.NotPanics(t, func() { panel.Draw(ebiten.NewImage(16, 9)) })
		assert.Len(t, state.GetPersistentEffectStatuses(), 2, "gameplay still projects Crown suppression and starvation")
		assert.Len(t, panel.visibleEffects, 1, "the worn Crown deduplicates only its presentation")
		assert.Len(t, panel.iconBounds, 1)
		assert.Len(t, panel.worn, 1)
		assert.Len(t, panel.wornBounds, 1)
	}
}

func TestWornArtifactImageSourcesAndNativeSpriteCrop(t *testing.T) {
	sheet := syntheticStatusSpriteSheet(t)
	panel := NewActiveEffectsPanel(nil, sheet, image.Rect(0, 0, 318, 65))
	for _, test := range []struct {
		id    game_state.ToggleItemType
		index indexes.SpriteIndex
	}{{game_state.ToggleItemCrown, 437}, {game_state.ToggleItemAmulet, 439}} {
		art := wornArtifactImage(panel.spriteSheet, test.id)
		assert.Equal(t, image.Pt(16, 16), art.Bounds().Size())
		assert.Equal(t, syntheticStatusColor(test.index), art.At(art.Bounds().Min.X, art.Bounds().Min.Y))
	}
	assert.Same(t, gametext.GetBitmapFonts().IBM.GetCharImage(29), wornArtifactImage(panel.spriteSheet, game_state.ToggleItemBadge))
	assert.Equal(t, image.Pt(8, 8), wornArtifactImage(panel.spriteSheet, game_state.ToggleItemBadge).Bounds().Size())
	assert.Panics(t, func() { wornArtifactImage(panel.spriteSheet, game_state.ToggleItemType(0)) })
	assert.Nil(t, unwrappedStatusSprite(nil))
}

func TestStatusBandDrawsOnlyActuallyWornArtifactsInCompactOrder(t *testing.T) {
	state := &game_state.GameState{}
	state.PartyState.Inventory.Provisions.Food.Set(1)
	panel := NewActiveEffectsPanel(state, syntheticStatusSpriteSheet(t), image.Rect(0, 0, 318, 65))
	screen := ebiten.NewImage(318, 65)
	t.Cleanup(screen.Deallocate)
	panel.Draw(screen)
	assert.Empty(t, panel.worn)
	assert.Empty(t, panel.wornBounds)
	assertStatusPixelsEmpty(t, screen, panel.layout.content)

	state.PartyState.Inventory.QuestItems.Set(references.Crown, 1)
	screen.Clear()
	panel.Draw(screen)
	assert.Empty(t, panel.worn, "owned but unworn artifacts are not previewed")
	assertStatusPixelsEmpty(t, screen, panel.layout.content)

	state.AmuletState = game_state.ToggleItemState{IsActive: true, TurnsRemaining: 255}
	state.BadgeState = state.AmuletState
	screen.Clear()
	panel.Draw(screen)
	require.Equal(t, []game_state.ToggleItemType{game_state.ToggleItemAmulet, game_state.ToggleItemBadge},
		[]game_state.ToggleItemType{panel.worn[0].ID, panel.worn[1].ID})
	require.Len(t, panel.wornBounds, 2)
	assert.Equal(t, panel.layout.content.Min.X, panel.wornBounds[0].Min.X)
	assert.Equal(t, activeEffectIconLayout(panel.layout.content, 1)[0].Size(), panel.wornBounds[0].Size())
	for _, slot := range panel.wornBounds {
		_, _, _, cornerAlpha := screen.At(slot.Min.X, slot.Min.Y).RGBA()
		assert.Zero(t, cornerAlpha, "worn artwork has no surrounding box")
		inset := statusCellInset(slot)
		assertStatusPixelsPresent(t, screen, slot.Inset(inset))
	}
}

func TestStatusArtworkPreservesAspectAndNearestNativePixels(t *testing.T) {
	screen := ebiten.NewImage(24, 24)
	t.Cleanup(screen.Deallocate)
	art := ebiten.NewImage(16, 8)
	t.Cleanup(art.Deallocate)
	art.Fill(ucolor.EGAWhite)
	drawStatusArtwork(screen, art, image.Rect(2, 2, 18, 18), persistentEffectPresentation{tintR: 1, tintG: 1, tintB: 1})
	assertStatusPixelsEmpty(t, screen, image.Rect(2, 2, 18, 6))
	assertStatusPixelsEmpty(t, screen, image.Rect(2, 14, 18, 18))
	assert.Equal(t, ucolor.EGAWhite.ToRGBA(), screen.At(2, 6))
	assert.Equal(t, ucolor.EGAWhite.ToRGBA(), screen.At(17, 13))
	art.SubImage(image.Rect(7, 0, 8, 8)).(*ebiten.Image).Fill(ucolor.EGACyan)
	screen.Clear()
	drawStatusArtwork(screen, art, screen.Bounds(), persistentEffectPresentation{tintR: 1, tintG: 1, tintB: 1})
	pixels := make([]byte, 24*24*4)
	screen.ReadPixels(pixels)
	white, cyan := 0, 0
	for offset := 0; offset < len(pixels); offset += 4 {
		if pixels[offset+3] == 0 {
			continue
		}
		pixel := color.RGBA{R: pixels[offset], G: pixels[offset+1], B: pixels[offset+2], A: pixels[offset+3]}
		require.Contains(t, []color.RGBA{ucolor.EGAWhite.ToRGBA(), ucolor.EGACyan.ToRGBA()}, pixel, "fractional scaling must not blend adjacent native pixels")
		if pixel == ucolor.EGAWhite.ToRGBA() {
			white++
		} else {
			cyan++
		}
	}
	require.Positive(t, white)
	require.Positive(t, cyan)
}

func TestStatusBandFullCapacityRenderIsContainedAndLeavesGapClear(t *testing.T) {
	state := &game_state.GameState{ElapsedMs: 1234}
	sheet := sharedPresentationTestSpriteSheet()
	for _, size := range []image.Point{{1280, 720}, {3840, 2160}} {
		t.Run(size.String(), func(t *testing.T) {
			band := calculateRightRailLayout(image.Rectangle{Max: size}).activeBorder
			// Render a band-sized target instead of retaining full 4K GPU surfaces.
			band = band.Sub(band.Min)
			panel := NewActiveEffectsPanel(state, sheet, band)
			panel.visibleEffects = fullStatusBandTestEffects()
			panel.iconBounds = activeEffectIconLayout(panel.layout.content, maximumPersistentEffects)
			panel.worn = []game_state.WornArtifactStatus{
				{ID: game_state.ToggleItemCrown, Owned: true, Worn: true},
				{ID: game_state.ToggleItemAmulet, Owned: true, Worn: true},
				{ID: game_state.ToggleItemBadge, Owned: true, Worn: true},
			}
			panel.wornBounds = statusIconRow(panel.layout.content, len(panel.worn), maximumWornArtifacts, false)
			screen := newPresentationRaster(band.Dx(), band.Dy())
			t.Cleanup(screen.Dispose)
			panel.drawStatusBand(screen)
			panel.drawEffects(screen)
			for _, icon := range panel.iconBounds {
				assertStatusPixelsPresent(t, screen, icon.Inset(statusCellInset(icon)))
				_, _, _, cornerAlpha := screen.At(icon.Min.X, icon.Min.Y).RGBA()
				assert.Zero(t, cornerAlpha, "effect artwork has no surrounding box")
				inset := statusCellInset(icon)
				assertStatusPixelsEmpty(t, screen, image.Rect(icon.Min.X, icon.Max.Y-inset, icon.Max.X, icon.Max.Y))
			}
			for _, icon := range panel.wornBounds {
				assert.Equal(t, panel.iconBounds[0].Size(), icon.Size())
				_, _, _, cornerAlpha := screen.At(icon.Min.X, icon.Min.Y).RGBA()
				assert.Zero(t, cornerAlpha, "worn artwork has no surrounding box")
				inset := statusCellInset(icon)
				assertStatusPixelsPresent(t, screen, icon.Inset(inset))
			}
			gap := image.Rect(panel.wornBounds[len(panel.wornBounds)-1].Max.X, panel.layout.content.Min.Y,
				panel.iconBounds[0].Min.X, panel.layout.content.Max.Y)
			require.False(t, gap.Empty())
			assertStatusPixelsEmpty(t, screen, gap)
			pixels := make([]byte, band.Dx()*band.Dy()*4)
			screen.ReadPixels(pixels)
			for y := range band.Dy() {
				for x := range band.Dx() {
					if !image.Pt(x, y).In(panel.layout.content) {
						require.Zero(t, pixels[(y*band.Dx()+x)*4+3], "status art/frame outside the shared content rail")
					}
				}
			}
		})
	}
}

func TestStatusBandRendersCrownOnceWithoutRemovingAuthoritativeSuppression(t *testing.T) {
	state := &game_state.GameState{CrownState: game_state.ToggleItemState{IsActive: true, TurnsRemaining: 255}}
	state.PartyState.Inventory.Provisions.Food.Set(1)
	panel := NewActiveEffectsPanel(state, nil, image.Rect(0, 0, 318, 65))
	panel.refreshLayout()

	require.Equal(t, []game_state.PersistentEffectStatus{{ID: game_state.PersistentEffectCrownSuppression}},
		state.GetPersistentEffectStatuses())
	require.Len(t, panel.worn, 1)
	assert.Equal(t, game_state.ToggleItemCrown, panel.worn[0].ID)
	assert.Empty(t, panel.visibleEffects)
	assert.Empty(t, panel.iconBounds)

	state.CrownState = game_state.ToggleItemState{}
	state.ActiveDurationEffect = game_state.DurationEffect{
		EffectType: game_state.DurationEffectNegateMagic, TurnsRemaining: 10,
	}
	panel.refreshLayout()
	assert.Empty(t, panel.worn)
	require.Len(t, panel.visibleEffects, 1)
	assert.Equal(t, game_state.PersistentEffectNegateMagic, panel.visibleEffects[0].ID)
}

func fullStatusBandTestEffects() []game_state.PersistentEffectStatus {
	return []game_state.PersistentEffectStatus{
		{ID: game_state.PersistentEffectTorch, Atmosphere: game_state.EffectAtmosphereFading},
		{ID: game_state.PersistentEffectMagicLight, Atmosphere: game_state.EffectAtmosphereCritical},
		{ID: game_state.PersistentEffectProtection, Atmosphere: game_state.EffectAtmosphereSteady},
		{ID: game_state.PersistentEffectIntoxication, Atmosphere: game_state.EffectAtmosphereDanger},
		{ID: game_state.PersistentEffectShadowlord, Atmosphere: game_state.EffectAtmosphereOminous},
		{ID: game_state.PersistentEffectStarvation, Atmosphere: game_state.EffectAtmosphereDanger},
		{ID: game_state.PersistentEffectHMSCape, Atmosphere: game_state.EffectAtmosphereSteady},
	}
}

func syntheticStatusSpriteSheet(t *testing.T) *sprites.SpriteSheet {
	t.Helper()
	atlas := image.NewRGBA(image.Rect(0, 0, 512, 256))
	for index := range 512 {
		for y := range 16 {
			for x := range 16 {
				atlas.SetRGBA(index%32*16+x, index/32*16+y, syntheticStatusColor(indexes.SpriteIndex(index)))
			}
		}
	}
	sheet := &sprites.SpriteSheet{SpriteImage: ebiten.NewImageFromImage(atlas)}
	t.Cleanup(sheet.SpriteImage.Dispose)
	return sheet
}

func syntheticStatusColor(index indexes.SpriteIndex) color.RGBA {
	return color.RGBA{R: uint8(index % 251), G: uint8(index/251) * 80, B: 100, A: 255}
}

func assertStatusPixelsEmpty(t *testing.T, screen *ebiten.Image, bounds image.Rectangle) {
	t.Helper()
	pixels := make([]byte, bounds.Dx()*bounds.Dy()*4)
	screen.SubImage(bounds).(*ebiten.Image).ReadPixels(pixels)
	for offset := 3; offset < len(pixels); offset += 4 {
		require.Zero(t, pixels[offset], "unexpected pixel in %v at offset %d", bounds, offset)
	}
}

func assertStatusPixelsPresent(t *testing.T, screen *ebiten.Image, bounds image.Rectangle) {
	t.Helper()
	pixels := make([]byte, bounds.Dx()*bounds.Dy()*4)
	screen.SubImage(bounds).(*ebiten.Image).ReadPixels(pixels)
	for offset := 3; offset < len(pixels); offset += 4 {
		if pixels[offset] != 0 {
			return
		}
	}
	t.Fatalf("expected visible status pixels in %v", bounds)
}
