package audio

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestAuthoredRoutingRoundTripAndAtomicRejection(t *testing.T) {
	editor, _ := editorFixture(t)
	original := bytes.Clone(editor.saved)
	profile := AmbientProfile{RadiusTiles: 6, FadeInMs: 300, FadeOutMs: 900}
	require.NoError(t, editor.AddAmbient("machinery", "Machinery", profile))
	routing := editor.Routing()
	routing.Tiles["216"] = TileAudio{AmbientCue: "ambient/machinery", Surface: "stone"}
	routing.Music["world/underworld"] = new(string)
	*routing.Music["world/underworld"] = "music/track/worlds"
	routing.Music["location/2"] = nil
	require.NoError(t, editor.SetRouting(routing))
	require.True(t, editor.Undo())
	require.NotEqual(t, routing.Tiles["216"], editor.Routing().Tiles["216"])
	require.True(t, editor.Redo())
	require.NoError(t, editor.Save())
	reopened, err := OpenLibraryEditor(editor.Path())
	require.NoError(t, err)
	require.Equal(t, routing, reopened.Routing())
	require.Equal(t, "worlds", routing.musicFor(references.BritanniaUnderworld, references.FloorNumber(references.UNDERWORLD)))
	cue, explicit := reopened.ResolveAssignment("location/2", false)
	require.Empty(t, cue)
	require.True(t, explicit)
	before := reopened.Routing()
	invalid := reopened.Routing()
	invalid.Tiles["217"] = TileAudio{AmbientCue: "ambient/machinery"}
	require.Error(t, reopened.SetRouting(invalid), "animation members cannot override their canonical family")
	require.Equal(t, before, reopened.Routing(), "invalid edit must not mutate the working document")
	invalid = reopened.Routing()
	delete(invalid.Music, "location/2")
	require.NoError(t, reopened.SetRouting(invalid))
	require.NoError(t, reopened.Save())
	cue, explicit = reopened.ResolveAssignment("location/2", false)
	require.Equal(t, "music/track/overworld", cue)
	require.False(t, explicit)
	// An assignment edit must not rewrite the user's existing recording entries.
	beforeRecords := objectSpans(objectSpans(original)["recordings"].data)
	afterRecords := objectSpans(objectSpans(editor.saved)["recordings"].data)
	for id, record := range beforeRecords {
		require.Equal(t, record.data, afterRecords[id].data, id)
	}
}

func TestRoutingRejectsWrongCategoriesAndInvalidProfiles(t *testing.T) {
	for name, change := range map[string]func(*Routing){
		"unknown tile":     func(r *Routing) { r.Tiles["99999"] = TileAudio{Surface: "stone"} },
		"unknown cue":      func(r *Routing) { r.Tiles["216"] = TileAudio{AmbientCue: "ambient/missing"} },
		"bad surface":      func(r *Routing) { r.DefaultSurface = "missing" },
		"missing default":  func(r *Routing) { delete(r.Music, "default") },
		"negative fade":    func(r *Routing) { r.Ambient[string(AmbientFountain)] = AmbientProfile{RadiusTiles: 8, FadeInMs: -1} },
		"wrong background": func(r *Routing) { value := string(AmbientFountain); r.Dungeon["default"] = &value },
		"wrong music":      func(r *Routing) { value := string(AmbientFountain); r.Music["default"] = &value },
		"wrong footstep":   func(r *Routing) { r.Surfaces["stone"] = []string{"music/track/town"} },
		"missing naval":    func(r *Routing) { delete(r.Naval, "skiff") },
		"unknown naval":    func(r *Routing) { r.Naval["rowboat"] = nil },
		"naval music as ambience": func(r *Routing) {
			cue := "music/track/hornpipe"
			r.Naval["frigate"] = &cue
		},
		"spatial ambience as onboard loop": func(r *Routing) {
			cue := "ambient/AmbientFire"
			r.Naval["skiff"] = &cue
		},
		"ambience as sailing music": func(r *Routing) {
			cue := "ambient/AmbientFountain"
			r.Naval["sailing"] = &cue
		},
	} {
		t.Run(name, func(t *testing.T) {
			editor, _ := editorFixture(t)
			routing := editor.Routing()
			change(&routing)
			require.Error(t, editor.SetRouting(routing))
			require.False(t, editor.Dirty())
		})
	}
}

func TestNavalSituationsAuthorDistinctOnboardLoopsAndOneSailingMoment(t *testing.T) {
	editor, _ := editorFixture(t)
	routing := editor.Routing()
	require.Equal(t, "ambient/frigate_onboard", *routing.Naval[NavalFrigate])
	require.Equal(t, "ambient/skiff_onboard", *routing.Naval[NavalSkiff])
	require.Nil(t, editor.Recording("frigate_onboard_01").File, "authored moment is silent until its recording is supplied")
	require.Nil(t, editor.Recording("skiff_onboard_01").File)
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	position := references.Position{X: 2, Y: 3}
	for _, vessel := range []string{NavalFrigate, NavalSkiff} {
		source, exists := manager.NavalAmbientSource(vessel, position)
		require.True(t, exists, "the onboard situation owns a prepared ambient moment")
		cue := productionLibrary().Cues[*productionLibrary().Routing.Naval[vessel]]
		require.Equal(t, productionLibrary().recordingPath(cue.RecordingID), source.SoundFile,
			"onboard playback follows the currently assigned recording, including an empty slot")
		require.Zero(t, source.MaxDistance)
	}
	require.Equal(t, "music/track/hornpipe", *routing.Naval["sailing"])
	require.Equal(t, "overworld", routing.musicFor(references.BuccaneersDen, 0),
		"hornpipe belongs to raising sails rather than a pirate location")
	profile := AmbientProfile{RadiusTiles: 0, FadeInMs: 300, FadeOutMs: 900}
	require.NoError(t, editor.AddAmbient("frigate_deck", "Frigate Deck", profile))
	require.NoError(t, editor.AddAmbient("skiff_water", "Skiff Water", profile))
	routing = editor.Routing()
	frigate, skiff := "ambient/frigate_deck", "ambient/skiff_water"
	routing.Naval["frigate"], routing.Naval["skiff"] = &frigate, &skiff
	require.NoError(t, editor.SetRouting(routing))
	require.NoError(t, editor.Save())
	reopened, err := OpenLibraryEditor(editor.Path())
	require.NoError(t, err)
	require.Equal(t, routing.Naval, reopened.Routing().Naval)
	require.Contains(t, reopened.RoutingUsages(frigate), Usage{Workspace: "naval", Target: "frigate", Label: "Aboard frigate · state-driven"})
	definitions := map[AmbientSoundType]AmbientSoundDefinition{
		AmbientSoundType(frigate): {SoundType: AmbientSoundType(frigate), FileName: "assets/sounds/magic/healing_01.wav",
			BaseVolume: 0.5, LoopSound: true, FadeInTime: 300 * time.Millisecond, FadeOutTime: 900 * time.Millisecond},
		AmbientSoundType(skiff): {SoundType: AmbientSoundType(skiff), FileName: "assets/sounds/magic/healing_01.wav",
			BaseVolume: 0.4, LoopSound: true, FadeInTime: 300 * time.Millisecond, FadeOutTime: 900 * time.Millisecond},
	}
	for _, vessel := range []string{"frigate", "skiff"} {
		source, exists := navalAmbientSource(routing, definitions, vessel, position)
		require.True(t, exists)
		require.Zero(t, source.MaxDistance)
		require.Equal(t, position, source.Position)
		require.Equal(t, AmbientSoundType(*routing.Naval[vessel]), source.SoundType)
	}
	ambient := manager.ambientSounds
	now := time.Unix(5, 0)
	ambient.now = func() time.Time { return now }
	frigateSource, _ := navalAmbientSource(routing, definitions, "frigate", position)
	ambient.UpdatePlayerPositionWithDynamicSources(position, nil, []AmbientSoundSource{frigateSource})
	key := AmbientSoundType("background/" + frigate)
	voice := ambient.activeSounds[key]
	require.NotNil(t, voice)
	now = now.Add(time.Second)
	frigateSource.Position = references.Position{X: 4, Y: 3}
	ambient.UpdatePlayerPositionWithDynamicSources(frigateSource.Position, nil, []AmbientSoundSource{frigateSource})
	require.Same(t, voice, ambient.activeSounds[key], "the onboard loop survives movement and raised/furled sail updates")
	ambient.UpdatePlayerPositionWithDynamicSources(position, nil, nil)
	now = now.Add(time.Second)
	ambient.UpdatePlayerPositionWithDynamicSources(position, nil, nil)
	require.NotContains(t, ambient.activeSounds, key, "disembark completes the fade")
	source, exists := navalAmbientSource(routing, definitions, "", position)
	require.False(t, exists)
	require.False(t, source.IsActive)
}

func TestFountainFramesAndNearestSourceRetainOnePlayer(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	now := time.Unix(10, 0)
	ambient := manager.ambientSounds
	ambient.now = func() time.Time { return now }
	listener := references.Position{X: 4, Y: 4}
	var first *AmbientSoundInstance
	for frame := indexes.Fountain1; frame <= indexes.Fountain4; frame++ {
		tiles := []TileWithPosition{
			{Tile: &references.Tile{Index: indexes.SpriteIndex(frame)}, Position: listener},
			{Tile: &references.Tile{Index: indexes.Fountain1}, Position: references.Position{X: 6, Y: 4}},
		}
		ambient.UpdatePlayerPosition(listener, tiles)
		require.Len(t, ambient.activeSounds, 1)
		instance := ambient.activeSounds[AmbientFountain]
		require.NotNil(t, instance)
		if first == nil {
			first = instance
		} else {
			require.Same(t, first, instance)
			require.Equal(t, time.Unix(10, 0), instance.FadeStart)
			require.Greater(t, instance.FadeGain, 0.0)
		}
		now = now.Add(200 * time.Millisecond)
	}
	ambient.UpdatePlayerPosition(listener, []TileWithPosition{
		{Tile: &references.Tile{Index: indexes.Fountain4}, Position: references.Position{X: 6, Y: 4}},
	})
	require.Same(t, first, ambient.activeSounds[AmbientFountain])
	ambient.UpdatePlayerPosition(listener, nil)
	now = now.Add(first.Definition.FadeOutTime)
	ambient.UpdatePlayerPosition(listener, nil)
	require.Empty(t, ambient.activeSounds)
}

func TestReassignedCombatTrackRetainsCombatPolicy(t *testing.T) {
	policy, player, advance := newPolicyTest(t)
	policy.combatTrack = "stones"
	policy.setCombatEncounter(1)
	require.NoError(t, policy.update())
	require.Equal(t, []string{"stones"}, player.starts)
	advance(musicFadeDuration)
	require.NoError(t, policy.update())
	require.InDelta(t, 1.0, policy.gain, 1e-9)
	player.playing = false
	require.NoError(t, policy.update())
	require.NoError(t, policy.update())
	require.Len(t, player.starts, 1, "one finite presentation per encounter")
}

func TestDungeonBackgroundResolutionAndPersistentListenerMotion(t *testing.T) {
	editor, _ := editorFixture(t)
	require.NoError(t, editor.AddAmbient("cave", "Cave", AmbientProfile{RadiusTiles: 0, FadeInMs: 100, FadeOutMs: 200}))
	routing := editor.Routing()
	cue := "ambient/cave"
	routing.Dungeon["default"] = &cue
	delete(routing.Dungeon, "location/33") // Exercise inheritance independently of the production Deceit assignment.
	routing.Dungeon["location/40"] = nil
	require.NoError(t, editor.SetRouting(routing))
	require.Equal(t, cue, routing.dungeonCue(references.Deceit))
	require.Empty(t, routing.dungeonCue(references.Doom), "explicit silence overrides the default")
	require.Empty(t, routing.dungeonCue(references.Britain), "towns never inherit dungeon ambience")
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	ambient := manager.ambientSounds
	now := time.Unix(1, 0)
	ambient.now = func() time.Time { return now }
	// Use an existing decoded test recording with background (zero-radius) semantics.
	source := AmbientSoundSource{
		SoundType: AmbientWater, SoundFile: "assets/sounds/magic/healing_01.wav",
		IsActive: true, IsDynamic: true, MaxDistance: 0, baseVolume: 0.3,
		VolumeModifier: 1, FadeInTime: 100 * time.Millisecond, FadeOutTime: 200 * time.Millisecond,
	}
	key := AmbientSoundType("background/" + string(AmbientWater))
	ambient.UpdatePlayerPositionWithDynamicSources(source.Position, nil, []AmbientSoundSource{source})
	instance := ambient.activeSounds[key]
	require.NotNil(t, instance)
	now = now.Add(time.Second)
	source.Position = references.Position{X: 5, Y: 5}
	ambient.UpdatePlayerPositionWithDynamicSources(source.Position, nil, []AmbientSoundSource{source})
	require.Same(t, instance, ambient.activeSounds[key])
	require.Len(t, ambient.activeSounds, 1)
	require.InDelta(t, 0.3, instance.Volume, 1e-9)
	ambient.UpdatePlayerPositionWithDynamicSources(source.Position, nil, nil)
	now = now.Add(time.Second)
	ambient.UpdatePlayerPositionWithDynamicSources(source.Position, nil, nil)
	require.Empty(t, ambient.activeSounds)
}

func TestTileAmbientOverridesRoundTripWithoutChangingSharedSound(t *testing.T) {
	editor, _ := editorFixture(t)
	routing := editor.Routing()
	shared := routing.Ambient[string(AmbientFire)]
	radius, gain := 12.0, 0.5
	assignment := routing.Tiles["178"]
	assignment.RadiusTiles, assignment.Gain = &radius, &gain
	routing.Tiles["178"] = assignment
	require.NoError(t, editor.SetRouting(routing))
	profile, level := editor.Routing().TileAmbientSettings("178")
	require.Equal(t, radius, profile.RadiusTiles)
	require.Equal(t, gain, level)
	profile, level = editor.Routing().TileAmbientSettings("269")
	require.Equal(t, shared, profile, "torch keeps shared defaults")
	require.Equal(t, 1.0, level)
	require.NoError(t, editor.Save())
	reopened, err := OpenLibraryEditor(editor.Path())
	require.NoError(t, err)
	require.Equal(t, assignment, reopened.Routing().Tiles["178"])
	require.Equal(t, shared, reopened.Routing().Ambient[string(AmbientFire)])
	for _, invalid := range []float64{-1, 0, 33} {
		bad := reopened.Routing()
		assignment.RadiusTiles = &invalid
		bad.Tiles["178"] = assignment
		require.Error(t, reopened.SetRouting(bad))
	}
	bad := reopened.Routing()
	tooLoud := 2.1
	assignment.RadiusTiles, assignment.Gain = &radius, &tooLoud
	bad.Tiles["178"] = assignment
	require.Error(t, reopened.SetRouting(bad))
}

func TestStrongerBrazierWinsSharedFireCueWithoutRestartingLoop(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	ambient := manager.ambientSounds
	now := time.Unix(100, 0)
	ambient.now = func() time.Time { return now }
	torch := AmbientSoundDefinition{
		SoundType: AmbientFire, FileName: "assets/sounds/magic/healing_01.wav",
		MaxDistance: 2, BaseVolume: 0.25, LoopSound: true, FadeOutTime: time.Second,
	}
	brazier := torch
	brazier.MaxDistance, brazier.BaseVolume = 8, 0.8
	ambient.AddCustomAmbientSound(indexes.ItemTorch, torch)
	ambient.AddCustomAmbientSound(indexes.Brazier, brazier)
	listener := references.Position{}
	tiles := []TileWithPosition{
		{Tile: &references.Tile{Index: indexes.ItemTorch}, Position: references.Position{X: 1}},
		{Tile: &references.Tile{Index: indexes.Brazier}, Position: references.Position{X: 4}},
	}
	ambient.UpdatePlayerPosition(listener, tiles)
	require.Len(t, ambient.activeSounds, 1)
	voice := ambient.activeSounds[AmbientFire]
	require.Equal(t, tiles[1].Position, voice.Position, "farther brazier is more audible than the nearby torch")
	require.InDelta(t, 0.6, voice.Volume, 1e-9)
	now = now.Add(time.Second)
	ambient.UpdatePlayerPosition(listener, tiles[:1])
	require.Same(t, voice, ambient.activeSounds[AmbientFire])
	require.Equal(t, torch.MaxDistance, voice.Definition.MaxDistance)
	require.InDelta(t, 0.25, voice.Volume, 1e-9)
	tiles[0].Position.X = 3
	ambient.UpdatePlayerPosition(listener, tiles[:1])
	now = now.Add(time.Second)
	ambient.UpdatePlayerPosition(listener, tiles[:1])
	require.Empty(t, ambient.activeSounds, "the torch's shorter radius controls departure")
}
