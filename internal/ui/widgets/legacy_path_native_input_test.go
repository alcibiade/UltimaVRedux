package widgets

import (
	"fmt"
	"image"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/exp/textinput"
	"github.com/stretchr/testify/require"

	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

func TestNativeEditorSelectionIsVisibleWithoutChangingText(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1280, 720)
	for _, reversed := range []bool{false, true} {
		m, _ := newNativeEditorTest(t, true, "ABCD", map[ebiten.Key]bool{})
		if reversed {
			m.setFieldTextAndSelection("ABCD", 3, 1)
		} else {
			m.setFieldTextAndSelection("ABCD", 1, 3)
		}
		screen := ebiten.NewImage(1280, 720)
		t.Cleanup(screen.Deallocate)
		m.Draw(screen)
		rect := m.pathFieldContentRect(m.calculateRects().field.Inset(m.pixelUnit()))
		prefix, _ := m.font.MeasureString("A")
		selected, _ := m.font.MeasureString("BC")
		pixel := screen.At(rect.Min.X+int(prefix+selected/2), rect.Min.Y+1)
		require.Equal(t, ucolor.From(ucolor.EGAWhite).RGBA(), pixel, "selected range needs a visible background")
		require.Equal(t, "ABCD", m.GetText())
	}
}

// Use the actual pinned Field for committed text and selection storage. Only
// platform event delivery is scripted. In particular, do NOT sort its ranges
// when committing: v2.9.9 Field.commit slices exactly these supplied endpoints.
type scriptedNativeTextField struct {
	textinput.Field
	preedit       string
	nextInput     func()
	externalEdits int
}

func (f *scriptedNativeTextField) HandleInputWithBounds(image.Rectangle) (bool, error) {
	if f.nextInput == nil {
		return false, nil
	}
	f.nextInput()
	f.nextInput = nil
	return true, nil
}

func (f *scriptedNativeTextField) CompositionSelection() (int, int, bool) {
	return len(f.preedit), len(f.preedit), f.IsFocused() && f.preedit != ""
}

func (f *scriptedNativeTextField) TextForRendering() string {
	if !f.IsFocused() || f.preedit == "" {
		return f.Text()
	}
	start, end := f.Selection()
	return f.Text()[:start] + f.preedit + f.Text()[end:]
}

func (f *scriptedNativeTextField) SetTextAndSelection(value string, start, end int) {
	// The production method cleans up the input session, discarding preedit.
	f.externalEdits++
	f.preedit = ""
	f.Field.SetTextAndSelection(value, start, end)
}

func (f *scriptedNativeTextField) commit(value string) {
	start, end := f.Selection()
	committed := f.Text()[:start] + value + f.Text()[end:]
	caret := start + len(value)
	f.Field.SetTextAndSelection(committed, caret, caret)
	f.preedit = ""
}

func newNativeEditorTest(t *testing.T, save bool, initial string, held map[ebiten.Key]bool) (*LegacyPathEntryModal, *scriptedNativeTextField) {
	t.Helper()
	field := new(scriptedNativeTextField)
	keySource := func(key ebiten.Key) bool { return held[key] }
	m := newFocusedLegacyPathEntryModal(LegacyPathEntryModalConfig{
		InitialPath: initial, Keyboard: input.NewKeyboardWithKeyStateSource(0, keySource),
		IsKeyPressed: keySource, OnResult: func(string, bool) {},
	}, field)
	m.saveDescription = save
	m.maxRunes = 80
	var stack DialogStack
	stack.PushModalDialog(m)
	t.Cleanup(m.Dismiss)
	return m, field
}

func TestLegacyEditorCompositionBackspaceDoesNotDeleteCommittedText(t *testing.T) {
	for _, save := range []bool{false, true} {
		t.Run(fmt.Sprintf("save=%t", save), func(t *testing.T) {
			held := map[ebiten.Key]bool{ebiten.KeyBackspace: true}
			m, field := newNativeEditorTest(t, save, "abc", held)
			field.preedit = "xy"
			edits := field.externalEdits
			field.nextInput = func() { field.preedit = "x" }
			m.Update()
			require.Equal(t, "abc", m.GetText())
			require.Equal(t, "abcx", field.TextForRendering())
			require.Equal(t, edits, field.externalEdits, "native preedit session must not be cleaned up")
			for range legacyEntryBackspaceDelay + legacyEntryBackspaceInterval {
				m.Update() // No event this tick, but composition still owns Backspace.
			}
			require.Equal(t, "abcx", field.TextForRendering())
			field.nextInput = func() { field.preedit = "" }
			m.Update()
			for range legacyEntryBackspaceDelay + legacyEntryBackspaceInterval {
				m.Update() // The same held key must not spill into committed text.
			}
			require.Equal(t, "abc", m.GetText())
			require.Equal(t, edits, field.externalEdits)
			held[ebiten.KeyBackspace] = false
			m.Update()
			held[ebiten.KeyBackspace] = true
			m.Update()
			require.Equal(t, "ab", m.GetText(), "fresh Backspace resumes local editing")
		})
	}
}

func TestLegacyEditorCompositionStartCommitAndOwnedShortcuts(t *testing.T) {
	for _, key := range []ebiten.Key{ebiten.KeyBackspace, ebiten.KeyU, ebiten.KeyEnter} {
		t.Run(key.String(), func(t *testing.T) {
			held := map[ebiten.Key]bool{key: true, ebiten.KeyControl: key == ebiten.KeyU}
			m, field := newNativeEditorTest(t, true, "abc", held)
			edits := field.externalEdits
			field.nextInput = func() { field.preedit = "xy" }
			m.Update() // Composition can start during HandleInputWithBounds.
			require.Equal(t, "abcxy", field.TextForRendering())
			require.Equal(t, edits, field.externalEdits)
			field.nextInput = func() { field.commit("x") }
			m.Update()
			for range 30 {
				m.Update()
			}
			require.Equal(t, "abcx", m.GetText())
			require.False(t, m.IsFinished(), "accepting IME text must not submit the modal")
			require.Equal(t, edits, field.externalEdits)
		})
	}
}

func TestLegacyEditorHandledNativeCommitIsNotReinterpretedAsBackspace(t *testing.T) {
	held := map[ebiten.Key]bool{ebiten.KeyBackspace: true}
	m, field := newNativeEditorTest(t, true, "abc", held)
	// A composition can start and commit entirely inside one native input call.
	field.nextInput = func() { field.commit("x") }
	m.Update()
	for range 30 {
		m.Update()
	}
	require.Equal(t, "abcx", m.GetText())
}

func TestLegacyEditorNativeCommitReplacesBackwardSelection(t *testing.T) {
	for _, save := range []bool{false, true} {
		t.Run(fmt.Sprintf("save=%t", save), func(t *testing.T) {
			held := map[ebiten.Key]bool{ebiten.KeyShift: true}
			m, field := newNativeEditorTest(t, save, "abc", held)
			m.keyboard.SimulateKeyPress(ebiten.KeyLeft)
			m.Update()
			start, end := field.Selection()
			require.Equal(t, 2, start)
			require.Equal(t, 3, end)
			anchor, caret := m.directedSelection()
			require.Equal(t, 3, anchor)
			require.Equal(t, 2, caret)
			field.nextInput = func() { field.commit("x") }
			m.Update()
			require.Equal(t, "abx", m.GetText(), "native replacement must not produce abcxc")
			require.False(t, m.selectionReversed)
		})
	}
}

func TestLegacyEditorShiftSelectionDirectionAndUTF8Replacement(t *testing.T) {
	held := map[ebiten.Key]bool{ebiten.KeyShift: true}
	m, field := newNativeEditorTest(t, true, "a\u00e9\u754cz", held)
	m.moveCursorLeft(false) // Anchor before z, allowing selection to cross both ways.
	for _, step := range []struct {
		key   ebiten.Key
		caret int
	}{
		{ebiten.KeyLeft, len("a\u00e9")},
		{ebiten.KeyLeft, len("a")},
		{ebiten.KeyRight, len("a\u00e9")},
		{ebiten.KeyRight, len("a\u00e9\u754c")},
		{ebiten.KeyRight, len("a\u00e9\u754cz")},
		{ebiten.KeyLeft, len("a\u00e9\u754c")},
		{ebiten.KeyLeft, len("a\u00e9")},
	} {
		m.keyboard.SimulateKeyPress(step.key)
		m.Update()
		anchor, caret := m.directedSelection()
		require.Equal(t, len("a\u00e9\u754c"), anchor)
		require.Equal(t, step.caret, caret)
		start, end := field.Selection()
		require.Equal(t, min(anchor, caret), start)
		require.Equal(t, max(anchor, caret), end)
	}
	field.nextInput = func() { field.commit("X") }
	m.Update()
	require.Equal(t, "a\u00e9Xz", m.GetText())
	require.True(t, utf8.ValidString(m.GetText()))
}

func TestSaveDescriptionNativeReplacementAtRuneLimit(t *testing.T) {
	for _, selectAll := range []bool{false, true} {
		t.Run(fmt.Sprintf("selectAll=%t", selectAll), func(t *testing.T) {
			held := map[ebiten.Key]bool{ebiten.KeyShift: true}
			initial := strings.Repeat("\u754c", 80)
			m, field := newNativeEditorTest(t, true, initial, held)
			count := 1
			replacement := "X"
			expected := strings.Repeat("\u754c", 79) + replacement
			if selectAll {
				count = 80
				replacement = strings.Repeat("\u00e9", 80)
				expected = replacement
			}
			for range count {
				m.keyboard.SimulateKeyPress(ebiten.KeyLeft)
				m.Update()
			}
			field.nextInput = func() { field.commit(replacement) }
			m.Update()
			require.Equal(t, expected, m.GetText())
			require.Equal(t, 80, utf8.RuneCountInString(m.GetText()))
			require.Empty(t, m.GetValidationMessage())
		})
	}
}

func TestSaveDescriptionNativeClearShortcutAtRuneLimit(t *testing.T) {
	held := map[ebiten.Key]bool{ebiten.KeyControl: true, ebiten.KeyU: true}
	m, field := newNativeEditorTest(t, true, strings.Repeat("x", 80), held)
	field.nextInput = func() { field.commit("u") }
	m.Update()
	require.Empty(t, m.GetText(), "the shortcut must clear, not reject a transient 81st character")
	require.Empty(t, m.GetValidationMessage())
}
