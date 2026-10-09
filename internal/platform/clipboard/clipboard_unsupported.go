//go:build !windows && !darwin

package clipboard

// ReadText reports that clipboard access is unavailable in this build.
func ReadText() (string, error) {
	return "", ErrUnavailable
}

// Supported reports whether this build has native clipboard access.
func Supported() bool {
	return false
}
