package audio

import (
	"errors"
	"fmt"
	"maps"
	"math"
	"time"
)

// Provisional listening values, relative to the calibrated track mix.
// These are presentation policy, never user volume or gameplay time/RNG.
const (
	musicQuietInterval = 5 * time.Minute
	musicFadeDuration  = 2 * time.Second
	musicLocalRelease  = 3 * time.Second
	musicInitialDB     = -10.0
	musicRepeatDB      = -16.0
	musicCombatDB      = 0.0 // Combat uses its authored cue mix, without exploration attenuation.
	musicDBDivisor     = 20.0
	musicDBBase        = 10.0
	musicCombatTrack   = "combat"
)

// MusicSuppression identifies independent presentation owners. Releasing one
// owner cannot release another owner's interruption.
type MusicSuppression string

// Independent scene owners are distinct from workflow, mute and device holds.
const (
	MusicMenu      MusicSuppression = "menu"
	MusicJournal   MusicSuppression = "journal"
	MusicCinematic MusicSuppression = "cinematic"
	musicSystem    MusicSuppression = "disabled"
	musicMuted     MusicSuppression = "muted"
	musicWorkflow  MusicSuppression = "workflow"
)

// MusicHistory is detached, application-session presentation memory. It is not
// part of saves: loading cannot restore players, interruptions or louder intros.
// A theme gets its subdued introduction only once per application session.
type MusicHistory struct {
	heard          map[string]bool
	quietRemaining time.Duration
	disabled       bool
}

type musicPlayback interface {
	Update() error
	Play(track string) error
	Stop() error
	IsPlaying() bool
	Pause() time.Duration
	Resume(track string, position time.Duration) error
	SetVolume(gain float64) error
}

// musicPolicy owns one finite score voice. The caller serializes access; tests
// supply a fake player and clock and never wait for a device or real wall time.
type musicPolicy struct {
	output          musicPlayback
	now             func() time.Time
	last            time.Time
	history         MusicHistory
	holds           map[MusicSuppression]bool
	exploration     string
	cue             string
	track           string
	trackIsCue      bool
	gain            float64
	envelope        float64
	suspended       bool
	position        time.Duration
	localRemaining  time.Duration
	combatEncounter uint64
	combatFinished  bool
	combatVictory   bool
	combatTrack     string
	sailingRaised   bool
	sailingPlayed   bool
	sailingTrack    string
	sailingEpoch    uint64
	trackSailEpoch  uint64
}

func newMusicPolicy(output musicPlayback, now func() time.Time) *musicPolicy {
	if output == nil || now == nil {
		panic("music policy requires playback and a presentation clock")
	}

	return &musicPolicy{
		output: output, now: now, last: now(),
		history: MusicHistory{heard: make(map[string]bool), quietRemaining: 0, disabled: false},
		holds:   make(map[MusicSuppression]bool), exploration: "", cue: "", track: "",
		trackIsCue: false, gain: 0, envelope: 0, suspended: false, position: 0, localRemaining: 0,
		combatEncounter: 0, combatFinished: false, combatVictory: false, combatTrack: musicCombatTrack,
		sailingRaised: false, sailingPlayed: false, sailingTrack: "", sailingEpoch: 0, trackSailEpoch: 0,
	}
}

func (policy *musicPolicy) desired() (string, bool) {
	if policy.combatEncounter != 0 && policy.combatVictory {
		return "", true // The battlefield stays quiet through looting and menu interruptions.
	}
	if policy.cue != "" {
		return policy.cue, true
	}
	if policy.combatEncounter != 0 {
		if policy.combatFinished {
			return "", true // Exploration cannot resume while still on the battlefield.
		}

		return policy.combatTrack, true
	}
	if policy.sailingRaised {
		if policy.sailingPlayed {
			if policy.trackSailEpoch == policy.sailingEpoch && policy.track != "" {
				return policy.sailingTrack, true // Let this hoist's finite track finish naturally.
			}

			return "", true // Sailing music is finite: no exploration score until sails are lowered.
		}

		return policy.sailingTrack, true
	}

	return policy.exploration, false
}

func (policy *musicPolicy) update() error {
	now := policy.now()
	delta := max(time.Duration(0), now.Sub(policy.last))
	policy.last = now
	if err := policy.output.Update(); err != nil {
		if policy.suspended || policy.track == "" {
			// A failed preview cannot erase suspended score ownership.
			return fmt.Errorf("service music transport: %w", err)
		}

		return policy.failed(err)
	}
	if policy.track != "" && !policy.suspended && !policy.output.IsPlaying() {
		policy.completeCombatPresentation()

		return policy.finish()
	}
	if policy.combatEncounter != 0 && policy.combatVictory && policy.track != "" {
		return policy.finish() // Victory stops any pending or active score without a transition fade.
	}
	if policy.history.disabled || len(policy.holds) != 0 {
		policy.suspend()

		return nil
	}
	if policy.localRemaining > 0 {
		policy.localRemaining = max(time.Duration(0), policy.localRemaining-delta)
		policy.suspend()

		return nil // Resume on the next update; do not spend the held time twice.
	}
	if policy.track == "" {
		policy.history.quietRemaining = max(time.Duration(0), policy.history.quietRemaining-delta)

		return policy.startEligible()
	}

	return policy.updateTrack(delta)
}

func (policy *musicPolicy) updateTrack(delta time.Duration) error {
	desired, isCue := policy.desired()
	if policy.trackSailEpoch != 0 && policy.trackSailEpoch != policy.sailingEpoch {
		return policy.finish() // A new hoist must start its own presentation.
	}
	if policy.track != desired || policy.trackIsCue != isCue {
		if policy.suspended {
			return policy.finish()
		}
		policy.envelope = max(0, policy.envelope-float64(delta)/float64(musicFadeDuration))
		if policy.envelope == 0 {
			return policy.finish()
		}

		return policy.applyGain()
	}
	if policy.suspended {
		// Preview playback may have replaced the device player. Resume restores
		// the saved position, never a preview's identity or the score's opening.
		if err := policy.output.SetVolume(0); err != nil {
			return fmt.Errorf("reset resumed music gain: %w", err)
		}
		if err := policy.output.Resume(policy.track, policy.position); err != nil {
			return policy.failed(err)
		}
		policy.suspended = false
		policy.envelope = 0
	}
	if !policy.output.IsPlaying() {
		policy.completeCombatPresentation()

		return policy.finish()
	}
	policy.envelope = min(1, policy.envelope+float64(delta)/float64(musicFadeDuration))

	return policy.applyGain()
}

func (policy *musicPolicy) applyGain() error {
	if err := policy.output.SetVolume(policy.gain * policy.envelope); err != nil {
		return fmt.Errorf("apply score envelope: %w", err)
	}

	return nil
}

func (policy *musicPolicy) startEligible() error {
	track, isCue := policy.desired()
	if track == "" || (!isCue && policy.history.quietRemaining > 0) {
		return nil
	}
	level := musicInitialDB
	if !isCue && policy.history.heard[track] {
		level = musicRepeatDB
	}
	if isCue && policy.combatEncounter != 0 && policy.cue == "" {
		level = musicCombatDB
	}
	if err := policy.output.SetVolume(0); err != nil {
		return fmt.Errorf("reset starting music gain: %w", err)
	}
	if err := policy.output.Play(track); err != nil {
		return policy.failed(err)
	}
	if !policy.output.IsPlaying() {
		// An authored null-file slot is silent. Retry no faster than a normal
		// quiet interval, and never turn a missing special cue into a loop.
		policy.completeCombatPresentation()
		policy.completeSailingPresentation()
		policy.history.quietRemaining = musicQuietInterval
		policy.cue = ""

		return nil
	}
	policy.startTrack(track, isCue, level)

	return nil
}

func (policy *musicPolicy) startTrack(track string, isCue bool, level float64) {
	policy.track, policy.trackIsCue = track, isCue
	policy.trackSailEpoch = 0
	if policy.sailingRaised && policy.combatEncounter == 0 && policy.cue == "" {
		policy.sailingPlayed = true
		policy.trackSailEpoch = policy.sailingEpoch
	}
	policy.gain = math.Pow(musicDBBase, level/musicDBDivisor)
	policy.envelope, policy.position, policy.suspended = 0, 0, false
	if !isCue {
		policy.history.heard[track] = true
	}
}

func (policy *musicPolicy) finish() error {
	if policy.trackIsCue && policy.cue == policy.track {
		policy.cue = ""
	}
	policy.track, policy.suspended = "", false
	policy.trackSailEpoch = 0
	policy.history.quietRemaining = musicQuietInterval

	if err := policy.output.Stop(); err != nil {
		return fmt.Errorf("retire finished score: %w", err)
	}

	return nil
}

func (policy *musicPolicy) failed(err error) error {
	policy.completeCombatPresentation()
	policy.completeSailingPresentation()
	policy.track, policy.cue, policy.suspended = "", "", false
	policy.history.quietRemaining = musicQuietInterval

	return errors.Join(err, policy.output.Stop())
}

// A hoist owns one attempted finite presentation, just like an encounter.
// Failure consumes it rather than retrying a broken stream on every frame.
func (policy *musicPolicy) completeSailingPresentation() {
	if policy.sailingRaised && policy.combatEncounter == 0 && policy.cue == "" {
		policy.sailingPlayed = true
	}
}

// Combat gets one finite presentation per encounter, independent of exploration
// cooldown/history. Repeated frame/turn updates must not restart it after EOF.
func (policy *musicPolicy) setCombatEncounter(encounter uint64) {
	if policy.combatEncounter == encounter {
		return
	}
	policy.cue = "" // A new encounter/exit supersedes the preceding score request.
	policy.combatEncounter = encounter
	policy.combatFinished = false
	policy.combatVictory = false
}

func (policy *musicPolicy) setCombatVictory(victory bool) {
	if policy.combatEncounter != 0 && victory {
		policy.combatVictory = true
		policy.cue = ""
	}
}

// setSailing follows sail-state edges, independently of automatic movement.
// One finite score is allowed for each hoist, including after a combat handoff.
func (policy *musicPolicy) setSailing(raised bool, track string) {
	if policy.sailingRaised != raised {
		policy.cue = "" // A sail-state transition supersedes an older requested score.
	}
	if raised && !policy.sailingRaised {
		policy.sailingEpoch++
		policy.sailingPlayed = false
	}
	policy.sailingRaised = raised
	policy.sailingTrack = track
}

func (policy *musicPolicy) completeCombatPresentation() {
	if policy.combatEncounter != 0 && policy.cue == "" {
		policy.combatFinished = true
	}
}

func (policy *musicPolicy) suspend() {
	if policy.track != "" && !policy.suspended {
		policy.position = policy.output.Pause()
		policy.suspended = true
	}
}

func (policy *musicPolicy) setSuppressed(owner MusicSuppression, suppressed bool) {
	if policy.holds[owner] == suppressed {
		return
	}
	if suppressed {
		policy.holds[owner] = true
		policy.suspend()
	} else {
		delete(policy.holds, owner)
	}
	policy.last = policy.now()
}

func (policy *musicPolicy) localMusic() {
	policy.localRemaining = musicLocalRelease
	policy.suspend()
}

func (policy *musicPolicy) exportHistory() MusicHistory {
	history := policy.history
	history.heard = maps.Clone(history.heard)
	if policy.track != "" {
		history.quietRemaining = musicQuietInterval
	}

	return history
}

func (policy *musicPolicy) restoreHistory(history MusicHistory) {
	if policy.track != "" {
		panic("music history must be restored before scene activation")
	}
	policy.history = history
	policy.history.heard = maps.Clone(history.heard)
	if policy.history.heard == nil {
		policy.history.heard = make(map[string]bool)
	}
	policy.last = policy.now()
}
