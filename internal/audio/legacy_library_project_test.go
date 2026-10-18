package audio

import (
	"fmt"
	"strings"
)

func (library *Library) readFile(name string) ([]byte, error) {
	if !strings.HasPrefix(name, library.prefix) {
		return nil, fmt.Errorf("%w: asset %q is outside the library mount", ErrAudioLibrary, name)
	}
	data, exists := library.assets[strings.TrimPrefix(name, library.prefix)]
	if !exists {
		return nil, fmt.Errorf("%w: asset %q is not in this library", ErrAudioLibrary, name)
	}
	return data, nil
}
