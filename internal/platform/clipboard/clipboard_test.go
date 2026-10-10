package clipboard_test

import (
	"runtime"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/platform/clipboard"
)

func TestDirectAPI(t *testing.T) {
	// Match the consumer's callback contract without reading a live clipboard.
	var read func() (string, error) = clipboard.ReadText
	if read == nil {
		t.Fatal("missing ReadText callback")
	}
	want := runtime.GOOS == "darwin" || runtime.GOOS == "windows"
	if clipboard.Supported() != want {
		t.Fatalf("Supported() = %v, want %v", clipboard.Supported(), want)
	}
}
