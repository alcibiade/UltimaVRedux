package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/require"
)

func TestSoloInputProjectionDoesNotClaimSchedulerOrConsumeRandomness(t *testing.T) {
	state, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	forceNextMonsterTurnForTest(t, state, 0)
	state.CombatScene.SetSoloMoveCharacter(0)
	state.BeginCombatTurns(CombatTurnsPresented)
	randomBefore := state.GetRNG().Calls()
	dateBefore := state.DateTime
	first, eligible := state.ProjectSoloCombatInput()
	require.True(t, eligible)
	require.True(t, first.Waiting)
	for range 3 {
		again, ok := state.ProjectSoloCombatInput()
		require.True(t, ok)
		require.Equal(t, first, again)
	}
	require.Equal(t, randomBefore, state.GetRNG().Calls())
	require.Equal(t, dateBefore, state.DateTime)
	step, reviewed := state.ReviewScheduledCombatStep()
	require.True(t, reviewed, "passive input projection must leave review unclaimed")
	for range 100 {
		if step.Kind == CombatPresentationMonsterTurn {
			_, err := state.ResolveScheduledCombatStep(step.Token)
			require.NoError(t, err)
		}
		require.NoError(t, state.CompleteScheduledCombatStep(step.Token))
		projection, ok := state.ProjectSoloCombatInput()
		require.True(t, ok)
		require.Equal(t, first.Opportunity, projection.Opportunity)
		if projection.Ready {
			return
		}
		step, reviewed = state.ReviewScheduledCombatStep()
		require.True(t, reviewed)
	}
	t.Fatal("selected opportunity was not reached")
}

func TestSoloInputProjectionRejectsOwnActionSkippedOpportunityAndReplacedEncounter(t *testing.T) {
	state, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	state.CombatScene.SetSoloMoveCharacter(0)
	forceNextCombatActor(state.CombatScene.TurnQueue, 0, true)
	state.BeginCombatTurns(CombatTurnsPresented)
	ready, eligible := state.ProjectSoloCombatInput()
	require.True(t, eligible)
	require.True(t, ready.Ready)
	completion := state.FinishCombatTurnDeferred()
	_, eligible = state.ProjectSoloCombatInput()
	require.False(t, eligible, "the player's committed action cannot admit typeahead")
	require.NoError(t, state.CompleteScheduledCombatStep(completion))
	state.CombatScene.PartySleeping[0] = true
	_, eligible = state.ProjectSoloCombatInput()
	require.False(t, eligible)
	state.CombatScene.PartySleeping[0] = false
	state.CombatScene.EncounterID++
	changed, eligible := state.ProjectSoloCombatInput()
	if eligible {
		require.NotEqual(t, ready.Opportunity, changed.Opportunity)
	}
	state.CombatScene.ClearSoloMoveMode()
	_, eligible = state.ProjectSoloCombatInput()
	require.False(t, eligible)
}

func TestCombatInputGenerationCountsOnlyActualEntryDispatchIncludingSkippedTurns(t *testing.T) {
	queue := NewCombatTurnQueue()
	player := newMockPC(35)
	queue.AddCombatant(player, 0)
	queue.AddCombatant(newMockMonster(35), 0)
	selected := queue.GetEntry(0, true)
	enemy := queue.GetEntry(0, false)
	require.Zero(t, selected.dispatchGeneration)
	require.Zero(t, enemy.dispatchGeneration)
	require.Same(t, selected, queue.AdvanceToNextActor())
	require.Equal(t, uint64(1), selected.dispatchGeneration)
	require.Zero(t, enemy.dispatchGeneration)
	player.canTakeTurn = false
	require.Same(t, enemy, queue.AdvanceToNextActor())
	require.Equal(t, uint64(1), selected.dispatchGeneration, "enemy dispatch cannot age the selected entry")
	require.Equal(t, uint64(1), enemy.dispatchGeneration)
	require.Same(t, selected, queue.AdvanceToNextActor())
	require.Equal(t, uint64(2), selected.dispatchGeneration, "non-actionable scheduled opportunities still count")
	queue.MarkDead(0, true)
	require.Same(t, enemy, queue.AdvanceToNextActor())
	require.Equal(t, uint64(2), selected.dispatchGeneration, "unselected dead entries do not dispatch")
}

type inputProjectionDexObserver struct {
	CombatantProvider
	reads int
}

func (provider *inputProjectionDexObserver) GetDexterity() int {
	provider.reads++

	return provider.CombatantProvider.GetDexterity()
}

func TestSoloInputProjectionDoesNotReadFutureActorDexterityOrMutateQueue(t *testing.T) {
	state, _ := newSoloInputGenerationTestState(t)
	queue := state.CombatScene.TurnQueue
	var observers []*inputProjectionDexObserver
	for _, entry := range queue.entries {
		observer := &inputProjectionDexObserver{CombatantProvider: entry.Combatant}
		entry.Combatant = observer
		observers = append(observers, observer)
	}
	before := make([]TurnQueueEntry, len(queue.entries))
	for index, entry := range queue.entries {
		before[index] = *entry
	}
	cursor, actions := queue.currentIndex, queue.actionCount
	rngCalls, date := state.GetRNG().Calls(), state.DateTime
	for range 3 {
		projection, eligible := state.ProjectSoloCombatInput()
		require.True(t, eligible)
		require.True(t, projection.Waiting)
	}
	for index, observer := range observers {
		require.Zero(t, observer.reads, "input must not simulate even one future DEX reset")
		require.Equal(t, before[index], *queue.entries[index])
	}
	require.Equal(t, cursor, queue.currentIndex)
	require.Equal(t, actions, queue.actionCount)
	require.Equal(t, rngCalls, state.GetRNG().Calls())
	require.Equal(t, date, state.DateTime)
	require.False(t, state.CombatScene.pendingPresentationStep.reviewed)
}

func TestSoloInputOpportunitySurvivesInterveningEnemyChanges(t *testing.T) {
	for _, change := range []string{"enemy killed", "enemy removed", "enemy DEX changed", "Time Stop"} {
		t.Run(change, func(t *testing.T) {
			state, secondEnemy := newSoloInputGenerationTestState(t)
			combat := state.CombatScene
			selected := combat.TurnQueue.GetEntry(0, true)
			before, eligible := state.ProjectSoloCombatInput()
			require.True(t, eligible)
			require.True(t, before.Waiting)
			require.Equal(t, selected.dispatchGeneration+1, before.Opportunity.generation)
			switch change {
			case "enemy killed":
				secondEnemy.SetCurrentHP(0)
				combat.TurnQueue.MarkDead(1, false)
			case "enemy removed":
				combat.TurnQueue.MarkFled(1, false)
				require.Same(t, secondEnemy, combat.MonsterManager.RemoveMonster(1))
			case "enemy DEX changed":
				secondEnemy.currentDex = 1
			case "Time Stop":
				require.True(t, state.ActivateTimeStopEffect(20))
			}
			afterChange, eligible := state.ProjectSoloCombatInput()
			require.True(t, eligible)
			require.Equal(t, before.Opportunity, afterChange.Opportunity, "an enemy change cannot retire the selected PC's next opportunity")
			sawSuppressedEnemy := false
			for stepCount := 0; stepCount < 20; stepCount++ {
				step, reviewed := state.ReviewScheduledCombatStep()
				require.True(t, reviewed)
				require.False(t, step.Actor.IsPartyMember)
				if change == "Time Stop" && step.Actor.EntityIndex == 1 {
					sawSuppressedEnemy = true
					require.Equal(t, combatPresentationIdleMonster, combat.pendingPresentationStep.action)
				}
				_, err := state.ResolveScheduledCombatStep(step.Token)
				require.NoError(t, err)
				require.NoError(t, state.CompleteScheduledCombatStep(step.Token))
				now, valid := state.ProjectSoloCombatInput()
				require.True(t, valid)
				require.Equal(t, before.Opportunity, now.Opportunity, "ordinary intervening steps retain the same exact token")
				if now.Ready {
					break
				}
				require.Equal(t, before.Opportunity.generation-1, selected.dispatchGeneration)
			}
			ready, eligible := state.ProjectSoloCombatInput()
			require.True(t, eligible)
			require.True(t, ready.Ready)
			require.Equal(t, before.Opportunity, ready.Opportunity)
			require.Equal(t, before.Opportunity.generation, selected.dispatchGeneration)
			if change == "Time Stop" {
				require.True(t, sawSuppressedEnemy, "the fixture must actually dispatch a Time Stop-skipped enemy")
			}
			state.FinishCombatAction()
			next, eligible := state.ProjectSoloCombatInput()
			require.True(t, eligible)
			require.NotEqual(t, before.Opportunity, next.Opportunity, "the token cannot replay after its selected opportunity completes")
			require.Equal(t, before.Opportunity.generation+1, next.Opportunity.generation)
		})
	}
}

func TestSoloInputSkippedOpportunityCannotReplayAfterSameDispatchWake(t *testing.T) {
	state, _ := newSoloInputGenerationTestState(t)
	combat := state.CombatScene
	waiting, eligible := state.ProjectSoloCombatInput()
	require.True(t, eligible)
	step, reviewed := state.ReviewScheduledCombatStep()
	require.True(t, reviewed)
	_, err := state.ResolveScheduledCombatStep(step.Token)
	require.NoError(t, err)
	// Sleep is acquired and recovered entirely between input observations. The
	// queued opportunity still expires: waking consumes that scheduled turn.
	forceNextCombatActor(combat.TurnQueue, 0, true)
	state.setCombatPCSleeping(waiting.Opportunity.Actor, true)
	seed := findCombatWakeSeed(t, state, combatPCWakeRollMaximum, func(roll int) bool {
		return roll < combatPCWakeThreshold
	})
	state.SetRandomSeed(seed)
	require.NoError(t, state.CompleteScheduledCombatStep(step.Token))
	require.False(t, combat.isCombatPCSleeping(waiting.Opportunity.Actor))
	require.Equal(t, waiting.Opportunity.generation, combat.TurnQueue.GetEntry(0, true).dispatchGeneration)
	_, eligible = state.ProjectSoloCombatInput()
	require.False(t, eligible, "a recovered-but-skipped turn is not ready input")
	skipped, reviewed := state.ReviewScheduledCombatStep()
	require.True(t, reviewed)
	require.Equal(t, CombatPresentationAutoPass, skipped.Kind)
	require.Equal(t, waiting.Opportunity.Actor, skipped.Actor)
	forceNextCombatActor(combat.TurnQueue, 0, false)
	require.NoError(t, state.CompleteScheduledCombatStep(skipped.Token))
	afterSkip, eligible := state.ProjectSoloCombatInput()
	require.True(t, eligible)
	require.True(t, afterSkip.Waiting)
	require.Equal(t, waiting.Opportunity.generation+1, afterSkip.Opportunity.generation)
	require.NotEqual(t, waiting.Opportunity, afterSkip.Opportunity)
	forceNextCombatActor(combat.TurnQueue, 0, true)
	step, reviewed = state.ReviewScheduledCombatStep()
	require.True(t, reviewed)
	_, err = state.ResolveScheduledCombatStep(step.Token)
	require.NoError(t, err)
	require.NoError(t, state.CompleteScheduledCombatStep(step.Token))
	ready, eligible := state.ProjectSoloCombatInput()
	require.True(t, eligible)
	require.True(t, ready.Ready)
	require.Equal(t, afterSkip.Opportunity, ready.Opportunity)
	require.NotEqual(t, waiting.Opportunity, ready.Opportunity, "the recovered actor's later ready opportunity must not accept old input")
}

// The second enemy is due several times before the selected PC. Mutating its
// participation or DEX therefore changes intervening dispatch counts, while the
// selected entry's actual next opportunity remains the same.
func newSoloInputGenerationTestState(t *testing.T) (*GameState, *CombatMonster) {
	t.Helper()
	state, _, firstEnemy := setupCombatWithMonster(t, references.MonsterRat, references.Down, 42)
	combat := state.CombatScene
	for index := 1; index < party_state.MaxActiveParty; index++ {
		combat.PartyActive[index] = false
		combat.TurnQueue.MarkFled(index, true)
	}
	character := state.PartyState.GetCharacter(0)
	character.SetStatus(party_state.Good)
	character.SetMaxHp(1000)
	character.SetHp(1000)
	character.Dexterity = 30
	combat.PartySleeping[0], combat.PartyPossessed[0] = false, false
	combat.SetPartyMemberPosition(0, references.Position{X: 5, Y: 5})
	firstEnemy.SetPosition(references.Position{X: 10, Y: 10})
	firstEnemy.currentDex = 30
	ref := *firstEnemy.EnemyRef()
	secondEnemy := NewCombatMonster(references.MonsterRat, &ref, references.Position{X: 10, Y: 9}, newFixedRNG(1))
	secondEnemy.currentDex = 35
	require.Equal(t, 1, combat.AddMonsterToQueue(secondEnemy))
	forceNextCombatActor(combat.TurnQueue, 0, false)
	combat.TurnQueue.GetEntry(0, true).turnCountdown = 4
	combat.TurnQueue.GetEntry(1, false).turnCountdown = 1
	combat.SetSoloMoveCharacter(0)
	state.BeginCombatTurns(CombatTurnsPresented)
	require.Equal(t, CombatTurnIdentity{EntityIndex: 0}, combat.GetCurrentTurnIdentity())
	require.True(t, combat.HasScheduledCombatStep())

	return state, secondEnemy
}
