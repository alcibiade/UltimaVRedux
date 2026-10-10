package doom_recovery

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
)

func TestRepositoryReplacesDedicatedCheckpointWithoutTouchingSlots(t *testing.T) {
	root := t.TempDir()
	repository := newTestRepository(t, root, nil)
	first := recoveryTestEnvelope(1, "run-one")
	second := recoveryTestEnvelope(2, "run-two")

	firstRecord, err := repository.Replace(first)
	if err != nil {
		t.Fatal(err)
	}
	secondRecord, err := repository.Replace(second)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := repository.Load()
	if err != nil {
		t.Fatal(err)
	}
	if firstRecord.Revision == secondRecord.Revision || loaded.Revision != secondRecord.Revision ||
		loaded.Envelope.Metadata.DoomRunID != "run-two" {
		t.Fatalf("replacement records = %#v, %#v, loaded %#v", firstRecord, secondRecord, loaded)
	}
	if _, err = os.Stat(filepath.Join(root, "slot-01.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("recovery write touched a player slot: %v", err)
	}
	path := filepath.Join(root, recoveryDirectory, recoveryFile)
	canonical, err := save_snapshot.EncodeCanonical(second)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(data, canonical) {
		t.Fatalf("checkpoint bytes are not canonical: %v", err)
	}
	if runtime.GOOS != "windows" {
		assertRecoveryMode(t, root, 0o700)
		assertRecoveryMode(t, filepath.Dir(path), 0o700)
		assertRecoveryMode(t, path, 0o600)
	}
}

func TestRepositoryFailedReplacementRestoresPriorBytes(t *testing.T) {
	repository := newTestRepository(t, t.TempDir(), nil)
	if _, err := repository.Replace(recoveryTestEnvelope(1, "run-one")); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(repository.path())
	if err != nil {
		t.Fatal(err)
	}
	syncCalls := 0
	repository.syncDirectory = func(string) error {
		syncCalls++
		if syncCalls == 1 {
			return errors.New("injected sync failure")
		}

		return nil
	}

	if _, err = repository.Replace(recoveryTestEnvelope(2, "run-two")); err == nil {
		t.Fatal("Replace() error = nil, want sync failure")
	}
	after, err := os.ReadFile(repository.path())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("failed replacement did not preserve prior checkpoint bytes")
	}
}

func TestRepositoryFirstWriteSyncsRecoveryParentBeforePublishing(t *testing.T) {
	root := t.TempDir()
	repository := newTestRepository(t, root, nil)
	parentSyncErr := errors.New("injected parent sync failure")
	repository.syncDirectory = func(path string) error {
		if path != root {
			t.Fatalf("first sync path = %q, want player-save root %q", path, root)
		}
		if _, err := os.Stat(repository.path()); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("checkpoint was published before parent sync: %v", err)
		}

		return parentSyncErr
	}

	if _, err := repository.Replace(recoveryTestEnvelope(1, "run")); !errors.Is(err, parentSyncErr) {
		t.Fatalf("Replace() error = %v, want parent sync failure", err)
	}
	if _, err := os.Stat(repository.recoveryRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("uncommitted recovery root remains after parent sync failure: %v", err)
	}

	repository.syncDirectory = func(string) error { return nil }
	if _, err := repository.Replace(recoveryTestEnvelope(1, "run")); err != nil {
		t.Fatalf("Replace() retry error = %v", err)
	}
	loaded, err := repository.Load()
	if err != nil || loaded.Envelope.Metadata.DoomRunID != "run" {
		t.Fatalf("Load() after retry = %#v, %v", loaded, err)
	}
}

func TestRepositoryLoadClassifiesMissingCorruptAndIncompatible(t *testing.T) {
	root := t.TempDir()
	compatibilityErr := errors.New("runtime data mismatch")
	repository := newTestRepository(t, root, func(save_snapshot.SaveSnapshot) error { return compatibilityErr })
	if _, err := repository.Load(); !errors.Is(err, ErrCheckpointNotFound) {
		t.Fatalf("Load(missing) error = %v", err)
	}
	if err := repository.ensureRoots(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(repository.path(), []byte("not JSON"), fileMode); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Load(); !errors.Is(err, save_snapshot.ErrInvalidEnvelope) {
		t.Fatalf("Load(corrupt) error = %v", err)
	}
	if _, err := repository.Replace(recoveryTestEnvelope(1, "run")); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Load(); !errors.Is(err, ErrCheckpointIncompatible) || !errors.Is(err, compatibilityErr) {
		t.Fatalf("Load(incompatible) error = %v", err)
	}
}

func newTestRepository(t *testing.T, root string, validator CompatibilityValidator) *Repository {
	t.Helper()
	repository, err := NewRepository(root, validator)
	if err != nil {
		t.Fatal(err)
	}

	return repository
}

func recoveryTestEnvelope(day int, runID string) save_snapshot.Envelope {
	snapshot := save_snapshot.SaveSnapshot{
		Version:      save_snapshot.CurrentVersion,
		Presentation: journal.EmptyPresentation(),
		Redux: save_snapshot.ReduxSnapshot{
			DungeonDiscoveries: []save_snapshot.DungeonDiscoverySnapshot{},
			PlayerKnowledge:    []save_snapshot.PlayerKnowledgeSnapshot{},
			QuestJournal:       []save_snapshot.QuestJournalEventSnapshot{},
		},
		World: save_snapshot.WorldSnapshot{
			Calendar: save_snapshot.CalendarSnapshot{Year: 100, Month: 1, Day: 1},
		},
	}
	snapshot.Party.Roster[0] = save_snapshot.CharacterSnapshot{
		Occupied: true, Name: "Avatar", Gender: int(party_state.Male), Class: int(party_state.Avatar),
		Status: int(party_state.Good), Level: 1,
	}

	return save_snapshot.Envelope{
		Format: save_snapshot.EnvelopeFormat, EnvelopeVersion: save_snapshot.CurrentEnvelopeVersion,
		Metadata: save_snapshot.Metadata{
			Name: "Pre-Doom Recovery", Description: "test", DoomRunID: runID,
			CreatedAt: time.Date(2026, time.August, day, 12, 0, 0, 0, time.UTC), GameVersion: "test",
		},
		Snapshot: snapshot,
	}
}

func assertRecoveryMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s mode = %o, want %o", path, got, want)
	}
}
