package main

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/character_creation"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

type keyboardOwnerProbe struct {
	countingDialog
	keyboard *input.Keyboard
	accepted int
}

func (d *keyboardOwnerProbe) Update() {
	d.updates++
	if d.keyboard.IsKeyPressedAndRegister(ebiten.KeyEnter) {
		d.accepted++
	}
}

func TestSceneKeyboardOwnersServiceReleaseBeforeModalEarlyReturn(t *testing.T) {
	for _, kind := range []string{"game", "intro", "setup", "creation"} {
		t.Run(kind, func(t *testing.T) {
			held := map[ebiten.Key]bool{ebiten.KeyEnter: true, ebiten.KeyShiftLeft: true}
			now := int64(1000)
			keyboard := input.NewKeyboardWithKeyStateSource(165, func(key ebiten.Key) bool { return held[key] })
			keyboard.SetElapsedMillisecondsSource(func() int64 { return now })
			probe := &keyboardOwnerProbe{keyboard: keyboard}
			update, prepare := keyboardOwnerUpdate(t, kind, keyboard, probe)
			require.True(t, keyboard.TryToRegisterKeyPress(ebiten.KeyEnter))
			prepare() // Incoming scene activation can happen before its first Update.
			require.True(t, keyboard.IsWaitingForRelease())
			delete(held, ebiten.KeyEnter)
			require.NoError(t, update())
			require.True(t, keyboard.IsWaitingForRelease(), "the modifier must also release")
			clear(held)
			require.NoError(t, update())
			require.False(t, keyboard.IsWaitingForRelease(), "scene services release even when the modal owns the frame")
			now += 166
			held[ebiten.KeyEnter] = true
			require.NoError(t, update())
			require.Equal(t, 1, probe.accepted)
			require.False(t, keyboard.IsKeyPressedAndRegister(ebiten.KeyEnter), "scene and widget share consumption")
			require.NoError(t, update())
			require.Equal(t, 1, probe.accepted)
		})
	}
}

func keyboardOwnerUpdate(
	t *testing.T, kind string, keyboard *input.Keyboard, modal widgets.Widget,
) (func() error, func()) {
	t.Helper()
	switch kind {
	case "game":
		scene, _, _, _ := newRunicReferenceTestScene(t, references.BritanniaUnderworld)
		scene.keyboard = keyboard
		scene.dialogStack.PushModalDialog(modal)
		t.Cleanup(scene.gameState.EventQueueManager.Shutdown)

		return func() error { return scene.Update(nil) }, scene.PrepareInputForSceneActivation
	case "intro":
		scene := newIntroInputTestScene()
		scene.keyboard = keyboard
		scene.dialogStack.PushModalDialog(modal)

		return func() error { return scene.Update(nil) }, scene.PrepareInputForSceneActivation
	case "setup":
		scene, _ := newDataSetupTestScene(t, dataSetupFirstRun)
		scene.keyboard = keyboard
		scene.dialogStack.PushModalDialog(modal)
		t.Cleanup(scene.Dispose)

		return func() error { return scene.Update(nil) }, scene.PrepareInputForSceneActivation
	case "creation":
		fixture := newCharacterCreationSceneFixture(t, CharacterCreationCallbacks{
			OnComplete: func(character_creation.Result) {}, OnCancel: func() {},
		})
		scene := fixture.scene
		scene.keyboard = keyboard
		scene.dialogStack.PushModalDialog(modal)

		return func() error { return scene.Update(nil) }, scene.PrepareInputForSceneActivation
	default:
		t.Fatalf("unknown owner %q", kind)

		return nil, nil
	}
}
