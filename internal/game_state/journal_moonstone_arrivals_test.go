package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMoonstoneJournalArrivalSamplesSuccessfulDestination(t *testing.T) {
	for _, source := range []journal.MoonstoneSource{journal.MoonstoneSourceGate, journal.MoonstoneSourceSpell} {
		for _, test := range []struct {
			name               string
			origin, target     references.Location
			floor              uint8
			departure, arrival uint8
			minute             uint8
			sextant, measured  bool
		}{
			{"dawn boundary", references.BritanniaUnderworld, references.BritanniaUnderworld, 0, 5, 6, 0, true, false},
			{"dusk boundary", references.BritanniaUnderworld, references.BritanniaUnderworld, 0, 18, 19, 0, true, true},
			{"last night minute", references.BritanniaUnderworld, references.BritanniaUnderworld, 0, 5, 5, 59, true, true},
			{"last day minute", references.BritanniaUnderworld, references.BritanniaUnderworld, 0, 18, 18, 59, true, false},
			{"town destination", references.BritanniaUnderworld, references.Britain, 1, 19, 19, 0, true, false},
			{"underworld destination", references.BritanniaUnderworld, references.BritanniaUnderworld, 255, 19, 19, 0, true, false},
			{"dungeon destination", references.BritanniaUnderworld, references.Deceit, 1, 19, 19, 0, true, false},
			{"town departure", references.Britain, references.BritanniaUnderworld, 0, 19, 19, 0, true, true},
			{"missing instrument", references.BritanniaUnderworld, references.BritanniaUnderworld, 0, 19, 19, 0, false, false},
		} {
			t.Run(string(source)+"/"+test.name, func(t *testing.T) {
				g, callbacks := newMoonstoneJournalTestState(t)
				g.TransitionToLocation(test.origin, 0, references.Position{X: 3, Y: 4}, TransitionFullLocation, false)
				if !test.sextant {
					g.PartyState.Inventory.QuestItems.Set(references.Sextant, 0)
				}
				g.DateTime.Hour, g.DateTime.Minute = test.departure, 59
				// Capture identity before time advances. Colocation must not select phase zero.
				g.moonstoneSystem.BuryMoonstone(0, 3, 4, uint8(test.target), test.floor)
				g.moonstoneSystem.BuryMoonstone(7, 3, 4, uint8(test.target), test.floor)
				destination, err := g.GetMoongateDestinationForPhase(datetime.MoonPhaseCrescentWaning)
				require.NoError(t, err)
				sequence := g.BuildMoongateTeleportationSequence(destination)
				if source == journal.MoonstoneSourceSpell {
					sequence = g.BuildMoonstoneSpellTeleportationSequence(destination)
				}
				assert.Zero(t, g.moonstoneJournal, "building a sequence is not arrival")
				g.DateTime.Hour, g.DateTime.Minute = test.arrival, test.minute
				arrivalClock := g.DateTime
				at := journal.OptionalGameTime{Known: true, Time: journal.GameTime{
					Year: 139, Month: 4, Day: 12, Hour: int(test.arrival), Minute: int(test.minute)}}
				calls := g.GetRNG().Calls()
				callbacks.Reset()
				var committed journal.MoonstoneObservation
				g.SystemCallbacks.Visual.StabilizeTeleportationDestinationRenderer = func(MoongateDestination) {
					assert.Equal(t, arrivalClock, g.DateTime, "arrival adds no gameplay time")
					committed = g.moonstoneJournal[7]
					require.True(t, committed.Known, "knowledge exists before destination presentation")
					assert.Equal(t, at, committed.ObservedAt)
					assert.Equal(t, source, committed.FirstSource)
					assert.Equal(t, moonstoneJournalSite(g.MapState.PlayerLocation), committed.Site)
					assert.Equal(t, committed.Site, committed.LastContext.Observer)
					assert.Equal(t, test.measured, committed.Survey.Measured)
					// Cosmetic completion cannot resample eligibility or provenance.
					g.DateTime.Hour = (test.arrival + 12) % 24
				}
				g.EventQueueManager.Workflow("journal_arrival").Priority(event_queue.PriorityHigh).
					Sequential().First(func() {}).ThenSequence(sequence).Finally(func() {}).Start()
				for range 20 {
					g.ElapsedMs += 1000
					g.EventQueueManager.Update()
				}
				require.False(t, g.EventQueueManager.HasActiveWorkflows())
				require.True(t, committed.Known)
				assert.Equal(t, committed, g.moonstoneJournal[7])
				for phase := range 7 {
					assert.Zero(t, g.moonstoneJournal[phase])
				}
				assert.Equal(t, calls, g.GetRNG().Calls())
				arrivalClock.Hour = (test.arrival + 12) % 24
				assert.Equal(t, arrivalClock, g.DateTime, "presentation adds no gameplay time")
			})
		}
	}
}

func TestMoonstoneJournalTeleportCancellationAndGenericExclusion(t *testing.T) {
	for _, mode := range []string{"before arrival", "after arrival", "generic"} {
		t.Run(mode, func(t *testing.T) {
			g, _ := newMoonstoneJournalTestState(t)
			destination, err := g.GetMoongateDestinationForPhase(datetime.MoonPhaseFull)
			require.NoError(t, err)
			sequence := g.BuildMoonstoneSpellTeleportationSequence(destination)
			if mode == "generic" {
				sequence = g.BuildTeleportationSequence(destination)
			}
			arrived := false
			g.SystemCallbacks.Visual.StabilizeTeleportationDestinationRenderer = func(MoongateDestination) { arrived = true }
			g.EventQueueManager.Workflow("cancelled_arrival").Priority(event_queue.PriorityHigh).Sequential().First(func() {}).ThenSequence(sequence).
				Finally(func() {}).Start()
			g.EventQueueManager.Update()
			if mode != "before arrival" {
				for step := 0; step < 20 && !arrived; step++ {
					g.ElapsedMs += 1000
					g.EventQueueManager.Update()
				}
				require.True(t, arrived)
			}
			before := g.moonstoneJournal
			g.EventQueueManager.Shutdown()
			assert.Equal(t, before, g.moonstoneJournal)
			if mode == "after arrival" {
				assert.True(t, before[4].Known && before[4].Survey.Measured)
			} else {
				assert.Zero(t, before)
			}
		})
	}
}

func TestMoonstoneJournalEligibleSearchTargetsOnlyExposedPhase(t *testing.T) {
	g, callbacks := newMoonstoneJournalTestState(t)
	g.moongateStage = MoongateStageHidden // 19:00 permits sextant use before gates grow at 20:00.
	g.moonstoneSystem.BuryMoonstone(1, 224, 133, 0, 0)
	clock, calls := g.DateTime, g.GetRNG().Calls()
	require.True(t, g.ActionSearchLargeMap(references.Right))
	entry := g.CurrentQuestJournalDossier().Moonstones[0]
	require.True(t, entry.LastObservedSurvey.Measured)
	assert.Equal(t, 224, entry.LastObservedSurvey.Location.X)
	assert.Equal(t, 133, entry.LastObservedSurvey.Location.Y)
	assert.Equal(t, journal.MoonstoneSourceSearch, entry.SurveySource)
	assert.Zero(t, g.moonstoneJournal[1])
	clock.Minute++
	clock.Turn++
	assert.Equal(t, clock, g.DateTime)
	assert.Equal(t, calls, g.GetRNG().Calls())
	assert.Equal(t, []int{1}, callbacks.TimeAdvanced())
}
