package main

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/minimap"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

func TestPalaceGuardRoutesDirectlyToBlackthornInterrogation(t *testing.T) {
	scene, callbacks := newBlackthornWorkflowTestScene(t, 2)

	scene.startNPCGuardArrestWorkflow(nil)
	scene.startNPCGuardArrestWorkflow(nil)
	require.True(t, scene.gameState.EventQueueManager.HasWorkflow("blackthorn_interrogation"))
	assert.False(t, scene.gameState.EventQueueManager.HasWorkflow("guard_arrest"))
	dialog := waitForBlackthornPrompt(t, scene, "Your response?")
	assert.Contains(t, transcriptText(dialog), strings.Join(strings.Fields(game_state.BlackthornQuestMessage), " "))
	assert.NotContains(t, dialog.TranscriptText(), "Wilt thou come quietly?")
	assert.Equal(t, indexes.SpriteIndex(indexes.Manacles_Prisoner), scene.miniMapScene.GetActor(minimap.ActorAvatar).Sprite)
	assert.Equal(t, indexes.SpriteIndex(indexes.Manacles_Prisoner), scene.miniMapScene.GetActor("prisoner-1").Sprite)

	enterTranscriptText(t, scene, dialog, "AHM")
	drainBlackthornWorkflow(t, scene, dialog)

	assert.Equal(t, references.PalaceOfBlackthorn, scene.gameState.MapState.PlayerLocation.Location)
	assert.Equal(t, references.Basement, scene.gameState.MapState.PlayerLocation.Floor)
	assert.Equal(t, references.Position{X: 10, Y: 7}, scene.gameState.MapState.PlayerLocation.Position)
	assert.Zero(t, scene.dialogStack.Count())
	assert.Nil(t, scene.miniMapScene)
	assert.False(t, callbacks.InputBlocked)
	assert.False(t, scene.gameState.EventQueueManager.HasActiveWorkflows())
}

func TestBlackthornInterrogationShutdownRestoresGameplayOwnership(t *testing.T) {
	scene, callbacks := newBlackthornWorkflowTestScene(t, 2)
	scene.gameState.PartyState.Inventory.Provisions.Keys.Set(7)

	scene.startNPCGuardArrestWorkflow(nil)
	waitForBlackthornPrompt(t, scene, "Your response?")
	require.Equal(t, references.BlackthornPrison, scene.gameState.MapState.PlayerLocation.Location)
	require.True(t, callbacks.InputBlocked)
	require.NotNil(t, scene.miniMapScene)
	require.Equal(t, 1, scene.dialogStack.Count())

	require.NotPanics(t, scene.gameState.EventQueueManager.Shutdown)
	assert.Equal(t, references.PalaceOfBlackthorn, scene.gameState.MapState.PlayerLocation.Location)
	assert.Equal(t, references.Position{X: 10, Y: 7}, scene.gameState.MapState.PlayerLocation.Position)
	assert.Zero(t, scene.gameState.PartyState.Inventory.Provisions.Keys.Get())
	assert.Zero(t, scene.dialogStack.Count())
	assert.Nil(t, scene.miniMapScene)
	assert.False(t, callbacks.InputBlocked)
}

func TestBlackthornWrongAnswerWorkflowAdvancesHourglassAndExecutesCompanion(t *testing.T) {
	scene, _ := newBlackthornWorkflowTestScene(t, 2)
	scene.startNPCGuardArrestWorkflow(nil)
	dialog := waitForBlackthornPrompt(t, scene, "Your response?")
	require.Equal(t, indexes.SpriteIndex(indexes.Hourglass1), scene.miniMapScene.GetActor("hourglass").Sprite)

	for _, expectedSprite := range []indexes.SpriteIndex{
		indexes.Hourglass2,
		indexes.Hourglass4,
		indexes.Hourglass1,
	} {
		enterTranscriptText(t, scene, dialog, "WRONG")
		dialog = waitForBlackthornPrompt(t, scene, "Your response?")
		require.Equal(t, expectedSprite, scene.miniMapScene.GetActor("hourglass").Sprite)
	}
	enterTranscriptText(t, scene, dialog, "WRONG")
	for range 100 {
		scene.gameState.ElapsedMs += 1000
		scene.gameState.EventQueueManager.Update()
		if strings.HasPrefix(dialog.ControlHint(), "Enter/Esc:") {
			break
		}
	}
	require.Equal(t, indexes.SpriteIndex(indexes.TortureChair1), scene.miniMapScene.GetActor("pendulum").Sprite)
	drainBlackthornWorkflow(t, scene, dialog)

	require.Equal(t, 1, scene.gameState.PartyState.GetActivePartyCount())
	require.Len(t, scene.gameState.PartyState.MartyrNames(), 1)
}

func TestBlackthornEscapeOnFinalAttemptClearsWithoutSubmitting(t *testing.T) {
	scene, callbacks := newBlackthornWorkflowTestScene(t, 2)
	companionName := scene.gameState.PartyState.GetCharacter(1).GetNameAsString()
	scene.startNPCGuardArrestWorkflow(nil)
	dialog := waitForBlackthornPrompt(t, scene, "Your response?")
	for range 3 {
		enterTranscriptText(t, scene, dialog, "WRONG")
		dialog = waitForBlackthornPrompt(t, scene, "Your response?")
	}
	require.Contains(t, strings.Join(strings.Fields(dialog.ControlHint()), " "), "Esc: Clear text")
	before := dialog.TranscriptText()
	for _, key := range []ebiten.Key{ebiten.KeyX, ebiten.KeyEscape, ebiten.KeyEscape} {
		scene.keyboard.ClearFrameConsumption()
		scene.keyboard.SetForceWaitAnyKey(-1000)
		scene.keyboard.SimulateKeyPress(key)
		dialog.Update()
		scene.gameState.ElapsedMs += 1000
		scene.gameState.EventQueueManager.Update()
		require.False(t, dialog.InputComplete())
		require.False(t, dialog.Submitted())
		require.Equal(t, "Your response?", dialog.ActivePrompt())
		require.Equal(t, before, dialog.TranscriptText())
		require.Empty(t, scene.gameState.PartyState.MartyrNames())
		require.Equal(t, 2, scene.gameState.PartyState.GetActivePartyCount())
		require.True(t, callbacks.InputBlocked)
	}
	enterTranscriptText(t, scene, dialog, "AHM")
	require.Equal(t, "AHM", dialog.Response(), "Escape must discard the typed X")
	drainBlackthornWorkflow(t, scene, dialog)
	require.Equal(t, []string{companionName}, scene.gameState.PartyState.MartyrNames())
	require.False(t, callbacks.InputBlocked)
}

func TestBlackthornFirstWrongAnswerRunsOriginalTorturePresentationBeforeAcknowledgement(t *testing.T) {
	scene, _ := newBlackthornWorkflowTestScene(t, 2)
	scene.startNPCGuardArrestWorkflow(nil)
	dialog := waitForBlackthornPrompt(t, scene, "Your response?")
	require.Equal(t, references.Position{X: 1, Y: 5}, scene.miniMapScene.GetActor("guard-left").Position)
	require.Equal(t, references.Position{X: 9, Y: 9}, scene.miniMapScene.GetActor("guard-right").Position)

	enterTranscriptText(t, scene, dialog, "NO")
	startedAt := scene.gameState.ElapsedMs
	sawMovement := false
	for range 100 {
		scene.gameState.ElapsedMs += 100
		scene.gameState.EventQueueManager.Update()
		if scene.miniMapScene.GetActor("guard-left").Position != (references.Position{X: 1, Y: 5}) {
			sawMovement = true
		}
		if strings.HasPrefix(dialog.ControlHint(), "Enter/Esc:") {
			break
		}
	}

	require.True(t, sawMovement)
	require.GreaterOrEqual(t, scene.gameState.ElapsedMs-startedAt, int64(5900))
	require.Equal(t, references.Position{X: 1, Y: 5}, scene.miniMapScene.GetActor("guard-left").Position)
	require.Equal(t, references.Position{X: 9, Y: 9}, scene.miniMapScene.GetActor("guard-right").Position)
	companion := scene.miniMapScene.GetActor("prisoner-1")
	require.Equal(t, references.Position{X: 5, Y: 7}, companion.Position)
	require.False(t, companion.Visible)
	require.Equal(t, indexes.SpriteIndex(indexes.TortureTableWithBody1), scene.miniMapScene.GetActor("pendulum").Sprite)
	require.Equal(t, indexes.SpriteIndex(indexes.Hourglass2), scene.miniMapScene.GetActor("hourglass").Sprite)
	require.Contains(t, transcriptText(dialog), "until the sand has fallen")
	require.False(t, dialog.InputComplete(), "the original post-animation key acknowledgement must remain pending")
}

func newBlackthornWorkflowTestScene(
	t *testing.T,
	living int,
) (*GameScene, *game_state.MockSystemCallbacks) {
	t.Helper()
	scene, callbacks := newShrineTranscriptTestScene(t)
	scene.gameState.MapState.PlayerLocation = references.PlayerLocation{
		Location: references.PalaceOfBlackthorn,
		Position: references.Position{X: 20, Y: 20},
		Floor:    references.MainFloor,
	}
	for index := range party_state.NRosterSlots {
		character := scene.gameState.PartyState.GetCharacter(index)
		character.PartyMap = party_state.PartyMapUnrecruited
		character.SetStatus(party_state.Dead)
	}
	for index := range living {
		character := scene.gameState.PartyState.GetCharacter(index)
		character.PartyMap = party_state.PartyMapActive
		character.SetStatus(party_state.Good)
	}
	for index := range game_state.TotalShrines {
		scene.gameState.SetShrineRuined(index, false)
	}
	callbacks.Reset()

	return scene, callbacks
}

func waitForBlackthornPrompt(
	t *testing.T,
	scene *GameScene,
	prompt string,
) *widgets.InteractionTranscript {
	t.Helper()
	var dialog *widgets.InteractionTranscript
	for range 300 {
		scene.gameState.ElapsedMs += 1000
		scene.gameState.EventQueueManager.Update()
		current, ok := scene.dialogStack.PeekTopModalDialog().(*widgets.InteractionTranscript)
		if !ok {
			continue
		}
		dialog = current
		if dialog.ActivePrompt() == prompt && !dialog.InputComplete() {
			return dialog
		}
		if dialog.ActivePrompt() == "" && !dialog.InputComplete() &&
			strings.HasPrefix(dialog.ControlHint(), "Enter/Esc:") {
			acknowledgeBlackthornTranscript(t, scene, dialog)
		}
	}
	t.Fatalf("Blackthorn prompt %q was not shown; transcript: %q", prompt, transcriptText(dialog))

	return nil
}

func drainBlackthornWorkflow(
	t *testing.T,
	scene *GameScene,
	dialog *widgets.InteractionTranscript,
) {
	t.Helper()
	for range 300 {
		scene.gameState.ElapsedMs += 1000
		scene.gameState.EventQueueManager.Update()
		if !scene.gameState.EventQueueManager.HasWorkflow("blackthorn_interrogation") {
			return
		}
		if scene.dialogStack.PeekTopModalDialog() == dialog &&
			dialog.ActivePrompt() == "" && !dialog.InputComplete() &&
			strings.HasPrefix(dialog.ControlHint(), "Enter/Esc:") {
			acknowledgeBlackthornTranscript(t, scene, dialog)
		}
	}
	t.Fatalf("Blackthorn workflow did not complete; transcript: %q", transcriptText(dialog))
}

func acknowledgeBlackthornTranscript(
	t *testing.T,
	scene *GameScene,
	dialog *widgets.InteractionTranscript,
) {
	t.Helper()
	scene.keyboard.ClearFrameConsumption()
	scene.keyboard.SetForceWaitAnyKey(-1000)
	scene.keyboard.SimulateKeyPress(ebiten.KeyEnter)
	dialog.Update()
	require.True(t, dialog.InputComplete())
}

func transcriptText(dialog *widgets.InteractionTranscript) string {
	if dialog == nil {
		return ""
	}

	return strings.Join(strings.Fields(dialog.TranscriptText()), " ")
}
