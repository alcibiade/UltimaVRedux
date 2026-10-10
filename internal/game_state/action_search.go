package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

const (
	combatChestDifficultyDivisor = 2
	simpleTrapMaximumValue       = 10
	complexTrapMinimumValue      = 20
)

// DungeonSearchResult reports presentation-relevant details without transferring mutation ownership from GameState.
type DungeonSearchResult struct {
	Found                bool
	CrumbledSkeletonWall bool
	RevealedHiddenPit    bool
	Target               references.Position
}

func (g *GameState) ActionSearchSmallMap(direction references.Direction) bool {
	return g.ActionSearchSmallMapWithCharacter(direction, g.defaultSurfaceSearchCharacter())
}

// ActionSearchSmallMapWithCharacter resolves Search for the actor selected after direction input.
func (g *GameState) ActionSearchSmallMapWithCharacter(direction references.Direction, character *party_state.PlayerCharacter) bool {
	return g.actionSearchSurfaceWithCharacter(direction, character)
}

func (g *GameState) actionSearchSurfaceWithCharacter(direction references.Direction, character *party_state.PlayerCharacter) bool {
	characterIndex := g.surfaceSearchCharacterIndex(character)
	searchResponse := g.SearchManager.ExecuteSearchWithCharacter(direction, characterIndex)

	g.SystemCallbacks.UI.ShowSearchFeedback(searchResponse.Message)
	g.SystemCallbacks.Message.AddRowStr(searchResponse.Message)
	if searchResponse.TimeSpent > 0 {
		g.AdvanceOrdinarySurfaceCommandMinutes(searchResponse.TimeSpent)
	}

	// Return true if something was found, false otherwise
	return searchResponse.Result != SearchResultNotFound
}

func (g *GameState) ActionSearchLargeMap(direction references.Direction) bool {
	return g.ActionSearchLargeMapWithCharacter(direction, g.defaultSurfaceSearchCharacter())
}

// ActionSearchLargeMapWithCharacter resolves large-map Search for the selected actor.
func (g *GameState) ActionSearchLargeMapWithCharacter(direction references.Direction, character *party_state.PlayerCharacter) bool {
	return g.actionSearchSurfaceWithCharacter(direction, character)
}

func (g *GameState) defaultSurfaceSearchCharacter() *party_state.PlayerCharacter {
	index := g.defaultSurfaceSearchCharacterIndex()
	if index < 0 {
		return nil
	}

	return g.PartyState.GetCharacter(index)
}

func (g *GameState) defaultSurfaceSearchCharacterIndex() int {
	if g.isSurfaceSearchCharacterEligible(g.GetActivePlayerIndex()) {
		return g.GetActivePlayerIndex()
	}
	selected := -1
	g.PartyState.ForEachCharacter(func(index int, character *party_state.PlayerCharacter) {
		if selected == -1 && character != nil && character.IsInParty() && (character.IsGood() || character.IsPoisoned()) {
			selected = index
		}
	})

	return selected
}

func (g *GameState) surfaceSearchCharacterIndex(character *party_state.PlayerCharacter) int {
	if character == nil {
		return -1
	}
	selected := -1
	g.PartyState.ForEachCharacter(func(index int, candidate *party_state.PlayerCharacter) {
		if candidate == character {
			selected = index
		}
	})

	return selected
}

func (g *GameState) isSurfaceSearchCharacterEligible(index int) bool {
	if index < 0 || index >= party_state.NRosterSlots {
		return false
	}
	character := g.PartyState.GetCharacter(index)

	return character != nil && character.IsInParty() && (character.IsGood() || character.IsPoisoned())
}

func (g *GameState) ActionSearchCombatMap(direction references.Direction) bool {
	if !g.IsInCombat() || g.CombatScene == nil {
		g.SystemCallbacks.UI.ShowActionFeedback("Not now!", true)

		return false
	}

	combatantPosition := g.getActiveCombatantPosition()
	targetPosition := direction.GetNewPositionInDirection(&combatantPosition)
	if targetPosition == nil {
		return false
	}

	if chest, found := g.CombatScene.objectAtMatching(*targetPosition, CombatObject.IsChest); found {
		return g.searchCombatChest(chest)
	}

	corpse, found := g.CombatScene.takeObjectAt(*targetPosition, CombatObject.IsCorpse)
	if !found {
		g.SystemCallbacks.Message.AddRowStr("Search-Nothing!")

		return false
	}

	outcome := rollCombatCorpseSearch(g.RandomIntInRange)
	activeIndex := g.CombatScene.GetActiveCombatantIndex()
	character := g.PartyState.GetCharacter(activeIndex)
	if outcome.poisoned {
		statusBefore := character.GetStatus()
		character.SetStatus(party_state.Poisoned)
		changes := transientStatusChanges{}
		changes.observePoison(statusBefore, character)
		g.publishTransientStatusChanges(changes)
		g.SystemCallbacks.Audio.PlaySoundEffect(SoundPoison)
		g.SystemCallbacks.Screen.UpdateStatsDisplay()
	}
	if outcome.hasItem {
		itemReference := g.GameReferences.InventoryItemReferences.GetReferenceByItem(outcome.item)
		g.CombatScene.replaceObjectInOriginalSlot(corpse, NewCombatLooseItem(
			*targetPosition,
			outcome.item,
			outcome.quantity,
			itemReference.ItemSprite,
		))
	}

	message := "Thou dost find " + outcome.message
	g.SystemCallbacks.Message.AddRowStr(message)
	g.SystemCallbacks.UI.ShowActionFeedback(message, false)

	return true
}

func (g *GameState) searchCombatChest(chest CombatObject) bool {
	activeIndex := g.CombatScene.GetActiveCombatantIndex()
	character := g.PartyState.GetCharacter(activeIndex)
	liveChest := g.CombatScene.liveChest(chest)
	liveChest.chestAssessment = g.inspectObjectChest(liveChest.ChestAssessment(), liveChest.IsLockedAndTrapped(), liveChest.TreasureValue(), int(character.Intelligence))
	fullMessage := objectChestAssessmentMessage(liveChest.ChestAssessment(), liveChest.TreasureValue())
	g.SystemCallbacks.Message.AddRowStr(fullMessage)
	g.SystemCallbacks.UI.ShowActionFeedback(fullMessage, false)

	return true
}

func (g *GameState) ActionSearchDungeonMap(direction references.Direction) bool {
	activeIndex := g.GetActivePlayerIndex()
	if g.IsDungeonInteractionCharacterEligible(activeIndex) {
		return g.ActionSearchDungeonMapWithCharacter(direction, activeIndex)
	}
	indices := g.DungeonInteractionCharacterIndices()
	if len(indices) == 0 {
		g.claimDungeonCommandAction(DungeonFullEndPlayer)
		g.SystemCallbacks.UI.ShowActionFeedback("No one able!", true)

		return false
	}

	return g.ActionSearchDungeonMapWithCharacter(direction, indices[0])
}

// ActionSearchDungeonMapWithCharacter performs the selected character's search.
func (g *GameState) ActionSearchDungeonMapWithCharacter(direction references.Direction, characterIndex int) bool {
	return g.ResolveDungeonSearchWithCharacter(direction, characterIndex).Found
}

// ResolveDungeonSearchWithCharacter performs Search and reports presentation-relevant transitions.
func (g *GameState) ResolveDungeonSearchWithCharacter(
	direction references.Direction,
	characterIndex int,
) DungeonSearchResult {
	dungeon := g.requireActiveDungeon()
	g.claimDungeonCommandAction(DungeonFullEndPlayer)
	character := g.requireDungeonInteractionCharacter(characterIndex)

	if !g.CanSearchDungeonMap() {
		return DungeonSearchResult{}
	}

	target, ok := dungeonSearchTarget(g.MapState.PlayerLocation.Position, dungeon.facing, direction)
	if !ok {
		g.SystemCallbacks.UI.ShowActionFeedback(MessageCantSearchThere, false)
		g.SystemCallbacks.Message.AddRowStr("Search-" + MessageCantSearchThere)

		return DungeonSearchResult{}
	}
	result := DungeonSearchResult{Target: target}
	cell, ok := dungeon.Cell(g.MapState.PlayerLocation.Floor, target)
	if !ok {
		panic("wrapped dungeon search produced invalid target")
	}

	message := MessageNothingOfNote
	found := false
	switch {
	case cell.Kind() == references.DungeonHiddenDoor:
		_, err := g.TransformDungeonCell(
			g.MapState.PlayerLocation.Floor,
			target,
			func(before references.DungeonCell) references.DungeonCell {
				if before.Kind() != references.DungeonHiddenDoor {
					panic("dungeon hidden door changed during search")
				}

				return references.DungeonCell(0xe0 | (before.Raw() & 0x08))
			},
		)
		if err != nil {
			panic(err)
		}
		g.recordDungeonDiscovery(dungeonDiscoveryHiddenDoor, g.MapState.PlayerLocation.Floor, target)
		g.SystemCallbacks.Audio.Play(SoundUnlock).Start()
		message = "a hidden door!"
		found = true
	case cell.Raw() == 0x61:
		floor := g.MapState.PlayerLocation.Floor
		_, err := g.TransformDungeonCell(floor, target, func(before references.DungeonCell) references.DungeonCell {
			if before.Raw() != 0x61 {
				panic("dungeon hidden pit changed during search")
			}

			return 0x60
		})
		if err != nil {
			panic(err)
		}
		if floor < references.DungeonFloorCount-1 {
			_, err = g.TransformDungeonCell(floor+1, target, func(below references.DungeonCell) references.DungeonCell {
				return references.DungeonCell(below.Raw() | 0x08)
			})
			if err != nil {
				panic(err)
			}
		}
		g.recordDungeonDiscovery(dungeonDiscoveryHiddenPit, floor, target)
		message = "A pit!"
		found = true
		result.RevealedHiddenPit = true
	case cell.Raw() == 0x62:
		message, found = g.searchDungeonFloorBomb(
			g.MapState.PlayerLocation.Floor,
			target,
			cell,
			character,
		)
	case cell.Kind() == references.DungeonClosedChest:
		message = g.searchDungeonChest(target, cell, character)
		found = true
	case cell.Kind() == references.DungeonSpecialWall &&
		dungeonPresentationKind(g.MapState.PlayerLocation.Location) == dungeonPresentationSkeleton:
		_, err := g.TransformDungeonCell(
			g.MapState.PlayerLocation.Floor,
			target,
			func(before references.DungeonCell) references.DungeonCell {
				if before.Kind() != references.DungeonSpecialWall {
					panic("dungeon skeleton wall changed during search")
				}

				return references.DungeonCell(0xb0 | (before.Raw() & 0x08))
			},
		)
		if err != nil {
			panic(err)
		}
		message = "the bones crumble!"
		found = true
		result.CrumbledSkeletonWall = true
	}

	g.SystemCallbacks.UI.ShowActionFeedback(message, false)
	g.SystemCallbacks.Message.AddRowStr("Search-" + message)
	if found {
		g.SystemCallbacks.Screen.UpdateDisplay()
	}

	result.Found = found

	return result
}

func (g *GameState) searchDungeonChest(
	position references.Position,
	cell references.DungeonCell,
	character *party_state.PlayerCharacter,
) string {
	floor := g.MapState.PlayerLocation.Floor
	dungeon := g.requireActiveDungeon()
	assessment := dungeon.ChestAssessment(floor, position)
	if assessment == references.ChestUninspected {
		difficulty := dungeonInteractionDifficulty(floor, character.Dexterity)
		// REDUX DEVIATION: dungeon chest inspection rerolls, invents severity on
		// failure, and misclassifies safe 0x48. Keep its first DEX/depth check
		// but remember honest knowledge using the actual Open/Jimmy trap bits.
		// See ref:CMD-0007.
		assessment = chestInspectionResult(g.RandomIntInRange(1, dungeonInteractionRollSides) > difficulty, cell.Raw()&0x07 != 0)
		dungeon.SetChestAssessment(floor, position, assessment)
	}
	if assessment == references.ChestSafe {
		return chestSafeMessage
	}
	if assessment == references.ChestUncertain {
		return chestUncertainMessage
	}

	switch {
	case floor < 4:
		return "A simple trap"
	case floor >= 7:
		return "A complex trap"
	default:
		return "A trap"
	}
}

func (g *GameState) searchDungeonFloorBomb(
	floor references.FloorNumber,
	position references.Position,
	before references.DungeonCell,
	character *party_state.PlayerCharacter,
) (string, bool) {
	difficulty := dungeonInteractionDifficulty(floor, character.Dexterity)
	if g.RandomIntInRange(1, dungeonInteractionRollSides) <= difficulty {
		return MessageNothingOfNote, false
	}

	change, err := g.TransformDungeonCell(floor, position, func(current references.DungeonCell) references.DungeonCell {
		if current != before || current.Raw() != 0x62 {
			panic("dungeon trap changed during search")
		}

		return references.DungeonCell(current.Raw() & 0x08)
	})
	if err != nil {
		panic(err)
	}
	if change.Before == change.After {
		panic("dungeon trap disarm did not change the cell")
	}

	return "A bomb trap!", true
}

// CanSearchDungeonMap resolves the original light precondition before direction selection.
func (g *GameState) CanSearchDungeonMap() bool {
	g.requireActiveDungeon()
	if g.MapState.Lighting.HasAnyLight() {
		return true
	}

	g.SystemCallbacks.UI.ShowActionFeedback("You find darkness.", false)
	g.SystemCallbacks.Message.AddRowStr("You find darkness.")

	return false
}

func dungeonSearchTarget(
	position references.Position,
	facing references.DungeonFacing,
	direction references.Direction,
) (references.Position, bool) {
	if direction == references.Down {
		return position, true
	}
	targetFacing := facing
	switch direction {
	case references.Up:
	case references.Left:
		targetFacing = facing.TurnLeft()
	case references.Right:
		targetFacing = facing.TurnRight()
	case references.Down, references.NoneDirection:
		return references.Position{}, false
	default:
		return references.Position{}, false
	}
	target := targetFacing.Direction().GetNewPositionInDirection(&position)
	if target == nil {
		return references.Position{}, false
	}

	return *target.GetWrapped(references.XDungeonMapTiles, references.YDungeonMapTiles), true
}
