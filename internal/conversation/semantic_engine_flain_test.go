package conversation

import (
	"testing"

	conversationpatchdata "github.com/bradhannah/UltimaVRedux/data/conversation-patches"
	"github.com/bradhannah/UltimaVRedux/internal/conversation/catalog"
	"github.com/bradhannah/UltimaVRedux/internal/conversation/extractor"
	"github.com/bradhannah/UltimaVRedux/internal/conversation/semantic"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"
	"github.com/stretchr/testify/require"
)

func TestOriginalFlainDiscoveryPreservesEveryPresentedByteAndControl(t *testing.T) {
	directory := originaldata.Directory(t)
	imported, err := extractor.Import(directory)
	require.NoError(t, err)
	original := originalSemanticDefinition(t, imported, "original.towne.038")
	catalogue, err := catalog.LoadOriginal(directory, conversationpatchdata.Files)
	require.NoError(t, err)
	program, found := catalogue.Original(semantic.OriginalSourceTowne, 38)
	require.True(t, found)
	patched, err := program.Definition()
	require.NoError(t, err)
	for _, test := range []struct {
		name   string
		inputs []string
		learns bool
	}{
		{name: "Malifora", inputs: []string{"OPPR", "", "JUDG", "Y", "MALI", ""}, learns: true},
		{name: "Annon", inputs: []string{"OPPR", "", "DRYD", "Y", "ANNO", ""}, learns: true},
		{name: "Goeth", inputs: []string{"OPPR", "", "JUDG", "Y", "GOET", ""}, learns: true},
		{name: "Felespar", inputs: []string{"OPPR", "", "JUDG", "Y", "FELE", ""}, learns: true},
		{name: "Fiona", inputs: []string{"OPPR", "", "JUDG", "Y", "FION", ""}, learns: true},
		{name: "Sindar", inputs: []string{"OPPR", "", "JUDG", "Y", "SIND", ""}, learns: true},
		{name: "guessed password", inputs: []string{"IMPERA"}},
		{name: "wrong referral", inputs: []string{"OPPR", "", "Nobody"}},
		{name: "refusal", inputs: []string{"OPPR", "", "JUDG", "N"}},
		{name: "Hassad already known", inputs: []string{"OPPR", "", "JUDG", "Y", "HASS"}},
		{name: "wrong council name", inputs: []string{"OPPR", "", "JUDG", "Y", "Nobody"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			callbacks := &semanticTestCallbacks{avatarName: "Avatar"}
			before, beforeErr := NewSemanticConversationEngine(original, &semanticTestCallbacks{avatarName: "Avatar"}, false)
			require.NoError(t, beforeErr)
			after, afterErr := NewSemanticConversationEngine(patched, callbacks, false)
			require.NoError(t, afterErr)
			require.Equal(t, "Flain", after.ExtractNPCName())
			require.Equal(t, before.Start(4), after.Start(4))
			for index, input := range test.inputs {
				require.Empty(t, callbacks.discoveries, "no discovery before the final password-bearing response")
				want, got := before.ProcessInput(input), after.ProcessInput(input)
				require.NoError(t, got.Error)
				require.Equal(t, want, got, "step %d", index)
			}
			if test.learns {
				require.Equal(t, []semantic.KnowledgeID{semantic.KnowledgeJournalOppressionPassword}, callbacks.discoveries)
			} else {
				require.Empty(t, callbacks.discoveries)
			}
		})
	}
}
