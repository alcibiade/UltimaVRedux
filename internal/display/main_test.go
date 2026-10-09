package display

import (
	"os"
	"testing"
)

func TestMain(tests *testing.M) {
	// Ebitengine 2.10 defers native window-system initialization until the
	// first monitor/window query. Cocoa requires that query on the main thread,
	// just as main.go initializes the display before starting the game loop.
	GetManager()
	os.Exit(tests.Run())
}
