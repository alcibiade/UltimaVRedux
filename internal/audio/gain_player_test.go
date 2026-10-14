package audio

import (
	"bytes"
	"io"
	"testing"

	eaudio "github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/stretchr/testify/require"
)

func newGainPlayer(context *eaudio.Context, source io.ReadSeeker) (*gainPlayer, error) {
	return newGainPlayerWithBackend(EbitengineBackend{Context: context}, source)
}

func TestGainPlayerRetirementPreventsRestart(t *testing.T) {
	player, err := newGainPlayer(getSharedAudioContext(), bytes.NewReader(make([]byte, 4096)))
	require.NoError(t, err)
	player.Play()
	require.NoError(t, player.Close())
	require.False(t, player.IsPlaying())
	require.ErrorIs(t, player.Close(), errGainPlayerRetired)
	require.ErrorIs(t, player.Rewind(), errGainPlayerRetired)
	require.ErrorIs(t, player.SetPosition(0), errGainPlayerRetired)
	require.Panics(t, player.Play)
}
