package main

import (
	"errors"
	"strconv"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

type activePartyWipeMemberResult struct {
	Slot   int    `json:"slot"`
	Name   string `json:"name"`
	HP     uint16 `json:"hp"`
	Status string `json:"status"`
}

type activePartyWipeMemberChangeResult struct {
	Before activePartyWipeMemberResult `json:"before"`
	After  activePartyWipeMemberResult `json:"after"`
}

type activePartyWipeReviewResult struct {
	Fingerprint string                        `json:"fingerprint"`
	Members     []activePartyWipeMemberResult `json:"members"`
}

type activePartyWipeDebugResult struct {
	BeforeFingerprint string                              `json:"beforeFingerprint"`
	AfterFingerprint  string                              `json:"afterFingerprint"`
	Changes           []activePartyWipeMemberChangeResult `json:"changes"`
	Workflow          string                              `json:"workflow,omitempty"`
	Queued            bool                                `json:"queued"`
}

func (g *GameScene) executeSixthBatchDebugCommand(request debugging.Request) (debugging.Result, bool) {
	switch request.CommandID {
	case debugging.CommandDeathKillParty:
		return g.executeDebugActivePartyWipe(request, false), true
	case debugging.CommandDeathTestSequence:
		return g.executeDebugActivePartyWipe(request, true), true
	default:
		return debugging.Result{}, false
	}
}

func (g *GameScene) executeDebugActivePartyWipe(request debugging.Request, queueResurrection bool) debugging.Result {
	if blocked := g.deathWipeBlocked(request, queueResurrection); blocked != nil {
		return *blocked
	}

	operation, err := g.gameState.WipeActiveParty(mustDebugStringArgument(request, "expectedFingerprint"))
	if err != nil {
		if errors.Is(err, game_state.ErrActivePartyWipeStale) {
			return debugging.FailureResult(request, "stale_party_wipe", err.Error())
		}

		return debugging.FailureResult(request, "invalid_party_wipe", err.Error())
	}

	mode := debugging.ExecutionSynchronized
	summary := "Active party wiped."
	workflow := ""
	if queueResurrection {
		g.startDeathResurrectionWorkflow()
		mode = debugging.ExecutionWorkflow
		summary = "Active party wiped; death and resurrection workflow queued."
		workflow = deathResurrectionWorkflowName
	}

	data := activePartyWipeDebugResult{
		BeforeFingerprint: operation.BeforeFingerprint,
		AfterFingerprint:  operation.AfterFingerprint,
		Changes:           newActivePartyWipeChanges(operation.Changes),
		Workflow:          workflow,
		Queued:            queueResurrection,
	}
	result := debugSuccessResultWithMode(request, mode, summary, data)
	for _, change := range data.Changes {
		prefix := "party.roster." + strconv.Itoa(change.Before.Slot)
		appendSemanticChange(&result.Changes, prefix+".currentHp", change.Before.HP, change.After.HP)
		appendSemanticChange(&result.Changes, prefix+".status", change.Before.Status, change.After.Status)
	}

	return result
}

func (g *GameScene) deathWipeBlocked(request debugging.Request, queueResurrection bool) *debugging.Result {
	if g.gameState.IsInCombat() {
		result := debugging.FailureResult(request, "combat_active", "party wipe is unavailable during combat")

		return &result
	}
	if g.gameState.EventQueueManager == nil {
		result := debugging.FailureResult(request, "event_queue_unavailable", "gameplay event queue is unavailable")

		return &result
	}
	if g.gameState.EventQueueManager.HasActiveWorkflows() {
		result := debugging.FailureResult(
			request, "gameplay_workflow_active", "party wipe is unavailable during a gameplay workflow",
		)

		return &result
	}
	if queueResurrection {
		if g.dialogStack.HasOpenDialog() {
			result := debugging.FailureResult(request, "dialog_active", "death sequence is unavailable while a dialog is open")

			return &result
		}
		if g.miniMapScene != nil || g.gameState.ShrineScene != nil {
			result := debugging.FailureResult(
				request, "scripted_scene_active", "death sequence is unavailable during a scripted scene",
			)

			return &result
		}
		if g.gameState.EventQueueManager.HasWorkflow(deathResurrectionWorkflowName) {
			result := debugging.FailureResult(
				request, "death_workflow_owned", "death and resurrection workflow is already owned",
			)

			return &result
		}
		if g.output == nil || g.spriteSheet == nil {
			result := debugging.FailureResult(
				request, "scene_resources_unavailable", "death sequence scene resources are unavailable",
			)

			return &result
		}
	}

	return nil
}

func newActivePartyWipeReviewResult(review game_state.ActivePartyWipeReview) activePartyWipeReviewResult {
	members := make([]activePartyWipeMemberResult, 0, len(review.Members))
	for _, member := range review.Members {
		members = append(members, newActivePartyWipeMemberResult(member))
	}

	return activePartyWipeReviewResult{Fingerprint: review.Fingerprint, Members: members}
}

func newActivePartyWipeChanges(changes []game_state.ActivePartyWipeMemberChange) []activePartyWipeMemberChangeResult {
	result := make([]activePartyWipeMemberChangeResult, 0, len(changes))
	for _, change := range changes {
		result = append(result, activePartyWipeMemberChangeResult{
			Before: newActivePartyWipeMemberResult(change.Before), After: newActivePartyWipeMemberResult(change.After),
		})
	}

	return result
}

func newActivePartyWipeMemberResult(member game_state.ActivePartyWipeMemberSnapshot) activePartyWipeMemberResult {
	return activePartyWipeMemberResult{
		Slot: member.Slot, Name: member.Name, HP: member.HP, Status: characterStatusName(member.Status),
	}
}
