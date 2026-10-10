package player_save

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/repositoryfs"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlayerSaveRequiresIdentityAndValidNoteBeforeAnyWrite(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		mutate func(*save_snapshot.Metadata)
	}{
		{name: "missing identity", mutate: func(metadata *save_snapshot.Metadata) { metadata.PlaythroughID = "" }},
		{name: "invalid identity", mutate: func(metadata *save_snapshot.Metadata) { metadata.PlaythroughID = "other" }},
		{name: "uppercase identity", mutate: func(metadata *save_snapshot.Metadata) { metadata.PlaythroughID = strings.Repeat("A", 32) }},
		{name: "long note", mutate: func(metadata *save_snapshot.Metadata) { metadata.UserDescription = strings.Repeat("\u754c", 81) }},
		{name: "multiline note", mutate: func(metadata *save_snapshot.Metadata) { metadata.UserDescription = "first\nsecond" }},
		{name: "control note", mutate: func(metadata *save_snapshot.Metadata) { metadata.UserDescription = "tab\there" }},
		{name: "separator note", mutate: func(metadata *save_snapshot.Metadata) { metadata.UserDescription = "line\u2028break" }},
		{name: "invalid UTF-8 note", mutate: func(metadata *save_snapshot.Metadata) { metadata.UserDescription = "\xff" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := newTestRepository(t, filepath.Join(t.TempDir(), "saves"), nil)
			envelope := testEnvelope(testTime(1))
			test.mutate(&envelope.Metadata)
			_, err := repository.Save(Slot01, envelope, "")
			require.ErrorIs(t, err, save_snapshot.ErrInvalidEnvelope)
			_, err = os.Stat(repository.root)
			require.ErrorIs(t, err, os.ErrNotExist, "invalid metadata must not create the repository root")
			original, err := repository.Save(Slot01, testEnvelope(testTime(1)), "")
			require.NoError(t, err)
			_, err = repository.Save(Slot01, envelope, original.Revision)
			require.ErrorIs(t, err, save_snapshot.ErrInvalidEnvelope)
			data, err := os.ReadFile(repository.path(Slot01))
			require.NoError(t, err)
			assert.Equal(t, original.rawData, data, "invalid overwrite changed the slot")
		})
	}
}

func TestIdentityFreeEnvelopeTwoIsIncompatibleButRevisionSafe(t *testing.T) {
	t.Parallel()
	for _, explicitEmpty := range []bool{false, true} {
		t.Run(map[bool]string{false: "omitted", true: "explicit empty"}[explicitEmpty], func(t *testing.T) {
			compatibilityCalls := 0
			repository := newTestRepository(t, t.TempDir(), func(save_snapshot.SaveSnapshot) error {
				compatibilityCalls++
				return nil
			})
			envelope := testEnvelope(testTime(9))
			envelope.Metadata.PlaythroughID = ""
			envelope.Metadata.UserDescription = "Generic note remains inspectable"
			canonical, err := save_snapshot.EncodeCanonical(envelope)
			require.NoError(t, err, "generic identity-free envelopes remain valid")
			var document map[string]any
			require.NoError(t, json.Unmarshal(canonical, &document))
			if explicitEmpty {
				document["metadata"].(map[string]any)["playthroughId"] = ""
			}
			data, err := json.Marshal(document)
			require.NoError(t, err)
			data = append([]byte(" \n"), data...)
			require.NoError(t, os.WriteFile(repository.path(Slot01), data, fileMode))

			previous, err := repository.Load(Slot01)
			require.ErrorIs(t, err, ErrIncompatible)
			assert.ErrorContains(t, err, "playthroughId is required")
			assert.Equal(t, 0, compatibilityCalls, "identity-free records cannot reach runtime load validation")
			assert.Equal(t, envelope, previous.Envelope)
			assert.Equal(t, repositoryfs.ContentRevision(canonical), previous.Revision)
			assert.NotEqual(t, repositoryfs.ContentRevision(data), previous.Revision)
			summary := repository.List()[0]
			assert.Equal(t, SlotIncompatible, summary.Status)
			assert.Equal(t, previous.Revision, summary.Revision)
			assert.Empty(t, summary.Metadata.PlaythroughID)
			assert.Equal(t, envelope.Metadata.UserDescription, summary.Metadata.UserDescription)
			_, err = repository.Latest()
			require.ErrorIs(t, err, ErrSlotNotFound)
			ready, err := repository.Save(Slot02, testEnvelope(testTime(1)), "")
			require.NoError(t, err)
			latest, err := repository.Latest()
			require.NoError(t, err)
			assert.Equal(t, ready.Slot, latest.Slot, "newer identity-free files are never newest-ready candidates")
			unchanged, err := os.ReadFile(repository.path(Slot01))
			require.NoError(t, err)
			assert.Equal(t, data, unchanged, "inspection must not backfill or rewrite")

			_, err = repository.Save(Slot01, testEnvelope(testTime(2)), "stale")
			require.ErrorIs(t, err, ErrStaleRevision)
			require.ErrorIs(t, repository.Delete(Slot01, "stale"), ErrStaleRevision)
			replacement, err := repository.Save(Slot01, testEnvelope(testTime(2)), previous.Revision)
			require.NoError(t, err)
			require.NoError(t, repository.Restore(previous, replacement.Revision))
			restored, err := os.ReadFile(repository.path(Slot01))
			require.NoError(t, err)
			assert.Equal(t, data, restored, "activation rollback must restore exact identity-free bytes")
			require.NoError(t, repository.Delete(Slot01, previous.Revision))
			_, err = repository.Load(Slot01)
			require.ErrorIs(t, err, ErrSlotNotFound)
		})
	}
}

func TestPlayerSaveMetadataIsCheckpointSpecificAndRevisionProtected(t *testing.T) {
	t.Parallel()
	repository := newTestRepository(t, t.TempDir(), nil)
	first := testEnvelope(testTime(1))
	first.Snapshot.Party.Roster[0].Name = "Derp"
	first.Metadata.UserDescription = "Before Blackthorn"
	first.Metadata.DoomRunID = "independent-doom-run"
	record, err := repository.Save(Slot01, first, "")
	require.NoError(t, err)
	second := first
	second.Metadata.PlaythroughID = strings.Repeat("f", 32)
	second.Metadata.UserDescription = "Stocked up in Britain"
	sibling, err := repository.Save(Slot02, second, "")
	require.NoError(t, err)
	assert.NotEqual(t, record.Revision, sibling.Revision)
	catalog := repository.List()
	assert.Equal(t, catalog[0].Metadata.AvatarName, catalog[1].Metadata.AvatarName)
	assert.NotEqual(t, catalog[0].Metadata.PlaythroughID, catalog[1].Metadata.PlaythroughID, "matching names cannot link independent games")
	assert.Equal(t, first.Metadata.UserDescription, catalog[0].Metadata.UserDescription)
	assert.Equal(t, second.Metadata.UserDescription, catalog[1].Metadata.UserDescription)

	third := first
	third.Metadata.UserDescription = "About to go to Doom"
	third.Snapshot.Party.Roster[0].Name = "Renamed"
	thirdRecord, err := repository.Save(Slot03, third, "")
	require.NoError(t, err)
	assert.Equal(t, first.Metadata.PlaythroughID, thirdRecord.Envelope.Metadata.PlaythroughID, "renaming and Save As preserve caller identity")
	assert.Equal(t, first.Metadata.UserDescription, repository.List()[0].Metadata.UserDescription, "notes do not propagate to sibling checkpoints")
	for range 2 {
		loaded, loadErr := repository.Load(Slot03)
		require.NoError(t, loadErr)
		assert.Equal(t, third, loaded.Envelope, "repeated loading must not invent metadata")
	}

	// Only identity changes: it must still invalidate the reviewed revision.
	identityOnly := first
	identityOnly.Metadata.PlaythroughID = second.Metadata.PlaythroughID
	identityRecord, err := repository.Save(Slot01, identityOnly, record.Revision)
	require.NoError(t, err)
	assert.NotEqual(t, record.Revision, identityRecord.Revision)
	_, err = repository.Save(Slot01, first, record.Revision)
	require.ErrorIs(t, err, ErrStaleRevision)

	// Overwriting the other game uses the supplied source identity, not the destination's.
	overwritten, err := repository.Save(Slot02, first, sibling.Revision)
	require.NoError(t, err)
	assert.Equal(t, first.Metadata.PlaythroughID, overwritten.Envelope.Metadata.PlaythroughID)
	assert.Equal(t, first.Metadata.Description, overwritten.Envelope.Metadata.Description)
	assert.Equal(t, first.Metadata.DoomRunID, overwritten.Envelope.Metadata.DoomRunID)
	cleared := first
	cleared.Metadata.UserDescription = ""
	clearedRecord, err := repository.Save(Slot02, cleared, overwritten.Revision)
	require.NoError(t, err)
	assert.NotEqual(t, overwritten.Revision, clearedRecord.Revision, "note-only changes affect revisions")
	assert.NotContains(t, string(clearedRecord.rawData), `"userDescription"`)
	loaded, err := repository.Load(Slot02)
	require.NoError(t, err)
	assert.Equal(t, cleared, loaded.Envelope)
	assert.Empty(t, repository.List()[1].Metadata.UserDescription)
	require.NoError(t, repository.Restore(overwritten, clearedRecord.Revision))
	assert.Equal(t, first.Metadata.UserDescription, repository.List()[1].Metadata.UserDescription)
}

func TestPlayerRepositoryRejectsEnvelopeOneButAllowsExplicitReclaim(t *testing.T) {
	t.Parallel()
	repository := newTestRepository(t, t.TempDir(), nil)
	envelope := testEnvelope(testTime(1))
	envelope.EnvelopeVersion = 1
	envelope.Metadata.PlaythroughID = ""
	data, err := json.Marshal(envelope)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(repository.path(Slot01), data, fileMode))
	_, err = repository.Load(Slot01)
	require.ErrorIs(t, err, save_snapshot.ErrInvalidEnvelope)
	assert.ErrorContains(t, err, "unsupported envelope version 1")
	summary := repository.List()[0]
	assert.Equal(t, SlotCorrupt, summary.Status)
	assert.Equal(t, repositoryfs.ContentRevision(data), summary.Revision)
	_, err = repository.Latest()
	require.ErrorIs(t, err, ErrSlotNotFound)
	retained, err := os.ReadFile(repository.path(Slot01))
	require.NoError(t, err)
	assert.Equal(t, data, retained)
	_, err = repository.Save(Slot01, testEnvelope(testTime(2)), "")
	require.ErrorIs(t, err, ErrStaleRevision)
	_, err = repository.Save(Slot01, testEnvelope(testTime(2)), summary.Revision)
	require.NoError(t, err)
}

func TestPlayerRepositoryRejectsMalformedIdentityAndNoteOnInspection(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		member string
		value  any
	}{
		{member: "playthroughId", value: "bad"},
		{member: "playthroughId", value: nil},
		{member: "playthroughId", value: 123},
		{member: "userDescription", value: "bad\u2029note"},
		{member: "userDescription", value: nil},
		{member: "userDescription", value: strings.Repeat("x", 81)},
		{member: "unknown", value: "value"},
	} {
		repository := newTestRepository(t, t.TempDir(), nil)
		data, err := save_snapshot.EncodeCanonical(testEnvelope(testTime(1)))
		require.NoError(t, err)
		var document map[string]any
		require.NoError(t, json.Unmarshal(data, &document))
		document["metadata"].(map[string]any)[test.member] = test.value
		data, err = json.Marshal(document)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(repository.path(Slot01), data, fileMode))
		_, err = repository.Load(Slot01)
		require.ErrorIs(t, err, save_snapshot.ErrInvalidEnvelope)
		assert.Equal(t, SlotCorrupt, repository.List()[0].Status)
		retained, err := os.ReadFile(repository.path(Slot01))
		require.NoError(t, err)
		assert.True(t, bytes.Equal(data, retained))
	}
}

func TestPlayerSaveMetadataRollbackRetainsIdentityAndNote(t *testing.T) {
	t.Parallel()
	repository := newTestRepository(t, t.TempDir(), nil)
	original := testEnvelope(testTime(1))
	original.Metadata.UserDescription = "Committed checkpoint"
	record, err := repository.Save(Slot01, original, "")
	require.NoError(t, err)
	replacement := original
	replacement.Metadata.PlaythroughID = strings.Repeat("f", 32)
	replacement.Metadata.UserDescription = "Uncommitted note"
	syncCalls := 0
	repository.syncDirectory = func(string) error {
		syncCalls++
		if syncCalls == 1 {
			return errors.New("injected metadata publication failure")
		}
		return nil
	}
	_, err = repository.Save(Slot01, replacement, record.Revision)
	require.Error(t, err)
	loaded, err := repository.Load(Slot01)
	require.NoError(t, err)
	assert.Equal(t, original, loaded.Envelope)
	assert.Equal(t, record.Revision, loaded.Revision)
}
