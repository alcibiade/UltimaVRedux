package audio

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSearchMetadataRoundTripsWithoutChangingAudioTreatment(t *testing.T) {
	editor, _ := editorFixture(t)
	original := editor.Recording("healing_01")
	recording := original
	recording.SearchTerms = []string{"warm shimmer", "gentle"}
	require.NoError(t, editor.SetRecording("healing_01", recording))
	cue := editor.Cue("sfx/SoundHeal")
	cue.SearchTerms = []string{"restorative", "recovery"}
	cue.HowToHear = "A restorative shimmer accompanies successful healing of an injured character."
	require.NoError(t, editor.SetCue("sfx/SoundHeal", cue))
	require.Contains(t, editor.SearchText("healing_01", false), "injured character")
	require.Contains(t, editor.SearchText("sfx/SoundHeal", true), "warm shimmer")
	require.Contains(t, editor.CueSearchTerms("sfx/SoundHeal"), "restoration", "profile synonyms are additive")
	require.NoError(t, editor.Save())
	reopened, err := OpenLibraryEditor(editor.Path())
	require.NoError(t, err)
	require.Equal(t, cue, reopened.Cue("sfx/SoundHeal"))
	require.Equal(t, recording, reopened.Recording("healing_01"))
	recording.SearchTerms = original.SearchTerms
	require.Equal(t, original, recording, "only descriptive metadata changed")
}

func TestSearchTermsRejectInvalidMetadata(t *testing.T) {
	for _, terms := range [][]string{{""}, {"two", "TWO"}, {" spaced "}, {"line\nbreak"}, {"one,two"}} {
		require.ErrorIs(t, validateSearchTerms(terms), ErrAudioLibrary)
	}
	require.NoError(t, validateSearchTerms([]string{"hex grate", "foley"}))
	editor, _ := editorFixture(t)
	require.Contains(t, strings.ToLower(editor.SearchText("physical_unlock_01", false)), "lockpick")
	require.Contains(t, strings.ToLower(editor.SearchText("sfx/SoundWalkMetal1", true)), "grating")
}
