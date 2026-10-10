package conversation

import (
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/conversation/semantic"
	"github.com/stretchr/testify/require"
)

type deferredSemanticTestCallbacks struct {
	semanticTestCallbacks
	boundaries []DiscoveryBoundary
	knowledge  []semantic.KnowledgeID
}

func (c *deferredSemanticTestCallbacks) QueueKnowledge(id semantic.KnowledgeID, boundary DiscoveryBoundary) error {
	c.knowledge = append(c.knowledge, id)
	c.boundaries = append(c.boundaries, boundary)
	return nil
}

func TestSemanticDiscoveryBoundaryUsesOriginalSpeechAndResponseIdentity(t *testing.T) {
	definition := semanticTestConversation()
	definition.Job = semantic.Body{
		&semantic.SayAction{Content: semanticText("Befor\u00e9")},
		&semantic.DiscoverKnowledgeAction{Knowledge: semantic.KnowledgeShadowlordFaulineiName},
		&semantic.PauseAction{},
		&semantic.SayAction{Content: semantic.RichText{&semantic.RunicPart{Text: "SU"}, &semantic.RunicPart{Text: "MM"}}},
		&semantic.DiscoverKnowledgeAction{Knowledge: semantic.KnowledgeShrineHonorMantra},
		&semantic.WaitForKeyAction{},
		&semantic.SayAction{Content: semanticText("The complete instruction.")},
		&semantic.DiscoverKnowledgeAction{Knowledge: semantic.KnowledgeShadowlordFaulineiAssociation},
		&semantic.EndAction{},
	}
	callbacks := &deferredSemanticTestCallbacks{}
	engine, err := NewDeferredSemanticConversationEngine(definition, callbacks, false)
	require.NoError(t, err)
	initial := engine.Start(1)
	response := engine.ProcessInput("JOB")
	require.NoError(t, response.Error)
	require.True(t, response.WaitingForKeypress)
	require.Greater(t, response.Serial, initial.Serial)
	require.Empty(t, callbacks.discoveries, "production capability never executes immediate discoveries")
	require.Equal(t, []DiscoveryBoundary{
		{Response: response.Serial, SourceOffset: len("\n\n\"Befor\u00e9")},
		{Response: response.Serial, SourceOffset: strings.Index(response.Output, "SUMM") + len("SUMM")},
	}, callbacks.boundaries)
	require.Contains(t, response.Output[:callbacks.boundaries[1].SourceOffset], PauseMarker)
	terminal := engine.ProcessInput("")
	require.True(t, terminal.IsCompleteAwaitingDismissal)
	require.Greater(t, terminal.Serial, response.Serial)
	require.Equal(t, DiscoveryBoundary{Response: terminal.Serial, SourceOffset: len("The complete instruction.")}, callbacks.boundaries[2])
}

func TestDeferredSemanticFactoryRejectsImmediateOnlyCallbacks(t *testing.T) {
	_, err := NewDeferredSemanticConversationEngine(semanticTestConversation(), &semanticTestCallbacks{}, false)
	require.ErrorIs(t, err, errSemanticCallbacksMissing)
}
