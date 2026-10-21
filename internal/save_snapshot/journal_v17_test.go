package save_snapshot

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/stretchr/testify/require"
)

func TestV17ConversationRoundTrip(t *testing.T) {
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
}
