package save_snapshot

import (
	"strconv"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/stretchr/testify/require"
)

func TestExistingSaveDefaultsStonesToUnknownWithoutConversion(t *testing.T) {
	for _, version := range []int{17, 18} {
		t.Run(strconv.Itoa(version), func(t *testing.T) {
			snapshot := validSnapshot()
			snapshot.Version = version
			// Even a previously solved harpsichord puzzle does not invent a
			// learned clue. Preserve unrelated discoveries and their provenance.
			snapshot.Redux.LordBritishPassageOpen = true
			snapshot.Redux.PlayerKnowledge = []PlayerKnowledgeSnapshot{{
				Category: KnowledgeCategoryJournal, Key: JournalKnowledgeDomainFalsehood,
				Source:       KnowledgeSourceSnapshot{Kind: KnowledgeSourceDialogue, DisplayName: "Existing informant"},
				DiscoveredOn: &BritannianDateSnapshot{Year: 139, Month: 4, Day: 5},
			}}
			data := historicalJournalEnvelopeData(t, snapshot)
			decoded, err := Decode(data)
			require.NoError(t, err)
			snapshot.Version = CurrentVersion
			require.Equal(t, snapshot, decoded.Snapshot, "only the version marker changes; there is no data conversion")
			for _, fact := range decoded.Snapshot.Redux.PlayerKnowledge {
				require.NotEqual(t, JournalKnowledgeStonesMelody, fact.Key)
			}
		})
	}
}

func TestStonesMelodyV19CodecAndFrozenDomains(t *testing.T) {
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
	frozen, _ := compileNativeSaveSchemaFile(t, "native-save-e2-s18.schema.json")
	for _, version := range []int{16, 17, 18} {
		snapshot.Version = version
		require.NotEmpty(t, validateSnapshot(snapshot, version))
	}
	snapshot.Presentation = journal.EmptyPresentation()
	historical := historicalJournalEnvelopeData(t, snapshot)
	_, err = Decode(historical)
	require.Error(t, err)
	require.Error(t, frozen.Validate(decodeJSONSchemaInstance(t, historical)))
	snapshot.Redux.PlayerKnowledge = []PlayerKnowledgeSnapshot{}
	historical = historicalJournalEnvelopeData(t, snapshot)
	_, err = Decode(historical)
	require.Error(t, err, "an event alone cannot bypass the frozen knowledge catalog")
	require.Error(t, frozen.Validate(decodeJSONSchemaInstance(t, historical)))
	snapshot.Redux.QuestJournal = []QuestJournalEventSnapshot{}
	historical = historicalJournalEnvelopeData(t, snapshot)
	require.NoError(t, frozen.Validate(decodeJSONSchemaInstance(t, historical)))
	decoded, err = Decode(historical)
	require.NoError(t, err)
	snapshot.Version = CurrentVersion
	require.Equal(t, snapshot, decoded.Snapshot, "promotion never grants the tune")
}
