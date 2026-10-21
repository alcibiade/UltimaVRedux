package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

type scenarioStatusResult struct {
	PartyFingerprint       string                      `json:"partyFingerprint"`
	WipeReview             activePartyWipeReviewResult `json:"wipeReview"`
	LocationID             int                         `json:"locationId"`
	LocationName           string                      `json:"locationName"`
	MapType                string                      `json:"mapType"`
	Floor                  int                         `json:"floor"`
	Date                   string                      `json:"date"`
	Time                   string                      `json:"time"`
	Food                   uint16                      `json:"food"`
	CombatActive           bool                        `json:"combatActive"`
	DialogActive           bool                        `json:"dialogActive"`
	GameplayWorkflowActive bool                        `json:"gameplayWorkflowActive"`
	ScriptedSceneActive    bool                        `json:"scriptedSceneActive"`
	GameplayTerminal       bool                        `json:"gameplayTerminal"`
	FinaleActive           bool                        `json:"finaleActive"`
	AllPartyDead           bool                        `json:"allPartyDead"`
	Party                  []activeCharacterResult     `json:"party"`
}

func (g *GameScene) executeScenarioDebugCommand(request debugging.Request) (debugging.Result, bool) {
	if request.CommandID != debugging.CommandScenarioStatus {
		return debugging.Result{}, false
	}

	return debugSuccessResult(request, "Scenario Lab status loaded.", newScenarioStatusResult(g)), true
}

func appendWorldSetupChanges(changes *[]debugging.StateChange, operation game_state.WorldSetupResult) {
	appendSemanticChange(changes, "world.locationId", int(operation.Before.Location), int(operation.After.Location))
	appendSemanticChange(changes, "world.floor", int(operation.Before.Floor), int(operation.After.Floor))
	appendSemanticChange(changes, "world.position.x", int(operation.Before.Position.X), int(operation.After.Position.X))
	appendSemanticChange(changes, "world.position.y", int(operation.Before.Position.Y), int(operation.After.Position.Y))
	appendSemanticChange(changes, "effects.smashed", operation.SmashedBefore, operation.SmashedAfter)
}

func newScenarioStatusResult(scene *GameScene) scenarioStatusResult {
	state := scene.gameState
	location := state.MapState.PlayerLocation
	characters := state.ActivePartyCharacters()
	party := make([]activeCharacterResult, 0, len(characters))
	for _, character := range characters {
		party = append(party, newActiveCharacterResult(character))
	}

	return scenarioStatusResult{
		PartyFingerprint: state.CurrentPartyRosterSnapshot().Fingerprint,
		WipeReview:       newActivePartyWipeReviewResult(state.ReviewActivePartyWipe()),
		LocationID:       int(location.Location), LocationName: location.Location.String(),
		MapType: debugMapTypeName(location.Location.GetMapType()), Floor: int(location.Floor),
		Date: state.DateTime.GetDateAsString(), Time: state.DateTime.GetTimeAsString(),
		Food: state.PartyState.Inventory.Provisions.Food.Get(), CombatActive: state.IsInCombat(),
		DialogActive: scene.dialogStack.HasOpenDialog(), GameplayWorkflowActive: state.EventQueueManager.HasActiveWorkflows(),
		ScriptedSceneActive: scene.miniMapScene != nil || state.ShrineScene != nil,
		GameplayTerminal:    state.IsGameplayTerminal(),
		FinaleActive:        !state.CanAcceptGameplayCommands(),
		AllPartyDead:        state.PartyState.IsAllPartyDead(), Party: party,
	}
}
