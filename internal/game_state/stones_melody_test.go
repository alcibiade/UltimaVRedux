package game_state

import (
	"strconv"
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	"github.com/stretchr/testify/require"
)

func TestStonesMelodyChronicleRoundTripAndPlayableInstructions(t *testing.T) {
	state, _ := newLordBritishHarpsichordTestState(t, 17, 17)
	require.Empty(t, state.CurrentQuestJournalDossier().Chronicle)
	source := DialogueKnowledgeSource("Lord Kenneth")
	require.True(t, state.DiscoverJournalKnowledge("stones.melody", source))
	before := state.CurrentQuestJournalDossier().Chronicle
	require.Len(t, before, 1)
	require.Equal(t, "Stones: harpsichord melody", before[0].Title)
	require.Contains(t, before[0].Summary, "immediately north")
	require.Contains(t, before[0].Summary, "top-row")
	require.Contains(t, before[0].Summary, "numpad")
	require.False(t, state.lordBritishPassageOpen, "knowledge alone never plays the melody")
	require.False(t, state.DiscoverJournalKnowledge("stones.melody", DialogueKnowledgeSource("Later speaker")))
	snapshot, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)
	restored, err := NewGameStateFromSaveSnapshot(snapshot, state.GameConfig, state.GameReferences,
		state.MapState.XTilesVisibleOnGameScreen, state.MapState.YTilesVisibleOnGameScreen)
	require.NoError(t, err)
	require.Equal(t, before, restored.CurrentQuestJournalDossier().Chronicle)
	require.False(t, restored.DiscoverJournalKnowledge("stones.melody", source))
	key, valid := JournalKnowledgeKey("stones.melody")
	require.True(t, valid)
	entry, known := restored.JournalKnowledge(key.ID)
	require.True(t, known)
	require.Equal(t, key, entry.Key)
	require.Equal(t, source, entry.Source)
	// Follow the persisted player-facing instruction, not the engine melody constant.
	_, sequence, found := strings.Cut(before[0].Summary, ": ")
	require.True(t, found)
	require.True(t, state.CanPlayHarpsichord())
	for _, digit := range strings.Fields(strings.TrimSuffix(sequence, ".")) {
		note, parseErr := strconv.Atoi(digit)
		require.NoError(t, parseErr)
		require.True(t, state.ActionPlayHarpsichordNote(note))
	}
	require.True(t, state.lordBritishPassageOpen)
	drainHarpsichordWorkflow(t, state)
	var recaptured save_snapshot.SaveSnapshot
	restored.captureRedux(&recaptured)
	require.Equal(t, snapshot.Redux, recaptured.Redux)
}
