package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFlowCallbacksValidatesEachCapabilityGroup(t *testing.T) {
	valid := NewMockSystemCallbacks(t).ToSystemCallbacks().Flow
	tests := []struct {
		name      string
		remove    func(*FlowCallbacks)
		wantPanic string
	}{
		{
			name: "consequences",
			remove: func(callbacks *FlowCallbacks) {
				callbacks.Consequences.TriggerDeathSequence = nil
			},
			wantPanic: "FlowCallbacks.Consequences.TriggerDeathSequence is required - cannot be nil",
		},
		{
			name: "rest",
			remove: func(callbacks *FlowCallbacks) {
				callbacks.Rest.RequestCampDuration = nil
			},
			wantPanic: "FlowCallbacks.Rest.RequestCampDuration is required - cannot be nil",
		},
		{
			name: "world interaction",
			remove: func(callbacks *FlowCallbacks) {
				callbacks.WorldInteraction.ShowWellInteraction = nil
			},
			wantPanic: "FlowCallbacks.WorldInteraction.ShowWellInteraction is required - cannot be nil",
		},
		{
			name: "observation",
			remove: func(callbacks *FlowCallbacks) {
				callbacks.Observation.ShowGemViewInteraction = nil
			},
			wantPanic: "FlowCallbacks.Observation.ShowGemViewInteraction is required - cannot be nil",
		},
		{
			name: "Shadowlord",
			remove: func(callbacks *FlowCallbacks) {
				callbacks.Shadowlord.ShowShardUsageWorkflow = nil
			},
			wantPanic: "FlowCallbacks.Shadowlord.ShowShardUsageWorkflow is required - cannot be nil",
		},
		{
			name: "items",
			remove: func(callbacks *FlowCallbacks) {
				callbacks.Items.StartPotionWorkflow = nil
			},
			wantPanic: "FlowCallbacks.Items.StartPotionWorkflow is required - cannot be nil",
		},
		{
			name: "sacred sites",
			remove: func(callbacks *FlowCallbacks) {
				callbacks.SacredSites.StartCodexWorkflow = nil
			},
			wantPanic: "FlowCallbacks.SacredSites.StartCodexWorkflow is required - cannot be nil",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			callbacks := valid
			test.remove(&callbacks)

			assert.PanicsWithValue(t, test.wantPanic, func() {
				NewFlowCallbacks(callbacks)
			})
		})
	}
}

func TestNewSystemCallbacksRevalidatesNamedCapabilities(t *testing.T) {
	valid := NewMockSystemCallbacks(t).ToSystemCallbacks()
	flow := valid.Flow
	flow.Items.StartSceptreWorkflow = nil

	assert.PanicsWithValue(t,
		"FlowCallbacks.Items.StartSceptreWorkflow is required - cannot be nil",
		func() {
			_, _ = NewSystemCallbacks(
				valid.Message,
				valid.Visual,
				valid.Audio,
				valid.Screen,
				flow,
				valid.Talk,
				valid.UI,
				valid.Combat,
			)
		},
	)
}

func TestNewSystemCallbacksRequiresJournalChanged(t *testing.T) {
	valid := NewMockSystemCallbacks(t).ToSystemCallbacks()
	valid.UI.JournalChanged = nil
	_, err := NewSystemCallbacks(valid.Message, valid.Visual, valid.Audio, valid.Screen,
		valid.Flow, valid.Talk, valid.UI, valid.Combat)
	require.ErrorIs(t, err, errJournalChangedCallbackRequired)
}

func TestAudioCallbacksRequireEveryCapability(t *testing.T) {
	valid := NewMockSystemCallbacks(t).ToSystemCallbacks()
	tests := []struct {
		name        string
		remove      func(*AudioCallbacks)
		constructor bool
	}{
		{"Play", func(audio *AudioCallbacks) { audio.Play = nil }, false},
		{"Stop", func(audio *AudioCallbacks) { audio.Stop = nil }, false},
		{"PlaySoundEffect", func(audio *AudioCallbacks) { audio.PlaySoundEffect = nil }, false},
		{"RequestPlayback", func(audio *AudioCallbacks) { audio.RequestPlayback = nil }, true},
		{"RequestVoice", func(audio *AudioCallbacks) { audio.RequestVoice = nil }, true},
		{"RequestStop", func(audio *AudioCallbacks) { audio.RequestStop = nil }, true},
		{"UpdateLocationMusic", func(audio *AudioCallbacks) { audio.UpdateLocationMusic = nil }, true},
		{"PauseBackgroundMusic", func(audio *AudioCallbacks) { audio.PauseBackgroundMusic = nil }, true},
		{"ResumeBackgroundMusic", func(audio *AudioCallbacks) { audio.ResumeBackgroundMusic = nil }, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			audio := valid.Audio
			test.remove(&audio)
			wantPanic := "AudioCallbacks." + test.name + " is required - cannot be nil"
			assert.PanicsWithValue(t, wantPanic, func() {
				_, _ = NewSystemCallbacks(valid.Message, valid.Visual, audio, valid.Screen,
					valid.Flow, valid.Talk, valid.UI, valid.Combat)
			})
			if test.constructor {
				assert.PanicsWithValue(t, wantPanic, func() {
					NewAudioCallbacks(audio.RequestPlayback, audio.RequestStop, audio.UpdateLocationMusic,
						audio.PauseBackgroundMusic, audio.ResumeBackgroundMusic, audio.RequestVoice)
				})
			}
		})
	}
}

func TestNewSystemCallbacksPreservesSuppliedAudioCapabilities(t *testing.T) {
	valid := NewMockSystemCallbacks(t).ToSystemCallbacks()
	audio := valid.Audio
	var calls []string
	audio.Play = func(effect SoundEffect) *SoundBuilder {
		calls = append(calls, "Play")
		return valid.Audio.Play(effect)
	}
	audio.Stop = func(effect SoundEffect) *StopBuilder {
		calls = append(calls, "Stop")
		return valid.Audio.Stop(effect)
	}
	audio.PlaySoundEffect = func(SoundEffect) { calls = append(calls, "PlaySoundEffect") }
	audio.RequestPlayback = func(SoundPlaybackRequest) { calls = append(calls, "RequestPlayback") }
	audio.RequestVoice = func(SoundPlaybackRequest) SoundVoice {
		calls = append(calls, "RequestVoice")
		return SilentSoundVoice{}
	}
	audio.RequestStop = func(SoundStopRequest) { calls = append(calls, "RequestStop") }
	audio.UpdateLocationMusic = func() { calls = append(calls, "UpdateLocationMusic") }
	audio.PauseBackgroundMusic = func() { calls = append(calls, "PauseBackgroundMusic") }
	audio.ResumeBackgroundMusic = func() { calls = append(calls, "ResumeBackgroundMusic") }

	callbacks, err := NewSystemCallbacks(valid.Message, valid.Visual, audio, valid.Screen,
		valid.Flow, valid.Talk, valid.UI, valid.Combat)
	require.NoError(t, err)
	require.NotNil(t, callbacks)
	assert.Empty(t, calls, "validation must not invoke capabilities")
	require.NotNil(t, callbacks.Audio.Play(SoundOpenDoor))
	require.NotNil(t, callbacks.Audio.Stop(SoundOpenDoor))
	callbacks.Audio.PlaySoundEffect(SoundOpenDoor)
	callbacks.Audio.RequestPlayback(SoundPlaybackRequest{Effect: SoundOpenDoor})
	callbacks.Audio.RequestVoice(SoundPlaybackRequest{Effect: SoundOpenDoor}).Stop(0)
	callbacks.Audio.RequestStop(SoundStopRequest{Effect: SoundOpenDoor})
	callbacks.Audio.UpdateLocationMusic()
	callbacks.Audio.PauseBackgroundMusic()
	callbacks.Audio.ResumeBackgroundMusic()
	assert.Equal(t, []string{"Play", "Stop", "PlaySoundEffect", "RequestPlayback", "RequestVoice", "RequestStop",
		"UpdateLocationMusic", "PauseBackgroundMusic", "ResumeBackgroundMusic"}, calls)
}

func TestNewSystemCallbacksRequiresTransientIconCallback(t *testing.T) {
	valid := NewMockSystemCallbacks(t).ToSystemCallbacks()
	valid.UI.ShowTransientIcon = nil

	_, err := NewSystemCallbacks(
		valid.Message,
		valid.Visual,
		valid.Audio,
		valid.Screen,
		valid.Flow,
		valid.Talk,
		valid.UI,
		valid.Combat,
	)

	assert.ErrorIs(t, err, errTransientIconCallbackRequired)
}

func TestNamedConstructorsRejectIncompleteLateWiredCapabilities(t *testing.T) {
	valid := NewMockSystemCallbacks(t).ToSystemCallbacks()

	t.Run("visual projectile", func(t *testing.T) {
		callbacks := valid.Visual
		callbacks.StartProjectile = nil

		assert.PanicsWithValue(t,
			"VisualCallbacks.StartProjectile is required - cannot be nil",
			func() { NewVisualCallbacks(callbacks) },
		)
	})

	t.Run("screen dialog", func(t *testing.T) {
		callbacks := valid.Screen
		callbacks.CreateAndPushMessageDialog = nil

		assert.PanicsWithValue(t,
			"ScreenCallbacks.CreateAndPushMessageDialog is required - cannot be nil",
			func() { NewScreenCallbacks(callbacks) },
		)
	})

	t.Run("message rich output", func(t *testing.T) {
		callbacks := valid.Message
		callbacks.AppendStyledMessage = nil

		assert.PanicsWithValue(t,
			"MessageCallbacks.AppendStyledMessage is required - cannot be nil",
			func() { NewMessageCallbacks(callbacks) },
		)
	})

	t.Run("typed dialog creation", func(t *testing.T) {
		var createDialog func() = valid.Screen.CreateAndPushMessageDialog
		require.NotNil(t, createDialog)
	})
	t.Run("camping result dialog", func(t *testing.T) {
		callbacks := valid.Screen
		callbacks.CreateAndPushCampingResultDialog = nil
		assert.PanicsWithValue(t, "ScreenCallbacks.CreateAndPushCampingResultDialog is required - cannot be nil",
			func() { NewScreenCallbacks(callbacks) })
	})
}
