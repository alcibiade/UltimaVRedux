package audio

import (
	"errors"
	"fmt"
	"io"

	"github.com/bradhannah/berad-sdk/modules/berad-audio/backend/ebitengine"
	sdkcatalog "github.com/bradhannah/berad-sdk/tools/berad-audio/catalog"
)

const decodedPCMBytesPerFrame = 4
const millisecondsPerSecond = 1000

var errSoundStreamUnavailable = errors.New("sound stream unavailable")

func unknownRecordingProvenance() RecordingProvenance {
	return RecordingProvenance{OriginalFilename: provenanceUnknown, SourceType: provenanceUnknown,
		SourceURL: provenanceUnknown, SourceCreator: provenanceUnknown, License: provenanceUnknown,
		ImportedFrom: provenanceUnknown, Processing: []ProcessingStep{}}
}

func decodedStreamLength(stream io.ReadSeeker) (int64, error) {
	length, ok := stream.(interface{ Length() int64 })
	if !ok || length.Length() <= 0 || length.Length()%decodedPCMBytesPerFrame != 0 {
		return 0, fmt.Errorf("%w: recording requires a positive whole-frame decoded PCM length", ErrAudioLibrary)
	}
	return length.Length(), nil
}

func inspectCatalogAsset(id, path string, measure bool) (CatalogAsset, error) {
	return sdkcatalog.InspectAsset(id, path, soundFiles, sdkcatalog.Options{Decode: ebitengine.Decode, Measure: measure})
}
