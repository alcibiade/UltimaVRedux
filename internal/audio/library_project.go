package audio

import (
	"bytes"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"sync"

	berad "github.com/bradhannah/berad-sdk/modules/berad-audio"
)

// LibraryDocument is the versioned authoring schema. Loading makes a detached
// immutable snapshot; changing this value cannot change an existing runtime.
type LibraryDocument = audioLibrary

// LibraryValidator adds consumer completeness, vocabulary and routing rules to
// structural validation. It receives a detached document and must be deterministic.
type LibraryValidator func(LibraryDocument) error

// LibraryOptions selects an explicit asset mount and optional consumer policy.
// AssetRoot is an fs path ("." for the supplied filesystem root), not a machine path.
type LibraryOptions struct {
	AssetRoot string
	Validate  LibraryValidator
}

// Library owns an immutable document and one snapshot of each referenced encoded
// asset. Filesystem changes after construction cannot silently retarget playback.
type Library struct {
	document      *audioLibrary
	assets        map[string][]byte
	prefix        string
	encodedBytes  int64
	playbackOnce  sync.Once
	playback      *berad.Library
	playbackError error
}

// NewLibrary validates a project without consulting Redux's embedded manifest.
// Reused recordings share encoded bytes; each voice gets its own decoder/cursor.
func NewLibrary(data []byte, files fs.FS, options LibraryOptions) (*Library, error) {
	if files == nil {
		return nil, fmt.Errorf("%w: asset filesystem is required", ErrAudioLibrary)
	}
	document, err := parseLibraryDocument(data, options.Validate)
	if err != nil {
		return nil, err
	}
	root := options.AssetRoot
	if root == "" {
		root = "."
	}
	if !fs.ValidPath(root) || strings.Contains(root, "\\") {
		return nil, fmt.Errorf("%w: invalid asset root %q", ErrAudioLibrary, root)
	}
	rooted, err := fs.Sub(files, root)
	if err != nil {
		return nil, fmt.Errorf("mount audio assets: %w", err)
	}
	library := new(Library)
	library.document, library.assets = document, make(map[string][]byte)
	if root != "." {
		library.prefix = root + "/"
	}
	for _, recordingID := range sortedKeys(document.Recordings) {
		recording := document.Recordings[recordingID]
		if recording.File == nil {
			continue
		}
		name := *recording.File
		if _, exists := library.assets[name]; exists {
			continue
		}
		encoded, err := fs.ReadFile(rooted, name)
		if err != nil {
			return nil, fmt.Errorf("snapshot recording %s: %w", recordingID, err)
		}
		library.assets[name] = bytes.Clone(encoded)
		library.encodedBytes += int64(len(encoded))
	}

	return library, nil
}

// Document returns a fully detached metadata view, never the runtime's live maps.
func (library *Library) Document() LibraryDocument { return *cloneAudioValue(library.document) }

// EncodedBytes reports the retained unique asset payload, excluding decoder buffers.
func (library *Library) EncodedBytes() int64 { return library.encodedBytes }

func (library *Library) recordingPath(id string) string {
	recording, exists := library.document.Recordings[id]
	if !exists {
		panic("unknown recording in validated library: " + id)
	}
	if recording.File == nil {
		return ""
	}

	return library.prefix + *recording.File
}

func validAssetPath(name string) bool {
	return fs.ValidPath(name) && !strings.Contains(name, "\\") && path.Clean(name) == name &&
		DetectAudioFormat(name) != AudioFormatUnknown
}
