package game_state

import (
	"sort"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// CombatPossessionReleaseReason distinguishes ordinary freedom from emergency sleep.
type CombatPossessionReleaseReason int

const (
	// CombatPossessionReleaseNatural is resistance at an own-turn boundary.
	CombatPossessionReleaseNatural CombatPossessionReleaseReason = iota
	// CombatPossessionReleaseControllerDefeated is authoritative controller death.
	CombatPossessionReleaseControllerDefeated
	// CombatPossessionReleaseControllerEscaped is an actual battlefield departure.
	CombatPossessionReleaseControllerEscaped
	// CombatPossessionReleaseControllerReplaced is successful rat replacement.
	CombatPossessionReleaseControllerReplaced
)

type combatPossessionEpisode struct {
	id         uint64
	controller CombatantID
	turns      int
}

// A completion belongs to a dispatched actor AND episode, never a current flag.
type combatPossessionTurn struct {
	encounter CombatEncounterID
	sequence  uint64
	actor     CombatTurnIdentity
	episode   uint64
}

// CombatPossessionProgress is a detached read; raw/Chaos possession has no timer.
// Episode IDs are encounter-local: retained consumers must also guard the owning
// CombatState or EncounterID before using an ID to validate presentation.
func (combat *CombatState) CombatPossessionProgress(identity CombatTurnIdentity) (uint64, int, bool) {
	if combat == nil {
		return 0, 0, false
	}
	episode, ok := combat.possessions[identity]

	return episode.id, episode.turns, ok
}

func (combat *CombatState) acquireCombatPossession(identity CombatTurnIdentity, controller CombatantID) bool {
	if controller == 0 || !combat.isCombatPCActive(identity) || combat.isCombatPCPossessed(identity) {
		return false
	}
	if combat.possessions == nil {
		combat.possessions = make(map[CombatTurnIdentity]combatPossessionEpisode)
	}
	combat.nextPossessionEpisode++
	combat.possessions[identity] = combatPossessionEpisode{id: combat.nextPossessionEpisode, controller: controller}
	combat.setCombatPCPossessed(identity, true)

	return true
}

func (combat *CombatState) cloneCombatPossession(source, clone CombatTurnIdentity) {
	if episode, ok := combat.possessions[source]; ok {
		combat.nextPossessionEpisode++
		episode.id = combat.nextPossessionEpisode
		combat.possessions[clone] = episode
	}
}

func (combat *CombatState) dispatchCombatPossessionTurn(identity CombatTurnIdentity) {
	combat.nextPossessionTurn++
	combat.possessionTurn = combatPossessionTurn{
		encounter: combat.EncounterID, sequence: combat.nextPossessionTurn,
		actor: identity, episode: combat.possessions[identity].id,
	}
}

// REDUX DEVIATION: COMBAT-0005. Original enemy possession is untimed. Redux
// bounds lost control to twelve own completed turns, using current INT to offer
// earlier resistance. These additional draws intentionally change RNG progression.
func combatPossessionThresholdBP(turns, intelligence int) int {
	base := 0
	switch {
	case turns >= 12:
		return 10000
	case turns == 11:
		base = 35
	case turns == 10:
		base = 20
	case turns >= 7:
		base = 8
	case turns >= 4:
		base = 3
	}

	return base * 100 * (20 + max(1, min(30, intelligence))) / 40
}

func (g *GameState) completeCombatPossessionTurn(ticket combatPossessionTurn) {
	g.completeCombatPossessionTurnWithRNG(ticket, g.AsRandomProvider())
}

func (g *GameState) completeCombatPossessionTurnWithRNG(ticket combatPossessionTurn, rng references.RandomProvider) {
	combat := g.CombatScene
	if combat == nil || ticket.encounter != combat.EncounterID || ticket.sequence == 0 ||
		ticket != combat.possessionTurn || ticket.sequence <= combat.completedPossessionTurn {
		return
	}
	combat.completedPossessionTurn = ticket.sequence
	episode, ok := combat.possessions[ticket.actor]
	if !ok || episode.id != ticket.episode || !combat.isCombatPCActive(ticket.actor) {
		return
	}
	character := g.PartyState.GetCharacter(ticket.actor.EntityIndex)
	entry := combat.TurnQueue.GetEntryByIdentity(ticket.actor)
	if character.IsDead() || entry == nil || entry.status != turnQueueActive {
		combat.setCombatPCPossessed(ticket.actor, false)

		return
	}
	episode.turns++
	combat.possessions[ticket.actor] = episode
	threshold := combatPossessionThresholdBP(episode.turns, int(character.Intelligence))
	if episode.turns >= 12 || (episode.turns >= 4 && rng.Intn(10000) < threshold) {
		g.releaseCombatPossession(ticket.actor, CombatPossessionReleaseNatural)

		return
	}
	if episode.turns == 3 || episode.turns == 7 || episode.turns == 10 {
		g.publishCombatPossessionEvent(ticket.actor, episode, CombatEventPossessionResistance, CombatPossessionReleaseNatural)
	}
}

func (g *GameState) releaseCombatPossession(identity CombatTurnIdentity, reason CombatPossessionReleaseReason) {
	episode, ok := g.CombatScene.possessions[identity]
	if !ok {
		return
	}
	g.CombatScene.setCombatPCPossessed(identity, false)
	// Preserve equipment, intrinsic allegiance, HP and unrelated conditions.
	g.publishCombatPossessionEvent(identity, episode, CombatEventPossessionRelease, reason)
}

func (g *GameState) publishCombatPossessionEvent(identity CombatTurnIdentity, episode combatPossessionEpisode, kind CombatEventKind, reason CombatPossessionReleaseReason) {
	g.SystemCallbacks.Combat.PublishEvent(CombatEvent{
		Kind: kind, Target: combatFeedbackIdentity(identity, g.CombatScene.GetCombatantPosition(identity)),
		PossessionEpisodeID: episode.id, PossessionTurns: episode.turns, PossessionReleaseReason: reason,
	})
}

func (g *GameState) releaseCombatController(controller CombatantID, reason CombatPossessionReleaseReason) {
	if controller == 0 || g.CombatScene == nil {
		return
	}
	identities := make([]CombatTurnIdentity, 0)
	for identity, episode := range g.CombatScene.possessions {
		if episode.controller == controller {
			identities = append(identities, identity)
		}
	}
	sort.Slice(identities, func(i, j int) bool {
		if identities[i].EntityIndex != identities[j].EntityIndex {
			return identities[i].EntityIndex < identities[j].EntityIndex
		}

		return identities[i].Slot < identities[j].Slot
	})
	for _, identity := range identities {
		g.releaseCombatPossession(identity, reason)
	}
}
