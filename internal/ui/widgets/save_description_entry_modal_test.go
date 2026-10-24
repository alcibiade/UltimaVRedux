package widgets

import (
	"fmt"
	"image"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

func newDescriptionEditorTest(t *testing.T, initial string, held map[ebiten.Key]bool) (*SaveDescriptionEntryModal, *fakeSaveDescriptionTextField) {
	t.Helper()
	keySource := func(key ebiten.Key) bool { return held[key] }
	field := new(fakeSaveDescriptionTextField)
	modal := newFocusedSaveDescriptionEntryModal(SaveDescriptionEntryModalConfig{
		Title: "Save To Slot 2", InitialText: initial, MaxRunes: 80,
		Keyboard:     input.NewKeyboardWithKeyStateSource(0, keySource),
		IsKeyPressed: keySource, OnResult: func(string, bool) {},
	}, field)
	var stack DialogStack
	stack.PushModalDialog(modal)
	t.Cleanup(func() { modal.Dismiss() })
	return modal, field
}

func TestSaveDescriptionBackspaceCadenceAndNativeArbitration(t *testing.T) {
	held := map[ebiten.Key]bool{}
	m, field := newDescriptionEditorTest(t, "abc\u00e9\u754c", held)
	held[ebiten.KeyBackspace] = true
	// Model platforms that deliver native deletion before explicit key handling.
	field.handleText = "abc\u00e9"
	m.Update()
	require.Equal(t, "abc\u00e9", m.GetText())
	for range saveDescriptionBackspaceDelay - 1 {
		field.handleText = "abc"
		m.Update()
		require.Equal(t, "abc\u00e9", m.GetText())
	}
	m.Update()
	require.Equal(t, "abc", m.GetText())
	for range saveDescriptionBackspaceInterval - 1 {
		m.Update()
		require.Equal(t, "abc", m.GetText())
	}
	m.Update()
	require.Equal(t, "ab", m.GetText())
	held[ebiten.KeyBackspace] = false
	for range 30 {
		m.Update()
	}
	require.Equal(t, "ab", m.GetText())
	held[ebiten.KeyBackspace] = true
	m.Update()
	require.Equal(t, "a", m.GetText(), "repress deletes immediately")
	for range 60 {
		m.Update()
	}
	require.Empty(t, m.GetText())
	require.True(t, utf8.ValidString(m.GetText()))
}

func TestSaveDescriptionBackspaceSelectionAndFocusReset(t *testing.T) {
	held := map[ebiten.Key]bool{ebiten.KeyBackspace: true}
	m, field := newDescriptionEditorTest(t, "a\u00e9\u754cz", held)
	field.SetTextAndSelection(m.GetText(), 1, len("a\u00e9\u754c"))
	m.Update()
	require.Equal(t, "az", m.GetText())
	start, end := field.Selection()
	require.Equal(t, 1, start)
	require.Equal(t, start, end)
	for range saveDescriptionBackspaceDelay {
		m.Update()
	}
	require.Equal(t, "z", m.GetText())
	m.SetText("again")
	m.onDialogFocusChanged(false)
	for range 30 {
		m.Update()
	}
	require.Equal(t, "again", m.GetText())
	require.False(t, m.backspaceHeld)
	m.onDialogFocusChanged(true)
	m.Update()
	require.Equal(t, "agai", m.GetText())
	m.Dismiss()
	require.False(t, m.backspaceHeld)
	for range 30 {
		m.Update()
	}
	require.Equal(t, "agai", m.GetText())
}

func TestSaveDescriptionClearChordDrainsPlatformAndDebounces(t *testing.T) {
	held := map[ebiten.Key]bool{ebiten.KeyControl: true, ebiten.KeyU: true}
	m, field := newDescriptionEditorTest(t, "Keep case", held)
	field.handleText = "Keep caseu"
	m.Update()
	require.Empty(t, m.GetText())
	start, end := field.Selection()
	require.Zero(t, start)
	require.Zero(t, end)
	m.SetText("new text")
	field.handleText = "new textu"
	m.Update()
	require.Equal(t, "new text", m.GetText())
	held[ebiten.KeyControl] = false
	m.Update()
	held[ebiten.KeyControl] = true
	m.Update()
	require.Empty(t, m.GetText())
	require.False(t, m.IsFinished())
	require.Equal(t, 4, field.handleCalls)
}

func TestSaveDescriptionWindowFocusStopsEditing(t *testing.T) {
	held := map[ebiten.Key]bool{ebiten.KeyBackspace: true}
	m, field := newDescriptionEditorTest(t, "abc", held)
	m.Update()
	require.Equal(t, "ab", m.GetText())
	m.isWindowFocused = func() bool { return false }
	for range 60 {
		m.Update()
	}
	require.Equal(t, "ab", m.GetText())
	require.False(t, m.backspaceHeld)
	require.Equal(t, 1, field.handleCalls)
	m.isWindowFocused = func() bool { return true }
	m.Update()
	require.Equal(t, "a", m.GetText())
}

func TestSaveDescriptionEscapeStopsHeldBackspace(t *testing.T) {
	held := map[ebiten.Key]bool{ebiten.KeyBackspace: true}
	m, _ := newDescriptionEditorTest(t, "abc", held)
	m.Update()
	require.Equal(t, "ab", m.GetText())
	held[ebiten.KeyEscape] = true
	m.Update()
	require.True(t, m.IsFinished())
	require.False(t, m.backspaceHeld)
	require.Equal(t, SaveDescriptionEntryResult{Text: "ab", Submitted: false}, m.GetResult())
}

func TestSaveDescriptionEnterOpeningReleaseAndSingleDispatch(t *testing.T) {
	for _, key := range []ebiten.Key{ebiten.KeyEnter, ebiten.KeyNumpadEnter} {
		t.Run(fmt.Sprint(key), func(t *testing.T) {
			held := map[ebiten.Key]bool{key: true}
			m, _ := newDescriptionEditorTest(t, "", held)
			calls := 0
			m.onResult = func(value string, submitted bool) {
				calls++
				require.True(t, submitted)
				require.Empty(t, value)
			}
			for range 60 {
				m.Update()
			}
			require.Zero(t, calls)
			held[key] = false
			m.keyboard.ClearFrameConsumption()
			m.Update()
			held[key] = true
			for range 60 {
				m.Update()
			}
			require.Equal(t, 1, calls)
		})
	}
}

func TestSaveDescriptionLimitsPreservePreviousValueAndSelection(t *testing.T) {
	m, field := newDescriptionEditorTest(t, "Keep case", nil)
	for _, invalid := range []string{strings.Repeat("\u754c", 81), "two\nlines", "trailing\n", "tab\t", "bad\x00", string([]byte{0xff})} {
		t.Run(fmt.Sprintf("%q", invalid), func(t *testing.T) {
			field.SetTextAndSelection("Keep case", 2, 6)
			field.handleText = invalid
			m.Update()
			require.Equal(t, "Keep case", m.GetText())
			start, end := field.Selection()
			require.Equal(t, 2, start)
			require.Equal(t, 6, end)
			require.NotEmpty(t, m.GetValidationMessage())
			m.SetText(invalid)
			require.Equal(t, "Keep case", m.GetText())
			start, end = field.Selection()
			require.Equal(t, 2, start)
			require.Equal(t, 6, end)
		})
	}
	valid := strings.Repeat("\u754c", 80)
	field.handleText = valid
	m.Update()
	require.Equal(t, valid, m.GetText())
	require.Empty(t, m.GetValidationMessage())
	m.SetText("About to go to Doom!")
	require.Equal(t, "About to go to Doom!", m.GetText())
}

func TestSaveDescriptionValidationCancelAndClearedSubmit(t *testing.T) {
	m, _ := newDescriptionEditorTest(t, "original", nil)
	m.validate = func(string) string { return "Review destination again." }
	m.Submit()
	require.False(t, m.IsFinished())
	require.Equal(t, "Review destination again.", m.GetValidationMessage())
	m.keyboard.SimulateKeyPress(ebiten.KeyEscape)
	m.Update()
	require.Equal(t, SaveDescriptionEntryResult{Text: "original", Submitted: false}, m.GetResult())
	m, _ = newDescriptionEditorTest(t, "original", nil)
	m.SetText("")
	m.Submit()
	require.Equal(t, SaveDescriptionEntryResult{Text: "", Submitted: true}, m.GetResult())
}

func TestSaveDescriptionMeasuredLayoutAtSupportedSizes(t *testing.T) {
	for _, size := range config.SupportedWindowResolutions() {
		t.Run(fmt.Sprintf("%dx%d", size.X, size.Y), func(t *testing.T) {
			setResponsiveDialogTestResolution(t, size.X, size.Y)
			m, _ := newDescriptionEditorTest(t, strings.Repeat("Checkpoint ", 7), nil)
			m.notice = "This slot belongs to another playthrough. Replaces Derp's save from Sep 7, 2026, 2:35 PM PDT. " +
				"Saving here permanently replaces that checkpoint, including its party, inventory, location, and progress."
			m.validationMessage = "Description must be at most 80 characters."
			rects := m.preparePresentation(size.X, size.Y)
			t.Cleanup(m.border.Dispose)
			require.True(t, m.border.GetBounds().In(image.Rect(0, 0, size.X, size.Y)))
			previousBottom := 0
			for _, rect := range []image.Rectangle{rects.title, rects.prompt, rects.field, rects.notice, rects.message, rects.hint} {
				require.True(t, rect.In(m.border.GetInteriorBounds()), "%v outside %v", rect, m.border.GetInteriorBounds())
				require.Greater(t, rect.Min.Y, previousBottom)
				previousBottom = rect.Max.Y
			}
			require.LessOrEqual(t, m.noticeOutput.GetTotalHeightForLines(m.noticeOutput.GetPanelContentLineCount()), float64(rects.notice.Dy()))
			require.Equal(t, GetScaledFont(FontSizeTitle).TextFace.Size, m.titleOutput.Font.TextFace.Size)
			require.Contains(t, strings.Join(strings.Fields(m.noticeOutput.GetOutputStr(false)), " "), m.notice)
			require.Contains(t, m.hintOutput.GetOutputStr(false), "[Ctrl+U] Clear description")
			require.Contains(t, m.hintOutput.GetOutputStr(false), "[Enter] Save")
			require.Contains(t, m.hintOutput.GetOutputStr(false), "[Esc] Back")
			if size.X != 1280 && size.X != 1366 {
				return
			}
			screen := ebiten.NewImage(size.X, size.Y)
			t.Cleanup(screen.Dispose)
			m.Draw(screen)
			requireImageRegionContainsColor(t, screen, rects.hint, DialogControlKeyColor)
			requireImageRegionContainsColor(t, screen, rects.hint, DialogControlHintColor)
			screen.Clear()
			m.titleOutput.DrawPanelContent(screen,
				image.Pt(rects.title.Min.X, centeredGlyphOutputY(rects.title, m.titleOutput, m.title)),
				rects.title.Dx(), etext.AlignCenter)
			glyphs := renderedNontransparentBounds(screen, m.border.GetBounds())
			require.True(t, glyphs.In(rects.title))
			require.InDelta(t, rects.title.Min.Y+rects.title.Max.Y, glyphs.Min.Y+glyphs.Max.Y, 3)
			require.InDelta(t, rects.title.Min.X+rects.title.Max.X, glyphs.Min.X+glyphs.Max.X, 3)
		})
	}
}
