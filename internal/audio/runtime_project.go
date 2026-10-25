package audio

import (
	"fmt"

	berad "github.com/bradhannah/berad-sdk/modules/berad-audio"
	"github.com/bradhannah/berad-sdk/modules/berad-audio/authoring"
)

// PlaybackLibrary lazily freezes one Berad Audio projection shared by this
// project's managers/lanes. It never opens production defaults for another project.
func (library *Library) PlaybackLibrary() (*berad.Library, error) {
	library.playbackOnce.Do(func() {
		definition, err := authoring.PlaybackDefinition(library.Document())
		if err != nil {
			library.playbackError = err

			return
		}
		library.playback, library.playbackError = berad.NewLibrary(definition,
			encodedLibraryFS{library: library}, max(1, library.encodedBytes))
	})
	if library.playbackError != nil {
		return nil, fmt.Errorf("freeze module playback library: %w", library.playbackError)
	}

	return library.playback, nil
}
