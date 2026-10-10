package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStyledMessageMockRecordsDetachedSegmentsAndPlainTranscript(t *testing.T) {
	mock := NewMockSystemCallbacks(t)
	callbacks := mock.ToSystemCallbacks()
	message := StyledMessage{
		LeadingBlank: true,
		Segments: []MessageSegment{
			{Text: "\"", Style: MessageQuotation},
			{Text: "Awake!", Style: MessageSpoken},
			{Text: "\"", Style: MessageQuotation},
		},
	}
	callbacks.Message.AppendStyledMessage(message)
	require.Len(t, mock.StyledMessages, 1)
	assert.Equal(t, message, mock.StyledMessages[0])
	message.Segments[1] = MessageSegment{Text: "changed", Style: MessageHighlight}
	assert.Equal(t, MessageSegment{Text: "Awake!", Style: MessageSpoken}, mock.StyledMessages[0].Segments[1])
	assert.Equal(t, []string{"\"Awake!\""}, mock.Messages)
	assert.Equal(t, 1, mock.MessageCallCount)
	mock.Reset()
	assert.Empty(t, mock.StyledMessages)
	assert.Empty(t, mock.Messages)
	assert.Zero(t, mock.MessageCallCount)
}

func TestMessageCallbacksRequireEveryMandatoryCapability(t *testing.T) {
	for name, clear := range map[string]func(*MessageCallbacks){
		"AddRowStr":             func(callbacks *MessageCallbacks) { callbacks.AddRowStr = nil },
		"AppendToCurrentRowStr": func(callbacks *MessageCallbacks) { callbacks.AppendToCurrentRowStr = nil },
		"ShowCommandPrompt":     func(callbacks *MessageCallbacks) { callbacks.ShowCommandPrompt = nil },
		"AppendStyledMessage":   func(callbacks *MessageCallbacks) { callbacks.AppendStyledMessage = nil },
	} {
		t.Run(name, func(t *testing.T) {
			valid := NewMockSystemCallbacks(t).ToSystemCallbacks()
			clear(&valid.Message)
			assert.PanicsWithValue(t, "MessageCallbacks."+name+" is required - cannot be nil", func() {
				NewMessageCallbacks(valid.Message)
			})
			if name == "AddRowStr" || name == "AppendToCurrentRowStr" {
				_, err := NewSystemCallbacks(valid.Message, valid.Visual, valid.Audio, valid.Screen,
					valid.Flow, valid.Talk, valid.UI, valid.Combat)
				require.ErrorContains(t, err, "MessageCallbacks."+name+" is required")

				return
			}
			assert.PanicsWithValue(t, "MessageCallbacks."+name+" is required - cannot be nil", func() {
				_, _ = NewSystemCallbacks(valid.Message, valid.Visual, valid.Audio, valid.Screen,
					valid.Flow, valid.Talk, valid.UI, valid.Combat)
			})
		})
	}
}
