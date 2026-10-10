package journal

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	// UnknownSpeaker is the neutral attribution until a name is presented.
	UnknownSpeaker = "Speaker"
	// MaxExchanges and MaxConversationBytes bound the recent conversation reader.
	MaxExchanges = 256
	// MaxConversationBytes bounds retained source including attribution.
	MaxConversationBytes = 256 * 1024
	// MaxExcerpts and MaxNotebookBytes never evict deliberately kept passages.
	MaxExcerpts = 512
	// MaxNotebookBytes bounds all kept source text in a playthrough.
	MaxNotebookBytes    = 4 * 1024 * 1024
	maxAttributionBytes = 256
)

// ErrNotebookFull means the player must remove an excerpt before keeping more.
var ErrNotebookFull = errors.New("excerpt notebook is full")

// ErrInvalidExchange identifies malformed or conflicting captured evidence.
var ErrInvalidExchange = errors.New("invalid conversation exchange")

// Passage preserves the script used to present text, without translating Runic.
type Passage struct {
	Text  string `json:"text"`
	Runic bool   `json:"runic"`
}

// Exchange contains only presented speech. ID is a presentation identity, not lore.
type Exchange struct {
	ID       string    `json:"id"`
	Speaker  string    `json:"speaker"`
	Location string    `json:"location"`
	Input    string    `json:"input"`
	Passages []Passage `json:"passages"`
	// Archived is notebook-only presentation state; older notebooks default active.
	Archived bool `json:"archived,omitempty"`
}

// Conversation retains the latest interaction independently of its live widgets.
type Conversation struct {
	Exchanges []Exchange `json:"exchanges"`
	Truncated bool       `json:"truncated"`
}

// Clone returns fully detached passage data.
func (e Exchange) Clone() Exchange {
	e.Passages = slices.Clone(e.Passages)

	return e
}

// Clone returns a detached conversation, preserving the canonical empty value.
func (c Conversation) Clone() Conversation {
	c.Exchanges = slices.Clone(c.Exchanges)
	for i := range c.Exchanges {
		c.Exchanges[i] = c.Exchanges[i].Clone()
	}

	return c
}

// Title uses only attribution and input already known to the player.
func (e Exchange) Title() string {
	topic := e.Input
	if topic == "" {
		topic = "Greeting"
	}

	return e.Speaker + " - " + topic
}

// SearchText deliberately excludes transliterated Runic and internal identities.
func (e Exchange) SearchText() string {
	var result strings.Builder
	result.WriteString(e.Title() + " " + e.Location)
	for _, passage := range e.Passages {
		if !passage.Runic {
			result.WriteByte(' ')
			result.WriteString(passage.Text)
		}
	}

	return result.String()
}

func (e Exchange) size() int {
	return len(e.ID) + len(e.Speaker) + len(e.Location) + len(e.Input) + e.speechSize()
}

func (e Exchange) speechSize() int {
	size := 0
	for _, passage := range e.Passages {
		size += len(passage.Text)
	}

	return size
}

func validText(value string, multiline bool) bool {
	if !utf8.ValidString(value) {
		return false
	}
	for _, char := range value {
		if unicode.IsControl(char) && (!multiline || char != '\n') {
			return false
		}
	}

	return true
}

// Validate rejects malformed persistence rather than repairing captured evidence.
func (e Exchange) Validate() error {
	if e.ID == "" || len(e.ID) > maxAttributionBytes || !validText(e.ID, false) {
		return fmt.Errorf("%w: identity", ErrInvalidExchange)
	}
	for _, value := range []string{e.Speaker, e.Location, e.Input} {
		if len(value) > maxAttributionBytes || !validText(value, false) {
			return fmt.Errorf("%w: attribution", ErrInvalidExchange)
		}
	}
	if e.Speaker == "" || len(e.Passages) == 0 || e.size() > MaxConversationBytes {
		return fmt.Errorf("%w: size or speaker", ErrInvalidExchange)
	}
	for _, passage := range e.Passages {
		if passage.Text == "" || !validText(passage.Text, true) {
			return fmt.Errorf("%w: passage", ErrInvalidExchange)
		}
	}

	return nil
}

// Validate enforces bounded, unique recent exchanges without notebook-only state.
func (c Conversation) Validate() error {
	for _, exchange := range c.Exchanges {
		if exchange.Archived {
			return fmt.Errorf("%w: conversation exchange cannot be archived", ErrInvalidExchange)
		}
	}

	return validateExchanges(c.Exchanges, MaxExchanges, MaxConversationBytes)
}

func validateExchanges(exchanges []Exchange, count, bytes int) error {
	if len(exchanges) > count {
		return ErrNotebookFull
	}
	seen := make(map[string]bool, len(exchanges))
	size := 0
	for _, exchange := range exchanges {
		if err := exchange.Validate(); err != nil {
			return fmt.Errorf("exchange: %w", err)
		}
		if seen[exchange.ID] {
			return fmt.Errorf("%w: duplicate identity", ErrInvalidExchange)
		}
		seen[exchange.ID] = true
		size += exchange.size()
	}
	if size > bytes {
		return ErrNotebookFull
	}

	return nil
}

// Present updates one exchange and drops oldest complete exchanges at the bound.
// An individually oversized exchange is rejected; callers must report overflow.
func (c *Conversation) Present(exchange Exchange) error {
	if exchange.Archived {
		return fmt.Errorf("%w: conversation exchange cannot be archived", ErrInvalidExchange)
	}
	if err := exchange.Validate(); err != nil {
		return err
	}
	if len(c.Exchanges) > 0 && c.Exchanges[len(c.Exchanges)-1].ID == exchange.ID {
		c.Exchanges[len(c.Exchanges)-1] = exchange.Clone()
	} else {
		c.Exchanges = append(c.Exchanges, exchange.Clone())
	}
	for c.Validate() != nil && len(c.Exchanges) > 1 {
		c.Exchanges = slices.Delete(c.Exchanges, 0, 1)
		c.Truncated = true
	}

	return c.Validate()
}
