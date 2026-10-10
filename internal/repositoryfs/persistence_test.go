package repositoryfs

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPublishTemporaryFilePublishesContentAndCleansTemporaryPath(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "published.json")
	data := []byte("published content")
	var temporaryPath string

	err := PublishTemporaryFile(root, ".publication-*.tmp", data, 0o600, func(path string) error {
		temporaryPath = path
		stored, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if string(stored) != string(data) {
			t.Fatalf("temporary content = %q, want %q", stored, data)
		}
		if runtime.GOOS != "windows" {
			info, statErr := os.Stat(path)
			if statErr != nil {
				return statErr
			}
			if got := info.Mode().Perm(); got != 0o600 {
				t.Fatalf("temporary mode = %o, want 600", got)
			}
		}

		return os.Link(path, destination)
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(temporaryPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temporary path remains after publication: %v", err)
	}
	stored, err := os.ReadFile(destination)
	if err != nil || string(stored) != string(data) {
		t.Fatalf("published content = %q, %v", stored, err)
	}
}

func TestPublishTemporaryFileCleansUpAfterPublicationFailure(t *testing.T) {
	root := t.TempDir()
	publishErr := errors.New("injected publication failure")
	var temporaryPath string

	err := PublishTemporaryFile(root, ".publication-*.tmp", []byte("content"), 0o600, func(path string) error {
		temporaryPath = path

		return publishErr
	})
	if !errors.Is(err, publishErr) {
		t.Fatalf("PublishTemporaryFile() error = %v, want injected failure", err)
	}
	if _, err = os.Stat(temporaryPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temporary path remains after failure: %v", err)
	}
}

func TestContentRevisionUsesLowercaseSHA256(t *testing.T) {
	const want = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if got := ContentRevision([]byte("abc")); got != want {
		t.Fatalf("ContentRevision() = %q, want %q", got, want)
	}
}

func TestPublicationFailureRetainsCleanupFailure(t *testing.T) {
	publishErr := errors.New("publication failed")
	err := PublishTemporaryFile(t.TempDir(), ".publication-*", []byte("content"), 0o600, func(path string) error {
		// A nonempty directory cannot be removed with os.Remove. This creates
		// a deterministic cleanup failure without relying on user permissions.
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "child"), nil, 0o600); err != nil {
			t.Fatal(err)
		}
		return publishErr
	})
	if !errors.Is(err, publishErr) || !strings.Contains(err.Error(), "remove temporary file") {
		t.Fatalf("expected publication and cleanup failures, got %v", err)
	}
}

func TestDecodeStrictJSONAcceptsOneDocumentAndRejectsUnknownOrTrailingData(t *testing.T) {
	type document struct {
		Name string `json:"name"`
	}
	var decoded document
	if err := DecodeStrictJSON([]byte(`{"name":"valid"}`), &decoded); err != nil || decoded.Name != "valid" {
		t.Fatalf("DecodeStrictJSON(valid) = %#v, %v", decoded, err)
	}
	for _, data := range []string{
		`{"name":"valid","unknown":true}`,
		`{"name":"valid"} []`,
		`{"name":"valid"} {`,
	} {
		if err := DecodeStrictJSON([]byte(data), &document{}); err == nil {
			t.Fatalf("DecodeStrictJSON(%q) error = nil", data)
		}
	}
}
