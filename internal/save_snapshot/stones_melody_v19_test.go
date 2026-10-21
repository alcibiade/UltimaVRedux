package save_snapshot

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/stretchr/testify/require"
)

func TestStonesMelodyV19CodecAndSchema(t *testing.T) {
	snapshot := validSnapshot()
	date := &BritannianDateSnapshot{Year: 139, Month: 4, Day: 5}
	snapshot.Redux.PlayerKnowledge = []PlayerKnowledgeSnapshot{{
		Category: KnowledgeCategoryJournal, Key: JournalKnowledgeStonesMelody,
		Source: KnowledgeSourceSnapshot{Kind: KnowledgeSourceDialogue, DisplayName: "Lord Kenneth"}, DiscoveredOn: date,
	}}
	snapshot.Redux.QuestJournal = []QuestJournalEventSnapshot{{
		Sequence: 1, Kind: QuestJournalEventKnowledgeLearned, Subject: "journal.stones.melody",
		OccurredOn: date, Source: QuestJournalSourceDialogue, SourceName: "Lord Kenneth",
	}}
	snapshot.Presentation.Visits.Record(journal.ChronicleTab, 50)
	data, err := EncodeCanonical(validEnvelope(snapshot))
	require.NoError(t, err)
	schema, _ := compileCurrentNativeSaveSchema(t)
	require.NoError(t, schema.Validate(decodeJSONSchemaInstance(t, data)))
	decoded, err := Decode(data)
	require.NoError(t, err)
	require.Equal(t, snapshot, decoded.Snapshot)
}
