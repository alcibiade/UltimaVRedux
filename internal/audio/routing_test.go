package audio

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/require"
)

func TestReduxRoutingRejectsUnknownWorldTargetsAndWrongCategories(t *testing.T) {
	for name, change := range map[string]func(*Routing){
		"unknown tile":       func(r *Routing) { r.Tiles["99999"] = TileAudio{Surface: "stone"} },
		"animation member":   func(r *Routing) { r.Tiles["217"] = TileAudio{AmbientCue: string(AmbientFountain)} },
		"unknown cue":        func(r *Routing) { r.Tiles["216"] = TileAudio{AmbientCue: "ambient/missing"} },
		"bad surface":        func(r *Routing) { r.DefaultSurface = "missing" },
		"missing default":    func(r *Routing) { delete(r.Music, "default") },
		"negative fade":      func(r *Routing) { r.Ambient[string(AmbientFountain)] = AmbientProfile{RadiusTiles: 8, FadeInMs: -1} },
		"spatial background": func(r *Routing) { r.Dungeon["default"] = new(string(AmbientFountain)) },
		"ambient music":      func(r *Routing) { r.Music["default"] = new(string(AmbientFountain)) },
		"music footstep":     func(r *Routing) { r.Surfaces["stone"] = []string{"music/track/town"} },
		"missing naval":      func(r *Routing) { delete(r.Naval, "skiff") },
		"unknown naval":      func(r *Routing) { r.Naval["rowboat"] = nil },
		"music onboard":      func(r *Routing) { r.Naval["frigate"] = new("music/track/hornpipe") },
		"spatial onboard":    func(r *Routing) { r.Naval["skiff"] = new(string(AmbientFire)) },
		"ambient sailing":    func(r *Routing) { r.Naval["sailing"] = new(string(AmbientFountain)) },
	} {
		t.Run(name, func(t *testing.T) {
			document := freshTestLibrary(t)
			change(&document.Routing)
			_, err := parseAudioLibraryDocument(encodeAudioValue(document))
			require.Error(t, err)
		})
	}
}

func TestReduxRoutingInheritanceSilenceAndIndependentTileOverrides(t *testing.T) {
	routing := cloneAudioValue(productionLibrary().Routing)
	routing.Music["location/2"] = nil
	cue, explicit := resolveAudioAssignment(routing.Music, "location/2")
	require.Empty(t, cue)
	require.True(t, explicit)
	delete(routing.Music, "location/2")
	cue, explicit = resolveAudioAssignment(routing.Music, "location/2")
	require.Equal(t, *routing.Music["default"], cue)
	require.False(t, explicit)
	routing.Dungeon["default"] = new("ambient/dungeon_cave")
	delete(routing.Dungeon, locationMusicKey(references.Deceit))
	routing.Dungeon[locationMusicKey(references.Doom)] = nil
	require.Equal(t, "ambient/dungeon_cave", routing.dungeonCue(references.Deceit))
	require.Empty(t, routing.dungeonCue(references.Doom))
	require.Empty(t, routing.dungeonCue(references.Britain))
	require.Equal(t, "overworld", routing.musicFor(references.BuccaneersDen, 0))
	require.Equal(t, "music/track/hornpipe", *routing.Naval["sailing"])
	shared := routing.Ambient[string(AmbientFire)]
	assignment := routing.Tiles["178"]
	assignment.RadiusTiles, assignment.Gain = new(12.0), new(0.5)
	routing.Tiles["178"] = assignment
	profile, gain := routing.TileAmbientSettings("178")
	require.Equal(t, 12.0, profile.RadiusTiles)
	require.Equal(t, 0.5, gain)
	profile, gain = routing.TileAmbientSettings("269")
	require.Equal(t, shared, profile)
	require.Equal(t, 1.0, gain)
}

func TestReassignedCombatTrackRetainsCombatPolicy(t *testing.T) {
	policy, player, advance := newPolicyTest(t)
	policy.combatTrack = "stones"
	policy.setCombatEncounter(1)
	advance(0)
	require.Equal(t, []string{"stones"}, player.starts)
	advance(musicFadeDuration)
	require.InDelta(t, 1.0, policy.gain, 1e-9)
	player.playing = false
	advance(0)
	advance(0)
	require.Len(t, player.starts, 1)
}
