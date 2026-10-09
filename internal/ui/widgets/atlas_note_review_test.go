package widgets

import (
	"math"
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func TestAtlasNoteEditorDrainsTextBeforeSameFrameControls(t *testing.T) {
	setQuestJournalTestResolution(t, 1280, 720)
	for _, test := range []struct {
		name               string
		focus              int
		keys               []ebiten.Key
		label, description string
		nextFocus, saves   int
	}{
		{name: "Tab", keys: []ebiten.Key{ebiten.KeyTab}, label: "Ax", description: "B", nextFocus: 1},
		{name: "Shift Tab", focus: 1, keys: []ebiten.Key{ebiten.KeyShiftLeft, ebiten.KeyTab}, label: "A", description: "Bx"},
		{name: "label Enter", keys: []ebiten.Key{ebiten.KeyEnter}, label: "Ax", description: "B"},
		{name: "description Enter", focus: 1, keys: []ebiten.Key{ebiten.KeyEnter}, label: "A", description: "Bx\n", nextFocus: 1},
		{name: "label save", keys: []ebiten.Key{ebiten.KeyControlLeft, ebiten.KeyEnter}, label: "Ax", description: "B", saves: 1},
		{name: "description save", focus: 1, keys: []ebiten.Key{ebiten.KeyControlLeft, ebiten.KeyEnter}, label: "A", description: "Bx", nextFocus: 1, saves: 1},
		{name: "backspace", focus: 1, keys: []ebiten.Key{ebiten.KeyBackspace}, label: "A", description: "B", nextFocus: 1},
		{name: "cancel", keys: []ebiten.Key{ebiten.KeyEscape}, label: "A", description: "B"},
		{name: "unowned shortcut", keys: []ebiten.Key{ebiten.KeyControlLeft, ebiten.KeyS}, label: "A", description: "B"},
	} {
		t.Run(test.name, func(t *testing.T) {
			keys := map[ebiten.Key]bool{}
			keyboard := input.NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool { return keys[key] })
			saves := 0
			editor := NewAtlasNoteEditor(AtlasNoteEditorConfig{
				Record: journal.AtlasNoteRecord{Label: "A", Description: "B"}, Keyboard: keyboard,
				Submit: func(label, description string) error {
					require.Equal(t, test.label, label)
					require.Equal(t, test.description, description)
					saves++
					return nil
				}, Cancel: func() {},
			})
			editor.focus = test.focus
			editor.appendChars = func(dst []rune) []rune { return append(dst, []rune("x\t\r\n\b")...) }
			for _, key := range test.keys {
				keys[key] = true
			}
			keyboard.ClearFrameConsumption()
			editor.Update()
			require.Equal(t, test.label, string(editor.fields[0]))
			require.Equal(t, test.description, string(editor.fields[1]))
			require.Equal(t, test.nextFocus, editor.focus)
			require.Equal(t, test.saves, saves)
		})
	}
}

func TestAtlasNoteEditorRejectedPendingTextCannotSubmit(t *testing.T) {
	setQuestJournalTestResolution(t, 1280, 720)
	keys := map[ebiten.Key]bool{ebiten.KeyControlLeft: true, ebiten.KeyEnter: true}
	keyboard := input.NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool { return keys[key] })
	saves := 0
	editor := NewAtlasNoteEditor(AtlasNoteEditorConfig{Record: journal.AtlasNoteRecord{Label: strings.Repeat("a", 32)},
		Keyboard: keyboard, Submit: func(string, string) error { saves++; return nil }, Cancel: func() {}})
	editor.appendChars = func(dst []rune) []rune { return append(dst, 'x') }
	editor.Update()
	require.Zero(t, saves)
	require.False(t, editor.finished)
	require.Contains(t, editor.message, "32")
	require.Equal(t, strings.Repeat("a", 32), string(editor.fields[0]))
}

func TestAtlasNoteEditorVerticalSoftWrapCaretAffinity(t *testing.T) {
	setQuestJournalTestResolution(t, 1280, 720)
	editor := NewAtlasNoteEditor(AtlasNoteEditorConfig{Keyboard: newWidgetTestKeyboard(0),
		Submit: func(string, string) error { return nil }, Cancel: func() {}})
	editor.focus = 1
	width, _ := editor.font.MeasureString("MMMM")
	fieldInsets := editor.fieldRects[1].Dx() - editor.fieldLayout(1).textWidth
	editor.fieldRects[1].Max.X = editor.fieldRects[1].Min.X + int(math.Ceil(width)) + fieldInsets
	editor.insert([]rune(strings.Repeat("M", 12)))
	lines := editor.lines(1)
	require.Equal(t, []atlasEditorLine{{0, 4}, {4, 8}, {8, 12}}, lines)
	for _, move := range []struct {
		key         ebiten.Key
		cursor, row int
	}{
		{ebiten.KeyUp, 8, 1}, {ebiten.KeyUp, 4, 0}, {ebiten.KeyDown, 8, 1}, {ebiten.KeyDown, 12, 2},
	} {
		editor.handleKey(move.key)
		require.Equal(t, move.cursor, editor.cursors[1])
		require.Equal(t, move.row, editor.cursorLine(1, lines))
	}
	editor.insert([]rune(strings.Repeat("M", 24)))
	lines = editor.lines(1)
	require.Len(t, lines, 9)
	editor.handleKey(ebiten.KeyPageUp)
	require.Equal(t, 16, editor.cursors[1])
	require.Equal(t, 3, editor.cursorLine(1, lines))
	editor.handleKey(ebiten.KeyPageDown)
	require.Equal(t, 36, editor.cursors[1])
	require.Equal(t, 8, editor.cursorLine(1, lines))
	editor.handleKey(ebiten.KeyUp)
	editor.handleKey(ebiten.KeyLeft)
	require.False(t, editor.upstream[1], "horizontal editing resets soft-wrap affinity")
	editor.insert([]rune("X"))
	require.False(t, editor.upstream[1])
	// Explicit newlines have distinct offsets and keep ordinary paragraph motion.
	editor.fields[1], editor.cursors[1] = []rune("MMMM\n\nMMMM"), 10
	editor.handleKey(ebiten.KeyUp)
	require.Equal(t, 5, editor.cursors[1])
	require.Equal(t, 1, editor.cursorLine(1, editor.lines(1)))
}

func TestAtlasNoteReaderPreservesExplicitPunctuationAndWhitespaceParagraphs(t *testing.T) {
	setQuestJournalTestResolution(t, 1280, 720)
	view := NewAtlasViewState()
	description := "Check here\n???\n\n   \n  Keep  spacing  \n!!!\n"
	note := journal.AtlasNoteRecord{ID: 1, Label: "Directions", Description: description}
	dialog := NewAtlasDialog(journal.AtlasDossier{Entries: []journal.AtlasEntry{{Key: "note-1", Name: note.Label,
		Category: journal.AtlasNotes, Kind: "note", Note: note}}}, nil, newWidgetTestKeyboard(0), &view, func() {})
	screen := ebiten.NewImage(1280, 720)
	defer screen.Deallocate()
	dialog.drawDetails(screen)
	expected := strings.Join(append([]string{"Description:"}, strings.Split(description, "\n")...), " \n")
	require.Contains(t, dialog.details.GetOutputStr(true), expected)
	require.NotContains(t, dialog.details.GetOutputStr(true), "Check here???")
	// Measured soft wrapping must also retain spaces, even at row boundaries.
	dialog.details.Reset()
	width, _ := dialog.font.MeasureString("MMMM")
	dialog.detailRect.Max.X = dialog.detailRect.Min.X + int(math.Ceil(width)) + 8
	paragraph := "AA  BB   CC  DD"
	dialog.appendNoteDescription(paragraph)
	output := dialog.details.GetOutputStr(true)
	_, body, found := strings.Cut(output, "Description: \n")
	require.True(t, found)
	require.Equal(t, paragraph, strings.ReplaceAll(body, " \n", ""))
}
