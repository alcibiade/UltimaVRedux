package main

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

const dungeonFountainIntroduction = "You see a fountain."

const dungeonFountainWorkflowName = "dungeon_fountain_look"

func (g *GameScene) startDungeonOpenWorkflow() {
	g.selectDungeonInteractionCharacter(
		"Open Chest: Choose Character",
		characterSelectionOpen,
		func(characterIndex int) {
			g.gameState.ActionOpenDungeonMap(characterIndex)
			g.completeDungeonInteraction(&g.dungeonOpenCommandToken)
		},
		func() { g.completeDungeonInteraction(&g.dungeonOpenCommandToken) },
	)
}

func (g *GameScene) startDungeonJimmyWorkflow() {
	g.selectDungeonInteractionCharacter(
		"Jimmy: Choose Character",
		characterSelectionJimmy,
		func(characterIndex int) {
			g.gameState.ActionJimmyDungeonMap(characterIndex)
			g.completeDungeonInteraction(&g.dungeonJimmyCommandToken)
		},
		func() { g.completeDungeonInteraction(&g.dungeonJimmyCommandToken) },
	)
}

func (g *GameScene) startDungeonSearchWorkflow() {
	g.selectDungeonInteractionCharacter(
		"Search: Choose Character",
		characterSelectionSearch,
		func(characterIndex int) {
			g.dungeonSearchCharacterIndex = characterIndex
			if !g.gameState.CanSearchDungeonMap() {
				g.completeDungeonInteraction(&g.dungeonSearchCommandToken)

				return
			}
			g.gameState.SystemCallbacks.Message.AddRowStr("Ahead Here Left Right Pass")
			g.secondaryKeyState = SearchDirectionInput
			g.keyboard.SetAllowKeyPressImmediately()
		},
		func() { g.completeDungeonInteraction(&g.dungeonSearchCommandToken) },
	)
}

func (g *GameScene) startDungeonLookWorkflow() {
	g.selectDungeonInteractionCharacter(
		"Look: Choose Character",
		characterSelectionLook,
		func(characterIndex int) {
			g.dungeonLookCharacterIndex = characterIndex
			if !g.gameState.CanLookDungeonMap() {
				g.completeDungeonInteraction(&g.dungeonLookCommandToken)

				return
			}
			g.gameState.SystemCallbacks.Message.AddRowStr("Ahead Here Left Right Pass")
			g.secondaryKeyState = LookDirectionInput
			g.keyboard.SetAllowKeyPressImmediately()
		},
		func() { g.completeDungeonInteraction(&g.dungeonLookCommandToken) },
	)
}

func (g *GameScene) dungeonLookSecondaryInput() {
	if g.gameState.IsAtDungeonCommandBoundary() {
		g.dungeonLookCommandToken = game_state.DungeonCommandCycle{}
		g.secondaryKeyState = PrimaryInput

		return
	}
	key := g.keyboard.GetBoundKeyPressed(&boundKeysGame)
	if key == nil || !g.keyboard.TryToRegisterKeyPress(*key) {
		return
	}

	var direction references.Direction
	//nolint:exhaustive // Only original Ahead/Here/Left/Right/Pass choices resolve Look.
	switch *key {
	case ebiten.KeyUp, ebiten.KeyNumpad8:
		direction = references.Up
	case ebiten.KeyDown, ebiten.KeyNumpad2:
		direction = references.Down
	case ebiten.KeyLeft, ebiten.KeyNumpad4:
		direction = references.Left
	case ebiten.KeyRight, ebiten.KeyNumpad6:
		direction = references.Right
	case ebiten.KeySpace, ebiten.KeyEscape:
		g.gameState.SystemCallbacks.Message.AddRowStr("Pass")
		g.completeDungeonInteraction(&g.dungeonLookCommandToken)

		return
	default:
		return
	}

	review, looked := g.gameState.ActionLookDungeonMap(direction, g.dungeonLookCharacterIndex)
	if !looked || !review.Fountain {
		g.completeDungeonInteraction(&g.dungeonLookCommandToken)

		return
	}
	g.secondaryKeyState = PrimaryInput
	g.promptDungeonFountainDrink(review, g.dungeonLookCharacterIndex)
}

func (g *GameScene) promptDungeonFountainDrink(review game_state.DungeonLookResult, characterIndex int) {
	dialog := widgets.NewInteractionTranscript(g.keyboard, widgets.InteractionTranscriptConfig{
		MaxLines:       5,
		MaxInputLength: 1,
		WidthSamples: []string{
			dungeonFountainIntroduction,
			"Drink?  > Yes",
			"Yes. Gulp!",
			"Poisoned!",
			"Bad taste.",
		},
	})
	dialog.AppendLine(dungeonFountainIntroduction)
	g.gameState.EventQueueManager.Workflow(dungeonFountainWorkflowName).
		Priority(event_queue.PriorityMedium).
		Sequential().BlockInput().First(func() {
		g.dialogStack.PushModalDialog(dialog)
	}).ThenSequence(g.thenPromptInteractionYesNo(
		dialog,
		"Drink?",
		func(yesQ *event_queue.SequentialEventBuilder) {
			yesQ.Then(func() {
				result := g.gameState.ApplyDungeonFountainChoice(characterIndex, review, true)
				dialog.AppendLine("Yes. Gulp!")
				dialog.AppendLine(result.Outcome)
			}).ThenSequence(g.thenWaitForInteractionAcknowledgement(dialog))
		},
		func(noQ *event_queue.SequentialEventBuilder) {
			noQ.Then(func() {
				g.gameState.ApplyDungeonFountainChoice(characterIndex, review, false)
			}).ThenSequence(g.thenWaitForInteractionAcknowledgement(dialog))
		},
	)).Finally(func() {
		g.dialogStack.RemoveWidgetInstance(dialog)
		g.keyboard.ResetReleaseDebouncing()
		g.completeDungeonInteraction(&g.dungeonLookCommandToken)
	}).Start()
}

func (g *GameScene) selectDungeonInteractionCharacter(
	title string,
	profile characterSelectionProfile,
	onSelect func(int),
	onCancel func(),
) {
	indices := g.gameState.DungeonInteractionCharacterIndices()
	if len(indices) == 0 {
		g.ShowActionFeedback("No one able!", true)
		onCancel()

		return
	}
	activeIndex := g.gameState.GetActivePlayerIndex()
	if g.gameState.IsDungeonInteractionCharacterEligible(activeIndex) {
		onSelect(activeIndex)

		return
	}
	if len(indices) == 1 {
		onSelect(indices[0])

		return
	}
	var dialog *widgets.CharacterSelectionDialog
	dialog = widgets.NewCharacterSelectionDialog(widgets.CharacterSelectionConfig{
		Title:   title,
		Options: g.characterSelectionOptions(characterSelectionContext{Profile: profile}),
		OnSelect: func(characterIndex int) {
			g.dialogStack.RemoveWidgetInstance(dialog)
			onSelect(characterIndex)
		},
		OnCancel: func() {
			g.dialogStack.RemoveWidgetInstance(dialog)
			onCancel()
		},
		Keyboard:    g.keyboard,
		SpriteSheet: g.spriteSheet,
		Placement:   gameScreenPercents,
	})
	g.dialogStack.PushModalDialog(dialog)
}

func (g *GameScene) completeDungeonInteraction(token *game_state.DungeonCommandCycle) {
	activeToken := *token
	*token = game_state.DungeonCommandCycle{}
	g.secondaryKeyState = PrimaryInput
	g.gameState.CompleteDungeonCommandCycle(activeToken, game_state.DungeonFullEndPlayer)
}
