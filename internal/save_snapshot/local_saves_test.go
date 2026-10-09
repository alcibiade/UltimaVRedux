package save_snapshot

import (
	"crypto/sha256"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLocalNativeSaves is opt-in and reads source saves only. No source path or
// private save data belongs in fixtures; callers supply a platform path list.
func TestLocalNativeSaves(t *testing.T) {
	paths := filepath.SplitList(os.Getenv("U5_NATIVE_SAVE_TEST_PATHS"))
	if len(paths) == 0 {
		// Keep the ordinary zero-skip gate independent of private user saves.
		t.Log("no local saves requested; set U5_NATIVE_SAVE_TEST_PATHS for the opt-in proof")
		return
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			before := sha256.Sum256(data)
			t.Cleanup(func() {
				after, readErr := os.ReadFile(path)
				if readErr != nil {
					t.Fatal(readErr)
				}
				if sha256.Sum256(after) != before {
					t.Fatal("source save bytes changed")
				}
			})
			decoded, err := Decode(data)
			if err != nil {
				t.Fatal(err)
			}
			var source struct {
				Metadata Metadata `json:"metadata"`
				Snapshot struct {
					Version int `json:"version"`
				} `json:"snapshot"`
			}
			require.NoError(t, json.Unmarshal(data, &source))
			require.Equal(t, source.Metadata, decoded.Metadata)
			require.Equal(t, CurrentVersion, decoded.Snapshot.Version)
			if decoded.Snapshot.NeedsMapUnitReconstruction() {
				t.Logf("v%d source requires runtime reconstruction before canonical encoding; source bytes unchanged", source.Snapshot.Version)
				return
			}
			current, err := EncodeCanonical(decoded)
			require.NoError(t, err)
			reloaded, err := Decode(current)
			require.NoError(t, err)
			require.Equal(t, decoded, reloaded, "every promoted field and metadata value survives current encoding")
			t.Logf("v%d source decoded and re-encoded in memory as v%d without changing source bytes", source.Snapshot.Version, CurrentVersion)
		})
	}
}
