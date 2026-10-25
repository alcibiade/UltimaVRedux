package audio

import (
	"path"
	"slices"
	"strings"
)

// provenanceUnknown is the authored placeholder for an unrecorded provenance fact.
const provenanceUnknown = "unknown"

// RecordingCredit is a detached metadata-only credit, independent of playback,
// cue multiplicity, calibration, diagnostic catalog scans, and audio decoding.
type RecordingCredit struct {
	Name       string
	Music      bool
	Provenance RecordingProvenance
}

// RecordingCredits projects only registered recordings with an authored file.
// It is safe to use at the title menu with sound disabled and no audio manager.
func RecordingCredits() []RecordingCredit {
	return recordingCredits(productionLibrary())
}

func recordingCredits(library *LibraryDocument) []RecordingCredit {
	ids := make([]string, 0, len(library.Recordings))
	for id, recording := range library.Recordings {
		if recording.File != nil {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	credits := make([]RecordingCredit, 0, len(ids))
	for _, id := range ids {
		recording := library.Recordings[id]
		provenance := recording.Provenance
		provenance.Processing = slices.Clone(provenance.Processing)
		name := provenance.OriginalFilename
		if name == provenanceUnknown {
			name = path.Base(*recording.File)
		}
		credits = append(credits, RecordingCredit{
			Name: name, Music: strings.HasPrefix(*recording.File, "music/"), Provenance: provenance,
		})
	}

	return credits
}
