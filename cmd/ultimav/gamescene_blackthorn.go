package main

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/minimap"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

const (
	blackthornTortureStepMs         = 170
	blackthornTortureOpeningWaitMs  = 1100
	blackthornTorturePendulumWaitMs = 150
	blackthornTortureGuardWaitMs    = 600
)

type blackthornActorMove struct {
	actorID   minimap.ActorID
	direction references.Direction
}

func (g *GameScene) startBlackthornInterrogationWorkflow() {
	g.startBlackthornInterrogationWorkflowWithReview(nil)
}

func (g *GameScene) startBlackthornInterrogationWorkflowWithReview(
	prepared *game_state.BlackthornInterrogationReview,
) {
	if g.gameState.EventQueueManager.HasWorkflow("blackthorn_interrogation") {
		return
	}
	workflowState := new(merchantTalkWorkflowState)
	transcriptPlacement := cinematicTranscriptPlacement()
	dialog := widgets.NewInteractionTranscript(g.keyboard, widgets.InteractionTranscriptConfig{
		MaxLines:       6,
		MaxInputLength: 14,
		PanelPlacement: &transcriptPlacement,
		// Original text entry: Escape clears the editor; only Return submits an answer.
		ClearTextOnEscape: true,
		WidthSamples: []string{
			game_state.BlackthornDraggedMessage,
			"Resistance is futile! Thou must yield the truth unto me!",
			"Your response?  > VERAMOCOR",
			game_state.BlackthornMercifulDeathMessage,
		},
	})
	workflowState.interactionTranscript = dialog
	var review game_state.BlackthornInterrogationReview
	var beginErr error
	var outcome game_state.BlackthornAnswerOutcome
	var answerErr error
	resolved := false

	workflow := g.gameState.EventQueueManager.Workflow("blackthorn_interrogation").
		Priority(event_queue.PriorityNormal).
		Sequential().BlockInput().
		First(func() {
			g.dialogStack.PushModalDialog(dialog)
			if prepared == nil {
				review, beginErr = g.gameState.BeginBlackthornInterrogation()
			} else {
				review = *prepared
			}
			if beginErr != nil {
				dialog.AppendLine("The guards lose their way!")

				return
			}
			dialog.AppendLine(game_state.BlackthornSubduedMessage)
			g.gameState.SystemCallbacks.Message.AddRowStr(game_state.BlackthornSubduedMessage)
			if !review.NeedsAnswer {
				return
			}
			g.miniMapScene = minimap.NewMiniMapScene()
			for index, position := range blackthornPartyPositions(review.LivingCount) {
				character := g.gameState.PartyState.GetCharacter(index)
				sprite := character.GetKeySpriteIndex()
				if sprite == 0 {
					sprite = indexes.AvatarKeyIndex
				}
				sprite = g.gameState.GetNPCSpriteReplacement(indexes.Manacles, sprite, position).Sprite
				actorID := minimap.ActorID(fmt.Sprintf("prisoner-%d", index))
				if index == 0 {
					actorID = minimap.ActorAvatar
				}
				g.miniMapScene.AddActor(actorID, position, sprite)
			}
			g.miniMapScene.
				AddActor("blackthorn", references.Position{X: 5, Y: 5}, indexes.Blackthorn1).
				AddActor("guard-left", references.Position{X: 1, Y: 5}, indexes.Guard_KeyIndex).
				AddActor("guard-right", references.Position{X: 9, Y: 9}, indexes.Guard_KeyIndex).
				AddActor("hourglass", references.Position{X: 5, Y: 9}, indexes.Hourglass1)
			dialog.AppendLine(game_state.BlackthornDraggedMessage)
			dialog.AppendLine(game_state.BlackthornChainedMessage)
			dialog.AppendLine(game_state.BlackthornFootstepsMessage)
			dialog.AppendLine(fmt.Sprintf(
				"Blackthorn says:\n\n\"Ah, %s!\n'Tis indeed an honour to meet thee at last!",
				review.AvatarName,
			))
		}).If(
		func() bool { return beginErr == nil && review.NeedsAnswer },
		func(q *event_queue.SequentialEventBuilder) {
			q.ThenSequence(g.thenWaitForInteractionContinuation(dialog)).Then(func() {
				title := "man"
				if review.AvatarFemale {
					title = "lady"
				}
				dialog.AppendLine("GUARD! Release this good " + title + " at once!\"")
				dialog.AppendLine(game_state.BlackthornQuestMessage)
			}).ThenSequence(g.thenWaitForInteractionContinuation(dialog))
		},
	)

	for attempt := range 4 {
		workflow.If(
			func() bool { return beginErr == nil && review.NeedsAnswer && !resolved },
			func(q *event_queue.SequentialEventBuilder) {
				q.Then(func() {
					dialog.AppendLine(game_state.BlackthornQuestion(attempt, review.VirtueName))
				}).ThenSequence(g.thenPromptInteractionText(
					dialog,
					"Your response?",
					func(response string, _ bool) {
						outcome, answerErr = g.gameState.SubmitBlackthornAnswer(review, attempt, response)
						resolved = answerErr != nil || outcome.Resolved
					},
				)).Then(func() {
					if answerErr != nil {
						dialog.AppendLine("Blackthorn's question is lost!")

						return
					}
					if !outcome.Correct && !outcome.LoneLiving {
						switch attempt {
						case 1:
							g.miniMapScene.SetActorSprite("hourglass", indexes.Hourglass4)
						case 2:
							g.miniMapScene.SetActorSprite("hourglass", indexes.Hourglass1)
						}
					}
					switch {
					case outcome.Correct && outcome.LoneLiving:
						dialog.AppendLine(game_state.BlackthornLoneCorrectMessage)
					case outcome.Correct:
						dialog.AppendLine(game_state.BlackthornMercifulDeathMessage)
					case outcome.LoneLiving:
						dialog.AppendLine(game_state.BlackthornLoneReleaseMessage)
					case outcome.StartedTorture:
						dialog.AppendLine(game_state.BlackthornTortureMessage)
					case outcome.Resolved:
						dialog.AppendLine(game_state.BlackthornPendulumMessage)
						dialog.AppendLine(outcome.MartyrName + " is sliced in half!")
						g.miniMapScene.SetActorSprite("pendulum", indexes.TortureChair1)
					}
				}).If(
					func() bool { return answerErr == nil && outcome.StartedTorture },
					func(waitQ *event_queue.SequentialEventBuilder) {
						waitQ.ThenSequence(g.thenRunBlackthornTortureAnimation()).Then(func() {
							dialog.AppendLine("\"I will ask thee until the sand has fallen. And then will " +
								g.gameState.PartyState.GetCharacter(1).GetNameAsString() + " die!\"")
						}).ThenSequence(g.thenWaitForInteractionContinuation(dialog))
					},
				).If(
					func() bool { return answerErr == nil && outcome.Resolved && !outcome.StartedTorture },
					func(waitQ *event_queue.SequentialEventBuilder) {
						waitQ.ThenSequence(g.thenWaitForInteractionContinuation(dialog))
					},
				).If(
					func() bool {
						return answerErr == nil && outcome.Resolved && !outcome.Correct && !outcome.LoneLiving
					},
					func(waitQ *event_queue.SequentialEventBuilder) {
						waitQ.Then(func() {
							dialog.AppendLine(game_state.BlackthornUnfairMessage)
						}).ThenSequence(g.thenWaitForInteractionContinuation(dialog))
					},
				)
			},
		)
	}

	workflow.Finally(func() {
		g.miniMapScene = nil
		g.gameState.FinishBlackthornInterrogation()
		g.finishMerchantWorkflow(workflowState)
	}).Start()
}

func (g *GameScene) thenRunBlackthornTortureAnimation() func(*event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
	const (
		companionID  minimap.ActorID = "prisoner-1"
		guardLeftID  minimap.ActorID = "guard-left"
		guardRightID minimap.ActorID = "guard-right"
	)

	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		q.Wait(blackthornTortureOpeningWaitMs)

		addMoveRound := func(moves ...blackthornActorMove) {
			q.Then(func() {
				for _, move := range moves {
					actor := g.miniMapScene.GetActor(move.actorID)
					if actor == nil {
						panic("thenRunBlackthornTortureAnimation: actor is unavailable")
					}
					target := blackthornMoveTarget(actor.Position, move.direction)
					g.miniMapScene.SetActorPosition(move.actorID, target)
					g.PlayWalkingSoundAt(target)
				}
			}).Wait(blackthornTortureStepMs)
		}
		addRepeatedMove := func(count int, moves ...blackthornActorMove) {
			for range count {
				addMoveRound(moves...)
			}
		}

		addMoveRound(blackthornActorMove{actorID: guardLeftID, direction: references.Down})
		addRepeatedMove(7, blackthornActorMove{actorID: guardLeftID, direction: references.Right})
		q.Then(func() {
			companionSprite := g.gameState.PartyState.GetCharacter(1).GetKeySpriteIndex()
			if companionSprite == 0 {
				companionSprite = indexes.AvatarKeyIndex
			}
			g.miniMapScene.SetActorSprite(companionID, companionSprite)
		})
		addMoveRound(blackthornActorMove{actorID: companionID, direction: references.Down})
		addRepeatedMove(2,
			blackthornActorMove{actorID: guardLeftID, direction: references.Left},
			blackthornActorMove{actorID: companionID, direction: references.Left},
		)
		addMoveRound(blackthornActorMove{actorID: companionID, direction: references.Down})
		q.Wait(blackthornTorturePendulumWaitMs).Then(func() {
			g.miniMapScene.SetActorVisible(companionID, false)
			g.miniMapScene.AddActor(
				"pendulum", references.Position{X: 5, Y: 7}, indexes.TortureTableWithBody1,
			)
		})
		addRepeatedMove(5, blackthornActorMove{actorID: guardLeftID, direction: references.Left})
		addMoveRound(blackthornActorMove{actorID: guardLeftID, direction: references.Up})
		q.Wait(blackthornTortureGuardWaitMs)
		addRepeatedMove(3, blackthornActorMove{actorID: guardRightID, direction: references.Left})
		q.Then(func() {
			g.miniMapScene.SetActorSprite("hourglass", indexes.Hourglass2)
		})
		addRepeatedMove(3, blackthornActorMove{actorID: guardRightID, direction: references.Right})

		return q
	}
}

func blackthornMoveTarget(position references.Position, direction references.Direction) references.Position {
	switch direction {
	case references.Up:
		return position.Up()
	case references.Right:
		return position.Right()
	case references.Down:
		return position.Down()
	case references.Left:
		return position.Left()
	case references.NoneDirection:
		panic("blackthornMoveTarget: direction is required")
	}

	panic("blackthornMoveTarget: unknown direction")
}

func blackthornPartyPositions(livingCount int) []references.Position {
	manacleIndexes := [6][]int{
		{4},
		{4, 5},
		{4, 5, 0},
		{4, 5, 3, 0},
		{4, 5, 2, 1, 0},
		{4, 5, 3, 2, 1, 0},
	}
	manaclePositions := [6]references.Position{
		{X: 0, Y: 1}, {X: 1, Y: 1}, {X: 9, Y: 1},
		{X: 10, Y: 1}, {X: 3, Y: 5}, {X: 7, Y: 5},
	}
	if livingCount < 1 || livingCount > len(manacleIndexes) {
		return nil
	}

	positions := make([]references.Position, 0, livingCount)
	for _, index := range manacleIndexes[livingCount-1] {
		positions = append(positions, manaclePositions[index])
	}

	return positions
}
