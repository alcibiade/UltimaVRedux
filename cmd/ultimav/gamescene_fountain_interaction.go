package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

const fountainWorkflowName = "fountain_workflow"

// ShowFountainInteraction handles the cosmetic fountain sequence on one persistent transcript.
// The character roster temporarily covers the transcript without replacing its interaction history.
func (g *GameScene) ShowFountainInteraction() {
	if g.gameState.EventQueueManager.HasWorkflow(fountainWorkflowName) {
		return
	}
	options := g.characterSelectionOptions(characterSelectionContext{Profile: characterSelectionFountain})
	widthSamples := []string{
		"Thou dost see a gurgling fountain!",
		"Who shall drink?  > Character",
		"Refreshing!",
	}
	for _, option := range options {
		widthSamples = append(widthSamples,
			"Who shall drink?  > "+option.Name,
			option.Name+" drinks from the fountain.",
		)
	}

	transcript := widgets.NewInteractionTranscript(g.keyboard, widgets.InteractionTranscriptConfig{
		MaxLines:       4,
		MaxInputLength: 1,
		WidthSamples:   widthSamples,
	})
	transcript.AppendLine("Thou dost see a gurgling fountain!")
	var characterSelectionDialog *widgets.CharacterSelectionDialog
	var selectionMade bool
	var selectedCharacterIndex int

	g.gameState.EventQueueManager.Workflow(fountainWorkflowName).
		Priority(event_queue.PriorityLow).
		Sequential().BlockInput().First(func() {
		g.dialogStack.PushModalDialog(transcript)
	}).ThenSequence(
		g.thenWaitForInteractionContinuation(transcript),
	).Then(func() {
		characterSelectionDialog = widgets.NewCharacterSelectionDialog(widgets.CharacterSelectionConfig{
			Title:   "Fountain: Who Shall Drink?",
			Options: options,
			OnSelect: func(characterIndex int) {
				g.dialogStack.RemoveWidgetInstance(characterSelectionDialog)
				selectedCharacterIndex = characterIndex
				selectionMade = true
			},
			OnCancel: func() {
				g.dialogStack.RemoveWidgetInstance(characterSelectionDialog)
				selectedCharacterIndex = -1
				selectionMade = true
			},
			Keyboard:    g.keyboard,
			SpriteSheet: g.spriteSheet,
			Placement:   gameScreenPercents,
		})
		g.dialogStack.PushModalDialog(characterSelectionDialog)
	}).While(
		func() bool { return !selectionMade },
		func() {},
		DialogPollingRateMs,
	).Then(func() {
		g.keyboard.SetForceWaitAnyKey(KeyboardDebounceMs)
	}).Wait(KeyboardDebounceShortMs).If(
		func() bool { return selectedCharacterIndex >= 0 },
		func(drinkQ *event_queue.SequentialEventBuilder) {
			drinkQ.Then(func() {
				selectedCharacter := g.gameState.PartyState.GetCharacter(selectedCharacterIndex)
				characterName := selectedCharacter.GetNameAsString()
				transcript.AppendResponse("Who shall drink?", characterName)
				g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundFountainDrink)
			}).Wait(SoundEffectPauseMs).Then(func() {
				selectedCharacter := g.gameState.PartyState.GetCharacter(selectedCharacterIndex)
				characterName := selectedCharacter.GetNameAsString()
				transcript.AppendLine(characterName + " drinks from the fountain.")
			}).ThenSequence(
				g.thenWaitForInteractionContinuation(transcript),
			).Then(func() {
				g.gameState.AdvanceGameplayMinutes(1)
				transcript.AppendLine("Refreshing!")
			}).ThenSequence(g.thenWaitForInteractionAcknowledgement(transcript))
		}).Finally(func() {
		if characterSelectionDialog != nil {
			g.dialogStack.RemoveWidgetInstance(characterSelectionDialog)
		}
		g.dialogStack.RemoveWidgetInstance(transcript)
		g.keyboard.ResetReleaseDebouncing()
		g.gameState.ProcessPendingProvisionHours()
	}).Start()
}
