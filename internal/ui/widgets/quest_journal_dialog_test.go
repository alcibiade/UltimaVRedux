package widgets

import (
	"image"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

func TestQuestJournalDialogMandatoryConfigPanics(t *testing.T) {
	valid := testQuestJournalConfig(newWidgetTestKeyboard(0))
	tests := []struct {
		name   string
		mutate func(*QuestJournalDialogConfig)
		panic  string
	}{
		{
			name:   "keyboard",
			mutate: func(config *QuestJournalDialogConfig) { config.Keyboard = nil },
			panic:  "QuestJournalDialog: keyboard is required",
		},
		{
			name:   "dismiss callback",
			mutate: func(config *QuestJournalDialogConfig) { config.OnDismiss = nil },
			panic:  "QuestJournalDialog: OnDismiss is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := valid
			test.mutate(&config)
			require.PanicsWithValue(t, test.panic, func() {
				NewQuestJournalDialog(testQuestJournalDossier(), config)
			})
		})
	}
}

func TestQuestJournalDialogNavigationPreservesPerTabSelectionAndResetsPages(t *testing.T) {
	setQuestJournalTestResolution(t, 320, 200)
	keyboard := newWidgetTestKeyboard(0)
	dossier := testQuestJournalDossier()
	dossier.Virtues[0].CodexAnswer = strings.Repeat("A learned answer fills the responsive detail panel. ", 40)
	dialog := NewQuestJournalDialog(dossier, testQuestJournalConfig(keyboard))

	pressQuestJournalKey(dialog, keyboard, ebiten.KeyDown)
	require.Equal(t, 1, dialog.GetSelectedIndex())
	pressQuestJournalKey(dialog, keyboard, ebiten.KeyTab)
	require.Equal(t, QuestJournalVirtuesTab, dialog.GetActiveTab())
	require.Zero(t, dialog.GetSelectedIndex())
	require.Greater(t, dialog.GetDetailPageCount(), 1)
	pressQuestJournalKey(dialog, keyboard, ebiten.KeySpace)
	require.Equal(t, 1, dialog.GetDetailPage())
	pressQuestJournalKey(dialog, keyboard, ebiten.KeyDown)
	require.Equal(t, 1, dialog.GetSelectedIndex())
	require.Zero(t, dialog.GetDetailPage())

	pressQuestJournalKey(dialog, keyboard, ebiten.KeyTab)
	require.Equal(t, QuestJournalShadowlordsTab, dialog.GetActiveTab())
	pressQuestJournalKey(dialog, keyboard, ebiten.KeyDown)
	require.Equal(t, 1, dialog.GetSelectedIndex())
	require.Equal(t, -1, questJournalTabDelta(true))
	require.Equal(t, 1, questJournalTabDelta(false))
	dialog.switchTabBy(questJournalTabDelta(true))
	require.Equal(t, QuestJournalVirtuesTab, dialog.GetActiveTab(), "Shift+Tab uses the reverse tab delta")
	require.Equal(t, 1, dialog.GetSelectedIndex())
	dialog.switchTabBy(questJournalTabDelta(true))
	require.Equal(t, QuestJournalChronicleTab, dialog.GetActiveTab())
	require.Equal(t, 1, dialog.GetSelectedIndex())
}

func TestQuestJournalDialogSpaceCyclesCapacityBoundPages(t *testing.T) {
	setQuestJournalTestResolution(t, 320, 200)
	keyboard := newWidgetTestKeyboard(0)
	dossier := testQuestJournalDossier()
	dossier.Virtues[0].CodexAnswer = strings.Repeat("The Codex answer remains readable without shrinking the standard font. ", 60)
	dialog := NewQuestJournalDialog(dossier, testQuestJournalConfig(keyboard))
	dialog.switchTabBy(1)

	pages := dialog.detailPages()
	require.Greater(t, len(pages), 1)
	capacity := dialog.detailOutput.GetLineCapacity(dialog.detailContentRect().Dy())
	for _, page := range pages {
		require.LessOrEqual(t, dialog.detailLineCount(page), capacity)
	}

	for expected := 1; expected < len(pages); expected++ {
		pressQuestJournalKey(dialog, keyboard, ebiten.KeySpace)
		require.Equal(t, expected, dialog.GetDetailPage())
	}
	pressQuestJournalKey(dialog, keyboard, ebiten.KeySpace)
	require.Zero(t, dialog.GetDetailPage())
}

func TestQuestJournalDialogUsesSemanticFontsAndResponsiveLayout(t *testing.T) {
	setQuestJournalTestResolution(t, 320, 200)
	keyboard := newWidgetTestKeyboard(0)
	dialog := NewQuestJournalDialog(testQuestJournalDossier(), testQuestJournalConfig(keyboard))
	dialog.switchTabBy(1)
	pressQuestJournalKey(dialog, keyboard, ebiten.KeyDown)
	dialog.ensureLayout()
	smallSize := dialog.font.TextFace.Size

	require.Equal(t, FontSizeMainScreen, dialog.GetFontSize())
	require.Equal(t, GetScaledFont(FontSizeDialogTitle).TextFace.Size, dialog.titleOutput.Font.TextFace.Size)
	for _, size := range []float64{
		dialog.tabOutput.Font.TextFace.Size,
		dialog.rowOutput.Font.TextFace.Size,
		dialog.detailOutput.Font.TextFace.Size,
		dialog.footerOutput.Font.TextFace.Size,
	} {
		require.Equal(t, smallSize, size)
	}
	for tab, rect := range dialog.layout.tabRects {
		width, _ := dialog.font.MeasureString(questJournalTabTitle(QuestJournalTab(tab)))
		require.GreaterOrEqual(t, float64(rect.Dx()), width)
	}
	require.True(t, dialog.layout.listRect.In(dialog.layout.panelRect))
	require.True(t, dialog.layout.detailRect.In(dialog.layout.panelRect))
	require.Less(t, dialog.layout.listRect.Max.X, dialog.layout.detailRect.Min.X)

	setResponsiveDialogTestResolution(t, 1280, 720)
	dialog.ensureLayout()
	require.Equal(t, QuestJournalVirtuesTab, dialog.GetActiveTab())
	require.Equal(t, 1, dialog.GetSelectedIndex())
	require.Less(t, dialog.GetDetailPage(), dialog.GetDetailPageCount())
	require.Greater(t, dialog.font.TextFace.Size, smallSize)
	largeBounds := dialog.outerBorder.GetBounds()
	require.InDelta(t, 1280*questJournalMarginX, largeBounds.Min.X, 1)
	require.InDelta(t, 720*questJournalMarginY, largeBounds.Min.Y, 1)
	require.InDelta(t, 1280*(1-questJournalMarginX), largeBounds.Max.X, 1)
	require.InDelta(t, 720*(1-questJournalMarginY), largeBounds.Max.Y, 1)
}

func TestQuestJournalDialogChronicleVirtuesAndShadowlordsUseSemanticProjection(t *testing.T) {
	dossier := testQuestJournalDossier()

	runtime := questJournalChronicleDetail(dossier.Chronicle[0])
	require.Equal(t, "Astaroth", runtime[0].value)
	require.Equal(t, "Date", runtime[2].label)
	require.Equal(t, "13-28-139", runtime[2].value)
	migrated := questJournalChronicleDetail(dossier.Chronicle[1])
	require.Equal(t, "Unknown Shadowlord I was permanently defeated.", migrated[1].value)
	require.Equal(t, "Previously defeated; date unknown", migrated[2].value)

	expectedStates := map[game_state.ShrineProgressState]string{
		game_state.ShrineProgressNotAssigned:     "Not started",
		game_state.ShrineProgressAssigned:        "In progress",
		game_state.ShrineProgressReadyToComplete: "Ready to complete",
		game_state.ShrineProgressOffering:        "Complete",
	}
	for state, expected := range expectedStates {
		actual, _ := questJournalVirtueStateText(state)
		require.Equal(t, expected, actual)
	}
	virtue := questJournalVirtueDetail(dossier.Virtues[0])
	require.Equal(t, "Ready to complete", virtue[1].value)
	require.Equal(t, "Shrine quest", virtue[1].label)
	require.Equal(t, "Destroyed", virtue[2].value, "ruin is an independent qualifier")
	require.Equal(t, DialogStatusWarningColor, virtue[2].color)
	require.Equal(t, "AHM", virtue[3].value)
	for state := range expectedStates {
		dossier.Virtues[0].Ruined = false
		dossier.Virtues[0].QuestState = state
		for _, line := range questJournalVirtueDetail(dossier.Virtues[0]) {
			require.NotEqual(t, "Shrine", line.label)
		}
	}

	living := questJournalShadowlordDetail(dossier.Shadowlords[0])
	defeated := questJournalShadowlordDetail(dossier.Shadowlords[1])
	require.Equal(t, "Falsehood", living[1].value)
	require.Equal(t, "Embodies", living[1].label)
	require.Equal(t, "Living", living[7].value)
	require.Equal(t, "Not learned", defeated[1].value)
	require.Equal(t, "Defeated", defeated[7].value)
}

func TestQuestJournalDialogClonesDossierAndDismissesIdempotently(t *testing.T) {
	dossier := testQuestJournalDossier()
	keyboard := newWidgetTestKeyboard(0)
	dismissCalls := 0
	config := testQuestJournalConfig(keyboard)
	config.OnDismiss = func() { dismissCalls++ }
	dialog := NewQuestJournalDialog(dossier, config)

	dossier.Chronicle[0].DisplayName = "Changed"
	dossier.Chronicle[0].OccurredOn.Day = 1
	dossier.Virtues[0].Name = "Changed"
	dossier.Shadowlords[0].DisplayName = "Changed"
	require.Equal(t, "Astaroth", dialog.dossier.Chronicle[0].DisplayName)
	require.Equal(t, byte(28), dialog.dossier.Chronicle[0].OccurredOn.Day)
	require.Equal(t, "Honesty", dialog.dossier.Virtues[0].Name)
	require.Equal(t, "Faulinei", dialog.dossier.Shadowlords[0].DisplayName)
	require.True(t, questJournalDismissShortcut(ebiten.KeyJ, true))
	require.True(t, questJournalDismissShortcut(ebiten.KeyEscape, false))
	require.False(t, questJournalDismissShortcut(ebiten.KeyJ, false))

	pressQuestJournalKey(dialog, keyboard, ebiten.KeyJ)
	require.False(t, dialog.IsDismissed(), "plain J remains isolated and does not close the modal")
	pressQuestJournalKey(dialog, keyboard, ebiten.KeyEscape)
	require.True(t, dialog.IsDismissed())
	dialog.Dismiss()
	require.Equal(t, 1, dismissCalls)
}

func testQuestJournalConfig(keyboard *input.Keyboard) QuestJournalDialogConfig {
	return QuestJournalDialogConfig{Keyboard: keyboard, OnDismiss: func() {}}
}

func pressQuestJournalKey(dialog *QuestJournalDialog, keyboard *input.Keyboard, key ebiten.Key) {
	keyboard.SimulateKeyPress(key)
	dialog.Update()
}

func setQuestJournalTestResolution(t *testing.T, width, height int) {
	t.Helper()
	setResponsiveDialogTestResolution(t, width, height)
}

func testQuestJournalDossier() game_state.QuestJournalDossier {
	date := datetime.BritannianDate{Year: 139, Month: 13, Day: 28}
	virtues := []game_state.QuestJournalVirtue{
		{Index: 0, Name: "Honesty", QuestState: game_state.ShrineProgressReadyToComplete, Ruined: true, MantraLearned: true, Mantra: "AHM", CodexAnswerLearned: true, CodexAnswer: "Truth"},
		{Index: 1, Name: "Compassion", QuestState: game_state.ShrineProgressAssigned},
		{Index: 2, Name: "Valor", QuestState: game_state.ShrineProgressOffering, CodexAnswerLearned: true, CodexAnswer: "Courage"},
		{Index: 3, Name: "Justice", QuestState: game_state.ShrineProgressNotAssigned},
		{Index: 4, Name: "Sacrifice", QuestState: game_state.ShrineProgressNotAssigned},
		{Index: 5, Name: "Honor", QuestState: game_state.ShrineProgressNotAssigned},
		{Index: 6, Name: "Spirituality", QuestState: game_state.ShrineProgressNotAssigned},
		{Index: 7, Name: "Humility", QuestState: game_state.ShrineProgressNotAssigned},
	}

	return game_state.QuestJournalDossier{
		Chronicle: []game_state.QuestJournalChronicleEntry{
			{DisplayName: "Astaroth", OccurredOn: &date},
			{DisplayName: "Unknown Shadowlord I"},
		},
		Virtues: virtues,
		Shadowlords: []game_state.QuestJournalShadowlord{
			{DisplayName: "Faulinei", Association: "Falsehood", State: game_state.QuestJournalShadowlordLiving},
			{DisplayName: "Astaroth", State: game_state.QuestJournalShadowlordDefeated},
			{DisplayName: "Unknown Shadowlord III", Association: "Cowardice", State: game_state.QuestJournalShadowlordLiving},
		},
	}
}

func alphaAt(screen *ebiten.Image, point image.Point) uint32 {
	_, _, _, alpha := screen.At(point.X, point.Y).RGBA()

	return alpha
}
