package widgets

import (
	"fmt"
	"image"
	"math"
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/stretchr/testify/require"
)

func TestAtlasNoteEditorRenderedFooterAndValidationFitSupportedSizes(t *testing.T) {
	labelError := journal.ValidateAtlasNoteText("", "")
	characterError := journal.ValidateAtlasNoteText("Label", "unsupported\tcontrol")
	require.Error(t, labelError)
	require.Error(t, characterError)
	// Actual field-validation and draft/operation errors, including the longest
	// capacity explanation. No arbitrary repeated-character layout surrogate.
	messages := []string{
		"",
		fmt.Errorf("save atlas note: %w", labelError).Error(),
		fmt.Errorf("save atlas note: %w", characterError).Error(),
		fmt.Errorf("%w: map notes full (%d); edit or delete a note",
			journal.ErrInvalidAtlasPersonal, journal.MaxAtlasNotes).Error(),
		fmt.Errorf("save atlas note: %w: captured fix has changed", journal.ErrInvalidAtlasPersonal).Error(),
		fmt.Errorf("save atlas note: %w: stale note session", journal.ErrInvalidAtlasPersonal).Error(),
		fmt.Errorf("save atlas note: %w: position already has a note", journal.ErrInvalidAtlasPersonal).Error(),
		fmt.Errorf("delete atlas note: %w: note changed or was deleted", journal.ErrInvalidAtlasPersonal).Error(),
		fmt.Sprintf("This field allows at most %d characters.", journal.MaxAtlasNoteDescriptionRunes),
	}
	for _, size := range config.SupportedWindowResolutions() {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			setQuestJournalTestResolution(t, size.X, size.Y)
			editor := NewAtlasNoteEditor(AtlasNoteEditorConfig{
				Record:   journal.AtlasNoteRecord{Label: strings.Repeat("M", 32), Description: strings.Repeat("Paragraph\n", 50)},
				Keyboard: newWidgetTestKeyboard(0), Submit: func(string, string) error { return nil }, Cancel: func() {},
			})
			var screen *ebiten.Image
			if size.X == 1280 || size.X == 1366 {
				screen = ebiten.NewImageWithOptions(editor.pane, nil)
				t.Cleanup(screen.Dispose)
			}
			for messageIndex, message := range messages {
				editor.message = message
				for _, cursor := range []int{0, 500} {
					editor.focus = 1
					editor.cursors[1] = cursor
					editor.ensureLayout()
					editor.prepareFieldScroll(1, editor.fieldLayout(1))
					var rendered *ebiten.Image
					if screen != nil && cursor == 500 && (messageIndex == 0 || messageIndex == 3 || messageIndex == len(messages)-1) {
						screen.Clear()
						editor.Draw(screen)
						rendered = screen
					}
					require.Equal(t, "[Tab/Shift+Tab] Field [Enter] Paragraph [Ctrl+Enter] Save [Esc] Cancel", atlasStatusWords(editor.output))
					require.Equal(t, 2, editor.output.GetPanelContentLineCount(), "actual measured key/action rows")
					require.Equal(t, strings.Join(strings.Fields(message), " "), atlasStatusWords(editor.validation))
					require.Equal(t, GetScaledFont(FontSizeBody).TextFace.Size, editor.font.TextFace.Size)
					require.Equal(t, editor.font.TextFace.Size, editor.validation.Font.TextFace.Size)
					require.True(t, editor.hintRect.In(editor.pane.Inset(ClassicUIPixelUnit(size.X, size.Y))))
					require.True(t, editor.validationRect.In(editor.pane))
					require.LessOrEqual(t, editor.fieldRects[1].Max.Y, editor.validationRect.Min.Y)
					require.LessOrEqual(t, editor.validationRect.Max.Y, editor.hintRect.Min.Y)
					require.Greater(t, editor.fieldRects[1].Dy(), editor.lineHeight)
					assertAtlasEditorTextBlockRendered(t, editor.output, editor.hintRect, rendered, editor.lineHeight)
					assertAtlasEditorTextBlockRendered(t, editor.validation, editor.validationRect, rendered, editor.lineHeight)
				}
			}
		})
	}
}

func TestAtlasNoteDeleteFooterKeepsBothActionsVisible(t *testing.T) {
	for _, size := range config.SupportedWindowResolutions() {
		setQuestJournalTestResolution(t, size.X, size.Y)
		editor := NewAtlasNoteEditor(AtlasNoteEditorConfig{Deleting: true, Record: journal.AtlasNoteRecord{Label: "Landing"},
			Keyboard: newWidgetTestKeyboard(0), Submit: func(string, string) error { return nil }, Cancel: func() {}})
		editor.ensureLayout()
		var screen *ebiten.Image
		if size.X == 1280 || size.X == 1366 {
			screen = ebiten.NewImageWithOptions(editor.pane, nil)
			editor.Draw(screen)
		}
		require.Equal(t, "[Enter] Delete [Esc] Keep", atlasStatusWords(editor.output))
		require.Equal(t, 1, editor.output.GetPanelContentLineCount())
		assertAtlasEditorTextBlockRendered(t, editor.output, editor.hintRect, screen, editor.lineHeight)
		if screen != nil {
			screen.Dispose()
		}
	}
}

func assertAtlasEditorTextBlockRendered(t *testing.T, output *text.Output, rect image.Rectangle, screen *ebiten.Image, padding int) {
	t.Helper()
	require.LessOrEqual(t, int(math.Ceil(output.GetTotalHeightForLines(output.GetPanelContentLineCount()))), rect.Dy())
	for _, line := range strings.Split(output.GetOutputStr(false), "\n") {
		width, _ := output.Font.MeasureString(strings.TrimSpace(line))
		require.LessOrEqual(t, width, float64(rect.Dx()))
	}
	if screen == nil {
		return
	}
	// Render onto a larger transparent surface first: glyphs beyond the reserved
	// right/bottom edges are visible to this assertion instead of being clipped
	// identically by the actual and reference images.
	unclipped := ebiten.NewImage(rect.Dx()+padding, rect.Dy()+padding)
	defer unclipped.Deallocate()
	output.DrawPanelContent(unclipped, image.Point{}, rect.Dx(), etext.AlignStart)
	pixels := atlasMarkerPixels(unclipped)
	var ink image.Rectangle
	for y := range pixels.Bounds().Dy() {
		for x := range pixels.Bounds().Dx() {
			if pixels.RGBAAt(x, y).A != 0 {
				ink = ink.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	if strings.TrimSpace(output.GetOutputStr(false)) != "" {
		require.False(t, ink.Empty())
	}
	if !ink.Empty() {
		require.True(t, ink.In(image.Rect(0, 0, rect.Dx(), rect.Dy())), "unclipped glyphs %v exceed reserved %v", ink, rect.Size())
	}
	expected := ebiten.NewImage(rect.Dx(), rect.Dy())
	defer expected.Deallocate()
	expected.Fill(ucolor.DialogBackground.RGBA())
	expected.DrawImage(unclipped, nil)
	require.Equal(t, atlasMarkerPixels(expected).Pix, atlasMarkerPixels(atlasSubImage(screen, rect)).Pix,
		"all retained hint/validation glyphs must actually render inside their reserved block")
}
