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

func TestOriginalChamfortDiscoveriesPreserveEveryPresentedByte(t *testing.T) {
	directory := originaldata.Directory(t)
	imported, err := extractor.Import(directory)
	require.NoError(t, err)
	original := originalSemanticDefinition(t, imported, "original.towne.017")
	catalogue, err := catalog.LoadOriginal(directory, conversationpatchdata.Files)
	require.NoError(t, err)
	program, found := catalogue.Original(semantic.OriginalSourceTowne, 17)
	require.True(t, found)
	patched, err := program.Definition()
	require.NoError(t, err)
	callbacks := &semanticTestCallbacks{avatarName: "Avatar"}
	before, err := NewSemanticConversationEngine(original, &semanticTestCallbacks{avatarName: "Avatar"}, false)
	require.NoError(t, err)
	after, err := NewSemanticConversationEngine(patched, callbacks, false)
	require.NoError(t, err)
	require.Equal(t, before.Start(4), after.Start(4))
	for _, input := range []string{"NAME", "RESI", "TERR", "Y", "Avatar", "MANT", "JERE", "BYE"} {
		want, got := before.ProcessInput(input), after.ProcessInput(input)
		require.NoError(t, want.Error)
		require.Equal(t, want, got, input)
	}
	require.Equal(t, []semantic.KnowledgeID{
		semantic.KnowledgeJournalResistancePassword, semantic.KnowledgeShrineJusticeMantra,
	}, callbacks.discoveries)
}
