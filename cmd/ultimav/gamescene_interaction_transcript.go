package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

func (g *GameScene) thenPromptInteractionYesNo(
	dialog *widgets.InteractionTranscript,
	prompt string,
	onYes func(*event_queue.SequentialEventBuilder),
	onNo func(*event_queue.SequentialEventBuilder),
) func(*event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
	return g.thenPromptInteractionYesNoWithBegin(dialog, prompt, dialog.BeginYesNo, onYes, onNo)
}

func (g *GameScene) thenPromptInteractionUppercaseYesNo(
	dialog *widgets.InteractionTranscript,
	prompt string,
	onYes func(*event_queue.SequentialEventBuilder),
	onNo func(*event_queue.SequentialEventBuilder),
) func(*event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
	return g.thenPromptInteractionYesNoWithBegin(dialog, prompt, dialog.BeginUppercaseYesNo, onYes, onNo)
}

func (g *GameScene) thenPromptInteractionYesNoWithBegin(
	dialog *widgets.InteractionTranscript,
	prompt string,
	begin func(string),
	onYes func(*event_queue.SequentialEventBuilder),
	onNo func(*event_queue.SequentialEventBuilder),
) func(*event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		var choseYes bool

		return q.Then(func() {
			begin(prompt)
		}).While(
			func() bool { return !dialog.InputComplete() },
			func() {},
			DialogPollingRateMs,
		).Then(func() {
			choseYes = dialog.Response() == "Yes"
			g.keyboard.SetForceWaitAnyKey(KeyboardDebounceMs)
		}).Wait(KeyboardDebounceMs).If(
			func() bool { return choseYes },
			onYes,
		).If(
			func() bool { return !choseYes },
			onNo,
		)
	}
}

func (g *GameScene) thenWaitForInteractionAcknowledgement(
	dialog *widgets.InteractionTranscript,
) func(*event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		return q.Then(func() {
			dialog.BeginAcknowledgement()
		}).While(
			func() bool { return !dialog.InputComplete() },
			func() {},
			DialogPollingRateMs,
		)
	}
}

func (g *GameScene) thenPromptInteractionText(
	dialog *widgets.InteractionTranscript,
	prompt string,
	onResult func(text string, submitted bool),
) func(*event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
	return g.thenPromptInteractionTextWithValidation(dialog, prompt, ".*", onResult)
}

func (g *GameScene) thenPromptInteractionTextWithValidation(
	dialog *widgets.InteractionTranscript,
	prompt string,
	validation string,
	onResult func(text string, submitted bool),
) func(*event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		return q.Then(func() {
			dialog.BeginText(prompt, validation)
		}).While(
			func() bool { return !dialog.InputComplete() },
			func() {},
			DialogPollingRateMs,
		).Then(func() {
			onResult(dialog.Response(), dialog.Submitted())
			g.keyboard.SetForceWaitAnyKey(KeyboardDebounceMs)
		}).Wait(KeyboardDebounceMs)
	}
}

func (g *GameScene) thenWaitForInteractionContinuation(
	dialog *widgets.InteractionTranscript,
) func(*event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		return q.Then(func() {
			dialog.BeginContinue()
		}).While(
			func() bool { return !dialog.InputComplete() },
			func() {},
			DialogPollingRateMs,
		)
	}
}
