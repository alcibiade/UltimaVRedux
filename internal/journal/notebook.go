package journal

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/repositoryfs"
)

const (
	notebookVersion       = 1
	playthroughIDLength   = 32
	notebookDirectoryMode = 0o700
	notebookFileMode      = 0o600
)

// ErrInvalidNotebook identifies unsupported storage or playthrough metadata.
var ErrInvalidNotebook = errors.New("invalid excerpt notebook")

// Notebook stores deliberate player selections, never gameplay authority.
type Notebook struct {
	Version  int        `json:"version"`
	Excerpts []Exchange `json:"excerpts"`
}

// NotebookRepository persists outside checkpoints, partitioned by playthrough.
type NotebookRepository struct {
	Root string
}

func notebookKey(id string) bool {
	if len(id) != playthroughIDLength {
		return false
	}
	for _, ch := range id {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return false
		}
	}

	return true
}

// Load never creates a file or overwrites an unreadable notebook.
func (r NotebookRepository) Load(id string) (Notebook, error) {
	if !notebookKey(id) {
		return Notebook{}, fmt.Errorf("%w: playthrough identity", ErrInvalidNotebook)
	}
	data, err := repositoryfs.ReadRegularFile(filepath.Join(r.Root, id+".json"))
	if errors.Is(err, os.ErrNotExist) {
		return Notebook{Version: notebookVersion, Excerpts: []Exchange{}}, nil
	}
	if err != nil {
		return Notebook{}, fmt.Errorf("read notebook: %w", err)
	}
	var notebook Notebook
	if err = repositoryfs.DecodeStrictJSON(data, &notebook); err != nil {
		return Notebook{}, fmt.Errorf("decode notebook: %w", err)
	}
	if notebook.Version != notebookVersion || notebook.Excerpts == nil {
		return Notebook{}, fmt.Errorf("%w: format", ErrInvalidNotebook)
	}
	if err = validateExchanges(notebook.Excerpts, MaxExcerpts, MaxNotebookBytes); err != nil {
		return Notebook{}, fmt.Errorf("validate notebook: %w", err)
	}

	return notebook, nil
}

// Keep atomically extends or adds an exchange without losing concurrent captures.
func (r NotebookRepository) Keep(id string, exchange Exchange) (Notebook, error) {
	if err := exchange.Validate(); err != nil {
		return Notebook{}, err
	}
	// Deliberately keeping an exchange restores it, even from an older checkpoint.
	exchange.Archived = false

	return r.update(id, func(notebook *Notebook) {
		for index, previous := range notebook.Excerpts {
			if previous.ID == exchange.ID {
				// An older checkpoint must not shorten an already-kept exchange.
				if exchange.speechSize() > previous.speechSize() {
					previous.Passages = exchange.Clone().Passages
				}
				if previous.Speaker == UnknownSpeaker && exchange.Speaker != UnknownSpeaker {
					previous.Speaker = exchange.Speaker
				}
				previous.Location = exchange.Location
				previous.Archived = false
				notebook.Excerpts[index] = previous

				return
			}
		}
		notebook.Excerpts = append(notebook.Excerpts, exchange.Clone())
	})
}

// SetArchived retains source and storage usage, changing only notebook visibility.
// Missing identities are a no-op, so a stale selection cannot resurrect a deletion.
func (r NotebookRepository) SetArchived(id, exchangeID string, archived bool) (Notebook, error) {
	return r.updateConditional(id, func(notebook *Notebook) bool {
		for index := range notebook.Excerpts {
			if notebook.Excerpts[index].ID == exchangeID && notebook.Excerpts[index].Archived != archived {
				notebook.Excerpts[index].Archived = archived

				return true
			}
		}

		return false
	})
}

// AttributeConversation updates already-kept exchanges after the speaker's name is presented.
func (r NotebookRepository) AttributeConversation(playthroughID, conversationID, speaker string) (Notebook, error) {
	if !notebookKey(conversationID) || speaker == UnknownSpeaker || speaker == "" ||
		len(speaker) > maxAttributionBytes || !validText(speaker, false) {
		return Notebook{}, fmt.Errorf("%w: speaker attribution", ErrInvalidNotebook)
	}
	prefix := conversationID + "-"

	return r.updateConditional(playthroughID, func(notebook *Notebook) bool {
		changed := false
		for index := range notebook.Excerpts {
			if strings.HasPrefix(notebook.Excerpts[index].ID, prefix) &&
				notebook.Excerpts[index].Speaker == UnknownSpeaker {
				notebook.Excerpts[index].Speaker = speaker
				changed = true
			}
		}

		return changed
	})
}

// Delete removes only the selected stable identity.
func (r NotebookRepository) Delete(id, exchangeID string) (Notebook, error) {
	return r.update(id, func(notebook *Notebook) {
		notebook.Excerpts = slices.DeleteFunc(notebook.Excerpts, func(e Exchange) bool { return e.ID == exchangeID })
	})
}

func (r NotebookRepository) update(playthrough string, mutate func(*Notebook)) (Notebook, error) {
	return r.updateConditional(playthrough, func(notebook *Notebook) bool {
		mutate(notebook)

		return true
	})
}

func (r NotebookRepository) updateConditional(playthrough string, mutate func(*Notebook) bool) (Notebook, error) {
	if !notebookKey(playthrough) {
		return Notebook{}, fmt.Errorf("%w: playthrough identity", ErrInvalidNotebook)
	}
	root := repositoryfs.Root{Path: r.Root, DirMode: notebookDirectoryMode, FileMode: notebookFileMode}
	unlock, err := repositoryfs.AcquireReferenceLocks([]repositoryfs.Root{root})
	if err != nil {
		return Notebook{}, fmt.Errorf("lock notebook: %w", err)
	}
	defer unlock()
	notebook, err := r.Load(playthrough)
	if err != nil {
		return Notebook{}, err
	}
	if !mutate(&notebook) {
		return notebook, nil
	}
	if err = validateExchanges(notebook.Excerpts, MaxExcerpts, MaxNotebookBytes); err != nil {
		return Notebook{}, err
	}
	data, err := json.Marshal(notebook)
	if err != nil {
		return Notebook{}, fmt.Errorf("encode notebook: %w", err)
	}
	err = repositoryfs.PublishTemporaryFile(r.Root, ".excerpt-*", data, root.FileMode, func(temporary string) error {
		return os.Rename(temporary, filepath.Join(r.Root, playthrough+".json"))
	})
	if err != nil {
		return Notebook{}, fmt.Errorf("publish notebook: %w", err)
	}
	if err = repositoryfs.SyncDirectory(r.Root); err != nil {
		return Notebook{}, fmt.Errorf("sync notebook: %w", err)
	}

	return notebook, nil
}
