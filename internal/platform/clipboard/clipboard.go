// Package clipboard provides platform-native access to clipboard text.
package clipboard

import "errors"

// ErrUnavailable indicates that clipboard access is not supported or available.
var ErrUnavailable = errors.New("clipboard unavailable")
