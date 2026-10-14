package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestShadowlordSummonPresentationUsesAuthoritativeTargetOnce(t *testing.T) {
	scene, callbacks := newShardWorkflowTestScene(t)
	prepareShadowlordSummonPresentationTest(scene)
	stopped := trackStoppedSounds(scene)
	rngCallsBefore := scene.gameState.GetRNG().Calls()

	require.True(t, scene.gameState.ActionYellSmallMap("FAULINEI"))
	require.True(t, scene.gameState.EventQueueManager.HasFeedback("shadowlord_summon"))
	scene.gameState.EventQueueManager.UpdateFeedback()

	assert.Equal(t, []game_state.SoundEffect{game_state.SoundShadowlordSummon}, callbacks.SoundEffectsPlayed)
	assert.Equal(t, []string{"A shadowlord appears!"}, callbacks.Messages)
	assert.NotContains(t, strings.ToLower(strings.Join(callbacks.Messages, "\n")), "an air of")
	require.NotNil(t, scene.shadowlordSummonPresentation)
	assert.Equal(t, (references.Position{X: 15, Y: 7}), scene.shadowlordSummonPresentation.position)
	require.Len(t, scene.activeTileDissolves, 1)
	assert.Equal(t, (references.Position{X: 15, Y: 7}), scene.activeTileDissolves[0].Position)
	assert.Equal(t, rngCallsBefore, scene.gameState.GetRNG().Calls(), "cosmetic fizz must not consume gameplay RNG")

	drainShadowlordSummonFeedback(t, scene)

	assert.False(t, scene.gameState.EventQueueManager.HasFeedback("shadowlord_summon"))
	assert.Nil(t, scene.shadowlordSummonPresentation)
	assert.Empty(t, scene.activeTileDissolves)
	assert.Contains(t, *stopped, game_state.SoundStopRequest{Effect: game_state.SoundShadowlordSummon})
	assert.Equal(t, rngCallsBefore, scene.gameState.GetRNG().Calls())
}

func TestShadowlordSummonPresentationFailureAndStaleStateStaySilent(t *testing.T) {
	t.Run("failed Yell", func(t *testing.T) {
		scene, callbacks := newShardWorkflowTestScene(t)
		prepareShadowlordSummonPresentationTest(scene)

		assert.False(t, scene.gameState.ActionYellSmallMap("RANDOM"))
		assert.False(t, scene.gameState.EventQueueManager.HasFeedback("shadowlord_summon"))
		assert.NotContains(t, callbacks.SoundEffectsPlayed, game_state.SoundShadowlordSummon)
		assert.Empty(t, scene.activeTileDissolves)
	})

	t.Run("location changed before feedback", func(t *testing.T) {
		scene, callbacks := newShardWorkflowTestScene(t)
		prepareShadowlordSummonPresentationTest(scene)

		require.True(t, scene.gameState.ActionYellSmallMap("FAULINEI"))
		scene.gameState.MapState.PlayerLocation.Location = references.Britain
		drainShadowlordSummonFeedback(t, scene)

		assert.NotContains(t, callbacks.SoundEffectsPlayed, game_state.SoundShadowlordSummon)
		assert.Empty(t, callbacks.Messages)
		assert.Nil(t, scene.shadowlordSummonPresentation)
		assert.Empty(t, scene.activeTileDissolves)
	})
}

func TestShadowlordSummonPresentationSceneDisposalCleansEffectAndAudio(t *testing.T) {
	scene, _ := newShardWorkflowTestScene(t)
	prepareShadowlordSummonPresentationTest(scene)
	stopped := trackStoppedSounds(scene)

	require.True(t, scene.gameState.ActionYellSmallMap("FAULINEI"))
	scene.gameState.EventQueueManager.UpdateFeedback()
	require.NotNil(t, scene.shadowlordSummonPresentation)
	require.NotEmpty(t, scene.activeTileDissolves)

	scene.inputBlocked = true
	scene.Dispose()

	assert.Nil(t, scene.shadowlordSummonPresentation)
	assert.Empty(t, scene.activeTileDissolves)
	assert.False(t, scene.inputBlocked)
	assert.Contains(t, *stopped, game_state.SoundStopRequest{Effect: game_state.SoundShadowlordSummon})
}

func TestShadowlordPresencePresentationRemainsSeparateAndCleansLifecycle(t *testing.T) {
	scene, callbacks := newShardWorkflowTestScene(t)
	pressed := false
	scene.gameState.SystemCallbacks.Visual.IsAnyKeyPressed = func() bool { return pressed }
	stopped := trackStoppedSounds(scene)

	scene.gameState.EventQueueManager.Workflow("shadowlord_presence_test").
		Priority(event_queue.PriorityNormal).
		Sequential().BlockInput().
		First(func() {}).
		ThenSequence(scene.ThenShadowlordPresenceEffect([]references.Shadowlord{
			references.Faulinei,
			references.Astaroth,
		})).
		Start()

	for range 30 {
		scene.gameState.ElapsedMs += 250
		scene.gameState.EventQueueManager.Update()
		if scene.shadowlordPresenceDialog != nil {
			break
		}
	}
	require.NotNil(t, scene.shadowlordPresenceDialog)
	assert.True(t, callbacks.InputBlocked)
	assert.Equal(t, []game_state.SoundEffect{game_state.SoundShadowlordPresence}, callbacks.SoundEffectsPlayed)
	assert.NotContains(t, callbacks.SoundEffectsPlayed, game_state.SoundShadowlordSummon)
	assert.True(t, scene.dialogStack.HasModalDialog())

	scene.gameState.EventQueueManager.Update()
	pressed = true
	scene.gameState.EventQueueManager.Update()
	pressed = false
	for range 20 {
		scene.gameState.ElapsedMs += 100
		scene.gameState.EventQueueManager.Update()
		if !scene.gameState.EventQueueManager.HasWorkflow("shadowlord_presence_test") {
			break
		}
	}

	assert.False(t, scene.gameState.EventQueueManager.HasWorkflow("shadowlord_presence_test"))
	assert.False(t, callbacks.InputBlocked)
	assert.False(t, scene.dialogStack.HasModalDialog())
	assert.Nil(t, scene.shadowlordPresenceDialog)
	assert.False(t, scene.shadowlordPresenceActive)
	require.NotEmpty(t, callbacks.ScreenFlashIntensities)
	assert.Zero(t, callbacks.ScreenFlashIntensities[len(callbacks.ScreenFlashIntensities)-1])
	assert.Contains(t, *stopped, game_state.SoundStopRequest{Effect: game_state.SoundShadowlordPresence})
}

func prepareShadowlordSummonPresentationTest(scene *GameScene) {
	scene.spriteSheet = sharedPresentationTestSpriteSheet()
	scene.gameState.ShadowlordState.Reset()
	scene.gameState.CurrentNPCAIController.GetNpcs().ReplaceWith(nil)
	scene.gameState.SystemCallbacks.Flow.Shadowlord.TriggerShadowlordSummoned = scene.TriggerShadowlordSummoned
}

func trackStoppedSounds(scene *GameScene) *[]game_state.SoundStopRequest {
	stopped := make([]game_state.SoundStopRequest, 0)
	audio := scene.gameState.SystemCallbacks.Audio
	scene.gameState.SystemCallbacks.Audio = game_state.NewAudioCallbacks(
		audio.RequestPlayback,
		func(request game_state.SoundStopRequest) {
			stopped = append(stopped, request)
			audio.RequestStop(request)
		},
		audio.UpdateLocationMusic,
		audio.PauseBackgroundMusic,
		audio.ResumeBackgroundMusic,
		audio.RequestVoice,
	)

	return &stopped
}

func drainShadowlordSummonFeedback(t *testing.T, scene *GameScene) {
	t.Helper()
	for range 10 {
		scene.gameState.ElapsedMs += 250
		scene.gameState.EventQueueManager.UpdateFeedback()
		if !scene.gameState.EventQueueManager.HasFeedback("shadowlord_summon") {
			return
		}
	}
	t.Fatal("Shadowlord summon feedback did not complete")
}
