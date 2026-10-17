package journal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bradhannah/UltimaVRedux/internal/repositoryfs"
)

const (
	notebookVersion     = 1
	playthroughIDLength = 32
)

// ErrInvalidNotebook identifies unsupported legacy storage or playthrough metadata.
var ErrInvalidNotebook = errors.New("invalid excerpt notebook")

// Notebook is the frozen external format, retained only for old-save import.
type Notebook struct {
	Version  int        `json:"version"`
	Excerpts []Exchange `json:"excerpts"`
}

// NotebookRepository is a read-only compatibility source. Runtime edits belong
// to Excerpts and native saves; this repository deliberately has no write API.
type NotebookRepository struct{ Root string }

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

// Load never creates a file or overwrites an unreadable legacy notebook.
func (r NotebookRepository) Load(id string) (Notebook, error) {
	if !notebookKey(id) || r.Root == "" {
		return Notebook{}, fmt.Errorf("%w: directory or playthrough identity", ErrInvalidNotebook)
	}
	data, err := repositoryfs.ReadRegularFile(filepath.Join(r.Root, id+".json"))
	if errors.Is(err, os.ErrNotExist) {
		return Notebook{Version: notebookVersion, Excerpts: nil}, nil
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
	if err = Excerpts(notebook.Excerpts).Validate(); err != nil {
		return Notebook{}, fmt.Errorf("validate notebook: %w", err)
	}

	return notebook, nil
}
