//go:build !darwin

package main

// setDockIcon has nothing to do outside macOS: Windows and Linux show the
// window icon, and Windows executables also embed the committed icon resource.
func setDockIcon(_ []byte) error {
	return nil
}
