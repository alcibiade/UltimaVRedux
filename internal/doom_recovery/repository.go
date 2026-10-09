// Package doom_recovery persists the single pre-Doom recovery checkpoint.
package doom_recovery

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bradhannah/UltimaVRedux/internal/repositoryfs"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
)

const (
	recoveryDirectory = "recovery"
	recoveryFile      = "pre-doom.json"
	rootMode          = os.FileMode(0o700)
	fileMode          = os.FileMode(0o600)
)

var (
	errInvalidRoot = errors.New("pre-Doom recovery repository requires the player-save root")
	// ErrCheckpointNotFound means no pre-Doom checkpoint has been prepared.
	ErrCheckpointNotFound = errors.New("pre-Doom recovery checkpoint not found")
	// ErrCheckpointIncompatible means the current runtime cannot hydrate the checkpoint.
	ErrCheckpointIncompatible = errors.New("pre-Doom recovery checkpoint is incompatible")
)

// Record contains the canonical recovery envelope and its SHA-256 revision.
type Record struct {
	Envelope save_snapshot.Envelope
	Revision string
}

// CompatibilityValidator verifies a decoded snapshot against runtime data.
type CompatibilityValidator func(save_snapshot.SaveSnapshot) error

// Repository owns one replaceable checkpoint below the player-save root.
type Repository struct {
	playerSaveRoot        string
	recoveryRoot          string
	validateCompatibility CompatibilityValidator
	mutationMu            sync.Mutex
	syncDirectory         func(string) error
}

// NewRepository constructs a dedicated recovery repository without creating directories.
func NewRepository(playerSaveRoot string, validateCompatibility CompatibilityValidator) (*Repository, error) {
	if strings.TrimSpace(playerSaveRoot) == "" {
		return nil, errInvalidRoot
	}
	root := filepath.Clean(playerSaveRoot)

	return &Repository{
		playerSaveRoot: root, recoveryRoot: filepath.Join(root, recoveryDirectory),
		validateCompatibility: validateCompatibility, mutationMu: sync.Mutex{},
		syncDirectory: repositoryfs.SyncDirectory,
	}, nil
}

// Load strictly decodes and compatibility-validates the current checkpoint.
func (repository *Repository) Load() (Record, error) {
	data, err := repositoryfs.ReadRegularFile(repository.path())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Record{}, ErrCheckpointNotFound
		}

		return Record{}, fmt.Errorf("read pre-Doom recovery checkpoint: %w", err)
	}
	envelope, err := save_snapshot.Decode(data)
	if err != nil {
		if errors.Is(err, save_snapshot.ErrUnsupportedSnapshotVersion) {
			return Record{}, fmt.Errorf("%w: %w", ErrCheckpointIncompatible, err)
		}

		return Record{}, fmt.Errorf("decode pre-Doom recovery checkpoint: %w", err)
	}
	if repository.validateCompatibility != nil {
		if err = repository.validateCompatibility(envelope.Snapshot); err != nil {
			return Record{}, fmt.Errorf("%w: %w", ErrCheckpointIncompatible, err)
		}
	}
	canonical, err := save_snapshot.EncodeCanonical(envelope)
	if err != nil {
		return Record{}, fmt.Errorf("canonicalize pre-Doom recovery checkpoint: %w", err)
	}

	return Record{Envelope: envelope, Revision: repositoryfs.ContentRevision(canonical)}, nil
}

// Replace atomically creates or supersedes the single recovery checkpoint.
// A failed directory sync rolls the file back to its prior exact bytes.
func (repository *Repository) Replace(envelope save_snapshot.Envelope) (Record, error) {
	data, err := save_snapshot.EncodeCanonical(envelope)
	if err != nil {
		return Record{}, fmt.Errorf("encode pre-Doom recovery checkpoint: %w", err)
	}

	repository.mutationMu.Lock()
	defer repository.mutationMu.Unlock()

	if err = repository.ensureRoots(); err != nil {
		return Record{}, err
	}
	previous, occupied, err := repository.readCurrentData()
	if err != nil {
		return Record{}, err
	}
	if err = repository.writeAndRename(data); err != nil {
		return Record{}, err
	}
	if err = repository.syncDirectory(repository.recoveryRoot); err != nil {
		persistErr := fmt.Errorf("persist pre-Doom recovery checkpoint: %w", err)
		if rollbackErr := repository.rollbackReplacement(previous, occupied); rollbackErr != nil {
			return Record{}, errors.Join(persistErr, fmt.Errorf("roll back pre-Doom recovery checkpoint: %w", rollbackErr))
		}

		return Record{}, persistErr
	}

	return Record{Envelope: envelope, Revision: repositoryfs.ContentRevision(data)}, nil
}

func (repository *Repository) ensureRoots() error {
	policy := func(path string) repositoryfs.Root {
		return repositoryfs.Root{Path: path, DirMode: rootMode, FileMode: fileMode}
	}
	if err := repositoryfs.EnsureRoot(policy(repository.playerSaveRoot)); err != nil {
		return fmt.Errorf("create player-save root for pre-Doom recovery: %w", err)
	}
	_, statErr := os.Stat(repository.recoveryRoot)
	recoveryRootCreated := errors.Is(statErr, os.ErrNotExist)
	if statErr != nil && !recoveryRootCreated {
		return fmt.Errorf("inspect pre-Doom recovery root: %w", statErr)
	}
	if err := repositoryfs.EnsureRoot(policy(repository.recoveryRoot)); err != nil {
		return fmt.Errorf("create pre-Doom recovery root: %w", err)
	}
	if recoveryRootCreated {
		if err := repository.syncDirectory(repository.playerSaveRoot); err != nil {
			persistErr := fmt.Errorf("persist pre-Doom recovery root: %w", err)
			if removeErr := os.Remove(repository.recoveryRoot); removeErr != nil {
				return errors.Join(persistErr, fmt.Errorf("remove uncommitted pre-Doom recovery root: %w", removeErr))
			}

			return persistErr
		}
	}

	return nil
}

func (repository *Repository) readCurrentData() ([]byte, bool, error) {
	data, err := repositoryfs.ReadRegularFile(repository.path())
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read current pre-Doom recovery checkpoint: %w", err)
	}

	return data, true, nil
}

func (repository *Repository) writeAndRename(data []byte) error {
	if err := repositoryfs.PublishTemporaryFile(
		repository.recoveryRoot,
		".pre-doom-*.tmp",
		data,
		fileMode,
		func(temporaryPath string) error {
			if renameErr := os.Rename(temporaryPath, repository.path()); renameErr != nil {
				return fmt.Errorf("replace pre-Doom recovery checkpoint: %w", renameErr)
			}

			return nil
		},
	); err != nil {
		return fmt.Errorf("write pre-Doom recovery checkpoint: %w", err)
	}

	return nil
}

func (repository *Repository) rollbackReplacement(previous []byte, occupied bool) error {
	if occupied {
		if err := repository.writeAndRename(previous); err != nil {
			return err
		}
	} else if err := os.Remove(repository.path()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove new pre-Doom recovery checkpoint: %w", err)
	}

	return repository.syncDirectory(repository.recoveryRoot)
}

func (repository *Repository) path() string {
	return filepath.Join(repository.recoveryRoot, recoveryFile)
}
