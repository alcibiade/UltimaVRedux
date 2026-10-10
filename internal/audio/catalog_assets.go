package audio

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
)

const catalogMeasurementSampleRate = 44100

func libraryFingerprint() string {
	data, err := soundFiles.ReadFile(libraryManifestPath)
	if err != nil {
		panic(err) // Mandatory embedded manifest; productionLibrary validates it as well.
	}
	digest := sha256.Sum256(data)

	return hex.EncodeToString(digest[:])
}

func inspectCatalogAsset(id, path string, measure bool) (CatalogAsset, error) {
	var asset CatalogAsset
	asset.ID, asset.Path, asset.Availability = id, path, CatalogPresent
	asset.Format = DetectAudioFormat(path).String()
	asset.ListeningReview, asset.LoopReview = CatalogUnreviewed, CatalogUnreviewed
	asset.Provenance = unknownRecordingProvenance()
	data, err := soundFiles.ReadFile(path)
	if err != nil {
		return asset, fmt.Errorf("read audio recording %s: %w", path, err)
	}
	digest := sha256.Sum256(data)
	asset.SHA256 = hex.EncodeToString(digest[:])
	stream, err := DecodeAudioStream(data, path, catalogMeasurementSampleRate)
	if err != nil {
		return asset, fmt.Errorf("inspect audio recording %s: %w", path, err)
	}
	length, ok := stream.(interface{ Length() int64 })
	if !ok || length.Length() < 0 {
		return asset, fmt.Errorf("%w: recording %s has no finite decoded length", ErrAudioCatalog, path)
	}
	asset.DurationMs = length.Length() * millisecondsPerSecond / (catalogMeasurementSampleRate * decodedPCMBytesPerFrame)
	if measure {
		peak, rms, err := measureCatalogPCM(stream)
		if err != nil {
			return asset, fmt.Errorf("measure recording %s: %w", path, err)
		}
		asset.PeakDBFS, asset.RMSDBFS = amplitudeDB(peak), amplitudeDB(rms)
		asset.PCMMeasured = true
	}

	return asset, nil
}

// measureCatalogPCM streams interleaved signed 16-bit PCM. These measurements
// describe the raw decoded asset, not LUFS, true peak, or perceived acceptance.
func measureCatalogPCM(stream io.Reader) (float64, float64, error) {
	const (
		bufferBytes = 8192
		fullScale   = 32768.0
		sampleBytes = 2
	)
	buffer := make([]byte, bufferBytes)
	var peak, squares float64
	var samples int64
	for {
		count, err := io.ReadFull(stream, buffer)
		if count%sampleBytes != 0 {
			return 0, 0, fmt.Errorf("%w: incomplete PCM sample", ErrAudioCatalog)
		}
		for offset := 0; offset < count; offset += sampleBytes {
			// PCM intentionally reinterprets the unsigned word as a signed sample.
			//nolint:gosec // This is a bit-preserving signed PCM conversion, not a narrowing length.
			sample := float64(int16(binary.LittleEndian.Uint16(buffer[offset:offset+sampleBytes]))) / fullScale
			peak = math.Max(peak, math.Abs(sample))
			squares += sample * sample
			samples++
		}
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			break
		}
		if err != nil {
			return 0, 0, fmt.Errorf("read decoded PCM: %w", err)
		}
	}
	if samples == 0 {
		return 0, 0, fmt.Errorf("%w: empty recording", ErrAudioCatalog)
	}

	return peak, math.Sqrt(squares / float64(samples)), nil
}

func amplitudeDB(amplitude float64) *float64 {
	if amplitude == 0 {
		return nil // Negative infinity is not valid JSON. Never substitute a finite level.
	}
	const amplitudeDecibelFactor = 20
	value := amplitudeDecibelFactor * math.Log10(amplitude)

	return &value
}
