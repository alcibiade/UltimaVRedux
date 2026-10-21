package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/character_creation"
	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/files"
	"github.com/bradhannah/UltimaVRedux/internal/player_save"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"
)

func TestCharacterCreationFlowBuildsOriginalNewGameAtIolosHut(t *testing.T) {
	directory := originaldata.Directory(t)
	templateBytes := make(map[string][]byte, 2)
	for _, filename := range []string{files.InitGam, files.InitOol} {
		path := filepath.Join(directory, filename)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("loading selected original %s: %v", filename, err)
		}
		templateBytes[filename] = data
	}

	configuration, err := config.NewUltimaVConfigurationWithPath(directory)
	require.NoError(t, err)
	configuration.SoundDisabled = true
	result := characterCreationFlowResult()
	fixture := newCharacterCreationSceneFixture(t, CharacterCreationCallbacks{
		OnComplete: func(character_creation.Result) {},
		OnCancel:   func() {},
	})
	scene := fixture.scene
	scene.config = configuration
	scene.state = characterCreationComplete
	scene.terminal = true
	scene.result = result
	scene.hasResult = true
	game := NewGame(scene)
	game.playthroughIDProvider = func() (string, error) { return testPlaythroughID, nil }
	repository := newCharacterCreationFlowRepository(player_save.Slot01, player_save.SlotEmpty, "")
	game.ConfigurePlayerSaveRepository(repository, func(
		configuration *config.UltimaVConfiguration,
		snapshot save_snapshot.SaveSnapshot,
	) (Scene, error) {
		return NewGameSceneFromSaveSnapshot(configuration, snapshot)
	})

	game.completeCharacterCreation(result)
	pressCharacterCreationFlowKey(t, game, scene, ebiten.KeyEnter)
	confirmCharacterCreationOverwrite(t, game, scene)

	gameplay, ok := game.currentScene.(*GameScene)
	require.True(t, ok)
	location := gameplay.gameState.MapState.PlayerLocation
	assert.Equal(t, references.IolosHut, location.Location)
	assert.Zero(t, location.Floor)
	assert.Equal(t, references.Coordinate(15), location.Position.X)
	assert.Equal(t, references.Coordinate(15), location.Position.Y)
	slot, revision, active := gameplay.playerSaveProvenance()
	assert.True(t, active)
	assert.Equal(t, player_save.Slot01, slot)
	assert.Equal(t, repository.records[player_save.Slot01].Revision, revision)
	require.Len(t, repository.saves, 1)
	assert.Equal(t, playerSaveNewGameDescription, repository.saves[0].envelope.Metadata.Description)
	assert.Equal(t, testPlaythroughID, gameplay.activePlaythrough.id)
	assertCanonicalPreparedPlayerSave(t, repository.saves[0].envelope.Snapshot)

	for filename, before := range templateBytes {
		after, err := os.ReadFile(filepath.Join(directory, filename))
		require.NoError(t, err)
		assert.Equal(t, before, after)
	}
}
