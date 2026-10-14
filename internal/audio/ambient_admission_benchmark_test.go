package audio

import (
	"bytes"
	"io"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

type benchmarkPCM struct{ *bytes.Reader }

func (stream benchmarkPCM) Length() int64 { return stream.Size() }

func BenchmarkAmbientReconciliationAndService(b *testing.B) {
	for _, count := range []int{8, 32, 128} {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			config, err := NewAudioConfiguration()
			require.NoError(b, err)
			manager, err := NewAudioManagerWithRuntime(config, getSharedAudioContext(), func() time.Time { return time.Time{} })
			require.NoError(b, err)
			b.Cleanup(func() { require.NoError(b, manager.Close()) })
			ambient := manager.ambientSounds
			pcm := make([]byte, 4096)
			ambient.openStream = func(string) (io.ReadSeeker, error) { return benchmarkPCM{bytes.NewReader(pcm)}, nil }
			sources := make([]AmbientSoundSource, count)
			for index := range sources {
				sources[index] = AmbientSoundSource{
					Emitter: "synthetic", Position: references.Position{X: references.Coordinate(index)},
					SoundType: AmbientFire, IsActive: true, IsDynamic: true, SoundFile: "synthetic.pcm",
					MaxDistance: 256, baseVolume: 1, VolumeModifier: 1,
				}
			}
			ambient.UpdatePlayerPositionWithDynamicSources(references.Position{}, nil, sources)
			b.ReportAllocs()
			for b.Loop() {
				ambient.UpdatePlayerPositionWithDynamicSources(references.Position{}, nil, sources)
				ambient.Update()
			}
			b.ReportMetric(float64(len(ambient.activeSounds)), "voices")
		})
	}
}
