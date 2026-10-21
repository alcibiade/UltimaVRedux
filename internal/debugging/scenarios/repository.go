package scenarios

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/repositoryfs"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
)

const (
	// SourceProject identifies scenarios shared from the project filesystem root.
	SourceProject Source = "project"
	// SourcePersonal identifies scenarios from the injected personal filesystem root.
	SourcePersonal Source = "personal"

	//nolint:unused // Referenced by package-level repository persistence tests.
	repositoryLockName = repositoryfs.ReferenceLockName
)

var (
	// ErrInvalidReference indicates an invalid source-qualified scenario reference.
	ErrInvalidReference = errors.New("invalid scenario reference")
	// ErrScenarioNotFound indicates that a referenced scenario file does not exist.
	ErrScenarioNotFound = errors.New("scenario not found")
	// ErrScenarioExists indicates that a create destination is already occupied.
	ErrScenarioExists = errors.New("scenario already exists")
	// ErrScenarioStale indicates that stored content no longer has the expected revision.
	ErrScenarioStale = errors.New("scenario revision is stale")
	// ErrSnapshotBaseline indicates an unavailable or non-portable named snapshot baseline.
	ErrSnapshotBaseline = errors.New("invalid scenario snapshot baseline")
	// ErrPendingMigration indicates that a source must be explicitly updated before duplication.
	ErrPendingMigration = errors.New("scenario has pending migrations")
)

// Source identifies one configured repository root.
type Source string

// Reference uniquely identifies a scenario and its source.
type Reference struct {
	Source Source
	ID     string
}

// Summary is deterministic list metadata for one stored definition.
type Summary struct {
	Reference   Reference
	Name        string
	Description string
	Baseline    Baseline
	Revision    string
	Parameters  []debugging.ParameterDefinition
	Steps       []Step
	Migrations  []Migration
}

// Record is a stored definition and the revision required for conditional mutation.
type Record struct {
	Definition Definition
	Revision   string
	Migrations []Migration
}

// Repository persists validated definitions beneath injected project and personal roots.
// Mutations are serialized across cooperating Repository instances and processes.
// Non-cooperating filesystem writers can still race after the final revision check;
// portable rename and remove APIs do not provide compare-and-swap semantics.
type Repository struct {
	projectRoot        string
	personalRoot       string
	snapshotRepository interface {
		Load(reference string) (save_snapshot.Record, error)
	}
	mutationMu sync.Mutex
}

// ConfigureSnapshotRepository enables named baseline existence validation.
func (repository *Repository) ConfigureSnapshotRepository(snapshotRepository interface {
	Load(reference string) (save_snapshot.Record, error)
}) {
	repository.snapshotRepository = snapshotRepository
	if coordinator, ok := snapshotRepository.(interface {
		ConfigureReferenceLockRoots(roots []repositoryfs.Root)
	}); ok {
		coordinator.ConfigureReferenceLockRoots(repository.ownRoots())
	}
}

// NewRepository constructs a repository without assuming either filesystem location.
func NewRepository(projectRoot, personalRoot string) (*Repository, error) {
	if strings.TrimSpace(projectRoot) == "" || strings.TrimSpace(personalRoot) == "" {
		return nil, errors.New("scenario repository requires project and personal roots")
	}

	return &Repository{projectRoot: filepath.Clean(projectRoot), personalRoot: filepath.Clean(personalRoot)}, nil
}

// ParseReference parses a source-qualified project:<id> or personal:<id> reference.
func ParseReference(value string) (Reference, error) {
	sourceText, id, found := strings.Cut(value, ":")
	reference := Reference{Source: Source(sourceText), ID: id}
	if !found || (reference.Source != SourceProject && reference.Source != SourcePersonal) || !slugPattern.MatchString(reference.ID) {
		return Reference{}, fmt.Errorf("%w: %q", ErrInvalidReference, value)
	}

	return reference, nil
}

// String returns the stable source-qualified representation.
func (reference Reference) String() string {
	return string(reference.Source) + ":" + reference.ID
}

// List returns all valid stored scenarios in source then ID order.
func (repository *Repository) List() ([]Summary, error) {
	summaries := make([]Summary, 0)
	for _, source := range []Source{SourceProject, SourcePersonal} {
		root := repository.root(source)
		info, err := os.Stat(root)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			return nil, fmt.Errorf("list %s scenarios: stat root: %w", source, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("list %s scenarios: root is not a directory", source)
		}
		entries, err := os.ReadDir(root)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			return nil, fmt.Errorf("list %s scenarios: %w", source, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
				continue
			}
			id := strings.TrimSuffix(entry.Name(), ".json")
			reference, err := ParseReference(string(source) + ":" + id)
			if err != nil {
				return nil, fmt.Errorf("list %s scenarios: %w", source, err)
			}
			record, err := repository.LoadRecord(reference.String())
			if err != nil {
				return nil, err
			}
			summaries = append(summaries, Summary{
				Reference: reference, Name: record.Definition.Name, Description: record.Definition.Description,
				Baseline: record.Definition.Baseline, Revision: record.Revision,
				Parameters: record.Definition.Parameters, Steps: record.Definition.Steps, Migrations: record.Migrations,
			})
		}
	}
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].Reference.Source != summaries[j].Reference.Source {
			return summaries[i].Reference.Source == SourceProject
		}

		return summaries[i].Reference.ID < summaries[j].Reference.ID
	})

	return summaries, nil
}

// LoadRecord reads a validated definition and returns its canonical content revision.
func (repository *Repository) LoadRecord(qualifiedReference string) (Record, error) {
	reference, err := ParseReference(qualifiedReference)
	if err != nil {
		return Record{}, err
	}
	path := repository.path(reference)
	data, err := repositoryfs.ReadRegularFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Record{}, fmt.Errorf("%w: %s", ErrScenarioNotFound, reference)
		}
		if errors.Is(err, repositoryfs.ErrNotRegular) {
			return Record{}, fmt.Errorf("%w: scenario path must be a regular file", ErrInvalidReference)
		}

		return Record{}, fmt.Errorf("inspect scenario %s: %w", reference, err)
	}
	definition, err := DecodeDefinition(data)
	if err != nil {
		return Record{}, fmt.Errorf("load scenario %s: %w", reference, err)
	}
	if definition.ID != reference.ID {
		return Record{}, fmt.Errorf("%w: file ID %q does not match reference %q", ErrInvalidDefinition, definition.ID, reference.ID)
	}
	canonical, err := encodeDefinition(reference, definition)
	if err != nil {
		return Record{}, err
	}
	migrated, migrations := MigrateDefinition(definition)

	return Record{Definition: migrated, Revision: repositoryfs.ContentRevision(canonical), Migrations: migrations}, nil
}

// Create stores a new definition only when the destination is absent.
func (repository *Repository) Create(qualifiedReference string, definition Definition) (Record, error) {
	repository.mutationMu.Lock()
	defer repository.mutationMu.Unlock()
	unlock, err := repository.lockMutations()
	if err != nil {
		return Record{}, err
	}
	defer unlock()

	return repository.create(qualifiedReference, definition)
}

func (repository *Repository) create(qualifiedReference string, definition Definition) (Record, error) {
	reference, err := ParseReference(qualifiedReference)
	if err != nil {
		return Record{}, err
	}
	data, err := encodeDefinition(reference, definition)
	if err != nil {
		return Record{}, err
	}
	if err = repository.validateBaseline(reference.Source, definition.Baseline); err != nil {
		return Record{}, err
	}
	if err = repository.writeCreate(reference, data, nil); err != nil {
		return Record{}, err
	}

	return Record{Definition: definition, Revision: repositoryfs.ContentRevision(data)}, nil
}

// Update replaces a definition only when expectedRevision matches stored content.
func (repository *Repository) Update(qualifiedReference, expectedRevision string, definition Definition) (Record, error) {
	repository.mutationMu.Lock()
	defer repository.mutationMu.Unlock()
	unlock, err := repository.lockMutations()
	if err != nil {
		return Record{}, err
	}
	defer unlock()

	reference, err := ParseReference(qualifiedReference)
	if err != nil {
		return Record{}, err
	}
	data, err := encodeDefinition(reference, definition)
	if err != nil {
		return Record{}, err
	}
	if err = repository.validateBaseline(reference.Source, definition.Baseline); err != nil {
		return Record{}, err
	}
	if _, err = repository.requireRevision(reference, expectedRevision); err != nil {
		return Record{}, err
	}
	if err = repository.writeConditionalReplace(reference, expectedRevision, data); err != nil {
		return Record{}, err
	}

	return Record{Definition: definition, Revision: repositoryfs.ContentRevision(data)}, nil
}

// Delete removes a regular scenario file only when its revision is current.
func (repository *Repository) Delete(qualifiedReference, expectedRevision string) error {
	repository.mutationMu.Lock()
	defer repository.mutationMu.Unlock()
	unlock, err := repository.lockMutations()
	if err != nil {
		return err
	}
	defer unlock()

	return repository.delete(qualifiedReference, expectedRevision)
}

func (repository *Repository) delete(qualifiedReference, expectedRevision string) error {
	reference, err := ParseReference(qualifiedReference)
	if err != nil {
		return err
	}
	// Recheck immediately before removal so a replaced path is never deleted blindly.
	if _, err = repository.requireRevision(reference, expectedRevision); err != nil {
		return err
	}
	if err = os.Remove(repository.path(reference)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrScenarioNotFound, reference)
		}

		return fmt.Errorf("delete scenario %s: %w", reference, err)
	}
	if err = repositoryfs.SyncDirectory(repository.root(reference.Source)); err != nil {
		return fmt.Errorf("persist deletion of scenario %s: %w", reference, err)
	}

	return nil
}

// DuplicateAs copies a current source definition to an absent destination and
// optionally replaces its display name in the same conditional transaction.
func (repository *Repository) DuplicateAs(sourceReference, sourceRevision, destinationReference, destinationName string) (Record, error) {
	repository.mutationMu.Lock()
	defer repository.mutationMu.Unlock()
	unlock, err := repository.lockMutations()
	if err != nil {
		return Record{}, err
	}
	defer unlock()

	return repository.duplicate(sourceReference, sourceRevision, destinationReference, destinationName)
}

func (repository *Repository) duplicate(sourceReference, sourceRevision, destinationReference, destinationName string) (Record, error) {
	source, err := ParseReference(sourceReference)
	if err != nil {
		return Record{}, err
	}
	destination, err := ParseReference(destinationReference)
	if err != nil {
		return Record{}, err
	}
	record, err := repository.requireRevision(source, sourceRevision)
	if err != nil {
		return Record{}, err
	}
	if len(record.Migrations) > 0 {
		return Record{}, fmt.Errorf("%w: explicitly update %s before duplicating", ErrPendingMigration, source)
	}
	record.Definition.ID = destination.ID
	if destinationName != "" {
		record.Definition.Name = destinationName
	}
	if err = repository.validateBaseline(destination.Source, record.Definition.Baseline); err != nil {
		return Record{}, err
	}
	data, err := encodeDefinition(destination, record.Definition)
	if err != nil {
		return Record{}, err
	}
	if err = repository.writeCreate(destination, data, func() error {
		_, revisionErr := repository.requireRevision(source, sourceRevision)

		return revisionErr
	}); err != nil {
		return Record{}, err
	}

	return Record{Definition: record.Definition, Revision: repositoryfs.ContentRevision(data)}, nil
}

func (repository *Repository) validateBaseline(source Source, baseline Baseline) error {
	if baseline.Type != BaselineSnapshot {
		return nil
	}
	reference, err := save_snapshot.ParseReference(baseline.Reference)
	if err != nil || reference.Source == save_snapshot.SourceRollback {
		return fmt.Errorf("%w: reference %q is not a named snapshot", ErrSnapshotBaseline, baseline.Reference)
	}
	if source == SourceProject && reference.Source == save_snapshot.SourcePersonal {
		return fmt.Errorf("%w: project scenarios cannot reference personal snapshots", ErrSnapshotBaseline)
	}
	if repository.snapshotRepository == nil {
		return nil
	}
	if _, err = repository.snapshotRepository.Load(baseline.Reference); err != nil {
		return fmt.Errorf("%w: load %s: %w", ErrSnapshotBaseline, baseline.Reference, err)
	}

	return nil
}

func encodeDefinition(reference Reference, definition Definition) ([]byte, error) {
	if definition.ID != reference.ID {
		return nil, fmt.Errorf("%w: definition ID %q does not match reference %q", ErrInvalidDefinition, definition.ID, reference.ID)
	}
	if err := definition.Validate(); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(definition, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode scenario %s: %w", reference, err)
	}

	return append(data, '\n'), nil
}

func (repository *Repository) requireRevision(reference Reference, expectedRevision string) (Record, error) {
	record, err := repository.LoadRecord(reference.String())
	if err != nil {
		return Record{}, err
	}
	if record.Revision != expectedRevision {
		return Record{}, fmt.Errorf("%w: %s", ErrScenarioStale, reference)
	}

	return record, nil
}

func (repository *Repository) writeCreate(reference Reference, data []byte, beforeCommit func() error) error {
	root := repository.root(reference.Source)
	policy := repository.rootPolicy(reference.Source)
	if err := repositoryfs.EnsureRoot(policy); err != nil {
		return fmt.Errorf("create scenario root: %w", err)
	}
	if err := repositoryfs.PublishTemporaryFile(root, ".scenario-*.tmp", data, policy.FileMode, func(temporaryPath string) error {
		if beforeCommit != nil {
			if commitErr := beforeCommit(); commitErr != nil {
				return commitErr
			}
		}
		if linkErr := os.Link(temporaryPath, repository.path(reference)); linkErr != nil {
			if errors.Is(linkErr, os.ErrExist) {
				return fmt.Errorf("%w: %s", ErrScenarioExists, reference)
			}

			return fmt.Errorf("create scenario %s: %w", reference, linkErr)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("write scenario %s: %w", reference, err)
	}
	if err := repositoryfs.SyncDirectory(root); err != nil {
		return fmt.Errorf("persist scenario %s: %w", reference, err)
	}

	return nil
}

func (repository *Repository) writeConditionalReplace(reference Reference, expectedRevision string, data []byte) error {
	root := repository.root(reference.Source)
	if err := repositoryfs.PublishTemporaryFile(
		root,
		".scenario-*.tmp",
		data,
		repository.rootPolicy(reference.Source).FileMode,
		func(temporaryPath string) error {
			if _, revisionErr := repository.requireRevision(reference, expectedRevision); revisionErr != nil {
				return revisionErr
			}
			if renameErr := os.Rename(temporaryPath, repository.path(reference)); renameErr != nil {
				return fmt.Errorf("replace scenario %s: %w", reference, renameErr)
			}

			return nil
		},
	); err != nil {
		return fmt.Errorf("write scenario %s: %w", reference, err)
	}
	if err := repositoryfs.SyncDirectory(root); err != nil {
		return fmt.Errorf("persist scenario %s: %w", reference, err)
	}

	return nil
}

func (repository *Repository) root(source Source) string {
	if source == SourceProject {
		return repository.projectRoot
	}

	return repository.personalRoot
}

func (repository *Repository) path(reference Reference) string {
	return filepath.Join(repository.root(reference.Source), reference.ID+".json")
}

func (repository *Repository) lockMutations() (func(), error) {
	roots := repository.ownRoots()
	if coordinator, ok := repository.snapshotRepository.(interface {
		ReferenceLockRoots() []repositoryfs.Root
	}); ok {
		roots = append(roots, coordinator.ReferenceLockRoots()...)
	}

	return repositoryfs.AcquireReferenceLocks(roots)
}

func (repository *Repository) ownRoots() []repositoryfs.Root {
	return []repositoryfs.Root{repository.rootPolicy(SourceProject), repository.rootPolicy(SourcePersonal)}
}

func (repository *Repository) rootPolicy(source Source) repositoryfs.Root {
	mode, fileMode := os.FileMode(0o700), os.FileMode(0o600)
	if source == SourceProject {
		mode, fileMode = 0o755, 0o644
	}

	return repositoryfs.Root{Path: repository.root(source), DirMode: mode, FileMode: fileMode}
}
