package audio

import (
	"strings"
)

// CueSearchTerms returns both per-moment keywords and shared profile synonyms.
func (e *LibraryEditor) CueSearchTerms(cueID string) []string {
	terms := append([]string(nil), e.library.Cues[cueID].SearchTerms...)

	return append(terms, e.library.Taxonomy.ProfileSearchTerms[e.CueProfile(cueID)]...)
}

func (e *LibraryEditor) cueSearchText(cueID string) string {
	cue := e.library.Cues[cueID]
	classification := e.CueClassification(cueID)

	return strings.Join([]string{cueID, cue.Name, cue.HowToHear, strings.Join(e.CueSearchTerms(cueID), " "),
		e.CueProfile(cueID), classification.TargetAcousticFamily, classification.Phase,
		classification.Outcome, classification.MixRole, strings.Join(classification.Emitters, " "),
		strings.Join(classification.Contexts, " ")}, " ")
}

func (e *LibraryEditor) recordingSearchText(recordingID string) string {
	recording := e.library.Recordings[recordingID]
	file := ""
	if recording.File != nil {
		file = *recording.File
	}

	return strings.Join([]string{recordingID, file, recording.Note, e.RecordingFamily(recordingID),
		strings.Join(recording.SearchTerms, " "), recording.Provenance.OriginalFilename,
		recording.Provenance.SourceCreator}, " ")
}

// SearchText links cue intent to actual recordings in either workspace without
// letting shared-file stand-ins change the authoritative classification facets.
func (e *LibraryEditor) SearchText(identity string, cue bool) string {
	var parts []string
	if cue {
		parts = append(parts, e.cueSearchText(identity))
		for _, recordingID := range e.library.Cues[identity].Recordings() {
			parts = append(parts, e.recordingSearchText(recordingID))
		}
	} else {
		parts = append(parts, e.recordingSearchText(identity))
		for _, cueID := range e.Usages(identity) {
			parts = append(parts, e.cueSearchText(cueID))
		}
	}
	text := strings.Join(parts, " ")

	return text + " " + strings.ReplaceAll(text, "_", " ")
}
