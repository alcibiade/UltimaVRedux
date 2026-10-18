package audio

import (
	"bytes"
	"context"
	"io"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
)

type recordingReaderFunc func([]byte) (int, error)

func (read recordingReaderFunc) Read(buffer []byte) (int, error) { return read(buffer) }

func TestRecordingReadCancellationDiscardsPartialAndFinalBytes(t *testing.T) {
	for _, final := range []bool{false, true} {
		t.Run(map[bool]string{false: "partial", true: "final"}[final], func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			reads := 0
			reader := recordingReaderFunc(func(buffer []byte) (int, error) {
				reads++
				require.LessOrEqual(t, len(buffer), 64<<10)
				cancel()
				if final {
					return copy(buffer, "final bytes"), io.EOF
				}
				return copy(buffer, "partial bytes"), nil
			})
			data, err := readRecordingBytes(ctx, reader)
			require.ErrorIs(t, err, context.Canceled)
			require.Nil(t, data)
			require.Equal(t, 1, reads, "cancellation must prevent another source read")
		})
	}
}

func TestRecordingReadDeadlineDuringRead(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		reader := recordingReaderFunc(func(buffer []byte) (int, error) {
			time.Sleep(2 * time.Second)
			return copy(buffer, "late bytes"), io.EOF
		})
		data, err := readRecordingBytes(ctx, reader)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.Nil(t, data)
	})
}

func TestRecordingReadPreservesShortReadsAndFailsWithoutProgress(t *testing.T) {
	input := bytes.Repeat([]byte("encoded recording"), 9000)
	source := bytes.NewReader(input)
	reader := recordingReaderFunc(func(buffer []byte) (int, error) {
		return source.Read(buffer[:min(7, len(buffer))])
	})
	data, err := readRecordingBytes(context.Background(), reader)
	require.NoError(t, err)
	require.Equal(t, input, data)
	_, err = readRecordingBytes(context.Background(), recordingReaderFunc(func([]byte) (int, error) {
		return 0, nil
	}))
	require.ErrorIs(t, err, io.ErrNoProgress)
}

func TestRecordingAnalysisAndVerificationRejectCancellationAtEOF(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reader := recordingReaderFunc(func(buffer []byte) (int, error) {
		cancel()
		return copy(buffer, []byte{0, 0, 0, 0}), io.EOF
	})
	peaks, loudness, err := recordingAnalysis(ctx, reader)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, peaks)
	require.Nil(t, loudness)
	ctx, cancelVerify := context.WithCancel(context.Background())
	defer cancelVerify()
	clip := calibrationTone(t, 0.1)
	err = clip.verifyBytes(ctx, recordingReaderFunc(func([]byte) (int, error) {
		cancelVerify()
		return 0, io.EOF
	}))
	require.ErrorIs(t, err, context.Canceled)
}
