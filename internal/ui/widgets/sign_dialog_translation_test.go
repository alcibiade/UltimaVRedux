package widgets

import (
	"encoding/binary"
	"image"
	stdcolor "image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func TestSignTranslationLocalModeCacheAndCompleteResponsiveContent(t *testing.T) {
	setGuidanceTestResolution(t, 1920, 1080)
	raw := []byte(strings.Repeat("[\\]_@ABCDEFGHIK\n", 13) + "END")
	original := append([]byte(nil), raw...)
	config := SignDialogConfig{Placement: sprites.PercentBasedPlacement{
		StartPercentX: .1, EndPercentX: .65, StartPercentY: .2, EndPercentY: .8}, RawText: raw}
	dialog := NewSignDialog(config)
	require.False(t, dialog.IsTranslated())
	for _, size := range signTranslationResolutions() {
		display.GetManager().PublishLogicalSize(size.X, size.Y)
		dialog.SetTranslated(true)
		dialog.Update()
		require.True(t, dialog.IsTranslated())
		content := signTranslationRowsText(dialog.translationRows)
		require.Equal(t, 13, strings.Count(content, "THEENGST"))
		require.Contains(t, content, "END")
		require.True(t, dialog.signBounds.In(dialog.border.GetInteriorBounds().Sub(dialog.bounds.Min)))
		buffer, rows := dialog.buffer, dialog.translationRows
		dialog.Update()
		dialog.SetTranslated(true)
		require.Same(t, buffer, dialog.buffer)
		require.Same(t, &rows[0][0], &dialog.translationRows[0][0])
		require.Equal(t, original, dialog.rawText)
	}
	raw[0] = 'Z'
	dialog.SetTranslated(false)
	require.Equal(t, original, dialog.rawText)
	require.False(t, dialog.IsTranslated())
	dialog.SetTranslated(true)
	require.False(t, NewSignDialog(config).IsTranslated(), "no shared preference across readings")
}

func TestSignTranslationFallbackAndMixedIBMContent(t *testing.T) {
	setGuidanceTestResolution(t, 1280, 720)
	raw := framedTranslationSignFixture()
	dialog := NewSignDialog(SignDialogConfig{Placement: sprites.PercentBasedPlacement{
		StartPercentX: .1, EndPercentX: .65, StartPercentY: .2, EndPercentY: .8}, RawText: raw})
	dialog.SetTranslated(true)
	content := signTranslationRowsText(dialog.translationRows)
	require.Contains(t, content, "LIVE BY THE")
	require.Contains(t, content, "EIGHT LAWS")
	require.Contains(t, content, "[George]")
	require.Contains(t, content, "?")
}

func TestSignTranslationLongestOriginalFitsAllResolutions(t *testing.T) {
	setGuidanceTestResolution(t, 1280, 720)
	refs, err := references.NewSignReferences(filepath.Join(originaldata.Directory(t, "SIGNS.DAT"), "SIGNS.DAT"))
	require.NoError(t, err)
	var longest *references.SignRecord
	longestLength := -1
	for _, signs := range refs.SignsByLocation {
		for _, sign := range signs {
			length := len(translatedSignText(sign.RawText))
			if length > longestLength {
				longest, longestLength = sign, length
			}
		}
	}
	require.NotNil(t, longest)
	requireSignTranslationFitAllResolutions(t, longest, false)
}

func TestSignTranslationResolvedAliasAndIBMWantedPosterFitAllResolutions(t *testing.T) {
	setGuidanceTestResolution(t, 1280, 720)
	requireSignTranslationFitAllResolutions(t, translationAliasFixture(t), false)
	requireSignTranslationFitAllResolutions(t, &references.SignRecord{RawText: wantedTranslationSignFixture()}, true)
}

func requireSignTranslationFitAllResolutions(t *testing.T, sign *references.SignRecord, unchanged bool) {
	t.Helper()
	dialog := NewSignDialog(SignDialogConfig{Placement: sprites.PercentBasedPlacement{
		StartPercentX: .1, EndPercentX: .65, StartPercentY: .2, EndPercentY: .8}, Sign: sign})
	defer func() { dialog.buffer.Deallocate() }()
	want := strings.Join(strings.Fields(translatedSignText(sign.RawText)), "")
	original := append([]byte(nil), sign.RawText...)
	for _, size := range signTranslationResolutions() {
		display.GetManager().PublishLogicalSize(size.X, size.Y)
		dialog.SetTranslated(false)
		dialog.Update()
		before, bounds := atlasMarkerPixels(dialog.buffer), dialog.signBounds
		dialog.SetTranslated(true)
		dialog.Update()
		require.Equal(t, want, strings.Join(strings.Fields(signTranslationRowsText(dialog.translationRows)), ""))
		require.Equal(t, bounds, dialog.signBounds)
		require.True(t, bounds.In(dialog.border.GetInteriorBounds().Sub(dialog.bounds.Min)))
		after := atlasMarkerPixels(dialog.buffer)
		requireSignFixedInk(t, before, after)
		if unchanged {
			require.Equal(t, before.Pix, after.Pix, "already-English wanted posters keep every pixel")
		}
		require.Equal(t, original, sign.RawText)
	}
}

func translationAliasFixture(t *testing.T) *references.SignRecord {
	t.Helper()
	data := make([]byte, 66)
	binary.LittleEndian.PutUint16(data[int(references.Britain)*2:], 66)
	data = append(data, byte(references.Britain), 0, 16, 27, '\n', 0)
	data = append(data, byte(references.Britain), 0, 15, 27)
	message := []byte{0x2c, 'R', 'E', 'A', 'D', '@', '[', 'I', 'S', '\n', 0x2d}
	data = append(data, message...)
	data = append(data, 0)
	path := filepath.Join(t.TempDir(), "SIGNS.DAT")
	require.NoError(t, os.WriteFile(path, data, 0o600))
	refs, err := references.NewSignReferences(path)
	require.NoError(t, err)
	alias := refs.GetSignAt(references.Britain, 0, 16, 27)
	require.NotNil(t, alias)
	require.Equal(t, message, alias.RawText)
	target := refs.GetSignAt(references.Britain, 0, 15, 27)
	require.NotNil(t, target)
	target.RawText[1] = 'X'
	require.Equal(t, message, alias.RawText, "resolved alias owns independent original bytes")
	return alias
}

func translatedSignText(raw []byte) string {
	return signTranslationRowsText(text.DecodeSignTranslation(raw))
}

func signTranslationRowsText(rows [][]text.ColorSegment) string {
	var result strings.Builder
	for _, row := range rows {
		for _, segment := range row {
			result.WriteString(segment.Text)
		}
		result.WriteByte('\n')
	}
	return result.String()
}

func signTranslationResolutions() []image.Point {
	return []image.Point{{1280, 720}, {1920, 1080}, {2560, 1440}, {3840, 2160}, {1365, 767}}
}

func framedTranslationSignFixture() []byte {
	raw := []byte("abbbbbbbbbbbbbbc\ng              g\ng  LIVE@BY@[E  g\ng  EIGHT@LAWS  g\ng              g\ng   ")
	for _, value := range []byte("[George]") {
		raw = append(raw, value|0x80)
	}
	return append(raw, []byte("   g\ng  ?           g\ng              g\ndeeeeeeeeeeeeeef\n")...)
}

func wantedTranslationSignFixture() []byte {
	raw := []byte("abbbbbbbbbbbbbc\n")
	for _, line := range []string{"g   Wanted:   g", "g             g", "g   Avatar    g", "g    Iolo     g", "g   Shamino   g", "g             g", "gDead or Aliveg"} {
		// Existing IBM lettering on wanted posters is already English.
		for index, value := range []byte(line) {
			if index != 0 && index != len(line)-1 {
				value |= 0x80
			}
			raw = append(raw, value)
		}
		raw = append(raw, '\n')
	}
	return append(raw, []byte("deeeeeeeeeeeeef")...)
}

func requireSignFixedInk(t *testing.T, before, after *image.RGBA) {
	t.Helper()
	require.Equal(t, before.Bounds(), after.Bounds())
	mask := func(pixels *image.RGBA) []byte {
		fixed := make([]byte, len(pixels.Pix))
		for y := range pixels.Bounds().Dy() {
			for x := range pixels.Bounds().Dx() {
				value := pixels.RGBAAt(x, y)
				if value == ucolor.EGABrown.ToRGBA() || value == ucolor.EGAWhite.ToRGBA() {
					index := pixels.PixOffset(x, y)
					copy(fixed[index:index+4], pixels.Pix[index:index+4])
				}
			}
		}
		return fixed
	}
	require.Equal(t, mask(before), mask(after), "original brown artwork and white IBM ink retain every source pixel")
}

func TestSignTranslationRestoresOriginalMixedFontArtwork(t *testing.T) {
	setGuidanceTestResolution(t, 1280, 720)
	dialog := NewSignDialog(SignDialogConfig{Placement: sprites.PercentBasedPlacement{
		StartPercentX: .1, EndPercentX: .65, StartPercentY: .2, EndPercentY: .8},
		RawText: []byte{0x2c, 'g', '[', '\\', '?', 'g' | 0x80, '[' | 0x80, '\n', 0x2d}})
	before := make([]byte, 4*dialog.bounds.Dx()*dialog.bounds.Dy())
	dialog.buffer.ReadPixels(before)
	dialog.SetTranslated(true)
	dialog.SetTranslated(false)
	after := make([]byte, len(before))
	dialog.buffer.ReadPixels(after)
	require.Equal(t, before, after)
}

func TestSignTranslationFramedInscriptionGeometryAndPixelsAcrossResolutions(t *testing.T) {
	setGuidanceTestResolution(t, 1280, 720)
	dialog := NewSignDialog(SignDialogConfig{Placement: sprites.PercentBasedPlacement{
		StartPercentX: .1, EndPercentX: .65, StartPercentY: .2, EndPercentY: .8}, RawText: framedTranslationSignFixture()})
	defer func() { dialog.buffer.Deallocate() }()
	for _, size := range signTranslationResolutions() {
		display.GetManager().PublishLogicalSize(size.X, size.Y)
		dialog.SetTranslated(false)
		dialog.Update()
		before, bounds := atlasMarkerPixels(dialog.buffer), dialog.signBounds
		require.Positive(t, atlasExactInkCount(before, ucolor.EGABrown))
		require.Positive(t, atlasExactInkCount(before, ucolor.EGAWhite))
		dialog.SetTranslated(true)
		after := atlasMarkerPixels(dialog.buffer)
		require.Equal(t, bounds, dialog.signBounds, "toggle must not move or resize the original sign")
		requireSignFixedInk(t, before, after)
		require.NotEqual(t, before.Pix, after.Pix, "the inscription actually changes")
		require.Positive(t, atlasExactInkCount(after, ucolor.EGAYellow))
		// Check the complete unexpanded row against yellow IBM source bitmaps.
		scale := fittingSignScale(dialog.border.GetInteriorBounds())
		reference := text.NewSignRenderer(float64(scale)).RenderTranslatedSign([]byte("EIGHT@LAWS"))
		expected := atlasMarkerPixels(reference.Image)
		reference.Image.Deallocate()
		start := bounds.Min.Add(image.Pt(3*text.BitmapCharWidth*scale, 3*text.BitmapCharHeight*scale))
		for y := range expected.Bounds().Dy() {
			for x := range expected.Bounds().Dx() {
				want := expected.RGBAAt(x, y)
				if want.A == 0 {
					want = stdcolor.RGBA{A: 255}
				}
				require.Equal(t, want, after.RGBAAt(start.X+x, start.Y+y))
			}
		}
		dialog.SetTranslated(false)
		require.Equal(t, before.Pix, atlasMarkerPixels(dialog.buffer).Pix)
	}
}

func TestSignTranslationOpaqueBackingAndRoundedFrameOverBrightScene(t *testing.T) {
	setGuidanceTestResolution(t, 1280, 720)
	dialog := NewSignDialog(SignDialogConfig{Placement: sprites.PercentBasedPlacement{
		StartPercentX: .1, EndPercentX: .65, StartPercentY: .2, EndPercentY: .8}, RawText: []byte("HELLO")})
	bright := stdcolor.RGBA{R: 255, B: 255, A: 255}
	screen := ebiten.NewImage(1280, 720)
	defer screen.Deallocate()
	for _, translated := range []bool{false, true, false} {
		dialog.SetTranslated(translated)
		screen.Fill(bright)
		dialog.Draw(screen)
		inside := dialog.border.GetInteriorBounds().Min.Add(image.Pt(dialog.pixelUnit*2, dialog.pixelUnit*2))
		require.Equal(t, stdcolor.RGBAModel.Convert(stdcolor.Black), stdcolor.RGBAModel.Convert(screen.At(inside.X, inside.Y)),
			"empty sign interior must occlude the bright scene in both modes")
		_, _, _, cornerAlpha := dialog.buffer.At(0, 0).RGBA()
		require.Zero(t, cornerAlpha, "the opaque backing must still receive the stepped corner mask")
		require.Equal(t, bright, stdcolor.RGBAModel.Convert(screen.At(dialog.bounds.Min.X, dialog.bounds.Min.Y)),
			"the outside corner retains the scene underneath")
		frame := image.Pt(dialog.bounds.Min.X+dialog.bounds.Dx()/2, dialog.bounds.Min.Y+dialog.pixelUnit/2)
		require.Equal(t, stdcolor.RGBAModel.Convert(DialogRuleColor), stdcolor.RGBAModel.Convert(screen.At(frame.X, frame.Y)),
			"translation retains the same outer frame")
	}
}
