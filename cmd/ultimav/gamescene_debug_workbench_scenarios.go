package main

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
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
	Guards                 guardStatusResult           `json:"guards"`
}

type starvationCharacterResult struct {
	PartyPosition int    `json:"partyPosition"`
	Name          string `json:"name"`
	HPBefore      uint16 `json:"hpBefore"`
	HPAfter       uint16 `json:"hpAfter"`
	Damage        uint16 `json:"damage"`
	StatusBefore  string `json:"statusBefore"`
	StatusAfter   string `json:"statusAfter"`
	NewlyDead     bool   `json:"newlyDead"`
}

type starvationScenarioResult struct {
	Seed         uint64                      `json:"seed"`
	ForcedHourly bool                        `json:"forcedHourly"`
	FoodBefore   uint16                      `json:"foodBefore"`
	FoodAfter    uint16                      `json:"foodAfter"`
	HourBefore   byte                        `json:"hourBefore"`
	HourAfter    byte                        `json:"hourAfter"`
	MinuteBefore byte                        `json:"minuteBefore"`
	MinuteAfter  byte                        `json:"minuteAfter"`
	Characters   []starvationCharacterResult `json:"characters"`
	AllPartyDead bool                        `json:"allPartyDead"`
}

type waterfallSetupDebugResult struct {
	Position  worldPositionResult `json:"position"`
	Trigger   references.Position `json:"trigger"`
	Waterfall references.Position `json:"waterfall"`
}

type bridgeTrollSetupDebugResult struct {
	queuedWorkflowResult
	Position        worldPositionResult `json:"position"`
	Bridge          references.Position `json:"bridge"`
	Seed            uint64              `json:"seed"`
	CharacterSlot   int                 `json:"characterSlot"`
	DexterityBefore byte                `json:"dexterityBefore"`
	DexterityAfter  byte                `json:"dexterityAfter"`
	GoldBefore      uint16              `json:"goldBefore"`
	GoldAfter       uint16              `json:"goldAfter"`
}

type wordOfPowerSetupDebugResult struct {
	Position    worldPositionResult `json:"position"`
	Word        string              `json:"word"`
	DungeonID   int                 `json:"dungeonId"`
	DungeonName string              `json:"dungeonName"`
	Entrance    references.Position `json:"entrance"`
}

type blackthornInterrogationDebugResult struct {
	queuedWorkflowResult
	Position           worldPositionResult `json:"position"`
	ShrineIndex        int                 `json:"shrineIndex"`
	ShrineRuinedBefore bool                `json:"shrineRuinedBefore"`
	ShrineRuinedAfter  bool                `json:"shrineRuinedAfter"`
}

type lordBritishFinaleDebugResult struct {
	queuedWorkflowResult
	Position        worldPositionResult `json:"position"`
	WoodenBoxBefore uint16              `json:"woodenBoxBefore"`
	WoodenBoxAfter  uint16              `json:"woodenBoxAfter"`
	Revived         []string            `json:"revived"`
}

type finaleCharacterBefore struct {
	name   string
	hp     uint16
	status party_state.CharacterStatus
}

func (g *GameScene) executeScenarioDebugCommand(request debugging.Request) (debugging.Result, bool) {
	switch request.CommandID {
	case debugging.CommandScenarioStatus:
		return debugSuccessResult(request, "Scenario Lab status loaded.", newScenarioStatusResult(g)), true
	case debugging.CommandScenarioStarvationTrigger, debugging.CommandScenarioStarvationHourly:
		if result, blocked := g.scenarioMutationBlockedResult(request); blocked {
			return result, true
		}
		forcedHourly := request.CommandID == debugging.CommandScenarioStarvationHourly
		operation, err := g.gameState.RunStarvationScenario(
			uint64(mustDebugIntegerArgument(request, "seed")), //nolint:gosec // Catalog validation requires a non-negative seed.
			forcedHourly,
		)
		if err != nil {
			return debugging.FailureResult(request, "invalid_starvation_scenario", err.Error()), true
		}

		return starvationDebugResult(request, operation), true
	case debugging.CommandScenarioGuardsActivate:
		if result, blocked := g.scenarioMutationBlockedResult(request); blocked {
			return result, true
		}
		status := newGuardStatusResult(g.gameState)
		if !status.Supported || status.PotentialGuardCount == 0 {
			message := status.UnavailableReason
			if message == "" {
				message = "No potential guards are present on the current map."
			}

			return debugging.FailureResult(request, "invalid_guard_scenario", message), true
		}
		criminalBefore := g.gameState.IsCriminal()
		operation, err := g.gameState.ActivateGuards()
		if err != nil {
			return debugging.FailureResult(request, "invalid_guard_scenario", err.Error()), true
		}
		g.gameState.SetCriminalStatus(true)
		result := debugSuccessResultWithMode(
			request, debugging.ExecutionLiveSafe, "Town guards activated.", newGuardStatusResult(g.gameState),
		)
		appendSemanticChange(&result.Changes, "party.criminal", criminalBefore, g.gameState.IsCriminal())
		appendSemanticChange(&result.Changes, "guards.hostilePotentialGuardCount", operation.HostileBefore, operation.HostileAfter)

		return result, true
	case debugging.CommandScenarioWaterfallSetup:
		if result, blocked := g.scenarioMutationBlockedResult(request); blocked {
			return result, true
		}
		operation, err := g.gameState.DebugSetupWaterfall()
		if err != nil {
			return debugging.FailureResult(request, "invalid_waterfall_setup", err.Error()), true
		}
		result := debugSuccessResultWithMode(request, debugging.ExecutionSynchronized, "Waterfall entry setup loaded.", waterfallSetupDebugResult{
			Position: newWorldPositionResult(g.gameState),
			Trigger:  references.Position{X: 54, Y: 136}, Waterfall: references.Position{X: 54, Y: 137},
		})
		appendWorldSetupChanges(&result.Changes, operation)

		return result, true
	case debugging.CommandScenarioBridgeTroll:
		if result, blocked := g.scenarioMutationBlockedResult(request); blocked {
			return result, true
		}
		operation, err := g.gameState.DebugSetupBridgeTroll(
			uint64(mustDebugIntegerArgument(request, "seed")), //nolint:gosec // Catalog validation requires a non-negative seed.
		)
		if err != nil {
			return debugging.FailureResult(request, "invalid_bridge_troll_setup", err.Error()), true
		}
		g.StartBridgeTrollEncounterWorkflow()
		result := debugSuccessResultWithMode(request, debugging.ExecutionWorkflow,
			"Bridge troll sneak and attack workflow started.", bridgeTrollSetupDebugResult{
				queuedWorkflowResult: queuedWorkflowResult{Workflow: "bridge_troll_encounter", Queued: true},
				Position:             newWorldPositionResult(g.gameState),
				Bridge:               operation.Bridge,
				Seed:                 operation.Seed,
				CharacterSlot:        operation.CharacterSlot,
				DexterityBefore:      operation.DexterityBefore,
				DexterityAfter:       operation.DexterityAfter,
				GoldBefore:           operation.GoldBefore,
				GoldAfter:            operation.GoldAfter,
			})
		appendWorldSetupChanges(&result.Changes, operation.WorldSetupResult)
		appendSemanticChange(&result.Changes, "inventory.gold", operation.GoldBefore, operation.GoldAfter)
		character := g.gameState.PartyState.GetCharacter(operation.CharacterSlot)
		appendSemanticChange(&result.Changes, "party.character."+character.GetNameAsString()+".dexterity",
			operation.DexterityBefore, operation.DexterityAfter)

		return result, true
	case debugging.CommandScenarioWordSetup:
		if result, blocked := g.scenarioMutationBlockedResult(request); blocked {
			return result, true
		}
		operation, err := g.gameState.DebugSetupWordOfPower(mustDebugStringArgument(request, "word"))
		if err != nil {
			return debugging.FailureResult(request, "invalid_word_of_power_setup", err.Error()), true
		}
		result := debugSuccessResultWithMode(request, debugging.ExecutionSynchronized, operation.Word+" setup loaded beside "+operation.Dungeon.String()+".", wordOfPowerSetupDebugResult{
			Position: newWorldPositionResult(g.gameState), Word: operation.Word,
			DungeonID: int(operation.Dungeon), DungeonName: operation.Dungeon.String(), Entrance: operation.Entrance,
		})
		appendWorldSetupChanges(&result.Changes, operation.WorldSetupResult)

		return result, true
	case debugging.CommandScenarioBlackthornInterrogation:
		if result, blocked := g.scenarioMutationBlockedResult(request); blocked {
			return result, true
		}
		if result, stale := g.scenarioPartyReviewResult(request); stale {
			return result, true
		}
		operation, review, err := g.gameState.DebugBeginBlackthornInterrogation()
		if err != nil {
			return debugging.FailureResult(request, "invalid_blackthorn_interrogation_setup", err.Error()), true
		}
		g.startBlackthornInterrogationWorkflowWithReview(&review)
		result := debugSuccessResultWithMode(
			request,
			debugging.ExecutionWorkflow,
			"Blackthorn interrogation workflow started.",
			blackthornInterrogationDebugResult{
				queuedWorkflowResult: queuedWorkflowResult{Workflow: "blackthorn_interrogation", Queued: true},
				Position:             newWorldPositionResult(g.gameState),
				ShrineIndex:          operation.ShrineIndex,
				ShrineRuinedBefore:   operation.ShrineRuinedBefore,
				ShrineRuinedAfter:    operation.ShrineRuinedAfter,
			},
		)
		appendWorldSetupChanges(&result.Changes, operation.WorldSetupResult)
		appendSemanticChange(
			&result.Changes,
			fmt.Sprintf("quest.shrines.%d.ruined", operation.ShrineIndex),
			operation.ShrineRuinedBefore,
			operation.ShrineRuinedAfter,
		)

		return result, true
	case debugging.CommandScenarioLordBritishFinale:
		if result, blocked := g.scenarioMutationBlockedResult(request); blocked {
			return result, true
		}
		if result, stale := g.scenarioPartyReviewResult(request); stale {
			return result, true
		}
		if g.gameConfig == nil || g.gameConfig.SavedConfigData == nil {
			return debugging.FailureResult(request, "invalid_lord_british_finale_assets", "original game data path is unavailable"), true
		}
		assets, err := sprites.LoadEndgameAssets(g.gameConfig.SavedConfigData.DataFilePath)
		if err != nil {
			return debugging.FailureResult(request, "invalid_lord_british_finale_assets", err.Error()), true
		}

		return g.startDebugLordBritishFinale(request, assets), true
	default:
		return debugging.Result{}, false
	}
}

func (g *GameScene) startDebugLordBritishFinale(
	request debugging.Request,
	assets *sprites.EndgameAssets,
) debugging.Result {
	before := make([]finaleCharacterBefore, g.gameState.PartyState.GetActivePartyCount())
	for index := range before {
		character := g.gameState.PartyState.GetCharacter(index)
		before[index] = finaleCharacterBefore{
			name: character.GetNameAsString(), hp: character.GetCurrentHp(), status: character.GetStatus(),
		}
	}
	operation, review, err := g.gameState.DebugBeginLordBritishFinale(
		mustDebugBooleanArgument(request, "woodenBox"),
	)
	if err != nil {
		return debugging.FailureResult(request, "invalid_lord_british_finale_setup", err.Error())
	}
	g.startLordBritishFinale(review, assets)
	revived := make([]string, 0)
	for _, actor := range review.Party {
		if actor.Revived {
			revived = append(revived, before[actor.PartyIndex].name)
		}
	}
	result := debugSuccessResultWithMode(
		request,
		debugging.ExecutionWorkflow,
		"Lord British finale workflow started.",
		lordBritishFinaleDebugResult{
			queuedWorkflowResult: queuedWorkflowResult{Workflow: lordBritishFinaleWorkflow, Queued: true},
			Position:             newWorldPositionResult(g.gameState),
			WoodenBoxBefore:      operation.WoodenBoxBefore,
			WoodenBoxAfter:       operation.WoodenBoxAfter,
			Revived:              revived,
		},
	)
	appendWorldSetupChanges(&result.Changes, operation.WorldSetupResult)
	appendSemanticChange(
		&result.Changes,
		"inventory.questItems.woodenBox",
		operation.WoodenBoxBefore,
		operation.WoodenBoxAfter,
	)
	for _, actor := range review.Party {
		if !actor.Revived {
			continue
		}
		character := g.gameState.PartyState.GetCharacter(actor.PartyIndex)
		prior := before[actor.PartyIndex]
		prefix := "party.character." + prior.name
		appendSemanticChange(&result.Changes, prefix+".currentHp", prior.hp, character.GetCurrentHp())
		appendSemanticChange(
			&result.Changes,
			prefix+".status",
			characterStatusName(prior.status),
			characterStatusName(character.GetStatus()),
		)
	}

	return result
}

func appendWorldSetupChanges(changes *[]debugging.StateChange, operation game_state.WorldSetupResult) {
	appendSemanticChange(changes, "world.locationId", int(operation.Before.Location), int(operation.After.Location))
	appendSemanticChange(changes, "world.floor", int(operation.Before.Floor), int(operation.After.Floor))
	appendSemanticChange(changes, "world.position.x", int(operation.Before.Position.X), int(operation.After.Position.X))
	appendSemanticChange(changes, "world.position.y", int(operation.Before.Position.Y), int(operation.After.Position.Y))
	appendSemanticChange(changes, "effects.smashed", operation.SmashedBefore, operation.SmashedAfter)
}

func (g *GameScene) scenarioMutationBlockedResult(request debugging.Request) (debugging.Result, bool) {
	if g.gameState.IsGameplayTerminal() {
		return debugging.FailureResult(request, "gameplay_terminal", "scenarios are unavailable after the terminal finale"), true
	}
	if !g.gameState.CanAcceptGameplayCommands() {
		return debugging.FailureResult(request, "finale_active", "scenarios are unavailable during the Lord British finale handoff"), true
	}
	if g.dialogStack.HasOpenDialog() {
		return debugging.FailureResult(request, "dialog_active", "scenarios are unavailable while a dialog is open"), true
	}
	if g.gameState.EventQueueManager.HasActiveWorkflows() {
		return debugging.FailureResult(request, "gameplay_workflow_active", "scenarios are unavailable during a gameplay workflow"), true
	}
	if g.miniMapScene != nil || g.gameState.ShrineScene != nil {
		return debugging.FailureResult(request, "scripted_scene_active", "scenarios are unavailable during a scripted scene"), true
	}

	return debugging.Result{}, false
}

func (g *GameScene) scenarioPartyReviewResult(request debugging.Request) (debugging.Result, bool) {
	reviewed := mustDebugStringArgument(request, "expectedPartyFingerprint")
	current := g.gameState.CurrentPartyRosterSnapshot().Fingerprint
	if reviewed != current {
		return debugging.FailureResult(request, "stale_party", "the active party changed after scenario review"), true
	}

	return debugging.Result{}, false
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
		AllPartyDead:        state.PartyState.IsAllPartyDead(), Party: party, Guards: newGuardStatusResult(state),
	}
}

func starvationDebugResult(request debugging.Request, operation game_state.StarvationScenarioResult) debugging.Result {
	characters := make([]starvationCharacterResult, 0, len(operation.Characters))
	for _, character := range operation.Characters {
		characters = append(characters, starvationCharacterResult{
			PartyPosition: character.Slot + 1, Name: character.Name,
			HPBefore: character.HPBefore, HPAfter: character.HPAfter, Damage: character.HPBefore - character.HPAfter,
			StatusBefore: characterStatusName(character.StatusBefore), StatusAfter: characterStatusName(character.StatusAfter),
			NewlyDead: character.StatusBefore != character.StatusAfter && character.StatusAfter == party_state.Dead,
		})
	}
	data := starvationScenarioResult{
		Seed: operation.Seed, ForcedHourly: operation.ForcedHourly,
		FoodBefore: operation.FoodBefore, FoodAfter: operation.FoodAfter,
		HourBefore: operation.HourBefore, HourAfter: operation.HourAfter,
		MinuteBefore: operation.MinuteBefore, MinuteAfter: operation.MinuteAfter,
		Characters: characters, AllPartyDead: operation.AllPartyDead,
	}
	result := debugSuccessResultWithMode(
		request, debugging.ExecutionAlteration, fmt.Sprintf("Starvation applied to %d active characters.", len(characters)), data,
	)
	appendSemanticChange(&result.Changes, "inventory.provisions.food", operation.FoodBefore, operation.FoodAfter)
	appendSemanticChange(&result.Changes, "calendar.hour", operation.HourBefore, operation.HourAfter)
	for _, character := range characters {
		prefix := "party.character." + character.Name
		appendSemanticChange(&result.Changes, prefix+".currentHp", character.HPBefore, character.HPAfter)
		appendSemanticChange(&result.Changes, prefix+".status", character.StatusBefore, character.StatusAfter)
	}

	return result
}
