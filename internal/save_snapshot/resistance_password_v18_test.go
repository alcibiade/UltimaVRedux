package save_snapshot

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/stretchr/testify/require"
)

func TestResistancePasswordV18CodecSchemaAndHistoricalRejection(t *testing.T) {
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
	for _, version := range []int{16, 17} {
		snapshot.Version = version
		require.NotEmpty(t, validateSnapshot(snapshot, version), "historical finite knowledge domains remain frozen")
	}
	snapshot.Presentation = journal.EmptyPresentation()
	snapshot.Version = 17
	historical := historicalJournalEnvelopeData(t, snapshot)
	_, err = Decode(historical)
	require.Error(t, err, "version 17 must reject the new knowledge key before promotion")
	frozen, _ := compileNativeSaveSchemaFile(t, "native-save-e2-s17.schema.json")
	require.Error(t, frozen.Validate(decodeJSONSchemaInstance(t, historical)))
	snapshot.Redux.PlayerKnowledge = []PlayerKnowledgeSnapshot{}
	historical = historicalJournalEnvelopeData(t, snapshot)
	_, err = Decode(historical)
	require.Error(t, err, "an event without its knowledge entry cannot bypass the historical catalog")
	require.Error(t, frozen.Validate(decodeJSONSchemaInstance(t, historical)))
}

func TestSnapshot17PromotionPreservesKnownStateWithoutPassword(t *testing.T) {
	snapshot := validSnapshot()
	snapshot.Version = 17
	data := historicalJournalEnvelopeData(t, snapshot)
	frozen, _ := compileNativeSaveSchemaFile(t, "native-save-e2-s17.schema.json")
	require.NoError(t, frozen.Validate(decodeJSONSchemaInstance(t, data)))
	decoded, err := Decode(data)
	require.NoError(t, err)
	snapshot.Version = CurrentVersion
	require.Equal(t, snapshot, decoded.Snapshot)
	require.Empty(t, decoded.Snapshot.Redux.PlayerKnowledge)
}
