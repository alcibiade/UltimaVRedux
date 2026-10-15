package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// ActorAnimationIdentity is an opaque runtime lifetime key, never save data.
// Actual actor pointers distinguish reused surface slots; encounter identity
// includes the complete clone slot and the stable monster ID, not its array slot.
type ActorAnimationIdentity struct {
	state       *GameState
	combat      *CombatState
	turn        CombatTurnIdentity
	combatantID CombatantID
	monster     *CombatMonster
	units       *map_units.MapUnits
	actor       *map_units.MapUnitDetails
	dungeon     *DungeonState
	wanderer    *DungeonWanderer
	location    references.Location
	floor       references.FloorNumber
}

// CombatActorAnimation returns (lifetime, slowed, live) independently of disclosure.
// It deliberately never calls combatTemporalActionSuppressed: Quickness there
// may consume gameplay RNG. Callers must still apply normal visibility rules.
func (g *GameState) CombatActorAnimation(identity CombatTurnIdentity, expectedID CombatantID) (ActorAnimationIdentity, bool, bool) {
	combat := g.CombatScene
	if combat == nil || identity.EntityIndex < 0 {
		return ActorAnimationIdentity{}, false, false
	}
	key := ActorAnimationIdentity{state: g, combat: combat, turn: identity}
	if identity.IsPartyMember {
		if expectedID != 0 || identity.EntityIndex >= len(combat.PartyActive) || !combat.IsCombatPCActive(identity) {
			return ActorAnimationIdentity{}, false, false
		}
		character := g.PartyState.GetCharacter(identity.EntityIndex)
		if character == nil || character.IsDead() || !character.IsInParty() {
			return ActorAnimationIdentity{}, false, false
		}
	} else {
		if combat.MonsterManager == nil || identity.Slot != 0 {
			return ActorAnimationIdentity{}, false, false
		}
		monster := combat.MonsterManager.GetMonster(identity.EntityIndex)
		if monster == nil || !monster.IsAlive() || (expectedID != 0 && monster.CombatantID() != expectedID) {
			return ActorAnimationIdentity{}, false, false
		}
		key.combatantID = monster.CombatantID()
		key.monster = monster
		// Storage compaction can move a surviving monster to another index.
		// Its stable ID and actual instance belong in the key, not that index.
		key.turn = CombatTurnIdentity{}
	}
	slow := g.HasActiveDurationEffect(DurationEffectTimeStop) &&
		g.combatIdentityLoyalty(identity) == CombatMonsterLoyaltyMonster

	return key, slow, true
}

// SurfaceActorAnimation classifies actors through their semantic references.
// Environmental enemies and vehicle/object map units retain ordinary timing.
func (g *GameState) SurfaceActorAnimation(unit map_units.MapUnit) (ActorAnimationIdentity, bool, bool) {
	player := g.MapState.PlayerLocation
	if g.CombatScene != nil || g.CurrentNPCAIController == nil || unit == nil ||
		!isAnimatedSurfaceActor(unit) || unit.Floor() != player.Floor {
		return ActorAnimationIdentity{}, false, false
	}
	units := g.CurrentNPCAIController.GetNpcs()
	if units == nil {
		return ActorAnimationIdentity{}, false, false
	}
	key := ActorAnimationIdentity{
		state: g, units: units, actor: unit.MapUnitDetails(), location: player.Location, floor: player.Floor,
	}

	return key, g.HasActiveDurationEffect(DurationEffectTimeStop), true
}

func isAnimatedSurfaceActor(unit map_units.MapUnit) bool {
	switch actor := unit.(type) {
	case *map_units.NPCEnemy:
		return actor != nil && actor.Variant() != map_units.EnemyVariantPirate &&
			actor.Variant() != map_units.EnemyVariantWhirlpool && !actor.EnemyReference.MonsterType.IsEnvironmentalHazard()
	case *map_units.NPCFriendly:
		if actor == nil || actor.NPCReference.GetNPCType() == references.Vehicle {
			return false
		}
		_, creature := actor.NPCReference.SmallMapCombatMonsterType()

		return creature
	default:
		return false
	}
}

// DungeonActorAnimation retains the actual wanderer through movement and turns.
func (g *GameState) DungeonActorAnimation() (ActorAnimationIdentity, bool, bool) {
	player := g.MapState.PlayerLocation
	dungeon := g.activeDungeon
	if g.CombatScene != nil || dungeon == nil || dungeon.location != player.Location ||
		dungeon.wanderer == nil || dungeon.wanderer.Floor != player.Floor {
		return ActorAnimationIdentity{}, false, false
	}

	return ActorAnimationIdentity{
		state: g, dungeon: dungeon, wanderer: dungeon.wanderer, location: player.Location, floor: player.Floor,
	}, g.HasActiveDurationEffect(DurationEffectTimeStop), true
}

// ForEachActorAnimation visits live actors, including hidden and offscreen ones.
// It is a pure timing projection, not a visibility or targeting API.
func (g *GameState) ForEachActorAnimation(visit func(ActorAnimationIdentity, bool)) {
	if g.CombatScene != nil {
		g.forEachCombatActorAnimation(visit)

		return
	}
	if key, slow, valid := g.DungeonActorAnimation(); valid {
		visit(key, slow)

		return
	}
	if g.CurrentNPCAIController == nil || g.CurrentNPCAIController.GetNpcs() == nil {
		return
	}
	for _, unit := range *g.CurrentNPCAIController.GetNpcs() {
		if key, slow, valid := g.SurfaceActorAnimation(unit); valid {
			visit(key, slow)
		}
	}
}

func (g *GameState) forEachCombatActorAnimation(visit func(ActorAnimationIdentity, bool)) {
	project := func(identity CombatTurnIdentity) {
		if key, slow, valid := g.CombatActorAnimation(identity, 0); valid {
			visit(key, slow)
		}
	}
	for index := range g.CombatScene.PartyActive {
		project(CombatTurnIdentity{IsPartyMember: true, EntityIndex: index})
	}
	g.CombatScene.ForEachPCClone(func(identity CombatTurnIdentity, _ references.Position) { project(identity) })
	if g.CombatScene.MonsterManager != nil {
		g.CombatScene.MonsterManager.ForEachActive(func(index int, _ *CombatMonster) {
			project(CombatTurnIdentity{EntityIndex: index})
		})
	}
}
