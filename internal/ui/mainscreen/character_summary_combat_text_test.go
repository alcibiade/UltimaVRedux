package mainscreen

import (
	"fmt"
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

func TestCharacterSummaryCombatTextPrecedenceAndSelection(t *testing.T) {
	party := summaryBaselineParty()
	character := party.GetCharacter(0)
	summary := &CharacterSummary{}
	for _, test := range []struct {
		name      string
		status    party_state.CharacterStatus
		possessed bool
		sleeping  bool
		label     string
		colour    color.Color
	}{
		{"healthy", party_state.Good, false, false, "Good", ucolor.White},
		{"possessed", party_state.Good, true, false, "Possessed", ucolor.CombatPossessed},
		{"possession over poison and sleep", party_state.Poisoned, true, true, "Possessed", ucolor.CombatPossessed},
		{"possession over shared sleep", party_state.Sleep, true, false, "Possessed", ucolor.CombatPossessed},
		{"death over all conditions", party_state.Dead, true, true, "Dead", ucolor.EGARed},
		{"persistent charm over sleep", party_state.Charmed, false, true, "Charmed", ucolor.EGABlue},
		{"poison over sleep", party_state.Poisoned, false, true, "Pois.", ucolor.EGAGreen},
		{"exact sleep", party_state.Good, false, true, "Sleep", ucolor.EGAMagenta},
		{"other copy asleep", party_state.Sleep, false, false, "Good", ucolor.White},
		{"source asleep", party_state.Sleep, false, true, "Sleep", ucolor.EGAMagenta},
	} {
		t.Run(test.name, func(t *testing.T) {
			character.SetStatus(test.status)
			before := *character
			summary.SetCombatText([party_state.MaxCharactersInParty]CharacterCombatText{
				{InCombat: true, Possessed: test.possessed, Sleeping: test.sleeping},
			})
			label, statusColor := summary.characterTextStatus(0, character)
			require.Equal(t, test.label, label)
			for _, selection := range []struct {
				name                                   string
				active, hasActive, solo, soloCharacter bool
				dim                                    bool
			}{
				{name: "round robin"},
				{name: "active", active: true, hasActive: true},
				{name: "inactive", hasActive: true, dim: true},
				{name: "solo", hasActive: true, solo: true, soloCharacter: true},
				{name: "solo overrides active", active: true, hasActive: true, solo: true, dim: true},
			} {
				got := selectedCharacterTextColor(statusColor, selection.active, selection.hasActive, selection.solo, selection.soloCharacter)
				want := test.colour
				if selection.dim {
					want = ucolor.From(want).Intensity(0.6).RGBA()
				}
				assert.Equal(t, ucolor.From(want).RGBA(), ucolor.From(got).RGBA(), selection.name)
			}
			assert.Equal(t, before, *character)
		})
	}
}

func TestCharacterSummaryCombatTextApplyClearMultipleMembersAndRenderCache(t *testing.T) {
	for _, width := range []int{960, 1280, 1920} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			party := summaryBaselineParty()
			summary := NewCharacterSummaryForWidth(sprites.NewSpriteSheet(), width)
			t.Cleanup(summary.Dispose)
			// Even portrait sizes keep the 18x18 artwork off exact nearest-neighbour
			// texel boundaries, whose rounding can change with native atlas placement.
			// Keep the resolution's font and status-cell width for the text/fit assertions;
			// the separate layout baseline test covers the original row geometry.
			bounds := summary.bounds
			rowHeight := bounds.Dy() / party_state.MaxCharactersInParty
			rowHeight -= rowHeight % 2
			bounds.Max.Y = bounds.Min.Y + rowHeight*party_state.MaxCharactersInParty
			summary.SetBounds(bounds)
			screen := ebiten.NewImage(width, width*9/16)
			t.Cleanup(screen.Dispose)
			draw := func() {
				screen.Clear()
				summary.Draw(party, screen, 0, false, -1)
			}
			draw()
			baseline := summaryPixels(screen, summary.bounds)
			portraits := summary.portraits
			fontBefore := *summary.ultimaFont.TextFace
			var characters [party_state.MaxCharactersInParty]party_state.PlayerCharacter
			var builds [party_state.MaxCharactersInParty]uint64
			for index := range characters {
				characters[index] = *party.GetCharacter(index)
				builds[index] = summary.textRows[index].builds
			}
			overlay := [party_state.MaxCharactersInParty]CharacterCombatText{
				{InCombat: true, Possessed: true, Sleeping: true},
				{InCombat: true, Possessed: true, Sleeping: true},
			}
			summary.SetCombatText(overlay)
			draw()
			for index := range 2 {
				row := &summary.textRows[index]
				require.Equal(t, builds[index]+1, row.builds)
				require.Contains(t, row.right.GetOutputStr(false), "Possessed")
				colour := ucolor.From(ucolor.CombatPossessed).RGBA()
				if index != 0 {
					colour = ucolor.From(colour).Intensity(0.6).RGBA()
				}
				_, hasNameColor := row.left.FirstPanelLineWithColor(0, 2, colour)
				_, hasStatusColor := row.right.FirstPanelLineWithColor(0, 2, colour)
				require.True(t, hasNameColor)
				require.True(t, hasStatusColor)
			}
			require.NotEqual(t, baseline, summaryPixels(screen, summary.bounds), "possession must visibly affect the panel")
			statusFace := *summary.textRows[0].right.Font.TextFace
			statusFace.Size *= summary.textRows[0].statusScale
			labelWidth, _ := etext.Measure("Possessed", &statusFace, 0)
			require.LessOrEqual(t, labelWidth, float64(characterRightTextBounds(summary.bounds).Dx()))
			requireCharacterCombatStatusInkFits(t, &summary.textRows[0], characterRightTextBounds(summary.bounds).Dx())
			overlay[0].Possessed = false
			draw()
			require.Equal(t, builds[0]+1, summary.textRows[0].builds, "overlay is detached from its caller")
			require.Equal(t, builds[1]+1, summary.textRows[1].builds, "unchanged frames reuse their text")
			summary.SetCombatText(overlay)
			draw()
			require.Contains(t, summary.textRows[0].right.GetOutputStr(false), "Sleep")
			require.Contains(t, summary.textRows[1].right.GetOutputStr(false), "Possessed")
			require.Equal(t, builds[1]+1, summary.textRows[1].builds, "one member's recovery must not rebuild another")
			overlay[1].Possessed = false
			summary.SetCombatText(overlay)
			draw()
			require.Contains(t, summary.textRows[1].right.GetOutputStr(false), "Pois.", "poison survives recovery despite sleep")
			summary.SetCombatText([party_state.MaxCharactersInParty]CharacterCombatText{})
			draw()
			require.Equal(t, baseline, summaryPixels(screen, summary.bounds), "exit immediately restores exploration text")
			for index := range characters {
				assert.Equal(t, characters[index], *party.GetCharacter(index))
				if index >= 2 {
					assert.Equal(t, builds[index], summary.textRows[index].builds)
				}
			}
			assert.Equal(t, portraits, summary.portraits, "combat text must not regenerate portraits")
			assert.Equal(t, fontBefore, *summary.ultimaFont.TextFace)
		})
	}
}

// Render into an oversized target: clipping to the cell itself would hide the
// overflow this regression test needs to catch. Inspect actual status-line ink.
func requireCharacterCombatStatusInkFits(t *testing.T, row *characterTextRow, width int) {
	t.Helper()
	const padding = 32
	screen := ebiten.NewImage(width+padding*2, int(row.right.GetLineSpacing())+padding*2)
	t.Cleanup(screen.Dispose)
	row.right.DrawPanelContentRange(screen, image.Pt(padding, padding), width, etext.AlignEnd, 0, 1)
	pixels := summaryPixels(screen, screen.Bounds())
	ink := 0
	for offset := 3; offset < len(pixels); offset += 4 {
		if pixels[offset] == 0 {
			continue
		}
		ink++
		x := (offset / 4) % screen.Bounds().Dx()
		require.GreaterOrEqual(t, x, padding, "status ink overflowed into the name column")
		require.Less(t, x, padding+width, "status ink overflowed the right inset")
	}
	require.Positive(t, ink, "the complete status label must actually render")
}

func TestCharacterSummaryCombatStatusFitTracksBoundsAndPreservesMP(t *testing.T) {
	summary := &CharacterSummary{}
	summary.ultimaFont, summary.output = newMainScreenOutput(1280, widgets.FontSizeXLarge, characterOutputMaxLines)
	bounds := image.Rect(0, 0, 282, 328)
	summary.SetBounds(bounds)
	character := summaryBaselineParty().GetCharacter(0)
	row := summary.prepareCharacterText(0, character, color.White)
	left, right := row.left, row.right
	faceBefore := *summary.ultimaFont.TextFace
	screen := ebiten.NewImage(160, 64)
	t.Cleanup(screen.Dispose)
	mpPixels := func() []byte {
		screen.Clear()
		row.right.DrawPanelContentRange(screen, image.Pt(0, 0), 160, etext.AlignEnd, 1, 1)

		return summaryPixels(screen, screen.Bounds())
	}
	baselineMP := mpPixels()
	summary.SetCombatText([party_state.MaxCharactersInParty]CharacterCombatText{{InCombat: true, Possessed: true}})
	summary.prepareCharacterText(0, character, color.White)
	initialScale := row.statusScale
	require.Less(t, initialScale, 1.0)
	require.Greater(t, initialScale, 0.0)
	builds := row.builds
	for _, cellBounds := range []image.Rectangle{image.Rect(0, 0, 210, 328), bounds} {
		summary.SetBounds(cellBounds)
		summary.prepareCharacterText(0, character, color.White)
		builds++
		require.Equal(t, builds, row.builds, "cell width participates in the cache key")
		face := *row.right.Font.TextFace
		face.Size *= row.statusScale
		width, _ := etext.Measure("Possessed", &face, 0)
		require.LessOrEqual(t, width, float64(characterRightTextBounds(cellBounds).Dx()))
		requireCharacterCombatStatusInkFits(t, row, characterRightTextBounds(cellBounds).Dx())
		require.Equal(t, baselineMP, mpPixels(), "only the status segment may shrink")
		require.Same(t, left, row.left)
		require.Same(t, right, row.right)
		require.Equal(t, faceBefore, *summary.ultimaFont.TextFace)
		summary.prepareCharacterText(0, character, color.White)
		require.Equal(t, builds, row.builds, "stable bounds retain the fitted text")
	}
	require.Equal(t, initialScale, row.statusScale)
	summary.SetCombatText([party_state.MaxCharactersInParty]CharacterCombatText{})
	summary.prepareCharacterText(0, character, color.White)
	require.Equal(t, 1.0, row.statusScale, "recovery restores the ordinary status typography")
	require.Equal(t, baselineMP, mpPixels())
}

func TestCharacterSummaryCombatTextCacheIncludesEffectiveLabel(t *testing.T) {
	summary := &CharacterSummary{}
	summary.ultimaFont, summary.output = newMainScreenOutput(1280, widgets.FontSizeXLarge, characterOutputMaxLines)
	character := summaryBaselineParty().GetCharacter(0)
	row := summary.prepareCharacterText(0, character, color.White)
	builds := row.builds
	summary.SetCombatText([party_state.MaxCharactersInParty]CharacterCombatText{{InCombat: true, Possessed: true}})
	// Hold RGBA constant to isolate the effective label's part of the cache key.
	summary.prepareCharacterText(0, character, color.White)
	require.Equal(t, builds+1, row.builds)
	require.Contains(t, row.right.GetOutputStr(false), "Possessed")
	// Selection changes RGBA without changing the label or persistent status.
	summary.prepareCharacterText(0, character, ucolor.CombatPossessed)
	require.Equal(t, builds+2, row.builds)
	dimmed := ucolor.From(ucolor.CombatPossessed).Intensity(0.6).RGBA()
	summary.prepareCharacterText(0, character, dimmed)
	require.Equal(t, builds+3, row.builds)
	_, hasDimmedColor := row.right.FirstPanelLineWithColor(0, 2, dimmed)
	require.True(t, hasDimmedColor)
	summary.prepareCharacterText(0, character, dimmed)
	require.Equal(t, builds+3, row.builds)
	summary.SetCombatText([party_state.MaxCharactersInParty]CharacterCombatText{})
	summary.prepareCharacterText(0, character, color.White)
	require.Equal(t, builds+4, row.builds)
	require.Contains(t, row.right.GetOutputStr(false), "Good")
}
