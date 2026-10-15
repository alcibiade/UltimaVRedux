package game_state

import "github.com/bradhannah/UltimaVRedux/internal/event_queue"

// DungeonInjuryKind identifies an already-resolved dungeon HP-loss source.
type DungeonInjuryKind uint8

const (
	// DungeonInjuryNone clears the dedicated injury overlay.
	DungeonInjuryNone DungeonInjuryKind = iota
	// DungeonInjuryFire identifies fire-field HP loss.
	DungeonInjuryFire
	// DungeonInjuryElectrical identifies electrical-contact HP loss.
	DungeonInjuryElectrical
	// DungeonInjuryPoison identifies a dungeon poison HP tick.
	DungeonInjuryPoison
)

// DungeonInjuryDurationMs is the presentation-only pulse lifetime.
const DungeonInjuryDurationMs int64 = 200

type dungeonInjuryResult struct {
	kind   DungeonInjuryKind
	hpLost int
}

func (result *dungeonInjuryResult) observe(before, after uint16) {
	if after < before {
		result.hpLost += int(before - after)
	}
}

func dungeonInjuryFeedbackIdentity(kind DungeonInjuryKind) (string, SoundEffect) {
	switch kind {
	case DungeonInjuryFire:
		return "dungeon_injury.fire", SoundFireBurn
	case DungeonInjuryElectrical:
		return "dungeon_injury.electrical", SoundElectricalDamage
	case DungeonInjuryPoison:
		return "dungeon_injury.poison", SoundPoison
	case DungeonInjuryNone:
		panic("cannot admit an empty dungeon injury")
	default:
		panic("invalid dungeon injury kind")
	}
}

func (g *GameState) presentDungeonInjury(result dungeonInjuryResult) {
	if result.hpLost == 0 {
		return
	}
	// Sound belongs to the authoritative injury admission, once per source, even
	// if a rapid repeat coalesces with an already-live visual pulse.
	name, sound := dungeonInjuryFeedbackIdentity(result.kind)
	g.SystemCallbacks.Audio.PlaySoundEffect(sound)
	owner := g.CaptureDungeonInjuryOwner()
	g.EventQueueManager.TriggerFeedback(name, event_queue.PriorityNormal,
		func(sequence *event_queue.SequentialEventBuilder) {
			shown := false
			clearPulse := func() {
				if shown {
					shown = false
					g.SystemCallbacks.Visual.ShowDungeonInjury(DungeonInjuryNone)
				}
			}
			sequence.First(func() {
				// Queued feedback must not cross a dungeon visit, floor, or combat
				// transition. Fatal HP loss still presents while this visit is live.
				if !g.MatchesDungeonInjuryOwner(owner) {
					return
				}
				shown = true
				g.SystemCallbacks.Visual.ShowDungeonInjury(result.kind)
			}).Wait(DungeonInjuryDurationMs).Then(clearPulse).Finally(clearPulse)
			// The delayed Then owns normal expiry; Finally also covers cancellation.
			// First followed only by Wait/Finally has no delayed successor to keep
			// this queue live. Both cleanup paths share exact-once pulse ownership.
		})
}
