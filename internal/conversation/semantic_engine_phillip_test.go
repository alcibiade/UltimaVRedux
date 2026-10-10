package conversation

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/conversation/extractor"
	"github.com/bradhannah/UltimaVRedux/internal/conversation/semantic"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"
	"github.com/stretchr/testify/require"
)

func TestSemanticAskNameReusesProtocolQuote(t *testing.T) {
	for _, preceding := range []semantic.Body{
		{},
		{&semantic.SayAction{Content: semanticText("Hello. ")}},
	} {
		definition := semanticTestConversation()
		definition.Farewell = append(preceding, &semantic.AskNameAction{})
		callbacks := &semanticTestCallbacks{avatarName: "Avatar"}
		engine, err := NewSemanticConversationEngine(definition, callbacks, false)
		require.NoError(t, err)
		engine.Start(19)
		response := engine.ProcessInput("bye")
		require.NoError(t, response.Error)
		require.NotContains(t, response.Output, `""`)
		require.Contains(t, response.Output, `What is thy name?"`)
		require.True(t, response.IsLabelQuestion)
		require.Empty(t, callbacks.metActorIDs)
		response = engine.ProcessInput("Avatar")
		require.NoError(t, response.Error)
		require.Equal(t, []int{19}, callbacks.metActorIDs)
	}
}

func TestOriginalPhillipFarewellNamePrompt(t *testing.T) {
	imported, err := extractor.Import(originaldata.Directory(t))
	require.NoError(t, err)
	definition := originalSemanticDefinition(t, imported, "original.castle.019")
	callbacks := &semanticTestCallbacks{avatarName: "Avatar"}
	engine, err := NewSemanticConversationEngine(definition, callbacks, false)
	require.NoError(t, err)
	engine.Start(7)
	response := engine.ProcessInput("bye")
	require.Equal(t, "\n\n\"What is thy name?\"", response.Output)
	assertSemanticResponseMirror(t, response)
	response = engine.ProcessInput("Avatar")
	require.NoError(t, response.Error)
	require.True(t, response.IsCompleteAwaitingDismissal)
	require.Equal(t, []int{7}, callbacks.metActorIDs)
}
