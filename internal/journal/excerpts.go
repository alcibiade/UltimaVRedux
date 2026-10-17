package journal

import (
	"fmt"
	"slices"
	"strings"
)

// Excerpts is the active game's kept speech. It is persisted only in snapshots.
// Mutations are transactional: a failed validation leaves the collection intact.
type Excerpts []Exchange

// Clone detaches the collection and every passage from its owner.
func (e Excerpts) Clone() Excerpts {
	cloned := slices.Clone(e)
	for i := range cloned {
		cloned[i] = cloned[i].Clone()
	}

	return cloned
}

// Validate preserves the existing notebook bounds and unique exchange identities.
func (e Excerpts) Validate() error {
	return validateExchanges(e, MaxExcerpts, MaxNotebookBytes)
}

// Keep restores or extends an exchange without shortening already kept speech.
func (e *Excerpts) Keep(exchange Exchange) error {
	if err := exchange.Validate(); err != nil {
		return err
	}
	next := e.Clone()
	exchange.Archived = false
	index := slices.IndexFunc(next, func(previous Exchange) bool { return previous.ID == exchange.ID })
	if index < 0 {
		next = append(next, exchange.Clone())
	} else {
		previous := &next[index]
		if exchange.speechSize() > previous.speechSize() {
			previous.Passages = exchange.Clone().Passages
		}
		if previous.Speaker == UnknownSpeaker && exchange.Speaker != UnknownSpeaker {
			previous.Speaker = exchange.Speaker
		}
		previous.Location = exchange.Location
		previous.Archived = false
	}
	if err := next.Validate(); err != nil {
		return err
	}
	*e = next

	return nil
}

// SetArchived changes only existing entries; a stale selection cannot recreate one.
func (e Excerpts) SetArchived(id string, archived bool) {
	for i := range e {
		if e[i].ID == id {
			e[i].Archived = archived

			return
		}
	}
}

// Delete removes only the selected identity.
func (e *Excerpts) Delete(id string) {
	*e = slices.DeleteFunc(*e, func(exchange Exchange) bool { return exchange.ID == id })
}

// AttributeConversation corrects existing notes without capturing unseen speech.
func (e *Excerpts) AttributeConversation(conversationID, speaker string) error {
	if !notebookKey(conversationID) || speaker == UnknownSpeaker || speaker == "" ||
		len(speaker) > maxAttributionBytes || !validText(speaker, false) {
		return fmt.Errorf("%w: speaker attribution", ErrInvalidExchange)
	}
	next := e.Clone()
	for i := range next {
		if strings.HasPrefix(next[i].ID, conversationID+"-") && next[i].Speaker == UnknownSpeaker {
			next[i].Speaker = speaker
		}
	}
	if err := next.Validate(); err != nil {
		return err
	}
	*e = next

	return nil
}
