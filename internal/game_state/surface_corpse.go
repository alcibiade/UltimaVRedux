package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// SurfaceCorpse is a non-carriable current-map object discovered by Search.
type SurfaceCorpse struct {
	SearchIndex int
	Class       references.SearchObjectClass
	Position    references.Position
	Location    references.Location
	Floor       references.FloorNumber
}

func (g *GameState) addSurfaceCorpse(result *AuthenticSearchResult) bool {
	if result == nil || (result.ObjectClass != references.SearchObjectDeadPC &&
		result.ObjectClass != references.SearchObjectDeadMonster) {
		return false
	}
	for _, corpse := range g.SurfaceCorpses {
		if corpse.SearchIndex == result.Index {
			return false
		}
	}
	g.SurfaceCorpses = append(g.SurfaceCorpses, SurfaceCorpse{
		SearchIndex: result.Index,
		Class:       result.ObjectClass,
		Position:    result.Position,
		Location:    g.MapState.PlayerLocation.Location,
		Floor:       g.MapState.PlayerLocation.Floor,
	})
	g.SearchManager.authenticLocationManager.setSurfaceCorpseActive(result.Index, true)

	return true
}

func (g *GameState) surfaceCorpseAt(position references.Position, class references.SearchObjectClass) (int, *SurfaceCorpse) {
	for index := range g.SurfaceCorpses {
		corpse := &g.SurfaceCorpses[index]
		if corpse.Location == g.MapState.PlayerLocation.Location &&
			corpse.Floor == g.MapState.PlayerLocation.Floor && corpse.Position == position && corpse.Class == class {
			return index, corpse
		}
	}

	return -1, nil
}

func (g *GameState) removeSurfaceCorpse(index int) {
	if index < 0 || index >= len(g.SurfaceCorpses) {
		panic("cannot remove invalid surface corpse")
	}
	g.SearchManager.authenticLocationManager.setSurfaceCorpseActive(g.SurfaceCorpses[index].SearchIndex, false)
	copy(g.SurfaceCorpses[index:], g.SurfaceCorpses[index+1:])
	g.SurfaceCorpses = g.SurfaceCorpses[:len(g.SurfaceCorpses)-1]
}

func (g *GameState) restoreSurfaceCorpsesForCurrentMap() {
	g.SurfaceCorpses = g.SurfaceCorpses[:0]
	if g.SearchManager == nil {
		return
	}
	locations := references.GetAllSearchLocations()
	for _, searchIndex := range g.SearchManager.authenticLocationManager.surfaceCorpseOrder() {
		location := locations[searchIndex]
		if references.Location(location.MapNumber) != g.MapState.PlayerLocation.Location ||
			references.FloorNumber(location.Level) != g.MapState.PlayerLocation.Floor {
			continue
		}
		g.SurfaceCorpses = append(g.SurfaceCorpses, SurfaceCorpse{
			SearchIndex: location.Index,
			Class:       location.ObjectClass,
			Position: references.Position{
				X: references.Coordinate(location.X), Y: references.Coordinate(location.Y),
			},
			Location: references.Location(location.MapNumber),
			Floor:    references.FloorNumber(location.Level),
		})
	}
}

func (g *GameState) clearSurfaceCorpsesInCurrentMap() {
	for _, corpse := range g.SurfaceCorpses {
		g.SearchManager.authenticLocationManager.setSurfaceCorpseActive(corpse.SearchIndex, false)
	}
	g.SurfaceCorpses = g.SurfaceCorpses[:0]
}

func (g *GameState) surfaceCorpseSpriteAt(position references.Position) (indexes.SpriteIndex, bool) {
	if _, corpse := g.surfaceCorpseAt(position, references.SearchObjectDeadMonster); corpse != nil {
		return indexes.Splat, true
	}
	if _, corpse := g.surfaceCorpseAt(position, references.SearchObjectDeadPC); corpse != nil {
		return indexes.DeadBody, true
	}

	return indexes.NoSprites, false
}
