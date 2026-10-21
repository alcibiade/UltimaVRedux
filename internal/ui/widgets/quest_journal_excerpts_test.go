package widgets

import (
	"image"
	"math"
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func TestExchangePagingFooterFitsSupportedResolutions(t *testing.T) {
	for _, resolution := range append(config.SupportedWindowResolutions(), config.ScreenResolution{X: 320, Y: 200}) {
		setQuestJournalTestResolution(t, resolution.X, resolution.Y)
		configuration := testQuestJournalConfig(newWidgetTestKeyboard(0))
		configuration.LastConversation = journal.Conversation{Exchanges: []journal.Exchange{{
			ID: "long", Speaker: "Gardner", Location: "Keep", Input: "JOB",
			Passages: []journal.Passage{{Text: strings.Repeat("A long passage\n", 128), Runic: false}},
		}}, Truncated: false}
		dialog := NewQuestJournalDialog(testQuestJournalDossier(), configuration)
		dialog.switchTabBy(int(QuestJournalConversationTab))
		dialog.pageSelectedExchange(1)
		require.Greater(t, dialog.GetDetailPageCount(), 2)
		width := dialog.footerContentRect().Dx()
		lines := packedFooterRows(dialog.font, dialog.footerGuidance(), width)
		footer := strings.Join(lines, "\n")
		require.LessOrEqual(t, len(lines), questJournalFooterLines, "%dx%d: %s", resolution.X, resolution.Y, footer)
		for _, line := range lines {
			measured, _ := dialog.font.MeasureString(line)
			require.LessOrEqual(t, measured, float64(width), "%dx%d: %s", resolution.X, resolution.Y, line)
		}
		require.Contains(t, footer, "Space")
		require.NotContains(t, footer, "PgUp")
		require.NotContains(t, footer, "PgDn")
	}
}

func TestRecordedRunesCanUseReferenceWithoutLosingReaderPosition(t *testing.T) {
	setQuestJournalTestResolution(t, 1280, 720)
	held := map[ebiten.Key]bool{}
	keyboard := input.NewKeyboardWithKeyStateSource(-1, func(key ebiten.Key) bool { return held[key] })
	configuration := testQuestJournalConfig(keyboard)
	configuration.LastConversation = journal.Conversation{Exchanges: []journal.Exchange{{
		ID: "runes", Speaker: "Iolo", Location: "Britain",
		Passages: []journal.Passage{{Text: strings.Repeat("RUNIC WORDS\n", 80), Runic: true}},
	}}}
	configuration.Excerpts = cloneJournalExchanges(configuration.LastConversation.Exchanges)
	configuration.DeleteExcerpt = func(string) ([]journal.Exchange, error) {
		t.Fatal("reference must not expose notebook deletion")

		return nil, nil
	}
	dialog := NewQuestJournalDialog(testQuestJournalDossier(), configuration)
	dialog.switchTabBy(int(QuestJournalExcerptsTab))
	dialog.pageSelectedExchange(1)
	page, row := dialog.detailPage, dialog.passageRow
	held[ebiten.KeyControl], held[ebiten.KeyR] = true, true
	dialog.Update()
	require.True(t, dialog.runicReferenceOpen)
	screen := ebiten.NewImage(1280, 720)
	t.Cleanup(screen.Deallocate)
	dialog.Draw(screen)
	require.Equal(t, dialog.layout.listRect, dialog.runicReference.bounds)
	require.False(t, dialog.runicReference.bounds.Overlaps(dialog.exchangeContentRect()))
	clear(held)
	keyboard.ClearFrameConsumption()
	dialog.Update()
	pressQuestJournalKey(dialog, keyboard, ebiten.KeyD)
	require.False(t, dialog.deletePending)
	pressQuestJournalKey(dialog, keyboard, ebiten.KeyEnter)
	pressQuestJournalKey(dialog, keyboard, ebiten.KeyEscape)
	require.False(t, dialog.runicReferenceOpen)
	require.False(t, dialog.dismissed)
	require.Equal(t, page, dialog.detailPage)
	require.Equal(t, row, dialog.passageRow)
}

func TestLastConversationReaderShowsWholeTranscriptAndKeepsSelectedExchange(t *testing.T) {
	setQuestJournalTestResolution(t, 640, 400)
	keyboard := newWidgetTestKeyboard(0)
	config := testQuestJournalConfig(keyboard)
	config.LastConversation = journal.Conversation{Exchanges: []journal.Exchange{
		{ID: "greeting", Speaker: "Gardner", Location: "SERPENT'S HOLD", Input: "",
			Passages: []journal.Passage{{Text: "You see a sincere man.", Runic: false}}},
		{ID: "name", Speaker: "Gardner", Location: "SERPENT'S HOLD", Input: "NAME",
			Passages: []journal.Passage{{Text: "My name is Gardner.", Runic: false}}},
		{ID: "job", Speaker: "Gardner", Location: "SERPENT'S HOLD", Input: "JOB",
			Passages: []journal.Passage{{Text: strings.Repeat("I tend the Flame.\n", 64), Runic: false}}},
	}, Truncated: false}
	var kept journal.Exchange
	config.KeepExcerpt = func(exchange journal.Exchange) ([]journal.Exchange, error) {
		kept = exchange

		return []journal.Exchange{exchange}, nil
	}
	dialog := NewQuestJournalDialog(testQuestJournalDossier(), config)
	dialog.switchTabBy(int(QuestJournalConversationTab))
	screen := ebiten.NewImage(640, 400)
	dialog.Draw(screen)
	transcript := dialog.exchangeOutput.GetOutputStr(false)
	for _, expected := range []string{"Gardner - SERPENT'S HOLD", "sincere man", "Avatar: NAME", "My name is Gardner", "Avatar: JOB", "I tend the Flame"} {
		require.Contains(t, transcript, expected)
	}
	require.NotContains(t, transcript, "—")
	require.Equal(t, "Gardner - Greeting", dialog.listRowText(0))
	require.Len(t, dialog.conversationRows, 3)
	dialog.moveSelection(2)
	dialog.Draw(screen)
	require.Equal(t, dialog.conversationRows[2], dialog.passageRow)
	pressQuestJournalKey(dialog, keyboard, ebiten.KeyK)
	require.Equal(t, "job", kept.ID)
	dialog.Draw(screen)
	dialog.moveSelection(-1)
	require.Equal(t, 1, dialog.GetSelectedIndex())
	pressQuestJournalKey(dialog, keyboard, ebiten.KeyK)
	require.Equal(t, "name", kept.ID)
}

func TestExchangePagingUsesOnlySpaceAndKeepsSelection(t *testing.T) {
	for _, tab := range []QuestJournalTab{QuestJournalConversationTab, QuestJournalExcerptsTab} {
		t.Run(questJournalTabTitle(tab), func(t *testing.T) {
			setQuestJournalTestResolution(t, 640, 400)
			held := map[ebiten.Key]bool{}
			keyboard := input.NewKeyboardWithKeyStateSource(-1, func(key ebiten.Key) bool { return held[key] })
			config := testQuestJournalConfig(keyboard)
			config.LastConversation = journal.Conversation{Exchanges: []journal.Exchange{
				{ID: "first", Speaker: "Gardner", Location: "Keep", Input: "", Passages: []journal.Passage{{Text: "Greeting", Runic: false}}},
				{ID: "long", Speaker: "Gardner", Location: "Keep", Input: "JOB",
					Passages: []journal.Passage{{Text: strings.Repeat("Long response\n", 80) + "The end.", Runic: false}}},
				{ID: "last", Speaker: "Gardner", Location: "Keep", Input: "BYE", Passages: []journal.Passage{{Text: "Farewell", Runic: false}}},
			}, Truncated: false}
			config.Excerpts = cloneJournalExchanges(config.LastConversation.Exchanges)
			var captured journal.Exchange
			config.KeepExcerpt = func(exchange journal.Exchange) ([]journal.Exchange, error) {
				captured = exchange.Clone()

				return []journal.Exchange{exchange.Clone()}, nil
			}
			dialog := NewQuestJournalDialog(testQuestJournalDossier(), config)
			dialog.switchTabBy(int(tab))
			dialog.moveSelection(1)
			pages := dialog.GetDetailPageCount()
			require.Greater(t, pages, 2)
			start, end := dialog.selectedExchangeSpan()
			capacity := dialog.exchangeOutput.GetLineCapacity(dialog.exchangeContentRect().Dy())
			press := func(key ebiten.Key, shift bool) {
				keyboard.SetAllowKeyPressImmediately()
				held[key], held[ebiten.KeyShift] = true, shift
				dialog.Update()
				delete(held, key)
				delete(held, ebiten.KeyShift)
				keyboard.ClearFrameConsumption()
			}
			covered := start
			for page := range pages {
				require.Equal(t, page, dialog.GetDetailPage())
				require.Equal(t, covered, dialog.passageRow, "every selected row must be reachable without overlap or omission")
				if tab == QuestJournalConversationTab {
					require.Equal(t, dialog.exchangeContentRect().Min.Y, dialog.exchangeHighlightRect().Min.Y)
				} else {
					require.True(t, dialog.exchangeHighlightRect().Empty())
				}
				require.Equal(t, 1, dialog.GetSelectedIndex())
				covered = min(end, covered+capacity)
				if page == 1 && tab == QuestJournalConversationTab {
					top := dialog.passageRow
					press(ebiten.KeyK, false)
					require.Equal(t, config.LastConversation.Exchanges[1], captured, "K keeps the whole exchange, not its visible page")
					require.Equal(t, page, dialog.GetDetailPage())
					require.Equal(t, top, dialog.passageRow, "capture feedback must not jump the reader")
				}
				press(ebiten.KeySpace, false)
			}
			require.Equal(t, end, covered)
			require.Equal(t, pages-1, dialog.GetDetailPage(), "Space at the end must not wrap")
			require.NotContains(t, dialog.footerGuidance(), "[Space] Next")
			require.Contains(t, dialog.footerGuidance(), "[Shift+Space] Previous")
			for range pages {
				press(ebiten.KeySpace, true)
			}
			require.Zero(t, dialog.GetDetailPage(), "Shift+Space must stop at the beginning")
			require.Equal(t, start, dialog.passageRow)
			for _, key := range []ebiten.Key{ebiten.KeyPageUp, ebiten.KeyPageDown} {
				press(key, false)
				require.Zero(t, dialog.GetDetailPage())
				require.Equal(t, 1, dialog.GetSelectedIndex())
			}
			held[ebiten.KeyControl] = true
			for _, key := range []ebiten.Key{ebiten.KeyUp, ebiten.KeyDown} {
				press(key, false)
				require.Equal(t, start, dialog.passageRow)
				require.Equal(t, 1, dialog.GetSelectedIndex())
			}
			delete(held, ebiten.KeyControl)
			press(ebiten.KeyDown, false)
			require.Equal(t, 2, dialog.GetSelectedIndex())
			require.Equal(t, 1, dialog.GetDetailPageCount())
			require.NotContains(t, dialog.footerGuidance(), "Space", "short exchanges need no paging hints")
			press(ebiten.KeySpace, false)
			require.Zero(t, dialog.GetDetailPage())
			require.Equal(t, 2, dialog.GetSelectedIndex())
		})
	}
}

func TestExchangeInversionCoversOnlySelectedRowsAndPreservesRunic(t *testing.T) {
	setQuestJournalTestResolution(t, 640, 400)
	config := testQuestJournalConfig(newWidgetTestKeyboard(0))
	config.LastConversation = journal.Conversation{Exchanges: []journal.Exchange{
		{ID: "first", Speaker: "Gardner", Location: "Keep", Input: "", Passages: []journal.Passage{{Text: "Hello", Runic: false}}},
		{ID: "runes", Speaker: "Gardner", Location: "Keep", Input: "WORD", Passages: []journal.Passage{{Text: "SUMM", Runic: true}}},
		{ID: "last", Speaker: "Gardner", Location: "Keep", Input: "BYE", Passages: []journal.Passage{{Text: "Farewell", Runic: false}}},
	}, Truncated: false}
	config.Excerpts = cloneJournalExchanges(config.LastConversation.Exchanges)
	dialog := NewQuestJournalDialog(testQuestJournalDossier(), config)
	dialog.switchTabBy(int(QuestJournalConversationTab))
	dialog.moveSelection(1)
	screen := ebiten.NewImage(640, 400)
	dialog.Draw(screen)
	content := dialog.exchangeContentRect()
	start, end := dialog.selectedExchangeSpan()
	spacing := dialog.exchangeOutput.GetLineSpacing()
	for row := range dialog.exchangeOutput.GetPanelContentLineCount() {
		y := content.Min.Y + int(math.Floor(float64(row-dialog.passageRow)*spacing)) + 1
		if y < content.Min.Y || y >= content.Max.Y {
			continue
		}
		red, green, blue, _ := screen.At(content.Max.X-2, y).RGBA()
		if row >= start && row < end {
			require.Equal(t, uint32(65535), red, "selected row %d must have white backing", row)
		} else {
			require.Zero(t, red, "attribution and other exchanges must remain outside the inversion")
		}
		require.Equal(t, red, green)
		require.Equal(t, red, blue)
	}
	// The Runic line itself must have dark glyphs on its white backing, not
	// invisible white text. Its source remains Runic and unchanged for capture.
	runicTop := content.Min.Y + int(math.Floor(float64(start+1-dialog.passageRow)*spacing))
	region := image.Rect(content.Min.X, runicTop, content.Max.X, runicTop+int(spacing)).Intersect(content)
	black := 0
	for y := region.Min.Y; y < region.Max.Y; y++ {
		for x := region.Min.X; x < region.Max.X; x++ {
			red, _, _, _ := screen.At(x, y).RGBA()
			if red == 0 {
				black++
			}
		}
	}
	require.Positive(t, black)
	require.Equal(t, config.LastConversation.Exchanges[1], dialog.selectedExchange())

	dialog.switchTabBy(1)
	require.Equal(t, QuestJournalExcerptsTab, dialog.GetActiveTab())
	dialog.moveSelection(1)
	screen.Clear()
	dialog.Draw(screen)
	require.True(t, dialog.exchangeHighlightRect().Empty(), "saved excerpts need no capture highlight")
	content = dialog.exchangeContentRect()
	white := 0
	for y := content.Min.Y; y < content.Max.Y; y++ {
		red, _, _, _ := screen.At(content.Max.X-2, y).RGBA()
		require.Zero(t, red, "the saved excerpt must retain its dark background")
		for x := content.Min.X; x < content.Max.X; x++ {
			red, green, blue, _ := screen.At(x, y).RGBA()
			if red == 65535 && red == green && red == blue {
				white++
			}
		}
	}
	require.Positive(t, white, "saved excerpt text must be readable in white")
}

func TestJournalSearchIncludesConversationAndExcerptsWithoutRunicTranslation(t *testing.T) {
	setQuestJournalTestResolution(t, 640, 400)
	config := testQuestJournalConfig(newWidgetTestKeyboard(0))
	config.LastConversation = journal.Conversation{Exchanges: []journal.Exchange{{
		ID: "one", Speaker: "Speaker", Location: "Britain", Input: "WORD",
		Passages: []journal.Passage{{Text: "A mysterious saying", Runic: false}, {Text: "SECRETRUNES", Runic: true}},
	}}, Truncated: false}
	config.KeepExcerpt = func(exchange journal.Exchange) ([]journal.Exchange, error) {
		return []journal.Exchange{exchange}, nil
	}
	dialog := NewQuestJournalDialog(testQuestJournalDossier(), config)
	dialog.beginSearch()
	dialog.setSearchQuery("secretrunes")
	require.Empty(t, dialog.searchResults)
	dialog.setSearchQuery("mysterious")
	require.Len(t, dialog.searchResults, 1)
	dialog.openSearchResult()
	require.Equal(t, QuestJournalConversationTab, dialog.activeTab)
	excerpts, err := config.KeepExcerpt(dialog.selectedExchange())
	dialog.excerptResult(excerpts, err, "Excerpt kept")
	require.True(t, dialog.exchangeKept("one"))
	require.Contains(t, dialog.listRowText(0), "Kept")
	dialog.beginSearch()
	dialog.setSearchQuery("saying")
	require.Len(t, dialog.searchResults, 2)
}

func TestExcerptReaderKeepsLongResponsesBeyondLegacyDetailLimit(t *testing.T) {
	setQuestJournalTestResolution(t, 640, 400)
	dialog := NewQuestJournalDialog(testQuestJournalDossier(), testQuestJournalConfig(newWidgetTestKeyboard(0)))
	exchange := journal.Exchange{ID: "long", Speaker: "Speaker", Location: "Britain", Input: "JOB",
		Passages: []journal.Passage{{Text: "START\n" + strings.Repeat("A whole line\n", 256) + "END", Runic: false}}}
	output, _ := journalExchangesOutput([]journal.Exchange{exchange}, dialog.font, dialog.rowOutput.GetLineSpacing(), 200, 0, "")
	require.Greater(t, output.GetPanelContentLineCount(), questJournalDetailMaxLines)
	require.Contains(t, output.GetOutputStr(false), "START")
	require.Contains(t, output.GetOutputStr(false), "END")
}
