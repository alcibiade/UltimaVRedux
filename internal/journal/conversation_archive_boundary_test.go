package journal

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConversationRejectsNotebookArchiveStateWithoutMutation(t *testing.T) {
	archived := testExchange("archived", "Kept notebook speech")
	archived.Archived = true
	require.NoError(t, archived.Validate(), "an archived exchange remains valid notebook data")
	invalid := Conversation{Exchanges: []Exchange{archived}, Truncated: false}
	require.ErrorIs(t, invalid.Validate(), ErrInvalidExchange)

	for _, test := range []struct {
		name         string
		conversation Conversation
	}{
		{name: "empty", conversation: Conversation{}},
		{name: "append", conversation: Conversation{Exchanges: []Exchange{testExchange("prior", "Prior speech")}}},
		{name: "replace", conversation: Conversation{Exchanges: []Exchange{testExchange("archived", "Original speech")}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			before := test.conversation.Clone()
			require.ErrorIs(t, test.conversation.Present(archived), ErrInvalidExchange)
			require.Equal(t, before, test.conversation, "rejection must precede replacement or retention eviction")
		})
	}
}
