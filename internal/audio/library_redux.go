package audio

import (
	"fmt"
	"slices"
	"strings"
	"sync"
)

// ReduxLibraryValidator supplies game completeness, vocabulary, asset layout and
// routing policy after shared structural validation. Neutral consumers omit it.
func ReduxLibraryValidator(document LibraryDocument) error {
	for family := range document.Taxonomy.RecordingFamilies {
		if !slices.Contains(acousticFamilies, family) {
			return fmt.Errorf("%w: invalid recording family %q", ErrAudioLibrary, family)
		}
	}
	for recordingID, recording := range document.Recordings {
		if recording.File != nil && !validRecordingPath(*recording.File) {
			return fmt.Errorf("%w: %s requires an exact sounds/ or music/ audio file", ErrAudioLibrary, recordingID)
		}
	}
	if err := validateLibraryCues(&document); err != nil {
		return err
	}
	if err := document.Taxonomy.validateProfiles(); err != nil {
		return err
	}

	return document.Routing.validate(&document)
}

func validRecordingPath(name string) bool {
	return validAssetPath(name) && (strings.HasPrefix(name, "sounds/") || strings.HasPrefix(name, "music/"))
}

func validateLibraryCues(library *audioLibrary) error {
	expected := knownLibraryCueIDs(library)
	for cueID, cue := range library.Cues {
		if !expected[cueID] {
			return fmt.Errorf("%w: unknown cue %s", ErrAudioLibrary, cueID)
		}
		if len(cue.RecordingIDs) > 0 && !strings.HasPrefix(cueID, "sfx/") {
			return fmt.Errorf("%w: %s requires either recordingId or an SFX cycle of at least two clips", ErrAudioLibrary, cueID)
		}
	}
	for cueID := range expected {
		if _, exists := library.Cues[cueID]; !exists {
			return fmt.Errorf("%w: missing known cue %s (use a null-file recording slot)", ErrAudioLibrary, cueID)
		}
	}

	return nil
}

func knownLibraryCueIDs(library *audioLibrary) map[string]bool {
	ids := make(map[string]bool)
	for _, symbol := range catalogEffectNames() {
		ids["sfx/"+symbol] = true
	}
	for cueID := range library.Routing.Ambient {
		ids[cueID] = true
	}
	for _, track := range musicTrackIDs() {
		ids["music/track/"+track] = true
	}

	return ids
}

func musicTrackIDs() []string {
	return []string{"title", "town", "dungeon", "combat", "overworld", "victory", "ambient", "blackthorn",
		"stones", "amiga", "rule", "hornpipe", "tarantella", "greyson", "reunion", "worlds"}
}

//nolint:gochecknoglobals // Only the default Redux adapter requests this immutable embedded asset snapshot.
var defaultRuntimeLibrary = sync.OnceValue(func() *Library {
	library, err := NewLibrary(encodeAudioValue(productionLibrary()), soundFiles,
		LibraryOptions{AssetRoot: "assets", Validate: ReduxLibraryValidator})
	if err != nil {
		panic(err)
	}

	return library
})
