package audio

import (
	"bytes"
	"io/fs"
	"path"
	"time"
)

// encodedLibraryFS exposes the immutable game source set without writable slices.
type encodedLibraryFS struct{ library *Library }

func (files encodedLibraryFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	data, exists := files.library.assets[name]
	if !exists {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}

	return &encodedFile{Reader: bytes.NewReader(data), name: name}, nil
}

type encodedFile struct {
	*bytes.Reader
	name string
}

func (file *encodedFile) Stat() (fs.FileInfo, error) { return file, nil }
func (*encodedFile) Close() error                    { return nil }
func (file *encodedFile) Name() string               { return path.Base(file.name) }
func (*encodedFile) Mode() fs.FileMode {
	const readOnlyFile = 0o444

	return readOnlyFile
}
func (*encodedFile) ModTime() time.Time { return time.Time{} }
func (*encodedFile) IsDir() bool        { return false }
func (*encodedFile) Sys() any           { return nil }
