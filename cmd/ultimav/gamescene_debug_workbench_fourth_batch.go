package main

import (
	"errors"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

type shardUseReviewResult struct {
	Shard              string `json:"shard"`
	Quantity           uint16 `json:"quantity"`
	LocationID         int    `json:"locationId"`
	LocationName       string `json:"locationName"`
	X                  int    `json:"x"`
	Y                  int    `json:"y"`
	Floor              int    `json:"floor"`
	Flame              string `json:"flame"`
	FlameMatches       bool   `json:"flameMatches"`
	ShadowlordSummoned bool   `json:"shadowlordSummoned"`
	Shadowlord         string `json:"shadowlord"`
	ShadowlordAlive    bool   `json:"shadowlordAlive"`
	ExpectedOutcome    string `json:"expectedOutcome"`
}

type shardUseQueuedResult struct {
	queuedWorkflowResult
	Review shardUseReviewResult `json:"review"`
}

func (g *GameScene) executeFourthBatchDebugCommand(request debugging.Request) (debugging.Result, bool) {
	switch request.CommandID {
	case debugging.CommandQuestShardUse:
		return g.executeDebugShardUse(request), true
	case debugging.CommandQuestInventoryPresetUsable:
		return g.executeDebugUsableInventoryPreset(request), true
	default:
		return debugging.Result{}, false //nolint:exhaustruct // Zero result marks an unhandled command.
	}
}

//nolint:cyclop // Shard use validates independent reviewed quest preconditions before queuing production flow.
func (g *GameScene) executeDebugShardUse(request debugging.Request) debugging.Result {
	if blocked := g.thirdBatchWorkflowBlocked(request); blocked != nil {
		return *blocked
	}
	shard, ok := debugQuestItem(mustDebugStringArgument(request, "shard"))
	if !ok || !isShard(shard) {
		return debugging.FailureResult(request, "invalid_shard", "unsupported shard identity")
	}
	review := newShardUseReviewResult(g.gameState, shard)
	expectedQuantity := mustDebugIntegerArgument(request, "expectedQuantity")
	if expectedQuantity < 0 || expectedQuantity > 99 || uint16(expectedQuantity) != review.Quantity {
		return debugging.FailureResult(request, "stale_shard_ownership", "selected shard ownership changed after review")
	}
	if review.Quantity == 0 {
		return debugging.FailureResult(request, "shard_not_owned", "selected shard is not in inventory")
	}
	if mustDebugIntegerArgument(request, "expectedLocationId") != review.LocationID ||
		mustDebugIntegerArgument(request, "expectedX") != review.X ||
		mustDebugIntegerArgument(request, "expectedY") != review.Y ||
		mustDebugIntegerArgument(request, "expectedFloor") != review.Floor {
		return debugging.FailureResult(request, "stale_shard_location", "player location changed after shard review")
	}
	if mustDebugStringArgument(request, "expectedFlame") != review.Flame {
		return debugging.FailureResult(request, "stale_shard_flame", "flame identity changed after shard review")
	}
	if mustDebugBooleanArgument(request, "expectedShadowlordSummoned") != review.ShadowlordSummoned ||
		mustDebugStringArgument(request, "expectedShadowlord") != review.Shadowlord ||
		mustDebugBooleanArgument(request, "expectedShadowlordAlive") != review.ShadowlordAlive {
		return debugging.FailureResult(request, "stale_shadowlord_state", "Shadowlord state changed after shard review")
	}

	g.StartShardWorkflow(shard)

	return debugSuccessResultWithMode(request, debugging.ExecutionWorkflow, "Shard workflow queued.", shardUseQueuedResult{
		queuedWorkflowResult: queuedWorkflowResult{Workflow: "shard_workflow", Queued: true},
		Review:               review,
	})
}

func (g *GameScene) executeDebugUsableInventoryPreset(request debugging.Request) debugging.Result {
	if g.dialogStack.HasOpenDialog() {
		return debugging.FailureResult(
			request, "dialog_active", "usable inventory preset is unavailable while a modal dialog is open",
		)
	}
	result, err := g.gameState.ApplyUsableInventoryPreset(mustDebugStringArgument(request, "expectedFingerprint"))
	if err != nil {
		if errors.Is(err, game_state.ErrUsableInventoryPresetStale) {
			return debugging.FailureResult(request, "stale_inventory", "usable inventory changed after review")
		}

		return debugging.FailureResult(request, "inventory_preset_unavailable", err.Error())
	}

	telemetry := newUsableInventoryPresetTelemetry(result.AfterFingerprint, result.Changes)
	debugResult := debugSuccessResultWithMode(
		request, debugging.ExecutionAlteration, "Usable inventory preset applied.", telemetry,
	)
	for _, change := range result.Changes {
		appendSemanticChange(&debugResult.Changes, "inventory."+change.Path+".quantity", change.Before, change.After)
	}

	return debugResult
}

func newShardUseReviewResult(state *game_state.GameState, shard references.QuestItem) shardUseReviewResult {
	location := state.MapState.PlayerLocation
	ritual := state.ReviewShardRitual(shard)
	flameName := "none"
	if ritual.Shadowlord != references.NoShadowlord {
		flameName = debugFlameName(ritual.Flame)
	}
	lord := references.NoShadowlord
	shadowlordSummoned := false
	shadowlordAlive := false
	if state.ShadowlordState != nil {
		lord = state.ShadowlordState.LastCalled
		shadowlordSummoned = state.ShadowlordState.Summoned
		shadowlordAlive = state.ShadowlordState.IsAlive(lord)
	}
	lordName := debugShadowlordName(lord)
	quantity := state.PartyState.Inventory.QuestItems.Get(shard)
	outcome := "no_effect"
	switch ritual.Result {
	case game_state.ShardUsageNotAttempted:
		outcome = "not_attempted"
	case game_state.ShardUsageSuccess:
		outcome = "success"
	case game_state.ShardUsageNoEffect:
		// Keep the default no-effect outcome.
	}
	key, _ := debugQuestItemIdentity(shard)

	return shardUseReviewResult{
		Shard: key, Quantity: quantity, LocationID: int(location.Location), LocationName: location.Location.String(),
		X: int(location.Position.X), Y: int(location.Position.Y), Floor: int(location.Floor), Flame: flameName,
		FlameMatches: ritual.AtMatchingFlame, ShadowlordSummoned: shadowlordSummoned,
		Shadowlord: lordName, ShadowlordAlive: shadowlordAlive, ExpectedOutcome: outcome,
	}
}

func isShard(item references.QuestItem) bool {
	return item >= references.ShardOfFalsehood && item <= references.ShardOfCowardice
}

func debugFlameName(flame references.FlameOfPrinciple) string {
	return strings.ToLower(flame.String())
}

func debugShadowlordName(lord references.Shadowlord) string {
	if lord == references.NoShadowlord {
		return "none"
	}

	return strings.ToLower(lord.String())
}
