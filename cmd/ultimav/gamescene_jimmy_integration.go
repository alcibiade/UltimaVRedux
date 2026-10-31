package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

// smallMapJimmySecondaryWithDialog runs the production small-map Jimmy workflow.
func (g *GameScene) smallMapJimmySecondaryWithDialog(
	direction references.Direction,
	onComplete func(surfaceCommandState),
) surfaceCommandState {
	if !g.gameState.PartyState.Inventory.Provisions.Keys.HasSome() {
		g.gameState.ActionJimmySmallMapWithCharacter(direction, nil)

		return surfaceCommandRejected
	}
	if !g.canJimmyInDirection(direction) {
		g.gameState.ActionJimmySmallMapWithCharacter(direction, nil)

		return surfaceCommandRejected
	}

	return g.surfaceCharacterActionWithDialog(
		"Jimmy: Choose Character",
		characterSelectionJimmy,
		func(character *party_state.PlayerCharacter) bool {
			return g.gameState.ActionJimmySmallMapWithCharacter(direction, character)
		},
		onComplete,
	)
}

func (g *GameScene) largeMapJimmySecondaryWithDialog(
	direction references.Direction,
	onComplete func(surfaceCommandState),
) surfaceCommandState {
	if !g.gameState.PartyState.Inventory.Provisions.Keys.HasSome() || !g.canJimmyInDirection(direction) {
		g.gameState.ActionJimmyLargeMapWithCharacter(direction, nil)

		return surfaceCommandRejected
	}

	return g.surfaceCharacterActionWithDialog(
		"Jimmy: Choose Character",
		characterSelectionJimmy,
		func(character *party_state.PlayerCharacter) bool {
			return g.gameState.ActionJimmyLargeMapWithCharacter(direction, character)
		},
		onComplete,
	)
}

func (g *GameScene) surfaceOpenSecondaryWithDialog(
	direction references.Direction,
	onComplete func(surfaceCommandState),
) surfaceCommandState {
	mapType := g.gameState.MapState.PlayerLocation.Location.GetMapType()
	if position, ok := g.gameState.StorageChestInDirection(direction); ok {
		// The party's own storage chest opens its contents instead (ref:CRAFT-0011).
		g.openStorageChest(position, mapType)

		return surfaceCommandDeferred
	}
	if !g.gameState.CanOpenSurfaceChest(direction) {
		if mapType == references.SmallMapType {
			if g.gameState.ActionOpenSmallMap(direction) {
				return surfaceCommandCompleted
			}
		} else if g.gameState.ActionOpenLargeMap(direction) {
			return surfaceCommandCompleted
		}

		return surfaceCommandRejected
	}

	return g.surfaceCharacterActionWithDialog(
		"Open Chest: Choose Character",
		characterSelectionOpen,
		func(character *party_state.PlayerCharacter) bool {
			if mapType == references.SmallMapType {
				return g.gameState.ActionOpenSmallMapWithCharacter(direction, character)
			}

			return g.gameState.ActionOpenLargeMapWithCharacter(direction, character)
		},
		onComplete,
	)
}

func (g *GameScene) surfaceSearchSecondaryWithDialog(
	direction references.Direction,
	onComplete func(surfaceCommandState),
) surfaceCommandState {
	mapType := g.gameState.MapState.PlayerLocation.Location.GetMapType()

	return g.surfaceCharacterActionWithDialog(
		"Search: Choose Character",
		characterSelectionSearch,
		func(character *party_state.PlayerCharacter) bool {
			if mapType == references.SmallMapType {
				return g.gameState.ActionSearchSmallMapWithCharacter(direction, character)
			}

			return g.gameState.ActionSearchLargeMapWithCharacter(direction, character)
		},
		onComplete,
	)
}

func (g *GameScene) surfaceCharacterActionWithDialog(
	prompt string,
	profile characterSelectionProfile,
	action func(*party_state.PlayerCharacter) bool,
	onComplete func(surfaceCommandState),
) surfaceCommandState {
	activePlayer := g.gameState.GetActivePlayer()
	if isCharacterActorEligible(activePlayer) {
		if action(activePlayer) {
			return surfaceCommandCompleted
		}

		return surfaceCommandRejected
	}
	if activePlayer != nil {
		g.gameState.ClearActivePlayer()
	}

	var availableCharacters []*party_state.PlayerCharacter
	g.gameState.PartyState.ForEachCharacter(func(_ int, character *party_state.PlayerCharacter) {
		if character.PartyMap == party_state.PartyMapActive &&
			isCharacterActorEligible(character) {
			// Check if character has a name (not empty slot)
			nameBytes := character.Name[:]
			nameEnd := 0
			for j, b := range nameBytes {
				if b == 0 {
					nameEnd = j

					break
				}
				if j == len(nameBytes)-1 {
					nameEnd = j + 1
				}
			}
			if string(nameBytes[:nameEnd]) != "" {
				availableCharacters = append(availableCharacters, character)
			}
		}
	})

	if len(availableCharacters) == 0 {
		action(nil)

		return surfaceCommandRejected
	}
	if len(availableCharacters) == 1 {
		if action(availableCharacters[0]) {
			return surfaceCommandCompleted
		}

		return surfaceCommandRejected
	}

	// Declare dialog variable first so it can be captured in closures
	var dialog *widgets.CharacterSelectionDialog

	// Create character selection dialog
	dialog = widgets.NewCharacterSelectionDialog(widgets.CharacterSelectionConfig{
		Title:   prompt,
		Options: g.characterSelectionOptions(characterSelectionContext{Profile: profile}),
		OnSelect: func(characterIndex int) {
			g.dialogStack.RemoveWidgetInstance(dialog)
			selectedCharacter := g.gameState.PartyState.GetCharacter(characterIndex)
			g.gameState.BeginSurfaceCommandRouting()
			success := func() bool {
				defer g.gameState.EndSurfaceCommandRouting()

				return action(selectedCharacter)
			}()
			if success {
				onComplete(surfaceCommandCompleted)
			} else {
				onComplete(surfaceCommandRejected)
			}
		},
		OnCancel: func() {
			g.dialogStack.RemoveWidgetInstance(dialog)
			g.output.AddText(prompt + " cancelled")
			onComplete(surfaceCommandCancelled)
		},
		Keyboard:    g.keyboard,
		SpriteSheet: g.spriteSheet,
		Placement:   gameScreenPercents,
	})

	// Push dialog to stack
	g.dialogStack.PushModalDialog(dialog)

	return surfaceCommandDeferred
}

// canJimmyInDirection checks if there's something that can be jimmied in the given direction
func (g *GameScene) canJimmyInDirection(direction references.Direction) bool {
	return g.gameState.CanJimmySurfaceTarget(direction)
}
