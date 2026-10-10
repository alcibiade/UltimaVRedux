package save_snapshot

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Exchange is shared with the notebook, but native v17/v18 conversation schemas
// do not admit archive metadata at all. Reject its presence before decoding can
// erase the distinction between an absent member and false (or null).
func validateConversationJSONMembers(data []byte) error {
	var conversation struct {
		Exchanges []map[string]json.RawMessage `json:"exchanges"`
	}
	if err := json.Unmarshal(data, &conversation); err != nil {
		return fmt.Errorf("%w: decode lastConversation members: %w", ErrInvalidEnvelope, err)
	}
	for index, exchange := range conversation.Exchanges {
		for member := range exchange {
			// encoding/json accepts case-insensitive field aliases too.
			if strings.EqualFold(member, "archived") {
				return fmt.Errorf("%w: lastConversation.exchanges[%d].%s is notebook-only", ErrInvalidEnvelope, index, member)
			}
		}
	}

	return nil
}
