package save_snapshot

import (
	"encoding/json"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/stretchr/testify/require"
)

func TestV17ConversationRoundTripAndV16Promotion(t *testing.T) {
	snapshot := validSnapshot()
	snapshot.Presentation.History = "Prior output"
	snapshot.LastConversation = journal.Conversation{Exchanges: []journal.Exchange{{
		ID: "exchange-1", Speaker: "Speaker", Location: "Britain", Input: "WORD",
		Passages: []journal.Passage{{Text: "AIPONI", Runic: true}},
	}}, Truncated: false}
	data, err := EncodeCanonical(validEnvelope(snapshot))
	require.NoError(t, err)
	loaded, err := Decode(data)
	require.NoError(t, err)
	require.Equal(t, snapshot, loaded.Snapshot)
	schema, _ := compileCurrentNativeSaveSchema(t)
	require.NoError(t, schema.Validate(decodeJSONSchemaInstance(t, data)))
	old := currentEnvelopeDocument(t, snapshot)
	body := old["snapshot"].(map[string]any)
	body["version"] = 16
	removeV20Fields(body)
	delete(body, "lastConversation")
	delete(body, "dungeonJournal")
	data, err = json.Marshal(old)
	require.NoError(t, err)
	loaded, err = Decode(data)
	require.NoError(t, err)
	require.Equal(t, snapshot.Presentation, loaded.Snapshot.Presentation)
	require.Empty(t, loaded.Snapshot.LastConversation.Exchanges)
	body["lastConversation"] = map[string]any{"exchanges": nil, "truncated": false}
	data, err = json.Marshal(old)
	require.NoError(t, err)
	_, err = Decode(data)
	require.Error(t, err, "frozen v16 must reject the new field")
}
