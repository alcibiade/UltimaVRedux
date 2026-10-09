package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

func TestIntroMenuPhysicalNavigationRepeatsAndRejectsPaging(t *testing.T) {
	held := map[ebiten.Key]bool{}
	scene := newIntroInputTestScene()
	scene.keyboard = input.NewKeyboardWithKeyStateSource(1000, func(key ebiten.Key) bool { return held[key] })
	held[ebiten.KeyDown] = true
	require.NoError(t, scene.Update(nil))
	require.Equal(t, 1, scene.nCurrentSelection)
	for range 17 {
		require.NoError(t, scene.Update(nil))
	}
	require.Equal(t, 1, scene.nCurrentSelection)
	require.NoError(t, scene.Update(nil))
	require.Equal(t, 2, scene.nCurrentSelection)
	clear(held)
	require.NoError(t, scene.Update(nil))
	held[ebiten.KeyShiftLeft], held[ebiten.KeyUp] = true, true
	for range 60 {
		require.NoError(t, scene.Update(nil))
	}
	require.Equal(t, 2, scene.nCurrentSelection)
}

func TestConfigurationPhysicalNavigationRepeatsWithoutChangingSettings(t *testing.T) {
	held := map[ebiten.Key]bool{}
	keyboard := input.NewKeyboardWithKeyStateSource(1000, func(key ebiten.Key) bool { return held[key] })
	configuration := dialogConfiguration(&config.AudioConfiguration{})
	before := configuration.GameplayConfiguration
	dialog := NewConfigurationDialog(
		&sprites.PercentBasedPlacement{StartPercentX: .1, EndPercentX: .9, StartPercentY: .1, EndPercentY: .9},
		keyboard, func() {}, configuration, &audioSettingsManagerStub{},
	)
	held[ebiten.KeyDown] = true
	dialog.Update()
	require.Equal(t, 1, dialog.focusedGameplayIndex)
	for range 18 {
		dialog.Update()
	}
	require.Equal(t, 2, dialog.focusedGameplayIndex)
	clear(held)
	keyboard.ClearFrameConsumption()
	dialog.Update()
	held[ebiten.KeyShiftRight], held[ebiten.KeyUp] = true, true
	for range 60 {
		dialog.Update()
	}
	require.Equal(t, 2, dialog.focusedGameplayIndex)
	require.Equal(t, before, configuration.GameplayConfiguration)
}
