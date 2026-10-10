package widgets

import (
	"fmt"
	"image"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

func TestQuestJournalHistoryStrictWindowAndWholeMessageNavigation(t *testing.T) {
	font := text.NewUltimaFont(16)
	for _, capacity := range []int{1, 10} {
		for _, count := range []int{capacity, 3 * capacity, 3*capacity + 1} {
			t.Run(fmt.Sprintf("C%d_rows%d", capacity, count), func(t *testing.T) {
				history := questJournalHistory{source: strings.TrimSuffix(strings.Repeat("same\n", count), "\n"), selected: -1}
				history.reflow(font, 500, capacity)
				require.Len(t, history.rows, count)
				require.Equal(t, max(0, count-3*capacity), history.window)
				require.Equal(t, count-1, history.selected)
				require.Equal(t, max(0, count-capacity), history.top)
				history.edge(false)
				require.Equal(t, history.window, history.selected)
				history.move(-1)
				require.Equal(t, history.window, history.selected)
				history.edge(true)
				history.move(1)
				require.Equal(t, count-1, history.selected)
			})
		}
	}
}

func TestQuestJournalHistoryOneRowWindowStartingOnSeparator(t *testing.T) {
	history := questJournalHistory{source: "old\n\nmiddle\nnew", selected: -1}
	history.reflow(text.NewUltimaFont(16), 500, 1)
	require.Equal(t, 1, history.window)
	require.Equal(t, -1, history.rows[history.window].Message)
	history.edge(false)
	require.Equal(t, 1, history.selected)
	require.Equal(t, "middle", history.rows[history.top].Text, "Home must reveal the selected message, not its preceding blank")
	history.move(-1)
	require.Equal(t, "middle", history.rows[history.top].Text)
}

func TestQuestJournalHistoryOversizedMessageEveryAccessibleRow(t *testing.T) {
	font := text.NewUltimaFont(16)
	for _, capacity := range []int{1, 10} {
		for _, words := range []int{20, 100} {
			history := questJournalHistory{source: "old\n\n" + strings.Repeat("wide ", words) + "\nnew", selected: -1}
			history.reflow(font, 80, capacity)
			history.edge(false)
			seen := make(map[int]bool)
			for step := 0; step < len(history.rows)*2; step++ {
				for row := history.top; row < min(len(history.rows), history.top+capacity); row++ {
					seen[row] = true
				}
				history.page(1)
			}
			for row := history.window; row < len(history.rows); row++ {
				require.True(t, seen[row], "C=%d words=%d row=%d", capacity, words, row)
			}
			require.Equal(t, history.spans[len(history.spans)-1].message, history.selected)
			clear(seen)
			for step := 0; step < len(history.rows)*2; step++ {
				for row := history.top; row < min(len(history.rows), history.top+capacity); row++ {
					seen[row] = true
				}
				history.page(-1)
			}
			for row := history.window; row < len(history.rows); row++ {
				require.True(t, seen[row], "reverse C=%d row=%d", capacity, row)
			}
		}
	}
}

func TestQuestJournalHistoryReflowKeepsIdentitySourceAndAnchor(t *testing.T) {
	font := text.NewUltimaFont(16)
	history := questJournalHistory{source: "old\n" + strings.Repeat("readable words ", 100) + "\ntail", selected: -1}
	history.reflow(font, 180, 10)
	history.move(-1)
	history.page(-1)
	selected, anchor, source := history.selected, history.anchor, history.source
	history.reflow(font, 240, 10)
	require.Equal(t, selected, history.selected)
	require.Equal(t, anchor, history.anchor)
	require.Equal(t, source, history.source)
	history.reflow(font, 80, 1)
	require.GreaterOrEqual(t, history.anchor, anchor)
	require.GreaterOrEqual(t, history.top, history.window)
	require.Equal(t, source, history.source)
	history.reflow(font, 180, 100)
	require.Zero(t, history.window, "source was not destructively trimmed")
}

func TestQuestJournalSearchRedactionStableIdentityAndVisits(t *testing.T) {
	setQuestJournalTestResolution(t, 640, 400)
	dossier := testQuestJournalDossier()
	dossier.Chronicle = []game_state.QuestJournalChronicleEntry{
		{Sequence: 1, Title: "Flame of Truth", Summary: "Discovered the Flame", Source: "Lycaeum"},
		{Sequence: 2, Title: "Shadowlord of Hatred", Summary: "An independent domain note"},
	}
	dossier.Virtues[1].Mantra = "SECRET-MANTRA"
	dossier.Virtues[1].CodexAnswer = "SECRET-ANSWER"
	dossier.Shadowlords[0].Opposes = "Truth"
	keyboard := newWidgetTestKeyboard(0)
	configuration := testQuestJournalConfig(keyboard)
	configuration.NewEntries = [3][]bool{{true, true}, {true}, {true}}
	dialog := NewQuestJournalDialog(dossier, configuration)
	configuration.NewEntries[0][0] = false
	require.True(t, dialog.entryIsNew(QuestJournalChronicleTab, 0), "opening markers are detached")
	dialog.beginSearch()
	require.Equal(t, "Type words to search the Journal.", dialog.searchStatus())
	for _, hidden := range []string{"secret-mantra", "secret-answer", "nosfentor", "astaroth hatred", "justice not learned"} {
		dialog.setSearchQuery(hidden)
		require.Empty(t, dialog.searchResults, hidden)
		require.Equal(t, "No matching entries.", dialog.searchStatus())
	}
	for _, query := range []string{"not started", "justice not started"} {
		dialog.setSearchQuery(query)
		var titles []string
		for _, result := range dialog.searchResults {
			titles = append(titles, dialog.searchEntries[result].title)
		}
		require.Contains(t, titles, "Justice", "visible state text is searchable regardless of its colour")
	}
	dialog.setSearchQuery("TRUTH flame lycaeum")
	require.Len(t, dialog.searchResults, 1)
	dialog.setSearchQuery("truth")
	require.Len(t, dialog.searchResults, 3)
	dialog.navigate(ebiten.KeyEnd)
	selected := dialog.searchResults[dialog.searchSelection]
	dialog.setSearchQuery("trut")
	require.Equal(t, selected, dialog.searchResults[dialog.searchSelection])
	setResponsiveDialogTestResolution(t, 1280, 720)
	require.NotPanics(t, dialog.ensureLayout, "search row indices must never index the underlying active-tab detail pane")
	require.Equal(t, selected, dialog.searchResults[dialog.searchSelection])
	require.Equal(t, [3]bool{true, false, false}, dialog.VisitedTabs())
	dialog.openSearchResult()
	require.Equal(t, QuestJournalShadowlordsTab, dialog.activeTab)
	require.Equal(t, [3]bool{true, false, true}, dialog.VisitedTabs())
	require.True(t, dialog.tabIsNew(QuestJournalChronicleTab))
	require.True(t, dialog.tabIsNew(QuestJournalShadowlordsTab))
	dialog.beginSearch()
	pressQuestJournalKey(dialog, keyboard, ebiten.KeyEscape)
	require.False(t, dialog.searching)
	require.False(t, dialog.dismissed)
}

func TestQuestJournalSearchTypingDoesNotRunTabOrDetailCommands(t *testing.T) {
	setQuestJournalTestResolution(t, 640, 400)
	keyboard := newWidgetTestKeyboard(0)
	dialog := NewQuestJournalDialog(testQuestJournalDossier(), testQuestJournalConfig(keyboard))
	pressQuestJournalKey(dialog, keyboard, ebiten.KeySlash)
	require.True(t, dialog.searching)
	for _, key := range []ebiten.Key{ebiten.KeyT, ebiten.KeyR, ebiten.KeySpace, ebiten.KeyU, ebiten.KeyTab} {
		pressQuestJournalKey(dialog, keyboard, key)
	}
	require.Equal(t, "tr u", strings.ToLower(dialog.searchQuery))
	require.Equal(t, QuestJournalChronicleTab, dialog.activeTab)
	require.Zero(t, dialog.detailPage)
	pressQuestJournalKey(dialog, keyboard, ebiten.KeyBackspace)
	require.Equal(t, "tr ", strings.ToLower(dialog.searchQuery))
	pressQuestJournalKey(dialog, keyboard, ebiten.KeyEscape)
	require.False(t, dialog.dismissed)
	pressQuestJournalKey(dialog, keyboard, ebiten.KeyEscape)
	require.True(t, dialog.dismissed)
}

func TestQuestJournalHistoryEntryAlwaysShowsNewestAndEnterDoesNothing(t *testing.T) {
	setQuestJournalTestResolution(t, 640, 400)
	keyboard := newWidgetTestKeyboard(0)
	configuration := testQuestJournalConfig(keyboard)
	configuration.History = strings.TrimSuffix(strings.Repeat("same\n", 64), "\n")
	dialog := NewQuestJournalDialog(testQuestJournalDossier(), configuration)
	dialog.switchTabBy(int(QuestJournalHistoryTab))
	require.Equal(t, QuestJournalHistoryTab, dialog.activeTab)
	require.Equal(t, 63, dialog.GetSelectedIndex())
	pressQuestJournalKey(dialog, keyboard, ebiten.KeyHome)
	require.Less(t, dialog.GetSelectedIndex(), 63)
	selected := dialog.GetSelectedIndex()
	pressQuestJournalKey(dialog, keyboard, ebiten.KeyEnter)
	require.Equal(t, selected, dialog.GetSelectedIndex())
	dialog.beginSearch()
	dialog.leaveSearch()
	require.Equal(t, selected, dialog.GetSelectedIndex(), "cancelled search must retain the reading anchor")
	pressQuestJournalKey(dialog, keyboard, ebiten.KeyHome)
	dialog.switchTabBy(1)
	dialog.switchTabBy(-1)
	require.Equal(t, 63, dialog.GetSelectedIndex())
	require.Len(t, dialog.frameRects, 3)
}

func TestQuestJournalLayoutAndCompleteFooterAtAllSizes(t *testing.T) {
	sizes := append(config.SupportedWindowResolutions(), config.ScreenResolution{X: 320, Y: 200})
	for _, size := range sizes {
		t.Run(fmt.Sprintf("%dx%d", size.X, size.Y), func(t *testing.T) {
			setQuestJournalTestResolution(t, size.X, size.Y)
			configuration := testQuestJournalConfig(newWidgetTestKeyboard(0))
			configuration.NewEntries = [3][]bool{{true}, {true}, {true}}
			dossier := testQuestJournalDossier()
			dossier.Shadowlords[2].DisplayName = "Shadowlord III"
			dialog := NewQuestJournalDialog(dossier, configuration)
			require.Equal(t, GetScaledFont(FontSizeDialogTitle).TextFace.Size, dialog.titleOutput.Font.TextFace.Size)
			for tab, rect := range dialog.layout.tabRects {
				width, _ := dialog.font.MeasureString(dialog.tabTitle(QuestJournalTab(tab)))
				require.LessOrEqual(t, width, float64(rect.Dx()))
				require.True(t, rect.In(dialog.layout.tabsRect))
			}
			for tab := range questJournalTabCount {
				dialog.activeTab = QuestJournalTab(tab)
				dialog.enterTab()
				guidance := fitModifierFooterGuidance(dialog.footerGuidance(), dialog.font, dialog.footerContentRect().Dx(), dialog.footerLineCount())
				for _, line := range strings.Split(guidance, "\n") {
					width, _ := dialog.font.MeasureString(line)
					require.LessOrEqual(t, width, float64(dialog.footerContentRect().Dx()), "%s", line)
				}
				require.Contains(t, guidance, "Close")
				require.NotContains(t, guidance, "S-J")
				if tab == int(QuestJournalShadowlordsTab) {
					require.True(t, dialog.shadowlordSpriteRect().In(dialog.layout.detailRect))
					require.Equal(t, "Shadowlord III", fitJournalLine("Shadowlord III", dialog.font, dialog.listContentRect().Dx()-dialog.layout.padding))
				}
			}
		})
	}
}

func TestQuestJournalCuratedPagesUseFullCapacityAndCenter(t *testing.T) {
	setQuestJournalTestResolution(t, 320, 200)
	dossier := testQuestJournalDossier()
	dossier.Chronicle = make([]game_state.QuestJournalChronicleEntry, 32)
	for index := range dossier.Chronicle {
		dossier.Chronicle[index].Title = fmt.Sprintf("Note %d", index)
	}
	dialog := NewQuestJournalDialog(dossier, testQuestJournalConfig(newWidgetTestKeyboard(0)))
	capacity := dialog.visibleRows(dialog.listContentRect())
	dialog.navigate(ebiten.KeyPageDown)
	require.Equal(t, capacity, dialog.GetSelectedIndex())
	start, _ := zStatsVisibleRange(dialog.GetSelectedIndex(), 32, capacity)
	require.Equal(t, capacity-capacity/2, start)
	dialog.navigate(ebiten.KeyEnd)
	require.Equal(t, 31, dialog.GetSelectedIndex())
	dialog.navigate(ebiten.KeyDown)
	require.Equal(t, 31, dialog.GetSelectedIndex())
	dialog.navigate(ebiten.KeyHome)
	require.Zero(t, dialog.GetSelectedIndex())
}

func TestQuestJournalHeldNavigationUsesRepeatRegistration(t *testing.T) {
	setQuestJournalTestResolution(t, 640, 400)
	held := map[ebiten.Key]bool{}
	keyboard := input.NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool { return held[key] })
	configuration := testQuestJournalConfig(keyboard)
	configuration.History = "first\nsecond\nthird"
	dialog := NewQuestJournalDialog(testQuestJournalDossier(), configuration)
	dialog.switchTabBy(1)
	held[ebiten.KeyDown] = true
	dialog.Update()
	require.Equal(t, 1, dialog.GetSelectedIndex())
	for range 18 {
		dialog.Update()
	}
	require.Equal(t, 2, dialog.GetSelectedIndex(), "held Down does not require physical release")
	delete(held, ebiten.KeyDown)
	dialog.switchTabBy(2)
	held[ebiten.KeyUp] = true
	dialog.Update()
	require.Equal(t, 1, dialog.GetSelectedIndex())
	for range 18 {
		dialog.Update()
	}
	require.Zero(t, dialog.GetSelectedIndex(), "held Up selects previous messages, not wrapped rows")
}

func TestQuestJournalSearchHeldPageKeysRespectRepeatDelay(t *testing.T) {
	setQuestJournalTestResolution(t, 640, 400)
	for _, key := range []ebiten.Key{ebiten.KeyPageDown, ebiten.KeyPageUp} {
		t.Run(key.String(), func(t *testing.T) {
			held := map[ebiten.Key]bool{}
			keyboard := input.NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool { return held[key] })
			dossier := testQuestJournalDossier()
			dossier.Chronicle = make([]game_state.QuestJournalChronicleEntry, 32)
			for index := range dossier.Chronicle {
				dossier.Chronicle[index].Title = fmt.Sprintf("Note %d", index)
			}
			dialog := NewQuestJournalDialog(dossier, testQuestJournalConfig(keyboard))
			dialog.beginSearch()
			dialog.searchInput.SetText("note")
			dialog.setSearchQuery("note")
			capacity := max(1, dialog.visibleRows(dialog.searchContentRect())/journalSearchResultRows)
			require.Less(t, capacity, len(dialog.searchResults)-1)
			expected := capacity
			if key == ebiten.KeyPageUp {
				dialog.navigate(ebiten.KeyEnd)
				expected = len(dialog.searchResults) - 1 - capacity
			}
			held[key] = true
			dialog.Update()
			require.Equal(t, expected, dialog.GetSelectedIndex())
			for range 17 {
				dialog.Update()
				require.Equal(t, expected, dialog.GetSelectedIndex(), "throttled page keys must not reach TextInput")
			}
			dialog.Update()
			if key == ebiten.KeyPageDown {
				expected = min(expected+capacity, len(dialog.searchResults)-1)
			} else {
				expected = max(0, expected-capacity)
			}
			require.Equal(t, expected, dialog.GetSelectedIndex(), "repeat resumes after the delay")
			require.Equal(t, "note", dialog.searchQuery)
		})
	}
}

func TestQuestJournalEmptyHistoryAndFocusOwnership(t *testing.T) {
	setQuestJournalTestResolution(t, 640, 400)
	held := map[ebiten.Key]bool{}
	keyboard := input.NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool { return held[key] })
	dialog := NewQuestJournalDialog(game_state.QuestJournalDossier{}, testQuestJournalConfig(keyboard))
	dialog.switchTabBy(int(QuestJournalHistoryTab))
	require.Equal(t, -1, dialog.GetSelectedIndex())
	for _, key := range []ebiten.Key{ebiten.KeyUp, ebiten.KeyDown, ebiten.KeyPageDown, ebiten.KeyHome, ebiten.KeyEnd, ebiten.KeyEnter} {
		pressQuestJournalKey(dialog, keyboard, key)
	}
	dialog.prepareSingleLine(dialog.historyContentRect(), "No recent messages.", DialogEmptyColor)
	require.Equal(t, "No recent messages.", dialog.rowOutput.GetOutputStr(false))
	stack := DialogStack{}
	stack.PushModalDialog(dialog)
	other := NewQuestJournalDialog(testQuestJournalDossier(), testQuestJournalConfig(keyboard))
	stack.PushModalDialog(other)
	require.False(t, dialog.focused)
	held[ebiten.KeyEscape] = true
	dialog.Update()
	require.False(t, dialog.dismissed)
	stack.MustRemoveWidgetInstance(other)
	require.True(t, dialog.focused)
	dialog.Update()
	require.False(t, dialog.dismissed, "the focusing key must first be released")
	delete(held, ebiten.KeyEscape)
	keyboard.ClearFrameConsumption()
	held[ebiten.KeyShift] = true
	held[ebiten.KeyJ] = true
	dialog.beginSearch()
	dialog.Update()
	require.True(t, dialog.dismissed, "Shift-J closes even while searching")
}

func TestQuestJournalWholeMessageBackingAndSelectedStarAreMonochrome(t *testing.T) {
	setQuestJournalTestResolution(t, 640, 400)
	configuration := testQuestJournalConfig(newWidgetTestKeyboard(0))
	configuration.History = "first\n\n" + strings.Repeat("wide message ", 40) + "\nlast"
	configuration.NewEntries = [3][]bool{{true}, nil, nil}
	dialog := NewQuestJournalDialog(testQuestJournalDossier(), configuration)
	dialog.switchTabBy(int(QuestJournalHistoryTab))
	dialog.history.move(-1)
	screen := ebiten.NewImage(640, 400)
	dialog.Draw(screen)
	content := dialog.historyContentRect()
	visibleSelected := 0
	for row := dialog.history.top; row < min(len(dialog.history.rows), dialog.history.top+dialog.history.capacity); row++ {
		point := image.Pt(content.Max.X-1, content.Min.Y+(row-dialog.history.top)*dialog.rowHeight()+1)
		red, green, blue, _ := screen.At(point.X, point.Y).RGBA()
		if dialog.history.rows[row].Message == dialog.history.selected {
			visibleSelected++
			require.Equal(t, uint32(65535), red)
			require.Equal(t, red, green)
			require.Equal(t, red, blue)
		} else {
			require.Zero(t, red)
		}
	}
	require.Greater(t, visibleSelected, 1, "the entire visible message, not one cursor row, is inverted")
	dialog.switchTabBy(-int(QuestJournalHistoryTab))
	rect := dialog.listContentRect()
	rect.Max.Y = rect.Min.Y + dialog.rowHeight()
	dialog.drawListRow(screen, rect, 0, true)
	starWidth, _ := dialog.font.MeasureString("*")
	black := 0
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Min.X+int(starWidth); x++ {
			red, green, blue, _ := screen.At(x, y).RGBA()
			require.Equal(t, red, green)
			require.Equal(t, red, blue, "selected star must not remain yellow")
			if red == 0 {
				black++
			}
		}
	}
	require.Positive(t, black)
}

func TestQuestJournalSharedShiftFooterPolicy(t *testing.T) {
	font := text.NewUltimaFont(16)
	for _, modifier := range []string{"Shift", "Shft", "Shf"} {
		expected := "[" + modifier + "-J] Close"
		width, _ := font.MeasureString(expected)
		require.Equal(t, expected, fitModifierFooterGuidance("[Shift-J] Close", font, int(width), 1))
	}
	for _, size := range append(config.SupportedWindowResolutions(), config.ScreenResolution{X: 320, Y: 200}) {
		setQuestJournalTestResolution(t, size.X, size.Y)
		item := NewItemSelectionDialog(ItemSelectionDialogConfig{
			Keyboard: newWidgetTestKeyboard(0), OnSelect: func(game_state.UseableItemInfo) {}, OnCancel: func() {},
		})
		PrepareKeyboardFooter(item.footerContentRect(),
			"[Tab/Shift-Tab] Category  [Space] Details\n"+
				"[Up/Dn] Item  [Shift+Up/Dn] Page  [Enter] Use  [Esc] Cancel",
			itemDialogFooterLines, item.footerOutput)
		guidance := item.footerOutput.GetOutputStr(true)
		require.NotContains(t, guidance, "S-Tab")
		require.Contains(t, guidance, "Category")
		require.Contains(t, guidance, "Cancel")
		for _, line := range strings.Split(guidance, "\n") {
			width, _ := item.font.MeasureString(line)
			require.LessOrEqual(t, width, float64(item.footerContentRect().Dx()))
		}
		dialog := NewQuestJournalDialog(testQuestJournalDossier(), testQuestJournalConfig(newWidgetTestKeyboard(0)))
		dialog.beginSearch()
		PrepareKeyboardFooter(dialog.footerContentRect(), dialog.footerGuidance(), dialog.footerLineCount(), dialog.footerOutput)
		for _, line := range strings.Split(dialog.footerOutput.GetOutputStr(true), "\n") {
			width, _ := dialog.font.MeasureString(line)
			require.LessOrEqual(t, width, float64(dialog.footerContentRect().Dx()))
		}
	}
}
