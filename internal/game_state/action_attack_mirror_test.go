package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestIntactMirrorBreakingConsumesNormalTownTurn(t *testing.T) {
	tests := []struct {
		name   string
		mirror indexes.SpriteIndex
	}{
		{name: "mirror", mirror: indexes.Mirror},
		{name: "avatar reflection", mirror: indexes.MirrorAvatar},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, callbacks := NewIntegrationTestBuilder(t).
				WithLocation(references.LordBritishsCastle).
				WithPlayerAt(5, 5).
				WithSystemCallbacks().
				Build()
			state.DateTime.Hour = 12
			state.DateTime.Minute = 0
			mirrorPosition := references.Position{X: 5, Y: 4}
			currentMap := state.GetLayeredMapByCurrentLocation()
			currentMap.SetTileByLayer(map_state.MapLayer, &mirrorPosition, test.mirror)
			currentMap.SetTileByLayer(map_state.MapOverrideLayer, &mirrorPosition, test.mirror)

			result := state.ActionAttackSmallMap(references.Up)

			require.True(t, result)
			assert.Equal(t, indexes.SpriteIndex(indexes.MirrorBroken), currentMap.GetTopTile(&mirrorPosition).Index)
			assert.Equal(t, "Broken!", callbacks.Messages[0])
			assert.Contains(t, callbacks.SoundEffectsPlayed, SoundGlassBreak)
			assert.Equal(t, uint8(1), state.DateTime.Minute,
				"attack_town returns a completed command after breaking a mirror")
			assert.Equal(t, []int{1}, callbacks.TimeAdvanced())
		})
	}
}

func TestIntactMirrorTakesPrecedenceOverOccupyingNPC(t *testing.T) {
	fixture := newDefenselessAttackFixture(t, indexes.Mirror, indexes.TownsPerson_KeyIndex, 1)
	currentMap := fixture.state.GetLayeredMapByCurrentLocation()
	currentMap.SetTileByLayer(
		map_state.MapOverrideLayer,
		&fixture.target,
		indexes.Mirror,
	)
	currentMap.SetTileByLayer(map_state.MapUnitLayer, &fixture.target, fixture.victim.GetKeySpriteIndex())
	require.Equal(t, fixture.victim.GetKeySpriteIndex(), currentMap.GetTopTile(&fixture.target).Index,
		"the projected NPC must hide the mirror from composited top-tile lookup")

	require.True(t, fixture.state.ActionAttackSmallMap(references.Up))

	assert.Equal(t, indexes.SpriteIndex(indexes.MirrorBroken),
		fixture.state.GetLayeredMapByCurrentLocation().
			GetTileByLayer(map_state.MapOverrideLayer, &fixture.target).Index)
	assert.Equal(t, "Broken!", fixture.callbacks.Messages[0])
	assert.Equal(t, 50, fixture.state.PartyState.Karma.Value)
	assert.False(t, fixture.state.IsCriminal())
	assert.False(t, fixture.state.PartyState.IsNPCDead(references.Britain, fixture.victimSlot))
	assert.True(t, mapUnitsContain(fixture.controller.GetNpcs(), fixture.victim))
	assert.Empty(t, fixture.kapows)
	assert.NotContains(t, fixture.callbacks.SoundEffectsPlayed, SoundWhiteNoiseDamage)
	assert.NotEqual(t, references.AiType(references.ExtortOrAttackOrFollow),
		fixture.guard.MapUnitDetails().GetOverriddenAiType())
	assert.Equal(t, uint8(1), fixture.state.DateTime.Minute)
}

func TestBrokenMirrorWithoutNPCIsNotBreakableAgain(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithPlayerAt(5, 5).
		WithSystemCallbacks().
		Build()
	brokenMirrorPosition := references.Position{X: 5, Y: 4}
	currentMap := state.GetLayeredMapByCurrentLocation()
	currentMap.SetTileByLayer(map_state.MapLayer, &brokenMirrorPosition, indexes.MirrorBroken)
	currentMap.SetTileByLayer(map_state.MapOverrideLayer, &brokenMirrorPosition, indexes.MirrorBroken)
	state.CurrentNPCAIController.GetNpcs().ReplaceWith([]map_units.MapUnit{})

	result := state.ActionAttackSmallMap(references.Up)

	assert.False(t, result)
	assert.Equal(t, "Nothing to attack!", callbacks.Messages[0])
	assert.NotContains(t, callbacks.SoundEffectsPlayed, SoundGlassBreak)
}
