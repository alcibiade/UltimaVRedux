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
// binary; window icons do not exist on macOS. Ebitengine turns the process into
// a Dock application only after launching finishes, and the Dock builds that
// tile from the default icon, so an icon set at startup alone is not shown. The
// icon is therefore applied again whenever the application becomes active.
func setDockIcon(atlasPNG []byte) error {
	encoded, err := appicon.DockIconPNG(atlasPNG)
	if err != nil {
		return fmt.Errorf("rendering Dock icon: %w", err)
	}
	alloc := objc.RegisterName("alloc")
	release := objc.RegisterName("release")
	pool := objc.ID(objc.GetClass("NSAutoreleasePool")).Send(alloc).Send(objc.RegisterName("init"))
	defer pool.Send(objc.RegisterName("drain"))

	data := objc.ID(objc.GetClass("NSData")).Send(alloc).Send(
		objc.RegisterName("initWithBytes:length:"),
		unsafe.Pointer(&encoded[0]), //nolint:gosec // AppKit copies the bytes during this call.
		uint64(len(encoded)),
	)
	if data == 0 {
		return fmt.Errorf("%w: AppKit could not copy the image data", errDockIcon)
	}
	defer data.Send(release)
	// The image is owned for the rest of the process; every activation reuses it.
	image := objc.ID(objc.GetClass("NSImage")).Send(alloc).Send(objc.RegisterName("initWithData:"), data)
	if image == 0 {
		return fmt.Errorf("%w: AppKit rejected the image", errDockIcon)
	}

	application := objc.ID(objc.GetClass("NSApplication")).Send(objc.RegisterName("sharedApplication"))
	setIcon := objc.RegisterName("setApplicationIconImage:")
	application.Send(objc.RegisterName("performSelectorOnMainThread:withObject:waitUntilDone:"), setIcon, image, false)

	notification := []byte("NSApplicationDidBecomeActiveNotification\x00")
	name := objc.ID(objc.GetClass("NSString")).Send(
		objc.RegisterName("stringWithUTF8String:"),
		unsafe.Pointer(&notification[0]), //nolint:gosec // NUL-terminated; AppKit copies the string.
	)
	center := objc.ID(objc.GetClass("NSNotificationCenter")).Send(objc.RegisterName("defaultCenter"))
	mainQueue := objc.ID(objc.GetClass("NSOperationQueue")).Send(objc.RegisterName("mainQueue"))
	// The notification center keeps the block for the rest of the process and
	// runs it on the main queue, where AppKit calls are safe.
	activated := objc.NewBlock(func(_ objc.Block, _ objc.ID) {
		application.Send(setIcon, image)
	})
	center.Send(objc.RegisterName("addObserverForName:object:queue:usingBlock:"), name, objc.ID(0), mainQueue, activated)

	return nil
}
