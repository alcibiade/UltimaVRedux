package scenarios

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
	"time"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/repositoryfs"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
)

const crossProcessScenarioRole = "U5_SCENARIO_SNAPSHOT_LOCK_CHILD"

type scenarioSnapshotResolverStub struct {
	records map[string]save_snapshot.Record
}

func (resolver scenarioSnapshotResolverStub) Load(reference string) (save_snapshot.Record, error) {
	record, exists := resolver.records[reference]
	if !exists {
		return save_snapshot.Record{}, save_snapshot.ErrSnapshotNotFound
	}

	return record, nil
}

func TestRepositoryCreateLoadAndDeterministicList(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	repository, err := NewRepository(filepath.Join(root, "project"), filepath.Join(root, "personal"))
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}

	projectZ := validDefinition()
	projectZ.ID, projectZ.Name = "zeta", "Project Zeta"
	projectA := validDefinition()
	projectA.ID, projectA.Name = "alpha", "Project Alpha"
	personalA := validDefinition()
	personalA.ID, personalA.Name = "alpha", "Personal Alpha"
	for reference, definition := range map[string]Definition{
		"project:zeta": projectZ, "project:alpha": projectA, "personal:alpha": personalA,
	} {
		if _, err = repository.Create(reference, definition); err != nil {
			t.Fatalf("Create(%q) error = %v", reference, err)
		}
	}

	loaded, err := repository.Load("personal:alpha")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Name != "Personal Alpha" || loaded.Baseline != CurrentBaseline() {
		t.Fatalf("Load() = %#v", loaded)
	}
	summaries, err := repository.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	want := []string{"project:alpha", "project:zeta", "personal:alpha"}
	if len(summaries) != len(want) {
		t.Fatalf("List() count = %d, want %d", len(summaries), len(want))
	}
	for index, summary := range summaries {
		if summary.Reference.String() != want[index] {
			t.Fatalf("List()[%d] = %q, want %q", index, summary.Reference, want[index])
		}
		if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(summary.Revision) {
			t.Fatalf("List()[%d].Revision = %q, want SHA-256 hex", index, summary.Revision)
		}
	}
	if len(summaries[0].Steps) != len(projectA.Steps) {
		t.Fatalf("List()[0].Steps = %d entries, want %d", len(summaries[0].Steps), len(projectA.Steps))
	}
}

func TestRepositoryLoadMigratesLegacyCharacterTokensWithoutWritingOrChangingStoredRevision(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	projectRoot := filepath.Join(root, "project")
	repository, err := NewRepository(projectRoot, filepath.Join(root, "personal"))
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	if err = os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	definition := validDefinition()
	definition.Steps = []Step{legacyCharacterStep(partyCharacterUpdateCommand, `"0.QXZhdGFy"`)}
	stored, err := encodeDefinition(Reference{Source: SourceProject, ID: definition.ID}, definition)
	if err != nil {
		t.Fatalf("encodeDefinition() error = %v", err)
	}
	path := filepath.Join(projectRoot, definition.ID+".json")
	if err = os.WriteFile(path, stored, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	record, err := repository.LoadRecord("project:" + definition.ID)
	if err != nil {
		t.Fatalf("LoadRecord() error = %v", err)
	}
	if record.Revision != repositoryfs.ContentRevision(stored) {
		t.Fatalf("Revision = %q, want stored canonical revision %q", record.Revision, repositoryfs.ContentRevision(stored))
	}
	if got := string(record.Definition.Steps[0].Arguments["character"].Literal); got != `"1"` {
		t.Fatalf("migrated character = %s, want %q", got, "1")
	}
	if len(record.Migrations) != 1 {
		t.Fatalf("Migrations = %#v, want one", record.Migrations)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(after) != string(stored) {
		t.Fatal("LoadRecord() rewrote the stored scenario")
	}
	summaries, err := repository.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(summaries) != 1 || len(summaries[0].Migrations) != 1 {
		t.Fatalf("List() migrations = %#v", summaries)
	}
}

func TestRepositoryLoadRemovesRuntimeWorldPreconditionsWithoutWritingOrChangingStoredRevision(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	projectRoot := filepath.Join(root, "project")
	repository, err := NewRepository(projectRoot, filepath.Join(root, "personal"))
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	if err = os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	definition := validDefinition()
	definition.Steps = []Step{{
		Type: StepCommand, CommandID: worldPositionSetCommand,
		Arguments: map[string]CommandArgument{
			"location":                  {Literal: json.RawMessage(`"33"`)},
			"expectedLocationId":        {Literal: json.RawMessage(`2`)},
			"expectedSourceFingerprint": {Literal: json.RawMessage(`"stored-review"`)},
		},
	}}
	stored, err := encodeDefinition(Reference{Source: SourceProject, ID: definition.ID}, definition)
	if err != nil {
		t.Fatalf("encodeDefinition() error = %v", err)
	}
	path := filepath.Join(projectRoot, definition.ID+".json")
	if err = os.WriteFile(path, stored, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	record, err := repository.LoadRecord("project:" + definition.ID)
	if err != nil {
		t.Fatalf("LoadRecord() error = %v", err)
	}
	if record.Revision != repositoryfs.ContentRevision(stored) {
		t.Fatalf("Revision = %q, want stored canonical revision %q", record.Revision, repositoryfs.ContentRevision(stored))
	}
	if _, exists := record.Definition.Steps[0].Arguments["expectedLocationId"]; exists {
		t.Fatal("loaded scenario retained expectedLocationId")
	}
	if _, exists := record.Definition.Steps[0].Arguments["expectedSourceFingerprint"]; exists {
		t.Fatal("loaded scenario retained expectedSourceFingerprint")
	}
	if len(record.Migrations) != 2 {
		t.Fatalf("Migrations = %#v, want two", record.Migrations)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(after) != string(stored) {
		t.Fatal("LoadRecord() rewrote the stored scenario")
	}
}

func TestRepositoryPendingMigrationRequiresUpdateBeforeDuplicate(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	projectRoot := filepath.Join(root, "project")
	repository, err := NewRepository(projectRoot, filepath.Join(root, "personal"))
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	if err = os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	definition := validDefinition()
	definition.Steps = []Step{legacyCharacterStep(partyCharacterUpdateCommand, `"0.QXZhdGFy"`)}
	stored, err := encodeDefinition(Reference{Source: SourceProject, ID: definition.ID}, definition)
	if err != nil {
		t.Fatalf("encodeDefinition() error = %v", err)
	}
	if err = os.WriteFile(filepath.Join(projectRoot, definition.ID+".json"), stored, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	record, err := repository.LoadRecord("project:" + definition.ID)
	if err != nil {
		t.Fatalf("LoadRecord() error = %v", err)
	}

	if _, err = repository.Duplicate("project:"+definition.ID, record.Revision, "personal:migrated-copy"); !errors.Is(err, ErrPendingMigration) {
		t.Fatalf("Duplicate() error = %v, want ErrPendingMigration", err)
	}
	if _, err = repository.LoadRecord("personal:migrated-copy"); !errors.Is(err, ErrScenarioNotFound) {
		t.Fatalf("destination error = %v, want ErrScenarioNotFound", err)
	}
	updated, err := repository.Update("project:"+definition.ID, record.Revision, record.Definition)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if len(updated.Migrations) != 0 {
		t.Fatalf("Update() migrations = %#v", updated.Migrations)
	}
	if _, err = repository.Duplicate("project:"+definition.ID, updated.Revision, "personal:migrated-copy"); err != nil {
		t.Fatalf("Duplicate(after update) error = %v", err)
	}
}

func TestRepositoryCreateUpdateAndExternalModificationConflict(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	projectRoot := filepath.Join(root, "project")
	repository, err := NewRepository(projectRoot, filepath.Join(root, "personal"))
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	definition := validDefinition()
	created, err := repository.Create("project:valid-id", definition)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	loaded, err := repository.LoadRecord("project:valid-id")
	if err != nil {
		t.Fatalf("LoadRecord() error = %v", err)
	}
	if loaded.Revision != created.Revision || loaded.Definition.Name != definition.Name {
		t.Fatalf("LoadRecord() = %#v, want revision %q", loaded, created.Revision)
	}

	updatedDefinition := definition
	updatedDefinition.Name = "Updated"
	updated, err := repository.Update("project:valid-id", created.Revision, updatedDefinition)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Revision == created.Revision {
		t.Fatal("Update() did not change revision")
	}

	externalDefinition := updatedDefinition
	externalDefinition.Description = "Changed outside the repository."
	externalData, err := encodeDefinition(Reference{Source: SourceProject, ID: definition.ID}, externalDefinition)
	if err != nil {
		t.Fatalf("encodeDefinition() error = %v", err)
	}
	path := filepath.Join(projectRoot, "valid-id.json")
	if err = os.WriteFile(path, externalData, 0o644); err != nil {
		t.Fatalf("WriteFile(external modification) error = %v", err)
	}
	staleUpdate := updatedDefinition
	staleUpdate.Name = "Must not be written"
	if _, err = repository.Update("project:valid-id", updated.Revision, staleUpdate); !errors.Is(err, ErrScenarioStale) {
		t.Fatalf("Update(stale) error = %v, want ErrScenarioStale", err)
	}
	entries, err := os.ReadDir(projectRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Name() != repositoryLockName || entries[1].Name() != "valid-id.json" {
		t.Fatalf("stale update left files behind: %v", entries)
	}
	after, err := repository.LoadRecord("project:valid-id")
	if err != nil {
		t.Fatalf("LoadRecord(after conflict) error = %v", err)
	}
	if after.Definition.Description != externalDefinition.Description || after.Definition.Name != externalDefinition.Name {
		t.Fatalf("stale update replaced external content: %#v", after.Definition)
	}
}

func TestRepositoryConcurrentInstancesAllowOneRevisionWinner(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	repository, err := NewRepository(filepath.Join(root, "project"), filepath.Join(root, "personal"))
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	created, err := repository.Create("project:valid-id", validDefinition())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	start := make(chan struct{})
	results := make(chan error, 2)
	for index, name := range []string{"First", "Second"} {
		writer := repository
		if index == 1 {
			writer, err = NewRepository(filepath.Join(root, "project"), filepath.Join(root, "personal"))
			if err != nil {
				t.Fatalf("NewRepository(second instance) error = %v", err)
			}
		}
		definition := validDefinition()
		definition.Name = name
		go func() {
			<-start
			_, updateErr := writer.Update("project:valid-id", created.Revision, definition)
			results <- updateErr
		}()
	}
	close(start)
	successes, stale := 0, 0
	for range 2 {
		err = <-results
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrScenarioStale):
			stale++
		default:
			t.Fatalf("Update() error = %v", err)
		}
	}
	if successes != 1 || stale != 1 {
		t.Fatalf("concurrent Update() results: successes=%d stale=%d", successes, stale)
	}
}

func TestRepositoryUsesUnlockedPersistentLockFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	projectRoot := filepath.Join(root, "project")
	repository, err := NewRepository(projectRoot, filepath.Join(root, "personal"))
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	if err = os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	lockPath := filepath.Join(projectRoot, repositoryLockName)
	if err = os.WriteFile(lockPath, []byte("abandoned"), 0o600); err != nil {
		t.Fatalf("WriteFile(lock) error = %v", err)
	}
	if _, err = repository.Create("project:valid-id", validDefinition()); err != nil {
		t.Fatalf("Create() with unlocked persistent lock file error = %v", err)
	}
}

func TestRepositoryCreateAndDuplicateCollisionsPreserveSource(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	repository, err := NewRepository(filepath.Join(root, "project"), filepath.Join(root, "personal"))
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	source, err := repository.Create("project:valid-id", validDefinition())
	if err != nil {
		t.Fatalf("Create(source) error = %v", err)
	}
	if _, err = repository.Create("project:valid-id", validDefinition()); !errors.Is(err, ErrScenarioExists) {
		t.Fatalf("Create(collision) error = %v, want ErrScenarioExists", err)
	}
	destination := validDefinition()
	destination.ID = "copy"
	destination.Name = "Existing destination"
	if _, err = repository.Create("personal:copy", destination); err != nil {
		t.Fatalf("Create(destination) error = %v", err)
	}
	if _, err = repository.Duplicate("project:valid-id", source.Revision, "personal:copy"); !errors.Is(err, ErrScenarioExists) {
		t.Fatalf("Duplicate(collision) error = %v, want ErrScenarioExists", err)
	}
	after, err := repository.LoadRecord("project:valid-id")
	if err != nil {
		t.Fatalf("LoadRecord(source) error = %v", err)
	}
	if after.Revision != source.Revision {
		t.Fatalf("failed duplicate changed source revision: got %q, want %q", after.Revision, source.Revision)
	}
	storedDestination, err := repository.Load("personal:copy")
	if err != nil {
		t.Fatalf("Load(destination) error = %v", err)
	}
	if storedDestination.Name != destination.Name {
		t.Fatalf("failed duplicate changed destination: %#v", storedDestination)
	}
}

func TestRepositoryDuplicateAcrossSources(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	repository, err := NewRepository(filepath.Join(root, "project"), filepath.Join(root, "personal"))
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	source, err := repository.Create("project:valid-id", validDefinition())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	duplicate, err := repository.Duplicate("project:valid-id", source.Revision, "personal:personal-copy")
	if err != nil {
		t.Fatalf("Duplicate() error = %v", err)
	}
	if duplicate.Definition.ID != "personal-copy" {
		t.Fatalf("Duplicate().Definition.ID = %q", duplicate.Definition.ID)
	}
	if _, err = repository.LoadRecord("project:valid-id"); err != nil {
		t.Fatalf("Duplicate() removed source: %v", err)
	}

}

func TestRepositoryDeleteRequiresCurrentRevisionAndRegularPath(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	projectRoot := filepath.Join(root, "project")
	repository, err := NewRepository(projectRoot, filepath.Join(root, "personal"))
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	record, err := repository.Create("project:valid-id", validDefinition())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err = repository.Delete("project:valid-id", "not-the-current-revision"); !errors.Is(err, ErrScenarioStale) {
		t.Fatalf("Delete(stale) error = %v, want ErrScenarioStale", err)
	}
	if _, err = repository.LoadRecord("project:valid-id"); err != nil {
		t.Fatalf("stale Delete() removed source: %v", err)
	}
	if err = repository.Delete("project:valid-id", record.Revision); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if err = repository.Delete("project:valid-id", record.Revision); !errors.Is(err, ErrScenarioNotFound) {
		t.Fatalf("Delete(missing) error = %v, want ErrScenarioNotFound", err)
	}

	outside := filepath.Join(root, "outside.json")
	if err = os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatalf("WriteFile(outside) error = %v", err)
	}
	if err = os.Symlink(outside, filepath.Join(projectRoot, "valid-id.json")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}
	if err = repository.Delete("project:valid-id", record.Revision); !errors.Is(err, ErrInvalidReference) {
		t.Fatalf("Delete(symlink) error = %v, want ErrInvalidReference", err)
	}
	data, err := os.ReadFile(outside)
	if err != nil || string(data) != "outside" {
		t.Fatalf("Delete(symlink) changed outside file: data=%q error=%v", data, err)
	}
	if err = os.Remove(filepath.Join(projectRoot, "valid-id.json")); err != nil {
		t.Fatalf("Remove(symlink) error = %v", err)
	}
	if err = os.Mkdir(filepath.Join(projectRoot, "valid-id.json"), 0o755); err != nil {
		t.Fatalf("Mkdir(nonregular path) error = %v", err)
	}
	if err = repository.Delete("project:valid-id", record.Revision); !errors.Is(err, ErrInvalidReference) {
		t.Fatalf("Delete(directory) error = %v, want ErrInvalidReference", err)
	}
}

func TestRepositoryRejectsTraversalMismatchesAndSymlinks(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	projectRoot := filepath.Join(root, "project")
	personalRoot := filepath.Join(root, "personal")
	repository, err := NewRepository(projectRoot, personalRoot)
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	definition := validDefinition()

	for _, reference := range []string{"project:../escape", "personal:/absolute", "unknown:valid-id", "project:bad_id", "project:"} {
		if _, err = repository.Create(reference, definition); !errors.Is(err, ErrInvalidReference) {
			t.Fatalf("Create(%q) error = %v, want ErrInvalidReference", reference, err)
		}
	}
	if _, err = repository.Create("project:different-id", definition); !errors.Is(err, ErrInvalidDefinition) {
		t.Fatalf("Create(mismatch) error = %v", err)
	}

	if err = os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	outside := filepath.Join(root, "outside.json")
	if err = os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	link := filepath.Join(projectRoot, definition.ID+".json")
	if err = os.Symlink(outside, link); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}
	if _, err = repository.Load("project:valid-id"); !errors.Is(err, ErrInvalidReference) {
		t.Fatalf("Load(symlink) error = %v, want ErrInvalidReference", err)
	}
	if _, err = repository.Create("project:valid-id", definition); !errors.Is(err, ErrScenarioExists) {
		t.Fatalf("Create(symlink) error = %v, want ErrScenarioExists", err)
	}
	data, err := os.ReadFile(outside)
	if err != nil || string(data) != "outside" {
		t.Fatalf("outside file changed: data=%q error=%v", data, err)
	}
}

func TestRepositoryFailedUpdatePreservesExistingFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	repository, err := NewRepository(filepath.Join(root, "project"), filepath.Join(root, "personal"))
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	definition := validDefinition()
	record, err := repository.Create("project:valid-id", definition)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	before, err := os.ReadFile(filepath.Join(root, "project", "valid-id.json"))
	if err != nil {
		t.Fatalf("ReadFile(before) error = %v", err)
	}
	definition.SchemaVersion++
	if _, err = repository.Update("project:valid-id", record.Revision, definition); !errors.Is(err, ErrInvalidDefinition) {
		t.Fatalf("invalid Update() error = %v", err)
	}
	after, err := os.ReadFile(filepath.Join(root, "project", "valid-id.json"))
	if err != nil {
		t.Fatalf("ReadFile(after) error = %v", err)
	}
	if string(after) != string(before) {
		t.Fatal("failed save changed existing scenario")
	}
	entries, err := os.ReadDir(filepath.Join(root, "project"))
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 2 || entries[0].Name() != repositoryLockName || entries[1].Name() != "valid-id.json" {
		t.Fatalf("temporary files remain after save: %v", entries)
	}
}

func TestRepositoryMissingRootsAndScenarios(t *testing.T) {
	t.Parallel()
	if _, err := NewRepository("", t.TempDir()); err == nil {
		t.Fatal("NewRepository() accepted an empty root")
	}
	root := t.TempDir()
	repository, err := NewRepository(filepath.Join(root, "project"), filepath.Join(root, "personal"))
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	summaries, err := repository.List()
	if err != nil || len(summaries) != 0 {
		t.Fatalf("List() = %v, %v", summaries, err)
	}
	if _, err = repository.Load("personal:missing"); !errors.Is(err, ErrScenarioNotFound) {
		t.Fatalf("Load(missing) error = %v", err)
	}
}

func TestRepositoryUsesPrivateModeForPersonalDefinitions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not support the repository permission policy")
	}
	root := t.TempDir()
	personalRoot := filepath.Join(root, "personal")
	repository, err := NewRepository(filepath.Join(root, "project"), personalRoot)
	if err != nil {
		t.Fatal(err)
	}
	definition := validDefinition()
	if _, err = repository.Create("personal:valid-id", definition); err != nil {
		t.Fatal(err)
	}
	rootInfo, err := os.Stat(personalRoot)
	if err != nil {
		t.Fatal(err)
	}
	fileInfo, err := os.Stat(filepath.Join(personalRoot, "valid-id.json"))
	if err != nil {
		t.Fatal(err)
	}
	if rootInfo.Mode().Perm() != 0o700 || fileInfo.Mode().Perm() != 0o600 {
		t.Fatalf("personal modes = dir %o file %o, want 700/600", rootInfo.Mode().Perm(), fileInfo.Mode().Perm())
	}
}

func TestRepositoryValidatesSnapshotBaselineExistenceAndPortabilityForMutations(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	repository, err := NewRepository(filepath.Join(root, "project"), filepath.Join(root, "personal"))
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	repository.ConfigureSnapshotRepository(scenarioSnapshotResolverStub{records: map[string]save_snapshot.Record{
		"project:shared":   {},
		"personal:private": {},
	}})

	project := validDefinition()
	project.ID = "project-snapshot"
	project.Baseline = SnapshotBaseline("project:shared")
	created, err := repository.Create("project:project-snapshot", project)
	if err != nil {
		t.Fatalf("Create(project snapshot) error = %v", err)
	}

	missing := validDefinition()
	missing.ID = "missing-snapshot"
	missing.Baseline = SnapshotBaseline("project:missing")
	if _, err = repository.Create("personal:missing-snapshot", missing); !errors.Is(err, ErrSnapshotBaseline) {
		t.Fatalf("Create(missing snapshot) error = %v, want ErrSnapshotBaseline", err)
	}

	updated := project
	updated.Baseline = SnapshotBaseline("personal:private")
	if _, err = repository.Update("project:project-snapshot", created.Revision, updated); !errors.Is(err, ErrSnapshotBaseline) {
		t.Fatalf("Update(non-portable snapshot) error = %v, want ErrSnapshotBaseline", err)
	}
	stored, err := repository.LoadRecord("project:project-snapshot")
	if err != nil || stored.Revision != created.Revision {
		t.Fatalf("failed Update changed stored scenario: record=%#v error=%v", stored, err)
	}

	personal := validDefinition()
	personal.ID = "personal-snapshot"
	personal.Baseline = SnapshotBaseline("personal:private")
	personalRecord, err := repository.Create("personal:personal-snapshot", personal)
	if err != nil {
		t.Fatalf("Create(personal snapshot) error = %v", err)
	}
	if _, err = repository.Duplicate("personal:personal-snapshot", personalRecord.Revision, "project:portable-copy"); !errors.Is(err, ErrSnapshotBaseline) {
		t.Fatalf("Duplicate(non-portable snapshot) error = %v, want ErrSnapshotBaseline", err)
	}
	if _, err = repository.LoadRecord("project:portable-copy"); !errors.Is(err, ErrScenarioNotFound) {
		t.Fatalf("failed Duplicate created destination: %v", err)
	}
}

func TestRepositoryCoordinatesSnapshotReferencesAcrossProcesses(t *testing.T) {
	if os.Getenv(crossProcessScenarioRole) == "1" {
		projectScenarios := os.Getenv("U5_PROJECT_SCENARIOS")
		personalScenarios := os.Getenv("U5_PERSONAL_SCENARIOS")
		snapshots, err := save_snapshot.NewRepository(os.Getenv("U5_PROJECT_SNAPSHOTS"), os.Getenv("U5_PERSONAL_SNAPSHOTS"), os.Getenv("U5_ROLLBACK_SNAPSHOTS"))
		if err != nil {
			t.Fatal(err)
		}
		repository, err := NewRepository(projectScenarios, personalScenarios)
		if err != nil {
			t.Fatal(err)
		}
		repository.ConfigureSnapshotRepository(snapshots)
		if err = os.WriteFile(os.Getenv("U5_CHILD_READY"), []byte("ready"), 0o600); err != nil {
			t.Fatal(err)
		}
		definition := validDefinition()
		definition.ID = "concurrent-reference"
		definition.Baseline = SnapshotBaseline("project:shared")
		_, err = repository.Create("project:concurrent-reference", definition)
		if !errors.Is(err, ErrSnapshotBaseline) {
			t.Fatalf("Create() error = %v, want ErrSnapshotBaseline after coordinated delete", err)
		}
		return
	}

	root := t.TempDir()
	projectScenarios := filepath.Join(root, "project-scenarios")
	personalScenarios := filepath.Join(root, "personal-scenarios")
	projectSnapshots := filepath.Join(root, "project-snapshots")
	personalSnapshots := filepath.Join(root, "personal-snapshots")
	rollbackSnapshots := filepath.Join(root, "rollback-snapshots")
	snapshots, err := save_snapshot.NewRepository(projectSnapshots, personalSnapshots, rollbackSnapshots)
	if err != nil {
		t.Fatal(err)
	}
	repository, err := NewRepository(projectScenarios, personalScenarios)
	if err != nil {
		t.Fatal(err)
	}
	repository.ConfigureSnapshotRepository(snapshots)
	record, err := snapshots.Create("project:shared", crossProcessSnapshotEnvelope())
	if err != nil {
		t.Fatal(err)
	}
	ready := filepath.Join(root, "child-ready")
	command := exec.Command(os.Args[0], "-test.run=^TestRepositoryCoordinatesSnapshotReferencesAcrossProcesses$")
	command.Env = append(os.Environ(),
		crossProcessScenarioRole+"=1",
		"U5_PROJECT_SCENARIOS="+projectScenarios,
		"U5_PERSONAL_SCENARIOS="+personalScenarios,
		"U5_PROJECT_SNAPSHOTS="+projectSnapshots,
		"U5_PERSONAL_SNAPSHOTS="+personalSnapshots,
		"U5_ROLLBACK_SNAPSHOTS="+rollbackSnapshots,
		"U5_CHILD_READY="+ready,
	)
	started := false
	err = snapshots.DeleteIfUnreferenced(record.Reference.String(), record.Revision, func() error {
		if err := command.Start(); err != nil {
			return err
		}
		started = true
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if _, statErr := os.Stat(ready); statErr == nil {
				return nil
			}
			time.Sleep(5 * time.Millisecond)
		}
		return errors.New("child did not reach scenario create")
	})
	if err != nil {
		t.Fatalf("DeleteIfUnreferenced() error = %v", err)
	}
	if !started {
		t.Fatal("child process was not started")
	}
	if waitErr := command.Wait(); waitErr != nil {
		t.Fatalf("child process error = %v", waitErr)
	}
	if _, err = repository.Load("project:concurrent-reference"); !errors.Is(err, ErrScenarioNotFound) {
		t.Fatalf("concurrent scenario unexpectedly established deleted snapshot reference: %v", err)
	}
}

func crossProcessSnapshotEnvelope() save_snapshot.Envelope {
	position := save_snapshot.PositionSnapshot{Location: 0, X: 10, Y: 20, Floor: 0}
	snapshot := save_snapshot.SaveSnapshot{
		Version:      save_snapshot.CurrentVersion,
		Presentation: journal.EmptyPresentation(),
		Party:        save_snapshot.PartySnapshot{Karma: 75},
		Redux: save_snapshot.ReduxSnapshot{
			DungeonDiscoveries: make([]save_snapshot.DungeonDiscoverySnapshot, 0),
			PlayerKnowledge:    make([]save_snapshot.PlayerKnowledgeSnapshot, 0),
			QuestJournal:       make([]save_snapshot.QuestJournalEventSnapshot, 0),
		},
		World:  save_snapshot.WorldSnapshot{Position: position, LastLargeMapPosition: position, Calendar: save_snapshot.CalendarSnapshot{Year: 100, Month: 4, Day: 12, Hour: 9, Minute: 30}},
		Wind:   save_snapshot.WindSnapshot{Direction: 1},
		Random: save_snapshot.RandomSnapshot{Seed: 1},
	}
	snapshot.Party.Roster[0] = save_snapshot.CharacterSnapshot{
		Occupied: true, Name: "Avatar", Gender: int(party_state.Male), Class: int(party_state.Avatar),
		Status: int(party_state.Good), Level: 1, MaximumHP: 10, CurrentHP: 10,
	}
	snapshot.Progression.Shadowlords.LastCalled = -1
	snapshot.Progression.Shadowlords.AutomaticPresence = -1
	return save_snapshot.Envelope{
		Format: save_snapshot.EnvelopeFormat, EnvelopeVersion: save_snapshot.CurrentEnvelopeVersion,
		Metadata: save_snapshot.Metadata{Name: "Shared", CreatedAt: time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC), GameVersion: "test"},
		Snapshot: snapshot,
	}
}
