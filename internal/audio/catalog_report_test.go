package audio

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGuideIsTableFirstWithStableSlotsAndGameplayInstructions(t *testing.T) {
	catalog, err := BuildCatalog(false)
	require.NoError(t, err)
	require.NoError(t, catalog.AttachSourceEvidence("../.."))
	guide := catalog.GenerationGuide()
	require.Less(t, strings.Index(guide, "| Cue / moment |"), strings.Index(guide, "## Recording briefs"))
	for _, moment := range catalog.Moments {
		require.Contains(t, guide, "<code>"+moment.ID+"</code>")
	}
	require.Contains(t, guide, "How to hear it in-game")
	require.Contains(t, guide, "magic_failure_fizzle_01")
	require.Contains(t, guide, "file: null")
	require.Contains(t, guide, "Not currently audible.")
	require.Contains(t, guide, "internal/audio/assets/library.json")
	require.Contains(t, guide, "tavern_lute_01", "the deferred tavern recording remains a named empty slot")
	// These game-owned empty slots have real triggers and authored recording briefs.
	for _, anchor := range []string{"brief-sfx-soundringvanish", "brief-ambient-ambientbard"} {
		require.Contains(t, guide, "[Brief](#"+anchor+")")
		require.Contains(t, guide, "<a id=\""+anchor+"\"></a>")
	}
	require.NotContains(t, guide, "<a id=\"brief-sfx-soundheal\"></a>", "assigned healing needs no missing-recording brief")
}
