package conversation

import (
	"fmt"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/conversation/semantic"
	"github.com/stretchr/testify/require"
)

type paymentRefusalCallbacks struct{ semanticTestCallbacks }

func (*paymentRefusalCallbacks) DeductGoldSimple(int, string) error {
	return fmt.Errorf("payment declined: %w", ErrInsufficientPayment)
}

func TestSemanticPaymentRefusalDiscardsNestedPaidContinuation(t *testing.T) {
	for _, next := range []string{"NO", "NAME", "BYE", "JOB"} {
		t.Run(next, func(t *testing.T) {
			definition := semanticTestConversation()
			definition.Questions["question_01"] = semantic.Question{
				Prompt: semantic.Body{&semantic.SayAction{Content: semanticText("Pay?")}},
				Answers: []semantic.Answer{{Keywords: []string{"yes"}, Response: semantic.Body{
					&semantic.BranchAction{Condition: &semantic.KarmaAtLeastCondition{Minimum: 0}, Then: semantic.Body{
						&semantic.TakeGoldAction{Amount: 100},
						&semantic.GiveItemAction{Item: "food", Quantity: 1},
					}},
					&semantic.AdjustKarmaAction{Delta: 1},
					&semantic.DiscoverKnowledgeAction{Knowledge: semantic.KnowledgeSpellInXenMani},
				}}},
				Default: semantic.Body{&semantic.SayAction{Content: semanticText("Declined.")}},
			}
			callbacks := &paymentRefusalCallbacks{}
			engine, err := NewSemanticConversationEngine(definition, callbacks, false)
			require.NoError(t, err)
			engine.Start(17)
			require.True(t, engine.ProcessInput("riddle").IsLabelQuestion)
			refusal := engine.ProcessInput("YES")
			require.NoError(t, refusal.Error)
			require.True(t, refusal.NeedsInput)
			require.False(t, refusal.IsLabelQuestion)
			require.Empty(t, engine.frames)
			require.Empty(t, engine.currentQuestion)
			response := engine.ProcessInput(next)
			require.NoError(t, response.Error)
			require.Empty(t, callbacks.items)
			require.Zero(t, callbacks.karmaUp)
			require.Empty(t, callbacks.discoveries)
		})
	}
}

func TestSemanticAskNameSourceFourBytePartyMatch(t *testing.T) {
	for _, test := range []struct {
		input, avatar string
		recognized    bool
	}{
		{"Avatar", "Avatar", true}, {"Avat", "Avatar", true}, {"A", "Avatar", false},
		{"Ava", "Avatar", false}, {"Iolo", "Avatar", true}, {"Iol", "Avatar", false},
		{"I am Iolo", "Avatar", true}, {"notIolo", "Avatar", false}, {"not\tIolo", "Avatar", false},
		{"notIolo Iolo", "Avatar", false},
		{"\tIolo", "Avatar", false}, {"Shamino", "Avatar", false}, {"Ed", "Ed", true},
		{"E", "Ed", false}, {"", "Avatar", false},
	} {
		t.Run(test.input+test.avatar, func(t *testing.T) {
			definition := semanticTestConversation()
			definition.Description = semantic.Body{&semantic.AskNameAction{}}
			callbacks := &semanticTestCallbacks{avatarName: test.avatar}
			engine, err := NewSemanticConversationEngine(definition, callbacks, false)
			require.NoError(t, err)
			engine.Start(17)
			require.NoError(t, engine.ProcessInput(test.input).Error)
			require.Equal(t, test.recognized, callbacks.hasMet)
			if test.recognized {
				require.Equal(t, []int{17}, callbacks.metActorIDs)
			}
			require.Equal(t, test.avatar, engine.substitute("{avatar_name}"))
		})
	}
}
