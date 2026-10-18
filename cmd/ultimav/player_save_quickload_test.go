package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/player_save"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type quickLoadReadFailureRepository struct {
	*playerSaveTestRepository
	err error
}

func (repository *quickLoadReadFailureRepository) Load(slot player_save.SlotID) (player_save.Record, error) {
	repository.loads = append(repository.loads, slot)

	return player_save.Record{}, repository.err
}

func TestQuickLoadUsesActiveReviewedSlotWithoutCaptureOrWrite(t *testing.T) {
	snapshot := validCommandTestSnapshot()
	record := playerSaveTestRecord(player_save.Slot07, "reviewed", snapshot)
	record.Envelope.Metadata.UserDescription = "Before the battle"
	repository := &playerSaveTestRepository{
		records: map[player_save.SlotID]player_save.Record{player_save.Slot07: record},
		latest:  playerSaveTestRecord(player_save.Slot01, "newer unrelated save", snapshot),
	}
	// This scene deliberately cannot capture at all: ordinary quickload must
	// not depend on the outgoing scene implementing the capture interface.
	old := &playerSaveNoCaptureScene{
		playthrough: playthroughMetadata{id: testPlaythroughID},
		slot:        player_save.Slot07, revision: "reviewed",
	}
	candidate := &playerSaveTestScene{}
	game := NewGame(old)
	game.ConfigurePlayerSaveRepository(repository, func(_ *config.UltimaVConfiguration, got save_snapshot.SaveSnapshot) (Scene, error) {
		assert.Equal(t, snapshot, got)
		assert.Same(t, old, game.currentScene)
		assert.Zero(t, old.disposals)

		return candidate, nil
	})
	generation := game.DebugSceneGeneration()

	slot, err := game.quickLoadPlayerSave()
	require.NoError(t, err)
	assert.Equal(t, player_save.Slot07, slot)
	assert.Same(t, candidate, game.currentScene)
	assert.Equal(t, generation+1, game.DebugSceneGeneration())
	assert.Equal(t, 1, old.disposals)
	assert.Equal(t, 1, candidate.inputPreparations)
	assert.Equal(t, "reviewed", candidate.revision)
	assert.Equal(t, "Before the battle", candidate.playthrough.userDescription)
	assert.Equal(t, testPlaythroughID, candidate.playthrough.id)
	assert.Equal(t, []player_save.SlotID{player_save.Slot07}, repository.loads)
	assert.Zero(t, repository.latestCalls)
	assert.Empty(t, repository.saves)
	assert.Empty(t, repository.restores)
	assert.Empty(t, repository.deletes)
	assert.Equal(t, record, repository.records[player_save.Slot07])
}

func TestQuickLoadFailuresRetainSceneGenerationAndProvenance(t *testing.T) {
	for _, failure := range []string{
		"no active slot", "unavailable", "stale", "deleted", "corrupt", "incompatible", "read failure",
		"construction", "nil candidate", "preflight", "activation",
	} {
		t.Run(failure, func(t *testing.T) {
			record := playerSaveTestRecord(player_save.Slot03, "reviewed", validCommandTestSnapshot())
			repository := &playerSaveTestRepository{records: map[player_save.SlotID]player_save.Record{record.Slot: record}}
			old := &playerSaveTestScene{
				playthrough: playthroughMetadata{id: testPlaythroughID, userDescription: "Unchanged"},
				slot:        record.Slot, revision: record.Revision,
				captureErr: game_state.ErrSaveSnapshotNotRepresentable,
			}
			candidate := &playerSaveTestScene{}
			game := NewGame(old)
			built := false
			game.ConfigurePlayerSaveRepository(repository, func(*config.UltimaVConfiguration, save_snapshot.SaveSnapshot) (Scene, error) {
				built = true
				if failure == "construction" {
					return candidate, errors.New("construction failed")
				}
				if failure == "nil candidate" {
					return nil, nil
				}

				return candidate, nil
			})
			switch failure {
			case "no active slot":
				old.slot, old.revision = "", ""
			case "unavailable":
				game.playerSaveRepository = nil
			case "stale":
				old.revision = "previously reviewed"
			case "deleted":
				delete(repository.records, record.Slot)
			case "corrupt":
				game.playerSaveRepository = &quickLoadReadFailureRepository{repository, save_snapshot.ErrInvalidEnvelope}
			case "incompatible":
				game.playerSaveRepository = &quickLoadReadFailureRepository{repository, player_save.ErrIncompatible}
			case "read failure":
				game.playerSaveRepository = &quickLoadReadFailureRepository{repository, fs.ErrPermission}
			case "preflight":
				candidate.preflightErr = errors.New("preflight failed")
			case "activation":
				candidate.activatePanic = true
			}
			beforeSlot, beforeRevision, beforeMetadata := old.slot, old.revision, old.playthrough
			generation := game.DebugSceneGeneration()
			_, err := game.quickLoadPlayerSave()
			require.Error(t, err)
			if failure == "no active slot" {
				var noActive *NoActivePlayerSaveSlotError
				require.ErrorAs(t, err, &noActive)
			}
			if failure == "stale" {
				require.ErrorIs(t, err, player_save.ErrStaleRevision)
				assert.False(t, built)
			}
			assert.Same(t, old, game.currentScene)
			assert.Equal(t, generation, game.DebugSceneGeneration())
			assert.Equal(t, beforeSlot, old.slot)
			assert.Equal(t, beforeRevision, old.revision)
			assert.Equal(t, beforeMetadata, old.playthrough)
			assert.Zero(t, old.disposals)
			assert.Zero(t, old.captures)
			assert.Empty(t, repository.saves)
			assert.Empty(t, repository.restores)
			assert.Zero(t, repository.latestCalls)
			if built && failure != "nil candidate" {
				assert.Equal(t, 1, candidate.disposals)
			}
		})
	}
}

func TestQuickLoadNativeSlotBytesAreNotRewritten(t *testing.T) {
	directory := t.TempDir()
	repository, err := player_save.NewRepository(directory, func(save_snapshot.SaveSnapshot) error { return nil })
	require.NoError(t, err)
	envelope := playerSaveTestRecord(player_save.Slot04, "", validCommandTestSnapshot()).Envelope
	record, err := repository.Save(player_save.Slot04, envelope, "")
	require.NoError(t, err)
	path := filepath.Join(directory, string(record.Slot)+".json")
	before, err := os.ReadFile(path)
	require.NoError(t, err)
	old := &playerSaveTestScene{
		playthrough: playthroughMetadata{id: testPlaythroughID}, slot: record.Slot, revision: record.Revision,
		captureErr: game_state.ErrSaveSnapshotNotRepresentable,
	}
	game := NewGame(old)
	game.ConfigurePlayerSaveRepository(repository, func(*config.UltimaVConfiguration, save_snapshot.SaveSnapshot) (Scene, error) {
		return &playerSaveTestScene{}, nil
	})
	_, err = game.quickLoadPlayerSave()
	require.NoError(t, err)
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, before, after)
	assert.Zero(t, old.captures)
}

type quickLoadFinalizingScene struct {
	playerSaveTestScene
	onDispose func()
}

func (scene *quickLoadFinalizingScene) Dispose() {
	scene.playerSaveTestScene.Dispose()
	scene.onDispose()
}

func TestQuickLoadRetiresOldOwnerBeforePublishingReplacement(t *testing.T) {
	record := playerSaveTestRecord(player_save.Slot02, "reviewed", validCommandTestSnapshot())
	old := &quickLoadFinalizingScene{playerSaveTestScene: playerSaveTestScene{
		slot: record.Slot, revision: record.Revision, playthrough: playthroughMetadata{id: testPlaythroughID},
	}}
	candidate := &playerSaveTestScene{}
	game := NewGame(old)
	generation := game.DebugSceneGeneration()
	old.onDispose = func() {
		assert.Same(t, old, game.currentScene, "shutdown callbacks must still address their old owner")
		assert.Equal(t, generation, game.DebugSceneGeneration())
		assert.Equal(t, 1, candidate.activations, "do not dispose the old scene before candidate activation succeeds")
	}
	game.ConfigurePlayerSaveRepository(&playerSaveTestRepository{records: map[player_save.SlotID]player_save.Record{record.Slot: record}},
		func(*config.UltimaVConfiguration, save_snapshot.SaveSnapshot) (Scene, error) { return candidate, nil })
	_, err := game.quickLoadPlayerSave()
	require.NoError(t, err)
	assert.Same(t, candidate, game.currentScene)
	assert.Equal(t, 1, old.disposals)
}
