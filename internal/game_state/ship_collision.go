package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

const (
	sourceHitHullDamageSides  = 30
	sourceHitPartyDamageSides = 8
)

// SourceHitOutcome identifies the authoritative result of outdoor collision damage.
// See ref:SHIP-0003.
type SourceHitOutcome int

const (
	// SourceHitPartyDamaged means one independent party-damage pass was applied.
	SourceHitPartyDamaged SourceHitOutcome = iota
	// SourceHitFrigateDamaged means the frigate survived with positive hull.
	SourceHitFrigateDamaged
	// SourceHitEscapedBySkiff means a carried skiff replaced the destroyed frigate.
	SourceHitEscapedBySkiff
	// SourceHitEscapedByCarpet means a carried carpet replaced the destroyed frigate.
	SourceHitEscapedByCarpet
	// SourceHitDrowned means the destroyed frigate had no fallback and the active party died.
	SourceHitDrowned
)

// SourceHitMemberDamage is one detached damage roll in source party order.
// Drowning can include multiple entries for the same member.
type SourceHitMemberDamage struct {
	MemberIndex int
	Damage      int
	HPBefore    uint16
	HPAfter     uint16
	Died        bool
}

// SourceHitResult is detached from GameState so presentation can safely retain it.
// PartyWiped is true only when this invocation removes the final living active member.
type SourceHitResult struct {
	Outcome         SourceHitOutcome
	VehicleBefore   references.VehicleType
	VehicleAfter    references.VehicleType
	DirectionBefore references.Direction
	DirectionAfter  references.Direction
	HullBefore      int
	HullAfter       int
	HullDamage      int
	MemberDamage    []SourceHitMemberDamage
	PartyWiped      bool
}

// ResolveSourceHit applies outdoor collision damage as one authoritative mutation.
// Frigates take one uniform 1..30 hull roll and sink on equality. Every other
// vehicle state applies an independent 1d8 to each living active member.
// See ref:SHIP-0003.
func (g *GameState) ResolveSourceHit() SourceHitResult {
	vehicle := g.PartyVehicle.GetVehicleDetails()
	result := SourceHitResult{
		VehicleBefore:   vehicle.VehicleType,
		VehicleAfter:    vehicle.VehicleType,
		DirectionBefore: vehicle.CurrentDirection(),
		DirectionAfter:  vehicle.CurrentDirection(),
	}
	if vehicle.VehicleType != references.FrigateVehicle {
		livingBefore := g.sourceLivingPartyCount()
		result.Outcome = SourceHitPartyDamaged
		result.MemberDamage = g.applySourcePartyDamageOnce()
		result.PartyWiped = livingBefore > 0 && g.sourceLivingPartyCount() == 0
		if len(result.MemberDamage) > 0 {
			g.SystemCallbacks.Screen.UpdateStatsDisplay()
		}

		return result
	}

	result.HullBefore = vehicle.GetHullHP()
	result.HullAfter = result.HullBefore
	result.HullDamage = g.RollDice(sourceHitHullDamageSides)
	if result.HullBefore > result.HullDamage {
		vehicle.SetHullHP(result.HullBefore - result.HullDamage)
		result.Outcome = SourceHitFrigateDamaged
		result.HullAfter = vehicle.GetHullHP()

		return result
	}

	// Stop before replacing the vessel or applying drowning damage. No pending
	// automatic phase may move the fallback or wait on its rough-seas workflow.
	g.StopAutomaticSailing()
	position := g.MapState.PlayerLocation.Position
	floor := g.MapState.PlayerLocation.Floor
	direction := vehicle.CurrentDirection()
	switch {
	case vehicle.HasAtLeastOneSkiff():
		vehicle.DecrementSkiffQuantity()
		replacement := map_units.NewNPCFriendlyVehiceNewRef(references.SkiffVehicle, position, floor)
		replacement.GetVehicleDetails().SetPartyVehicleDirection(direction)
		replacement.MapUnitDetails().Position = position
		replacement.MapUnitDetails().Floor = floor
		g.PartyVehicle = *replacement
		g.registerAtlasVessel(g.PartyVehicle.GetVehicleDetails())
		result.Outcome = SourceHitEscapedBySkiff
	case g.PartyState.Inventory.QuestItems.HasSome(references.Carpet):
		if !g.PartyState.Inventory.QuestItems.DecrementByOne(references.Carpet) {
			panic("available sinking fallback carpet could not be consumed")
		}
		carpetDirection := references.Right
		if g.RandomIntInRange(0, 1) == 1 {
			carpetDirection = references.Left
		}
		replacement := map_units.NewNPCFriendlyVehiceNewRef(references.CarpetVehicle, position, floor)
		replacement.GetVehicleDetails().SetPartyVehicleDirection(carpetDirection)
		replacement.MapUnitDetails().Position = position
		replacement.MapUnitDetails().Floor = floor
		g.PartyVehicle = *replacement
		result.Outcome = SourceHitEscapedByCarpet
	default:
		g.PartyVehicle = map_units.NewNPCFriendlyVehiceNoVehicle()
		result.Outcome = SourceHitDrowned
		livingBefore := g.sourceLivingPartyCount()
		for g.sourceLivingPartyCount() > 0 {
			result.MemberDamage = append(result.MemberDamage, g.applySourcePartyDamageOnce()...)
		}
		result.PartyWiped = livingBefore > 0
	}

	after := g.PartyVehicle.GetVehicleDetails()
	result.VehicleAfter = after.VehicleType
	result.DirectionAfter = after.CurrentDirection()
	result.HullAfter = 0
	if len(result.MemberDamage) > 0 {
		g.SystemCallbacks.Screen.UpdateStatsDisplay()
	}

	return result
}

// ResolveRangedSourceHit is called only after an enemy projectile reaches the
// party. Contact/collision callers keep their own interruption policies.
func (g *GameState) ResolveRangedSourceHit() SourceHitResult {
	wasSailing := g.isSailing && g.PartyVehicle.GetVehicleDetails().VehicleType == references.FrigateVehicle
	hit := g.ResolveSourceHit()
	if wasSailing {
		// REDUX DEVIATION: SHIP-0005: original ranged hits keep sailing.
		// Redux offers a decision point without furling surviving sails or
		// changing facing, so a fresh direction can resume at sailing speed.
		g.StopAutomaticSailing()
		g.SystemCallbacks.Message.AddRowStr("Under attack! Sailing interrupted.")
	}

	return hit
}

func (g *GameState) applySourcePartyDamageOnce() []SourceHitMemberDamage {
	damage := make([]SourceHitMemberDamage, 0, party_state.MaxActiveParty)
	for index := range party_state.MaxActiveParty {
		member := g.PartyState.GetCharacter(index)
		if member == nil || member.PartyMap != party_state.PartyMapActive || member.IsDead() {
			continue
		}
		before := member.GetCurrentHp()
		roll := g.RollDice(sourceHitPartyDamageSides)
		member.SubtractHp(uint16(roll))
		damage = append(damage, SourceHitMemberDamage{
			MemberIndex: index,
			Damage:      roll,
			HPBefore:    before,
			HPAfter:     member.GetCurrentHp(),
			Died:        member.IsDead(),
		})
	}

	return damage
}

func (g *GameState) sourceLivingPartyCount() int {
	count := 0
	for index := range party_state.MaxActiveParty {
		member := g.PartyState.GetCharacter(index)
		if member != nil && member.PartyMap == party_state.PartyMapActive && !member.IsDead() {
			count++
		}
	}

	return count
}

func (g *GameState) presentSourceHitOutcome(hit SourceHitResult) {
	switch hit.Outcome {
	case SourceHitPartyDamaged, SourceHitFrigateDamaged:
	case SourceHitEscapedBySkiff, SourceHitEscapedByCarpet:
		g.SystemCallbacks.Message.AddRowStr("Ship sunk!")
		g.SystemCallbacks.Message.AddRowStr("Abandon ship!")
	case SourceHitDrowned:
		g.SystemCallbacks.Message.AddRowStr("Ship sunk!")
		g.SystemCallbacks.Message.AddRowStr("DROWNING!!!")
	}
}

// SailingCollisionKind preserves original outdoor rules's distinction between safe piers,
// shoals, blocking objects, and other blocking terrain.
// See ref:SHIP-0003.
type SailingCollisionKind int

const (
	// SailingCollisionNone means the reviewed target is no longer blocked.
	SailingCollisionNone SailingCollisionKind = iota
	// SailingCollisionPier means the frigate docks safely without a hit.
	SailingCollisionPier
	// SailingCollisionShoal means shoal terrain blocked the frigate.
	SailingCollisionShoal
	// SailingCollisionObject means an object blocked otherwise navigable water.
	SailingCollisionObject
	// SailingCollisionTerrain means non-shoal terrain blocked the frigate.
	SailingCollisionTerrain
)

// SailingCollisionResult is detached state for collision presentation.
type SailingCollisionResult struct {
	Kind   SailingCollisionKind
	HasHit bool
	Hit    SourceHitResult
}

// ClassifySailingCollision classifies the blocked target without mutating it.
func (g *GameState) ClassifySailingCollision(target references.Position) SailingCollisionKind {
	// Original outdoor rules keeps the terrain tile separate from looklist object occupancy.
	// In particular, an object or dropped item over shoals is still a shoal hit.
	// See ref:SHIP-0003.
	tile := g.GetCurrentLayeredMap().GetTileTopMapOnlyTile(&target)
	if tile == nil {
		return SailingCollisionTerrain
	}
	if tile.Is(indexes.Dock) {
		return SailingCollisionPier
	}
	if tile.Is(indexes.WaterCoast) {
		return SailingCollisionShoal
	}
	if g.ObjectPresentAt(&target) {
		return SailingCollisionObject
	}
	if g.ItemStacksMap.HasItemStackAtPosition(&target) {
		return SailingCollisionObject
	}
	if hasQuestItem, item := g.IsQuestItemAtPosition(target); hasQuestItem && g.IsQuestItemVisible(target, item) {
		return SailingCollisionObject
	}
	if !tile.IsBoatPassable() {
		return SailingCollisionTerrain
	}

	return SailingCollisionNone
}

// ResolveSailingCollision furls at piers and applies hit2 elsewhere (ref:SHIP-0003).
// Other blocked movement retains hoisted sails. It does not advance time or present feedback.
func (g *GameState) ResolveSailingCollision(target references.Position) SailingCollisionResult {
	kind := g.ClassifySailingCollision(target)
	result := SailingCollisionResult{Kind: kind}
	if kind == SailingCollisionNone {
		return result
	}
	vehicle := g.PartyVehicle.GetVehicleDetails()
	if vehicle.VehicleType != references.FrigateVehicle {
		panic("sailing collision requires a frigate")
	}
	g.StopAutomaticSailing()
	if kind == SailingCollisionPier {
		vehicle.SetSailState(map_units.SailFurled)

		return result
	}
	result.HasHit = true
	result.Hit = g.ResolveSourceHit()

	return result
}
