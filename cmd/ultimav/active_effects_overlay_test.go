package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

func TestTransientIconPresentationMatrix(t *testing.T) {
	tests := []struct {
		kind       game_state.TransientIconKind
		sprite     indexes.SpriteIndex
		tint       ucolor.EGAColor
		durationMs int64
	}{
		{game_state.TransientIconEating, indexes.ItemFood, ucolor.EGAWhite, 1000},
		{game_state.TransientIconStarving, indexes.ItemFood, ucolor.EGALightRed, 500},
		{game_state.TransientIconHealing, indexes.SignHealer, ucolor.EGALightGreen, 1500},
		{game_state.TransientIconPoisonOnset, indexes.PoisonField, ucolor.EGAGreen, 1500},
		{game_state.TransientIconPoisonCure, indexes.PoisonField, ucolor.EGALightCyan, 1500},
		{game_state.TransientIconLevelUp, indexes.StarPattern, ucolor.EGAYellow, 3000},
		{game_state.TransientIconKarmaIncrease, indexes.Shrine, ucolor.EGALightGreen, 1500},
		{game_state.TransientIconKarmaDecrease, indexes.Shrine, ucolor.EGALightRed, 1500},
	}

	for _, test := range tests {
		presentation := transientIconPresentationFor(test.kind)

		assert.Equal(t, test.sprite, presentation.spriteIndex)
		assert.Equal(t, test.tint, presentation.colorTint)
		assert.Equal(t, test.durationMs, presentation.durationMs)
	}
	assert.PanicsWithValue(t, "unsupported transient icon kind", func() {
		transientIconPresentationFor(game_state.TransientIconKind(255))
	})
}

func TestActiveEffectsOverlayRefreshesAndPrunesWithoutReordering(t *testing.T) {
	overlay := NewActiveEffectsOverlay(nil)
	overlay.RegisterTransient(game_state.TransientIconEating, 100)
	overlay.RegisterTransient(game_state.TransientIconStarving, 100)

	overlay.RegisterTransient(game_state.TransientIconEating, 500)

	require.Len(t, overlay.effects, 2)
	assert.Equal(t, game_state.TransientIconEating, overlay.effects[0].kind)
	assert.Equal(t, int64(1500), overlay.effects[0].expiresAtMs)
	assert.Equal(t, game_state.TransientIconStarving, overlay.effects[1].kind)

	overlay.RegisterTransient(game_state.TransientIconHealing, 600)

	require.Len(t, overlay.effects, 2)
	assert.Equal(t, game_state.TransientIconEating, overlay.effects[0].kind)
	assert.Equal(t, game_state.TransientIconHealing, overlay.effects[1].kind)
	overlay.Update(1499)
	assert.Len(t, overlay.effects, 2)
	overlay.Update(1500)
	require.Len(t, overlay.effects, 1)
	assert.Equal(t, game_state.TransientIconHealing, overlay.effects[0].kind)
}

func TestActiveEffectsOverlayEvictsOldestDistinctKindAtCapacity(t *testing.T) {
	overlay := NewActiveEffectsOverlay(nil)
	kinds := []game_state.TransientIconKind{
		game_state.TransientIconEating,
		game_state.TransientIconStarving,
		game_state.TransientIconHealing,
		game_state.TransientIconPoisonOnset,
		game_state.TransientIconPoisonCure,
		game_state.TransientIconLevelUp,
	}
	for _, kind := range kinds {
		overlay.RegisterTransient(kind, 0)
	}

	require.Len(t, overlay.effects, transientIconCapacity)
	assert.Equal(t, game_state.TransientIconStarving, overlay.effects[0].kind)
	assert.Equal(t, game_state.TransientIconLevelUp, overlay.effects[len(overlay.effects)-1].kind)

	overlay.RegisterTransient(game_state.TransientIconPoisonOnset, 100)
	assert.Len(t, overlay.effects, transientIconCapacity)
	assert.Equal(t, game_state.TransientIconStarving, overlay.effects[0].kind)
	assert.Equal(t, game_state.TransientIconPoisonOnset, overlay.effects[2].kind)
}

func TestActiveEffectsOverlayKeepsOppositeEventsDistinctAndClears(t *testing.T) {
	overlay := NewActiveEffectsOverlay(nil)
	for _, kind := range []game_state.TransientIconKind{
		game_state.TransientIconPoisonOnset,
		game_state.TransientIconPoisonCure,
		game_state.TransientIconKarmaIncrease,
		game_state.TransientIconKarmaDecrease,
	} {
		overlay.RegisterTransient(kind, 0)
	}

	assert.Len(t, overlay.effects, 4)
	overlay.Clear()
	overlay.Clear()
	assert.Empty(t, overlay.effects)
}

func TestGameSceneClearsTransientIconsOnReplacementAndDisposal(t *testing.T) {
	scene := &GameScene{activeEffectsOverlay: NewActiveEffectsOverlay(nil)}
	scene.activeEffectsOverlay.RegisterTransient(game_state.TransientIconEating, 0)

	scene.PrepareForSceneReplacement()
	assert.Empty(t, scene.activeEffectsOverlay.effects)

	scene.activeEffectsOverlay.RegisterTransient(game_state.TransientIconHealing, 0)
	scene.Dispose()
	scene.Dispose()
	assert.Empty(t, scene.activeEffectsOverlay.effects)
}
