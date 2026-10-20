package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/character_creation"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func TestCharacterCreationSceneDrawsOriginalAssetsThroughEveryState(t *testing.T) {
	directory := originaldata.Directory(t, "CREATE.16", "PROPORT.PCS", "QUESTION.DAT")
	configuration := testCharacterCreationConfig(directory)
	introSprites, err := sprites.LoadIntroSprites(directory)
	require.NoError(t, err)
	scene, err := CreateCharacterCreationScene(
		configuration,
		beradrng.New(42),
		CharacterCreationCallbacks{
			OnComplete: func(character_creation.Result) {},
			OnCancel:   func() { t.Fatal("unexpected cancellation") },
		},
		newCharacterCreationIdentityPresentation(introSprites, 0),
	)
	require.NoError(t, err)
	scene.keyboard = newPresentationTestKeyboard()
	scene.appendInputChars = func(dst []rune) []rune { return dst }
	screen := ebiten.NewImage(320, 200)
	t.Cleanup(screen.Deallocate)
	t.Cleanup(scene.canvas.Deallocate)
	t.Cleanup(scene.Dispose)
	draw := func() {
		t.Helper()
		require.NotPanics(t, func() { scene.Draw(screen) })
	}
	press := func(key ebiten.Key) {
		t.Helper()
		scene.keyboard.SimulateKeyPress(key)
		require.NoError(t, scene.Update(nil))
	}

	draw()
	scene.name = []rune("Avatar")
	press(ebiten.KeyEnter)
	draw()
	press(ebiten.KeyF)
	draw()
	press(ebiten.KeySpace)
	for range characterCreationQuestionCount {
		draw()
		press(ebiten.KeyA)
	}
	draw()
	press(ebiten.KeySpace)
	draw()
}
