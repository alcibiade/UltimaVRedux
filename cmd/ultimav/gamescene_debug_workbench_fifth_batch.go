//nolint:nlreturn // Short validation helpers group immediate failure returns with their conditions.
package main

import (
	"errors"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

var (
	errTelescopeSunDamageCallbacks  = errors.New("telescope sun-damage callbacks are unavailable")
	errAstronomyWorkflowCallbacks   = errors.New("astronomy workflow callbacks are unavailable")
	errCrystalBallWorkflowCallbacks = errors.New("crystal-ball workflow callbacks are unavailable")
)

type commandWorkflowContextResult struct {
	Date         string `json:"date"`
	Time         string `json:"time"`
	Hour         int    `json:"hour"`
	Minute       int    `json:"minute"`
	IsNight      bool   `json:"isNight"`
	MapType      string `json:"mapType"`
	LocationID   int    `json:"locationId"`
	LocationName string `json:"locationName"`
	Floor        int    `json:"floor"`
}

type commandWorkflowQueuedResult struct {
	queuedWorkflowResult
	Context        commandWorkflowContextResult `json:"context"`
	ExpectedBranch string                       `json:"expectedBranch"`
}

func (g *GameScene) executeFifthBatchDebugCommand(request debugging.Request) (debugging.Result, bool) {
	switch request.CommandID {
	case debugging.CommandTestTelescope:
		return g.executeDebugTelescope(request), true
	case debugging.CommandTestSpyglass:
		return g.executeDebugSpyglass(request), true
	case debugging.CommandTestCrystalBall:
		return g.executeDebugCrystalBall(request), true
	default:
		return debugging.Result{}, false //nolint:exhaustruct // Zero result marks an unhandled command.
	}
}

func (g *GameScene) executeDebugTelescope(request debugging.Request) debugging.Result {
	if blocked := g.commandWorkflowBlocked(request); blocked != nil {
		return *blocked
	}
	if result := g.validateCommandWorkflowContext(request, false); result != nil {
		return *result
	}
	telescopeNight := !isTelescopeDaytime(g.gameState.DateTime.Hour)
	if err := validateTelescopeCallbacks(g.gameState, telescopeNight); err != nil {
		return debugging.FailureResult(request, "callbacks_unavailable", err.Error())
	}

	expectedBranch := "sun_damage"
	if telescopeNight {
		expectedBranch = "night_sky"
	}
	g.StartTelescopeWorkflow()

	return g.commandWorkflowQueuedResult(request, "Telescope workflow queued.", "telescope_workflow", expectedBranch)
}

func (g *GameScene) executeDebugSpyglass(request debugging.Request) debugging.Result {
	if blocked := g.commandWorkflowBlocked(request); blocked != nil {
		return *blocked
	}
	if result := g.validateCommandWorkflowContext(request, true); result != nil {
		return *result
	}
	quantity := g.gameState.PartyState.Inventory.QuestItems.GetQuantity(references.Spyglass)
	if quantity == nil || !(*quantity).HasSome() {
		return debugging.FailureResult(request, "spyglass_not_owned", "spyglass workflow requires a spyglass in inventory")
	}
	if err := validateSpyglassCallbacks(g.gameState); err != nil {
		return debugging.FailureResult(request, "callbacks_unavailable", err.Error())
	}

	expectedBranch := "no_stars"
	if g.gameState.DateTime.IsNight() {
		expectedBranch = "night_sky"
	}
	g.StartSpyglassWorkflow()

	return g.commandWorkflowQueuedResult(request, "Spyglass workflow queued.", "spyglass_workflow", expectedBranch)
}

func (g *GameScene) executeDebugCrystalBall(request debugging.Request) debugging.Result {
	if blocked := g.commandWorkflowBlocked(request); blocked != nil {
		return *blocked
	}
	if result := g.validateCommandWorkflowContext(request, false); result != nil {
		return *result
	}
	if g.gameState.PartyState.GetActivePartyCount() == 0 {
		return debugging.FailureResult(request, "party_unavailable", "crystal-ball workflow requires an active party")
	}
	if err := validateCrystalBallCallbacks(g.gameState); err != nil {
		return debugging.FailureResult(request, "callbacks_unavailable", err.Error())
	}
	g.StartCrystalBallWorkflow()

	return g.commandWorkflowQueuedResult(
		request, "Crystal-ball workflow queued.", "crystal_ball_workflow", "character_selection",
	)
}

func (g *GameScene) commandWorkflowBlocked(request debugging.Request) *debugging.Result {
	if blocked := g.thirdBatchWorkflowBlocked(request); blocked != nil {
		return blocked
	}
	if g.output == nil || g.keyboard == nil || g.spriteSheet == nil {
		result := debugging.FailureResult(
			request, "scene_resources_unavailable", "game-window workflow resources are unavailable",
		)
		return &result
	}
	return nil
}

func (g *GameScene) validateCommandWorkflowContext(request debugging.Request, aboveGround bool) *debugging.Result {
	location := g.gameState.MapState.PlayerLocation
	if location.Location.GetMapType() != references.SmallMapType {
		result := debugging.FailureResult(request, "unsupported_map_type", "command workflow requires a small-map location")
		return &result
	}
	if aboveGround && location.Floor < 0 {
		result := debugging.FailureResult(request, "unsupported_floor", "spyglass workflow requires an above-ground floor")
		return &result
	}
	return nil
}

func (g *GameScene) commandWorkflowQueuedResult(
	request debugging.Request,
	summary string,
	workflow string,
	expectedBranch string,
) debugging.Result {
	location := g.gameState.MapState.PlayerLocation
	dateTime := &g.gameState.DateTime
	return debugSuccessResultWithMode(request, debugging.ExecutionWorkflow, summary, commandWorkflowQueuedResult{
		queuedWorkflowResult: queuedWorkflowResult{Workflow: workflow, Queued: true},
		Context: commandWorkflowContextResult{
			Date: dateTime.GetDateAsString(), Time: dateTime.GetTimeAsString(), Hour: int(dateTime.Hour),
			Minute: int(dateTime.Minute), IsNight: dateTime.IsNight(), MapType: debugMapTypeName(location.Location.GetMapType()),
			LocationID: int(location.Location), LocationName: location.Location.String(), Floor: int(location.Floor),
		},
		ExpectedBranch: expectedBranch,
	})
}

func validateTelescopeCallbacks(state *game_state.GameState, night bool) error {
	if err := validateAstronomyCallbacks(state); err != nil {
		return err
	}
	if !night && (state.SystemCallbacks.Audio.PlaySoundEffect == nil ||
		state.SystemCallbacks.Visual.SetScreenFlashColor == nil ||
		state.SystemCallbacks.Visual.SetScreenFlashIntensity == nil) {
		return errTelescopeSunDamageCallbacks
	}
	return nil
}

func validateSpyglassCallbacks(state *game_state.GameState) error {
	return validateAstronomyCallbacks(state)
}

func validateAstronomyCallbacks(state *game_state.GameState) error {
	if state.SystemCallbacks == nil || state.SystemCallbacks.UI.ShowActionFeedback == nil {
		return errAstronomyWorkflowCallbacks
	}
	return nil
}

func validateCrystalBallCallbacks(state *game_state.GameState) error {
	if state.SystemCallbacks == nil || state.SystemCallbacks.UI.ShowActionFeedback == nil ||
		state.SystemCallbacks.Audio.PlaySoundEffect == nil ||
		state.SystemCallbacks.Visual.SetScreenFlashColor == nil ||
		state.SystemCallbacks.Visual.SetScreenFlashIntensity == nil {
		return errCrystalBallWorkflowCallbacks
	}
	return nil
}
