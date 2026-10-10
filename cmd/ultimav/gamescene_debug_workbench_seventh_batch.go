package main

import (
	"errors"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

type allShrineQuestStateChangeResult struct {
	Index      int               `json:"index"`
	VirtueName string            `json:"virtueName"`
	VirtueKey  string            `json:"virtueKey"`
	Before     shrineStatusEntry `json:"before"`
	After      shrineStatusEntry `json:"after"`
}

type allShrineQuestStateDebugResult struct {
	TargetState       string                            `json:"targetState"`
	BeforeFingerprint string                            `json:"beforeFingerprint"`
	AfterFingerprint  string                            `json:"afterFingerprint"`
	Changes           []allShrineQuestStateChangeResult `json:"changes"`
}

func (g *GameScene) executeSeventhBatchDebugCommand(request debugging.Request) (debugging.Result, bool) {
	if request.CommandID != debugging.CommandCodexQuestStateSetAll {
		return debugging.Result{}, false //nolint:exhaustruct // Zero result marks an unhandled command.
	}

	target := game_state.AllShrineQuestState(mustDebugStringArgument(request, "targetState"))
	operation, err := g.gameState.SetAllShrineQuestState(
		target,
		mustDebugStringArgument(request, "expectedFingerprint"),
	)
	if err != nil {
		if errors.Is(err, game_state.ErrAllShrineQuestStateStale) {
			return debugging.FailureResult(request, "stale_shrine_quest_state", err.Error()), true
		}

		return debugging.FailureResult(request, "invalid_shrine_quest_state", err.Error()), true
	}

	data := allShrineQuestStateDebugResult{
		TargetState:       string(target),
		BeforeFingerprint: operation.BeforeFingerprint,
		AfterFingerprint:  operation.AfterFingerprint,
		Changes:           make([]allShrineQuestStateChangeResult, 0, len(operation.Changes)),
	}
	result := debugSuccessResultWithMode(
		request,
		debugging.ExecutionAlteration,
		"All shrine quest states set to "+string(target)+".",
		data,
	)
	for _, change := range operation.Changes {
		before := newShrineStatusEntry(change.Index, change.Before)
		after := newShrineStatusEntry(change.Index, change.After)
		data.Changes = append(data.Changes, allShrineQuestStateChangeResult{
			Index: change.Index, VirtueName: before.VirtueName, VirtueKey: before.VirtueKey,
			Before: before, After: after,
		})
		prefix := "shrines." + before.VirtueKey
		appendSemanticChange(&result.Changes, prefix+".questState", before.QuestState, after.QuestState)
		appendSemanticChange(&result.Changes, prefix+".completed", before.Completed, after.Completed)
		appendSemanticChange(&result.Changes, prefix+".active", before.Active, after.Active)
	}
	result.Data = mustDebugJSON(data)

	return result, true
}
