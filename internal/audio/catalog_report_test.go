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
	for _, moment := range catalog.Moments {
		if needsRecordingBrief(moment) {
			require.Contains(t, guide, "[Brief](#"+briefAnchor(moment.ID)+")")
			require.Contains(t, guide, "<a id=\""+briefAnchor(moment.ID)+"\"></a>")
		}
	}
}

func TestTableEscapesAuthoredText(t *testing.T) {
	require.Equal(t, "wood \\| stone<br>&lt;quiet&gt;", tableCell("wood | stone\n<quiet>"))
}
