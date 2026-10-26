//go:build darwin

package main

import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/ebitengine/purego/objc"

	"github.com/bradhannah/UltimaVRedux/internal/appicon"
)

var errDockIcon = errors.New("dock icon")

// setDockIcon replaces the generic executable icon macOS shows for an unbundled
// binary; window icons do not exist on macOS. The change is queued on the main
// run loop so AppKit applies it after Ebitengine finishes launching the app.
func setDockIcon(atlasPNG []byte) error {
	encoded, err := appicon.DockIconPNG(atlasPNG)
	if err != nil {
		return fmt.Errorf("rendering Dock icon: %w", err)
	}
	release := objc.RegisterName("release")
	data := objc.ID(objc.GetClass("NSData")).Send(objc.RegisterName("alloc")).Send(
		objc.RegisterName("initWithBytes:length:"),
		unsafe.Pointer(&encoded[0]), //nolint:gosec // AppKit copies the bytes during this call.
		uint64(len(encoded)),
	)
	if data == 0 {
		return fmt.Errorf("%w: AppKit could not copy the image data", errDockIcon)
	}
	defer data.Send(release)
	image := objc.ID(objc.GetClass("NSImage")).Send(objc.RegisterName("alloc")).Send(
		objc.RegisterName("initWithData:"), data,
	)
	if image == 0 {
		return fmt.Errorf("%w: AppKit rejected the image", errDockIcon)
	}
	// performSelectorOnMainThread retains the image until the selector runs.
	defer image.Send(release)
	application := objc.ID(objc.GetClass("NSApplication")).Send(objc.RegisterName("sharedApplication"))
	application.Send(
		objc.RegisterName("performSelectorOnMainThread:withObject:waitUntilDone:"),
		objc.RegisterName("setApplicationIconImage:"), image, false,
	)

	return nil
}
