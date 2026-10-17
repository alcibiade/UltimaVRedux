package main

import (
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

const hmsCapeWorkflowName = "hms_cape_workflow"

type hmsCapeConfirmationState struct {
	dialog     *widgets.InputBox
	choiceMade bool
	confirmed  bool
	committed  bool
}

func (g *GameScene) startHMSCapeWorkflow(commit *game_state.HMSCapeUseCommit, complete func()) {
	confirmation := &hmsCapeConfirmationState{}
	g.gameState.EventQueueManager.Workflow(hmsCapeWorkflowName).
		Priority(event_queue.PriorityNormal).
		Sequential().BlockInput().
		First(func() {}).
		ThenSequence(g.thenConfirmHMSCape(commit, confirmation)).
		Finally(func() {
			g.cleanupHMSCapeConfirmation(confirmation)
			g.keyboard.ResetReleaseDebouncing()
			if complete == nil {
				if confirmation.committed {
					g.gameState.AdvanceGameplayMinutesAndProcessProvisions(1)
				}
			} else {
				complete()
			}
		}).
		Start()
}

func (g *GameScene) thenConfirmHMSCape(
	commit *game_state.HMSCapeUseCommit,
	confirmation *hmsCapeConfirmationState,
) event_queue.Sequence {
	if commit == nil || confirmation == nil {
		panic("HMS Cape confirmation requires a commit and workflow state")
	}

	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		return q.Then(func() {
			confirmation.dialog = widgets.NewInputBox(hmsCapeConfirmationPrompt(commit.Enables()), nil, g.keyboard).
				WithRegexValidation("(?i)^[yn]?$").
				WithResultCallback(func(response string, submitted bool) {
					confirmation.confirmed = submitted && strings.EqualFold(strings.TrimSpace(response), "y")
					confirmation.choiceMade = true
				}).
				WithAutoReplace(true).
				WithTextLines(5).
				WithCharsPerLine(42).
				AutoCloseAndPop(&g.dialogStack).
				Build()
			g.dialogStack.PushModalDialog(confirmation.dialog)
		}).While(
			func() bool { return !confirmation.choiceMade },
			func() {},
			DialogPollingRateMs,
		).If(
			func() bool { return confirmation.confirmed },
			func(confirmQ *event_queue.SequentialEventBuilder) {
				confirmQ.Then(func() {
					g.gameState.SystemCallbacks.Message.AddRowStr("Plans")
					switch g.gameState.CommitHMSCapeUse(commit) {
					case game_state.HMSCapeUseNoPlans, game_state.HMSCapeUseReady:
						panic("HMS Cape commit returned a review-only result")
					case game_state.HMSCapeUseEnabled:
						confirmation.committed = true
						g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundSummon)
						g.gameState.SystemCallbacks.UI.ShowActionFeedback("You put the H.M.S. Cape plans to use.", true)
					case game_state.HMSCapeUseDisabled:
						confirmation.committed = true
						g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundSummon)
						g.gameState.SystemCallbacks.UI.ShowActionFeedback("You put the H.M.S. Cape plans aside.", true)
					case game_state.HMSCapeUseStale:
						g.gameState.SystemCallbacks.UI.ShowActionFeedback("You cannot use the plans now.", true)
					default:
						panic("unsupported HMS Cape commit result")
					}
				})
			},
		)
	}
}

func hmsCapeConfirmationPrompt(enable bool) string {
	action := "Put these plans to use"
	if !enable {
		action = "Put these plans aside"
	}

	return "H.M.S. Cape Plans\n\n" + action + "? (Y/N)"
}

func (g *GameScene) cleanupHMSCapeConfirmation(confirmation *hmsCapeConfirmationState) {
	if confirmation != nil && confirmation.dialog != nil {
		g.dialogStack.RemoveWidgetInstance(confirmation.dialog)
	}
}
