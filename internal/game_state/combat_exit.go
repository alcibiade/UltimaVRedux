package game_state

import "github.com/bradhannah/UltimaVRedux/internal/references"

const combatMaxDungeonFloor = references.FloorNumber(7)

// CombatExitContext controls whether party members must share a physical exit.
type CombatExitContext int

const (
	// CombatExitSurface does not require party members to share an exit route.
	CombatExitSurface CombatExitContext = iota
	// CombatExitDungeonEncounter represents non-room dungeon combat.
	CombatExitDungeonEncounter
	// CombatExitDungeonRoom requires every physical departure to share one route.
	CombatExitDungeonRoom
)

// CombatExitRoute identifies the shared route used to leave combat.
type CombatExitRoute int

const (
	// CombatExitNone means no PC has physically left the battlefield.
	CombatExitNone CombatExitRoute = iota
	// CombatExitNorth records departure through the north edge.
	CombatExitNorth
	// CombatExitEast records departure through the east edge.
	CombatExitEast
	// CombatExitSouth records departure through the south edge.
	CombatExitSouth
	// CombatExitWest records departure through the west edge.
	CombatExitWest
	// CombatExitUp records departure through an up ladder.
	CombatExitUp
	// CombatExitDown records departure through a down ladder.
	CombatExitDown
	// CombatExitDoomMirror records absorption by the final-room mirror.
	CombatExitDoomMirror
)

func combatExitRouteFromDirection(direction references.Direction) CombatExitRoute {
	switch direction {
	case references.Up:
		return CombatExitNorth
	case references.Right:
		return CombatExitEast
	case references.Down:
		return CombatExitSouth
	case references.Left:
		return CombatExitWest
	case references.NoneDirection:
		return CombatExitNone
	}

	return CombatExitNone
}

func (combatState *CombatState) establishExitRoute(route CombatExitRoute) bool {
	if route == CombatExitNone {
		panic("cannot establish an empty combat exit route")
	}
	if combatState.ExitRoute == CombatExitNone {
		combatState.ExitRoute = route

		return true
	}
	if combatState.ExitContext == CombatExitDungeonRoom && combatState.ExitRoute != route {
		return false
	}

	return true
}

// ActiveCombatPartyMembers returns active party indexes in stable party order.
func (g *GameState) ActiveCombatPartyMembers() []int {
	if g.CombatScene == nil {
		return nil
	}
	members := make([]int, 0, len(g.CombatScene.PartyActive))
	for index, active := range g.CombatScene.PartyActive {
		if active {
			members = append(members, index)
		}
	}

	return members
}

// DepartCombatPartyMemberAfterVictory removes one PC during the victory exit sequence.
func (g *GameState) DepartCombatPartyMemberAfterVictory(pcIndex int) {
	if g.CombatScene == nil || !g.CombatScene.Victory {
		panic("victory departure requires victorious active combat")
	}
	g.departPCFromCombat(CombatTurnIdentity{IsPartyMember: true, EntityIndex: pcIndex})
}

func (g *GameState) applyCombatExitRoute(context CombatExitContext, route CombatExitRoute) {
	if context == CombatExitSurface || route == CombatExitNone ||
		g.MapState.PlayerLocation.Location.GetMapType() != references.DungeonMapType {
		return
	}

	position := g.MapState.PlayerLocation.Position
	if context == CombatExitDungeonRoom {
		newPosition, moved := applyDungeonCardinalExit(position, route)
		if moved {
			g.MapState.PlayerLocation.Position = newPosition
			if g.activeDungeon == nil {
				panic("dungeon room exit requires active dungeon authority")
			}
			g.activeDungeon.setFacingAndLastDirection(combatExitRouteFacing(route))

			return
		}
	}
	g.applyDungeonVerticalExit(route)
}

func combatExitRouteFacing(route CombatExitRoute) references.DungeonFacing {
	switch route {
	case CombatExitNorth:
		return references.DungeonFacingNorth
	case CombatExitEast:
		return references.DungeonFacingEast
	case CombatExitSouth:
		return references.DungeonFacingSouth
	case CombatExitWest:
		return references.DungeonFacingWest
	case CombatExitNone, CombatExitUp, CombatExitDown, CombatExitDoomMirror:
		panic("non-cardinal route has no dungeon facing")
	}
	panic("invalid combat exit route")
}

func (g *GameState) applyDungeonVerticalExit(route CombatExitRoute) {
	location := &g.MapState.PlayerLocation
	switch route {
	case CombatExitUp:
		if location.Floor > 0 {
			g.transitionDungeonFloor(location.Floor-1, references.DungeonArrivedFromBelow)

			return
		}
		g.exitDungeonToWorld(DungeonKlimbUp)
	case CombatExitDown:
		if location.Floor < combatMaxDungeonFloor {
			g.transitionDungeonFloor(location.Floor+1, references.DungeonArrivedFromAbove)

			return
		}
		g.exitDungeonToWorld(DungeonKlimbDown)
	case CombatExitNone, CombatExitNorth, CombatExitEast, CombatExitSouth, CombatExitWest, CombatExitDoomMirror:
	}
}

func applyDungeonCardinalExit(
	position references.Position,
	route CombatExitRoute,
) (references.Position, bool) {
	switch route {
	case CombatExitNorth:
		position.Y = (position.Y - 1 + references.YDungeonMapTiles) % references.YDungeonMapTiles
	case CombatExitEast:
		position.X = (position.X + 1) % references.XDungeonMapTiles
	case CombatExitSouth:
		position.Y = (position.Y + 1) % references.YDungeonMapTiles
	case CombatExitWest:
		position.X = (position.X - 1 + references.XDungeonMapTiles) % references.XDungeonMapTiles
	case CombatExitNone, CombatExitUp, CombatExitDown, CombatExitDoomMirror:
		return position, false
	}

	return position, true
}
