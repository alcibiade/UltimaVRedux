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

// Wisps, seahorses, daemons and ghosts light the map on every animation frame,
// each with its own marker and radius; other creatures emit no light.
func TestNPCLightInfoClassifiesLightEmittingCreatures(t *testing.T) {
	gs := &GameState{}
	for _, test := range []struct {
		name   string
		sprite indexes.SpriteIndex
		marker indexes.SpriteIndex
		radius int
	}{
		{name: "seahorse first frame", sprite: indexes.Seahorse1, marker: map_state.SeahorseMarkerIndex, radius: map_state.SeahorseLightDistance},
		{name: "seahorse last frame", sprite: indexes.Seahorse1 + 3, marker: map_state.SeahorseMarkerIndex, radius: map_state.SeahorseLightDistance},
		{name: "wisp first frame", sprite: indexes.Wisp1, marker: map_state.WispMarkerIndex, radius: map_state.WispLightDistance},
		{name: "wisp last frame", sprite: indexes.Wisp1 + 3, marker: map_state.WispMarkerIndex, radius: map_state.WispLightDistance},
		{name: "daemon first frame", sprite: indexes.Daemon1, marker: map_state.DaemonMarkerIndex, radius: map_state.DaemonLightDistance},
		{name: "daemon last frame", sprite: indexes.Daemon1 + 3, marker: map_state.DaemonMarkerIndex, radius: map_state.DaemonLightDistance},
		{name: "ghost first frame", sprite: indexes.Ghost1, marker: map_state.GhostMarkerIndex, radius: map_state.GhostLightDistance},
		{name: "ghost last frame", sprite: indexes.Ghost1 + 3, marker: map_state.GhostMarkerIndex, radius: map_state.GhostLightDistance},
		{name: "wizard", sprite: indexes.Wizard1},
		{name: "orc", sprite: indexes.Orc1},
		{name: "skeleton", sprite: indexes.Skeleton1},
		{name: "dragon", sprite: indexes.Dragon1},
		{name: "avatar", sprite: indexes.Avatar1},
		{name: "no sprite", sprite: 0},
	} {
		enemy := map_units.NewEnemyNPC(references.EnemyReference{KeyFrameTile: &references.Tile{Index: test.sprite}}, -1)
		enemy.SetPos(references.Position{X: 3, Y: 4})
		info := gs.getNPCLightInfoIfEmitsLight(&enemy)
		if test.marker == 0 {
			assert.Nil(t, info, test.name)

			continue
		}
		require.NotNil(t, info, test.name)
		assert.Equal(t, map_state.NPCLightInfo{
			Position: references.Position{X: 3, Y: 4}, MarkerIndex: test.marker, LightRadius: test.radius,
		}, *info, test.name)
	}
}

func TestNPCLightMarkersBreatheWithCreatureTints(t *testing.T) {
	for _, marker := range []indexes.SpriteIndex{
		map_state.SeahorseMarkerIndex, map_state.WispMarkerIndex, map_state.DaemonMarkerIndex, map_state.GhostMarkerIndex,
	} {
		props := map_state.GetLightSourceProperties(marker)
		assert.True(t, props.HasBreathing, "marker %d", marker)
		assert.Positive(t, props.BreathingPeriodMs, "marker %d", marker)
	}
	seahorse := map_state.GetLightSourceProperties(map_state.SeahorseMarkerIndex)
	assert.Greater(t, seahorse.TintColor.B, seahorse.TintColor.R, "seahorse glow is blue")
	daemon := map_state.GetLightSourceProperties(map_state.DaemonMarkerIndex)
	assert.Greater(t, daemon.TintColor.R, daemon.TintColor.B, "daemon glow is red")

	unknown := map_state.GetLightSourceProperties(indexes.SpriteIndex(1))
	assert.False(t, unknown.HasBreathing)
	assert.Equal(t, [3]float32{1, 1, 1}, [3]float32{unknown.TintColor.R, unknown.TintColor.G, unknown.TintColor.B})
}
