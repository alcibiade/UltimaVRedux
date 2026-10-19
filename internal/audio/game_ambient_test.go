package audio

import (
	"fmt"
	"slices"
	"testing"
	"time"

	berad "github.com/bradhannah/berad-sdk/modules/berad-audio"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func dynamicFixtureSource(manager *GameAudioManager, kind AmbientSoundType, id string) AmbientSoundSource {
	definition, _ := manager.GetAmbientSoundDefinition(kind)
	source := AmbientSoundSource{ID: id, SoundType: kind, SoundFile: definition.FileName, IsActive: true,
		IsDynamic: true, MaxDistance: definition.MaxDistance, VolumeModifier: 1,
		FadeInTime: definition.FadeInTime, FadeOutTime: definition.FadeOutTime}
	source.SetBaseVolume(definition.BaseVolume)
	return source
}

func TestLiveAmbientAdapterPreservesStableIDsAndBudget(t *testing.T) {
	manager, backend, now := gameAudioFixture(t)
	sources := make([]AmbientSoundSource, 32)
	for index := range sources {
		sources[index] = dynamicFixtureSource(manager, AmbientFire, fmt.Sprintf("source/%03d", index))
	}
	manager.UpdateAmbientSoundsWithDynamicSources(references.Position{}, nil, sources)
	require.Len(t, backend.players, AmbientVoiceLimit)
	slices.Reverse(sources)
	*now = now.Add(time.Second)
	manager.UpdateAmbientSoundsWithDynamicSources(references.Position{}, nil, sources)
	manager.Update()
	require.Len(t, backend.players, AmbientVoiceLimit, "ordering alone cannot replace admitted sources")
	require.NoError(t, manager.SetAmbientVolume(0))
	manager.Update()
	require.Len(t, backend.players, AmbientVoiceLimit)
	for _, source := range manager.ambience.Snapshot().Sources {
		require.Zero(t, source.Voice.Gain)
	}
	manager.SetAmbientSuppressed(true)
	for _, player := range backend.players {
		require.False(t, player.IsPlaying())
	}
	manager.SetAmbientSuppressed(false)
	manager.Update()
	require.Empty(t, manager.ambience.Snapshot().Sources, "restoration requires fresh world discovery")
	manager.UpdateAmbientSoundsWithDynamicSources(references.Position{}, nil, sources[:1])
	initial := len(backend.players)
	sources[0].Position.X = 255
	manager.UpdateAmbientSoundsWithDynamicSources(references.Position{X: 255}, nil, sources[:1])
	require.Len(t, backend.players, initial, "projected seam coordinates do not change source identity")
}

func TestFountainFramesAndStrongestFireRetainLiveVoices(t *testing.T) {
	manager, backend, now := gameAudioDocumentFixture(t, func(document *LibraryDocument) {
		for _, tile := range []struct {
			index        indexes.SpriteIndex
			radius, gain float64
		}{
			{indexes.ItemTorch, 2, 0.25}, {indexes.Brazier, 8, 0.8},
		} {
			assignment := document.Routing.Tiles[tileAudioKey(tile.index)]
			assignment.AmbientCue = string(AmbientFire)
			assignment.RadiusTiles, assignment.Gain = new(tile.radius), new(tile.gain)
			document.Routing.Tiles[tileAudioKey(tile.index)] = assignment
		}
	})
	for frame := indexes.Fountain1; frame <= indexes.Fountain4; frame++ {
		manager.UpdateAmbientSounds(references.Position{}, []TileWithPosition{
			{Tile: &references.Tile{Index: indexes.SpriteIndex(frame)}},
			{Tile: &references.Tile{Index: indexes.Fountain1}, Position: references.Position{X: 2}},
		})
		*now = now.Add(200 * time.Millisecond)
		manager.Update()
		require.Len(t, backend.players, 1, "canonical animation family owns one voice")
	}
	manager.SetAmbientSuppressed(true)
	manager.SetAmbientSuppressed(false)
	tiles := []TileWithPosition{
		{Tile: &references.Tile{Index: indexes.ItemTorch}, Position: references.Position{X: 1}},
		{Tile: &references.Tile{Index: indexes.Brazier}, Position: references.Position{X: 4}},
	}
	manager.UpdateAmbientSounds(references.Position{}, tiles)
	plans, err := manager.planner.Sources(references.Position{}, tiles, nil)
	require.NoError(t, err)
	require.Len(t, plans, 1)
	require.InDelta(t, 0.8*AmbientDistanceGain(4, 8), plans[0].Gain, 1e-9)
	count := len(backend.players)
	manager.UpdateAmbientSounds(references.Position{}, tiles[:1])
	require.Len(t, backend.players, count, "strongest static source changes gain, not voice identity")
	manager.UpdateAmbientSounds(references.Position{X: 4}, tiles[:1])
	*now = now.Add(time.Minute)
	manager.Update()
	require.False(t, backend.players[count-1].IsPlaying())
}

func TestLighthouseSharesTileVoiceAcrossMovementFloorAndExit(t *testing.T) {
	manager, backend, now := gameAudioFixture(t)
	definition, exists := manager.GetAmbientSoundDefinition(AmbientLighthouse)
	require.True(t, exists)
	require.Equal(t, 10.0, definition.MaxDistance)
	require.Equal(t, "seagulls_01", productionLibrary().Cues[string(AmbientLighthouse)].RecordingID)
	tiles := []TileWithPosition{{Tile: &references.Tile{Index: indexes.SpriteIndex(27)}}}
	manager.UpdateAmbientSounds(references.Position{}, tiles)
	*now = now.Add(definition.FadeInTime)
	manager.Update()
	for _, location := range []references.Location{references.Fogsbane, references.Stormcrow, references.Greyhaven, references.Waveguide} {
		for floor := range 3 {
			listener := references.Position{X: references.Coordinate(floor * 7), Y: references.Coordinate(floor * 5)}
			tiles[0].Position = listener
			source, active := manager.LighthouseAmbientSource(location, listener)
			require.True(t, active)
			require.Zero(t, source.MaxDistance)
			manager.UpdateAmbientSoundsWithDynamicSources(listener, tiles, []AmbientSoundSource{source})
			manager.Update()
			require.Len(t, backend.players, 1)
			require.InDelta(t, definition.BaseVolume, manager.ambience.Snapshot().Sources[0].Voice.Gain, 1e-9)
		}
	}
	outside := tiles[0].Position
	outside.X += 5
	manager.UpdateAmbientSounds(outside, tiles)
	manager.Update()
	require.Len(t, backend.players, 1)
	require.InDelta(t, definition.BaseVolume*AmbientDistanceGain(5, 10), manager.ambience.Snapshot().Sources[0].Voice.Gain, 1e-9)
	outside.X += 6
	manager.UpdateAmbientSounds(outside, tiles)
	*now = now.Add(definition.FadeOutTime)
	manager.Update()
	require.False(t, backend.players[0].IsPlaying())
	for _, location := range []references.Location{references.Britain, references.Deceit, references.BritanniaUnderworld} {
		_, active := manager.LighthouseAmbientSource(location, references.Position{})
		require.False(t, active)
	}
}

func TestDungeonAndNavalBedsUseGameRoutingAndLiveLifecycle(t *testing.T) {
	production, devices, _ := gameAudioLibraryFixture(t, defaultRuntimeLibrary())
	for _, test := range []struct {
		cue       string
		locations []references.Location
	}{
		{"ambient/dungeon_cave", []references.Location{references.Despise, references.Destard, references.Doom, references.Shame, references.Hythloth}},
		{"ambient/dungeon_brick", []references.Location{references.Deceit, references.Wrong, references.Covetous}},
	} {
		for _, location := range test.locations {
			source, exists := production.DungeonAmbientSource(location, references.Position{})
			require.True(t, exists)
			require.Equal(t, AmbientSoundType(test.cue), source.SoundType)
			require.Empty(t, source.SoundFile)
			production.UpdateAmbientSoundsWithDynamicSources(references.Position{}, nil, []AmbientSoundSource{source})
		}
	}
	require.Empty(t, devices.players, "unfilled dungeon slots never open transports")
	for _, location := range []references.Location{references.BritanniaUnderworld, references.Britain, references.CombatLocation} {
		_, active := production.DungeonAmbientSource(location, references.Position{})
		require.False(t, active)
	}
	manager, backend, now := gameAudioDocumentFixture(t, func(document *LibraryDocument) {
		for _, id := range []string{"dungeon_cave_01", "dungeon_brick_01", "frigate_onboard_01", "skiff_onboard_01"} {
			recording := document.Recordings[id]
			recording.File = new("sounds/generated.wav")
			document.Recordings[id] = recording
		}
	})
	for _, source := range []struct {
		name    string
		resolve func(references.Position) (AmbientSoundSource, bool)
	}{
		{"cave", func(p references.Position) (AmbientSoundSource, bool) {
			return manager.DungeonAmbientSource(references.Despise, p)
		}},
		{"brick", func(p references.Position) (AmbientSoundSource, bool) {
			return manager.DungeonAmbientSource(references.Deceit, p)
		}},
		{"frigate", func(p references.Position) (AmbientSoundSource, bool) {
			return manager.NavalAmbientSource(NavalFrigate, p)
		}},
		{"skiff", func(p references.Position) (AmbientSoundSource, bool) {
			return manager.NavalAmbientSource(NavalSkiff, p)
		}},
	} {
		t.Run(source.name, func(t *testing.T) {
			bed, exists := source.resolve(references.Position{})
			require.True(t, exists)
			require.Zero(t, bed.MaxDistance)
			manager.UpdateAmbientSoundsWithDynamicSources(references.Position{}, nil, []AmbientSoundSource{bed})
			count := len(backend.players)
			*now = now.Add(bed.FadeInTime)
			manager.Update()
			for _, point := range []references.Position{{X: 7, Y: 7}, {X: 200, Y: 100}} {
				bed.Position = point
				manager.UpdateAmbientSoundsWithDynamicSources(point, nil, []AmbientSoundSource{bed})
				require.Len(t, backend.players, count)
			}
			require.NoError(t, manager.SetAmbientVolume(0))
			manager.Update()
			require.Zero(t, manager.ambience.Snapshot().Sources[0].Voice.Gain)
			require.NoError(t, manager.SetAmbientVolume(0.5))
			require.NoError(t, manager.SetSoundEffectVolume(0))
			manager.Update()
			require.InDelta(t, bed.GetBaseVolume()*0.5, manager.ambience.Snapshot().Sources[0].Voice.Gain, 1e-9)
			manager.UpdateAmbientSounds(references.Position{}, nil)
			require.Equal(t, berad.VoiceReleasing, manager.ambience.Snapshot().Sources[0].Voice.State)
			*now = now.Add(bed.FadeOutTime)
			manager.Update()
			require.False(t, backend.players[count-1].IsPlaying())
		})
	}
	_, active := manager.NavalAmbientSource("", references.Position{})
	require.False(t, active)
}
