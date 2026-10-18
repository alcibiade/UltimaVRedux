package reduxadapter

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"testing/fstest"
	"time"

	gameaudio "github.com/bradhannah/UltimaVRedux/internal/audio"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	audio "github.com/bradhannah/UltimaVRedux/modules/berad-audio"
	"github.com/bradhannah/UltimaVRedux/modules/berad-audio/backend/ebitengine"
)

type player struct {
	playing bool
	gain    float64
	source  io.ReadSeeker
	closed  int
}

func (device *player) Play()                    { device.playing = true }
func (device *player) Pause()                   { device.playing = false }
func (device *player) PauseAndStopReading()     { device.playing = false; device.closed++ }
func (device *player) IsPlaying() bool          { return device.playing }
func (device *player) SetVolume(gain float64)   { device.gain = gain }
func (*player) SetPosition(time.Duration) error { return nil }
func (*player) Position() time.Duration         { return 0 }

type host struct{ players []*player }

func (*host) SampleRate() int { return 44100 }
func (backend *host) NewPlayer(source io.ReadSeeker) (audio.Transport, error) {
	device := &player{source: source}
	backend.players = append(backend.players, device)
	return device, nil
}

// Retain Redux's actual authored identities, taxonomy and routing while replacing
// payloads with one generated WAV. Neither original-game files nor production
// audio bytes enter the bridge library; null recording slots remain null.
func project(t *testing.T) (*gameaudio.Library, fstest.MapFS) {
	t.Helper()
	root := os.Getenv("BERAD_AUDIO_REDUX_ROOT")
	if root == "" {
		t.Fatal("gate must supply the reviewed Redux checkout")
	}
	data, err := os.ReadFile(filepath.Join(root, gameaudio.LibraryManifestRelativePath))
	if err != nil {
		t.Fatal(err)
	}
	var document gameaudio.LibraryDocument
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	file := "sounds/synthetic.wav"
	for identity, recording := range document.Recordings {
		if recording.File != nil {
			recording.File = &file
		}
		recording.PlaybackWindow = nil
		recording.CalibrationGainDB = nil
		recording.CalibrationPolicy = ""
		recording.CalibrationTargetDB = nil
		document.Recordings[identity] = recording
	}
	fire := document.Cues[string(gameaudio.AmbientFire)]
	fire.VolumeModifier = 0.5
	document.Cues[string(gameaudio.AmbientFire)] = fire
	recording := document.Recordings[fire.RecordingID]
	calibration := 6.0
	recording.CalibrationGainDB = &calibration
	recording.PlaybackWindow = &gameaudio.PlaybackWindow{StartMs: 20, EndMs: 40}
	document.Recordings[fire.RecordingID] = recording
	document.Routing.Ambient[string(gameaudio.AmbientFire)] = gameaudio.AmbientProfile{RadiusTiles: 8, FadeInMs: 1000, FadeOutMs: 1000}
	key := strconv.Itoa(int(indexes.CanonicalAnimationSprite(indexes.FireLogs)))
	assignment := document.Routing.Tiles[key]
	assignment.AmbientCue = string(gameaudio.AmbientFire)
	assignment.Gain = new(2.0)
	document.Routing.Tiles[key] = assignment
	data, err = json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	files := fstest.MapFS{file: {Data: syntheticWAV()}}
	library, err := gameaudio.NewLibrary(data, files, gameaudio.LibraryOptions{Validate: gameaudio.ReduxLibraryValidator})
	if err != nil {
		t.Fatal(err)
	}
	return library, files
}

func syntheticWAV() []byte {
	data := make([]byte, 44+44100*4/10)
	copy(data, "RIFF")
	binary.LittleEndian.PutUint32(data[4:], uint32(len(data)-8))
	copy(data[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(data[16:], 16)
	binary.LittleEndian.PutUint16(data[20:], 1)
	binary.LittleEndian.PutUint16(data[22:], 2)
	binary.LittleEndian.PutUint32(data[24:], 44100)
	binary.LittleEndian.PutUint32(data[28:], 44100*4)
	binary.LittleEndian.PutUint16(data[32:], 4)
	binary.LittleEndian.PutUint16(data[34:], 16)
	copy(data[36:], "data")
	binary.LittleEndian.PutUint32(data[40:], uint32(len(data)-44))
	return data
}

func TestReduxJSONProjectionAndStrongestFireUseOneCalibratedVoice(t *testing.T) {
	library, files := project(t)
	backend := new(host)
	now := time.Unix(100, 0)
	options := audio.DefaultAmbientOptions(backend, ebitengine.Decode)
	options.Playback.Now = func() time.Time { return now }
	adapter, err := NewAmbient(library, files, 1<<20, options)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := adapter.Close(); err != nil {
			t.Error(err)
		}
	})
	tiles := []gameaudio.TileWithPosition{{Tile: &references.Tile{Index: indexes.FireLogs}},
		{Tile: &references.Tile{Index: indexes.FireLogs}, Position: references.Position{X: 4}}}
	if err := adapter.Reconcile(references.Position{}, tiles, nil); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Second)
	if err := adapter.Update(); err != nil {
		t.Fatal(err)
	}
	snapshot := adapter.Snapshot()
	if len(backend.players) != 1 || len(snapshot.Sources) != 1 {
		t.Fatal("static fires not coalesced")
	}
	want := math.Pow(10, 6.0/20) // 0.5 cue * 2 source; calibration applies exactly once.
	if math.Abs(snapshot.Sources[0].Voice.Gain-want) > 1e-9 {
		t.Fatal(snapshot)
	}
	pcm, err := io.ReadAll(io.LimitReader(backend.players[0].source, 44100*4/50))
	if err != nil || len(pcm) != 44100*4/50 {
		t.Fatal("windowed decoder failed", len(pcm), err)
	}
	if err := adapter.Reconcile(references.Position{X: 4}, tiles[1:], nil); err != nil {
		t.Fatal(err)
	}
	if len(backend.players) != 1 {
		t.Fatal("strongest fire change restarted transport")
	}
	if err := adapter.SetVolume(0.5); err != nil {
		t.Fatal(err)
	}
	if math.Abs(adapter.Snapshot().Sources[0].Voice.Gain-want*0.5) > 1e-9 {
		t.Fatal(adapter.Snapshot())
	}
	if err := adapter.Reconcile(references.Position{}, nil, nil); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Second)
	if err := adapter.Update(); err != nil {
		t.Fatal(err)
	}
	if backend.players[0].closed != 1 || len(adapter.Snapshot().Sources) != 0 {
		t.Fatal("departure leaked source")
	}
}

func TestReduxLighthouseAndProjectedDynamicIdentityRemainGameOwned(t *testing.T) {
	library, _ := project(t)
	planner, err := gameaudio.NewAmbientPlanner(library)
	if err != nil {
		t.Fatal(err)
	}
	document := library.Document()
	makeSource := func(cueID, role string) gameaudio.AmbientSoundSource {
		cue := document.Cues[cueID]
		recording := document.Recordings[cue.RecordingID]
		gain := cue.VolumeModifier
		if recording.CalibrationGainDB != nil {
			gain *= math.Pow(10, *recording.CalibrationGainDB/20)
		}
		source := gameaudio.AmbientSoundSource{ID: "map/floor/entity", Emitter: role, SoundType: gameaudio.AmbientSoundType(cueID),
			IsActive: true, IsDynamic: true, VolumeModifier: 1, MaxDistance: 8, FadeInTime: time.Second, FadeOutTime: time.Second}
		if recording.File != nil {
			source.SoundFile = *recording.File
		}
		source.SetBaseVolume(gain)
		return source
	}
	lighthouse := makeSource(string(gameaudio.AmbientLighthouse), "lighthouse")
	lighthouse.MaxDistance = 0
	tiles := []gameaudio.TileWithPosition{{Tile: &references.Tile{Index: 27}}}
	plan, err := planner.Sources(references.Position{}, tiles, []gameaudio.AmbientSoundSource{lighthouse})
	if err != nil || len(plan) != 1 || plan[0].ID != string(gameaudio.AmbientLighthouse) || plan[0].Gain != 1 {
		t.Fatal(plan, err)
	}
	source := makeSource(string(gameaudio.AmbientFire), "moongate")
	source.Position.X = -1
	left, err := planner.Sources(references.Position{}, nil, []gameaudio.AmbientSoundSource{source})
	if err != nil {
		t.Fatal(err)
	}
	source.Position.X = 255
	right, err := planner.Sources(references.Position{X: 255}, nil, []gameaudio.AmbientSoundSource{source})
	if err != nil {
		t.Fatal(err)
	}
	if left[0].ID != right[0].ID || right[0].Emitter != "moongate" {
		t.Fatal(left, right)
	}
	source.MaxDistance = 0
	source.Position.X = 100
	bed, err := planner.Sources(references.Position{}, nil, []gameaudio.AmbientSoundSource{source})
	if err != nil || bed[0].ID != "background/"+string(gameaudio.AmbientFire) || bed[0].Gain != 1 {
		t.Fatal(bed, err)
	}
	for _, distance := range []float64{0, 1, 2, 4, 8, 9} {
		if gameaudio.AmbientDistanceGain(distance, 8) != audio.QuadraticDistanceGain(distance, 8) {
			t.Fatal("attenuation changed", distance)
		}
	}
}

func TestReduxProjectionPreservesNullSlotsAndOrderedVariants(t *testing.T) {
	library, _ := project(t)
	document := library.Document()
	definition, err := PlaybackDefinition(document)
	if err != nil {
		t.Fatal(err)
	}
	for id, recording := range document.Recordings {
		if recording.File == nil && definition.Recordings[audio.RecordingID(id)].File != "" {
			t.Fatal("empty slot filled", id)
		}
	}
	for id, cue := range document.Cues {
		actual := definition.Cues[audio.CueID(id)]
		if len(actual.Recordings) != len(cue.Recordings()) {
			t.Fatal("variant count", id)
		}
		for index, recording := range cue.Recordings() {
			if actual.Recordings[index] != audio.RecordingID(recording) {
				t.Fatal("variant order", id)
			}
		}
	}
	fire := document.Cues[string(gameaudio.AmbientFire)]
	window := definition.Recordings[audio.RecordingID(fire.RecordingID)].Window
	if window.Start != 20*time.Millisecond || window.End != 40*time.Millisecond {
		t.Fatal(window)
	}
	recording := document.Recordings[fire.RecordingID]
	recording.PlaybackWindow.EndMs = math.MaxInt64
	document.Recordings[fire.RecordingID] = recording
	if _, err := PlaybackDefinition(document); !errors.Is(err, gameaudio.ErrAudioLibrary) {
		t.Fatal("overflowed authored window accepted", err)
	}
}
