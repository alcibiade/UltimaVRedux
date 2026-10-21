package save_snapshot

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/stretchr/testify/require"
)

func TestResistancePasswordV18CodecAndSchema(t *testing.T) {
	snapshot := validSnapshot()
	date := &BritannianDateSnapshot{Year: 139, Month: 4, Day: 5}
	snapshot.Redux.PlayerKnowledge = []PlayerKnowledgeSnapshot{{
		Category: KnowledgeCategoryJournal, Key: JournalKnowledgeResistancePassword,
		Source: KnowledgeSourceSnapshot{Kind: KnowledgeSourceDialogue, DisplayName: "Speaker"}, DiscoveredOn: date,
	}}
	snapshot.Redux.QuestJournal = []QuestJournalEventSnapshot{{
		Sequence: 1, Kind: QuestJournalEventKnowledgeLearned, Subject: "journal.resistance.password",
		OccurredOn: date, Source: QuestJournalSourceDialogue, SourceName: "Speaker",
	}}
	snapshot.Presentation.Visits.Record(journal.ChronicleTab, 49)
	data, err := EncodeCanonical(validEnvelope(snapshot))
	require.NoError(t, err)
	schema, _ := compileCurrentNativeSaveSchema(t)
	require.NoError(t, schema.Validate(decodeJSONSchemaInstance(t, data)))
	decoded, err := Decode(data)
	require.NoError(t, err)
	require.Equal(t, snapshot, decoded.Snapshot)
}
