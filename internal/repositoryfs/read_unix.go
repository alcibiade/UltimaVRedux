//go:build aix || android || darwin || dragonfly || freebsd || illumos || ios || linux || netbsd || openbsd || solaris

package repositoryfs

import (
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/sys/unix"
)

// ReadRegularFile opens without following the final path component and reads
// from that verified descriptor, eliminating the Lstat/open swap window.
//
//nolint:nonamedreturns // The deferred close must preserve any read/stat error and reject a failed close.
func ReadRegularFile(path string) (data []byte, err error) {
	descriptor, err := unix.Open(path, unix.O_RDONLY|unix.O_NONBLOCK|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		if errors.Is(err, unix.ELOOP) {
			return nil, ErrNotRegular
		}

		return nil, &os.PathError{Op: "open", Path: path, Err: err}
	}
	file := os.NewFile(uintptr(descriptor), path)
	if file == nil {
		return nil, errors.Join(fmt.Errorf("open %s: invalid descriptor", path), unix.Close(descriptor))
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			data = nil
			err = errors.Join(err, fmt.Errorf("close %s: %w", path, closeErr))
		}
	}()
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return nil, ErrNotRegular
	}

	data, err = io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	return data, nil
}
