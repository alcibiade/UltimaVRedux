package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func (g *GameState) usesDaylightTinting() bool {
	if !g.CanAcceptGameplayCommands() || g.DateTime.GetAmbientTintDarkness() >= 1 {
		return false
	}
	location := g.MapState.PlayerLocation
	mapType := location.Location.GetMapType()

	return (mapType == references.SmallMapType && location.Floor >= 0) ||
		(mapType == references.LargeMapType && location.Floor == 0)
}

func (g *GameState) updateDaylightTinting(layer *map_state.LayeredMap) {
	g.daylightLayer = nil
	if !g.usesDaylightTinting() {
		if g.daylightTinting != nil {
			g.daylightTinting.Clear()
		}

		return
	}
	if g.daylightTinting == nil {
		g.daylightTinting = map_state.NewDaylightField(map_state.DefaultDaylightOptions())
	}
	// REDUX DEVIATION: Original calendar advancement and original visibility rules use global surface
	// sight and fixed glowing tiles, not indoor ambient targets or window daylight.
	// This field changes cosmetic tint only; it never enters visibility sources.
	// See ref:LIGHT-0007.
	if g.MapState.PlayerLocation.Location.GetMapType() == references.LargeMapType {
		g.daylightTinting.PrepareOutdoor()
	} else {
		g.daylightTinting.PrepareSmallMap(layer, g.MapState.IsPositionOpenDoor)
	}
	g.daylightLayer = layer
}

// DaylightTintTarget returns a prepared full-day cosmetic brightness target.
// Call UpdateLightSourceTinting after visibility preparation and before drawing.
// Map/floor transitions and full night cannot reuse another layer's targets.
// Exposed boundary surfaces use outdoor light when viewed from outdoor space;
// this does not reclassify the wall or brighten the room behind it.
func (g *GameState) DaylightTintTarget(pos references.Position) (float32, bool) {
	if g.daylightTinting == nil || !g.usesDaylightTinting() || g.daylightLayer != g.GetLayeredMapByCurrentLocation() {
		return 0, false
	}

	target, ok := g.daylightTinting.TargetAt(pos)
	layer := g.daylightLayer
	if !ok || layer.ExposureAt(pos) != map_state.ExposureBoundary {
		return target, ok
	}
	viewer := g.MapState.PlayerLocation.Position
	exposure := layer.ExposureAt(viewer)
	outdoorView := exposure == map_state.ExposureOutdoor
	if exposure == map_state.ExposureBoundary {
		// An exterior threshold belongs to the sunlit side for this whole-tile
		// presentation decision; enclosed internal doorways stay shaded.
		_, outdoorView = daylightOutdoorNeighbor(layer, viewer)
	}
	if outdoorView {
		if outside, found := daylightOutdoorNeighbor(layer, pos); found {
			return g.daylightTinting.TargetAt(outside)
		}
		// Wall corners can face outdoor space diagonally. This target-only check
		// must not turn an internal threshold into an outdoor viewpoint.
		for _, step := range [...]references.Position{{X: -1, Y: -1}, {X: 1, Y: -1}, {X: -1, Y: 1}, {X: 1, Y: 1}} {
			outside := references.Position{X: pos.X + step.X, Y: pos.Y + step.Y}
			if layer.ExposureAt(outside) == map_state.ExposureOutdoor {
				return g.daylightTinting.TargetAt(outside)
			}
		}
	}

	return target, true
}

func daylightOutdoorNeighbor(layer *map_state.LayeredMap, pos references.Position) (references.Position, bool) {
	for _, step := range [...]references.Position{{X: -1}, {X: 1}, {Y: -1}, {Y: 1}} {
		neighbor := references.Position{X: pos.X + step.X, Y: pos.Y + step.Y}
		if layer.ExposureAt(neighbor) == map_state.ExposureOutdoor {
			return neighbor, true
		}
	}

	return references.Position{}, false
}
