package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
)

const (
	surfaceCameraDefaultDurationMs  = 125
	surfaceCameraMountedSpeedFactor = 1.25
)

// Original town movement commits a whole-tile step and requests
// a LOS refresh. This opt-in prototype adds a camera tween after authoritative
// movement and contact finish, without retaining a gameplay completion token.
// The frame offset is sampled once so every world layer uses exactly the same projection.
type surfaceCameraProjection struct {
	active           bool
	destination      references.PlayerLocation
	layer            *map_state.LayeredMap
	delta            references.Position
	startedMs        int64
	durationMs       int64
	smoothstep       bool
	automaticSailing bool
	x, y             float64
}

func (p *surfaceCameraProjection) sample(now int64) {
	p.x, p.y = 0, 0
	if !p.active {
		return
	}
	if p.durationMs <= 0 || now-p.startedMs >= p.durationMs {
		*p = surfaceCameraProjection{}

		return
	}
	t := min(1.0, max(0.0, float64(now-p.startedMs)/float64(p.durationMs)))
	if p.smoothstep {
		t = t * t * (3 - 2*t)
	}
	// Eastward movement shifts the destination-centered world right initially.
	p.x = float64(p.delta.X) * sprites.TileSize * (1 - t)
	p.y = float64(p.delta.Y) * sprites.TileSize * (1 - t)
}

func (g *GameScene) canScrollSurface() bool {
	if !g.canProjectSurface() || g.gameState.IsSailing() || g.gameState.EventQueueManager.HasActiveWorkflows() {
		return false
	}
	vehicle := g.gameState.PartyVehicle.GetVehicleDetails()
	switch vehicle.VehicleType {
	case references.NoPartyVehicle, references.HorseVehicle, references.CarpetVehicle, references.SkiffVehicle:
		return true
	case references.FrigateVehicle:
		return vehicle.GetSailState() == map_units.SailFurled
	case references.NPC:
		return false
	default:
		return false
	}
}

func (g *GameScene) clearSurfaceCameraForInput(key ebiten.Key) {
	normalized := primaryMovementKey(key)
	isMovement := normalized == ebiten.KeyUp || normalized == ebiten.KeyDown ||
		normalized == ebiten.KeyLeft || normalized == ebiten.KeyRight
	if g.surfaceCamera.active && g.surfaceCamera.automaticSailing && g.gameState != nil &&
		g.gameState.IsSailing() && isMovement {
		return
	}
	g.surfaceCamera = surfaceCameraProjection{}
}

func (g *GameScene) canScrollAutomaticSailing() bool {
	if !g.canProjectSurface() || !g.gameState.IsSailing() ||
		!g.gameState.EventQueueManager.HasOnlyWorkflow(automaticSailingWorkflowName) {
		return false
	}
	vehicle := g.gameState.PartyVehicle.GetVehicleDetails()

	return vehicle.VehicleType == references.FrigateVehicle && vehicle.GetSailState() == map_units.SailRaised
}

func (g *GameScene) canProjectSurface() bool {
	if g.gameConfig == nil || g.gameConfig.GameplayConfiguration == nil || g.gameConfig.DebugConfiguration == nil ||
		!g.gameConfig.GameplayConfiguration.SmoothSurfaceMovement || g.disposed {
		return false
	}
	state := g.gameState
	if state == nil {
		return false
	}
	mapType := state.MapState.PlayerLocation.Location.GetMapType()
	if (mapType != references.SmallMapType && mapType != references.LargeMapType) || state.HasPendingOverworldContact() ||
		state.IsInCombat() || state.IsInCamping() || state.IsInMiniMap() ||
		g.miniMapScene != nil || g.finalePresentation != nil || !state.CanAcceptGameplayCommands() {
		return false
	}
	if state.IsFalling() || state.IsTeleportationActive() ||
		g.inputBlocked || g.secondaryKeyState != PrimaryInput || g.autoOpenDoorState.IsActive() ||
		g.activeProjectile != nil || len(g.visualEffects) != 0 || len(g.activeTileDissolves) != 0 ||
		g.avatarDissolve.GetLevel() != 0 || g.mapFadeLevel != 0 ||
		g.hideMapUnits || g.hideMoongates || g.hideItemStacks || g.hideQuestItems ||
		state.DebugOptions.LightOverlay || state.DebugOptions.SearchOverlay ||
		(g.overlayRenderer != nil && g.overlayRenderer.IsMoonstoneOverlayEnabled()) ||
		g.dialogStack.HasOpenDialog() {
		return false
	}
	background := state.GetAvatarSpriteAtPosition(state.MapState.PlayerLocation.Position).BackgroundIntegrated

	return !background
}

func (g *GameScene) handleSurfaceCameraMovement(key ebiten.Key) {
	origin := g.gameState.MapState.PlayerLocation
	eligible := g.canScrollSurface()
	originLayer := g.gameState.GetLayeredMapByCurrentLocation()
	direction := g.getKeyAsDirection(key)
	expected := origin
	expected.Position = *direction.GetNewPositionInDirection(&origin.Position)
	if origin.Location.GetMapType() == references.LargeMapType {
		expected.Position = *expected.Position.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
	}
	result := g.handleMovement(direction.GetDirectionCompassName(), key)
	g.completeSurfaceMovement(result)
	// Completion may synchronously push the Avatar back or enter combat. Never defer it.
	if eligible && result.State == surfaceCommandCompleted && result.MovementOccurred && !result.MapTransitioned &&
		g.gameState.MapState.PlayerLocation == expected && g.gameState.GetLayeredMapByCurrentLocation() == originLayer {
		g.installSurfaceCamera(origin, expected)
	}
}

func (g *GameScene) installSurfaceCamera(origin, expected references.PlayerLocation) {
	if g.gameState.MapState.PlayerLocation != expected || !g.canScrollSurface() ||
		origin.Location != expected.Location || origin.Floor != expected.Floor {
		return
	}
	delta := references.Position{X: expected.Position.X - origin.Position.X, Y: expected.Position.Y - origin.Position.Y}
	if origin.Location.GetMapType() == references.LargeMapType {
		delta = wrappedSurfaceDelta(expected.Position, origin.Position)
	}
	if math.Abs(float64(delta.X))+math.Abs(float64(delta.Y)) != 1 {
		return
	}
	settings := g.gameConfig.DebugConfiguration
	duration := settings.SurfaceMovementDurationMs
	if duration != 100 && duration != 125 && duration != 150 {
		duration = surfaceCameraDefaultDurationMs
	}
	vehicle := g.gameState.PartyVehicle.GetVehicleDetails().VehicleType
	if vehicle == references.HorseVehicle || vehicle == references.CarpetVehicle {
		// Cosmetic only: mounted scrolling is 25% faster without changing movement cadence or turn costs.
		duration = int(float64(duration) / surfaceCameraMountedSpeedFactor)
	}
	g.surfaceCamera = surfaceCameraProjection{
		active: true, destination: expected, delta: delta, startedMs: g.getElapsedMs(),
		layer:      g.gameState.GetLayeredMapByCurrentLocation(),
		durationMs: int64(duration), smoothstep: settings.SurfaceMovementCurve == "smoothstep",
	}
}

func (g *GameScene) installAutomaticSailingCamera(origin, expected references.PlayerLocation, layer *map_state.LayeredMap) {
	if g.gameState.MapState.PlayerLocation != expected || g.gameState.GetLayeredMapByCurrentLocation() != layer ||
		origin.Location != expected.Location || origin.Floor != expected.Floor || !g.canScrollAutomaticSailing() {
		return
	}
	delta := wrappedSurfaceDelta(expected.Position, origin.Position)
	if math.Abs(float64(delta.X))+math.Abs(float64(delta.Y)) != 1 {
		return
	}
	g.surfaceCamera = surfaceCameraProjection{
		active: true, destination: expected, layer: layer, delta: delta, startedMs: g.getElapsedMs(),
		durationMs:       automaticSailingPhaseIntervalMs,
		smoothstep:       g.gameConfig.DebugConfiguration.SurfaceMovementCurve == "smoothstep",
		automaticSailing: true,
	}
}

// Use the nearest world copy for movement and overlays at the 256-tile seams.
func wrappedSurfaceDelta(world, origin references.Position) references.Position {
	delta := references.Position{X: world.X - origin.X, Y: world.Y - origin.Y}
	delta.X = (delta.X%references.XLargeMapTiles+references.XLargeMapTiles*3/2)%references.XLargeMapTiles - references.XLargeMapTiles/2
	delta.Y = (delta.Y%references.YLargeMapTiles+references.YLargeMapTiles*3/2)%references.YLargeMapTiles - references.YLargeMapTiles/2

	return delta
}

func (g *GameScene) surfaceWorldRelativePosition(world references.Position) references.Position {
	player := g.gameState.MapState.PlayerLocation
	if g.gameConfig != nil && g.gameConfig.GameplayConfiguration != nil &&
		g.gameConfig.GameplayConfiguration.SmoothSurfaceMovement && player.Location.GetMapType() == references.LargeMapType {
		return wrappedSurfaceDelta(world, player.Position)
	}

	return references.Position{X: world.X - player.Position.X, Y: world.Y - player.Position.Y}
}

func (g *GameScene) updateSurfaceCamera() {
	if !g.surfaceCamera.active {
		return
	}
	eligible := g.canScrollSurface()
	if g.surfaceCamera.automaticSailing {
		eligible = g.canScrollAutomaticSailing()
	}
	if g.gameState.MapState.PlayerLocation != g.surfaceCamera.destination || !eligible ||
		g.gameState.GetLayeredMapByCurrentLocation() != g.surfaceCamera.layer {
		g.surfaceCamera = surfaceCameraProjection{}

		return
	}
	g.surfaceCamera.sample(g.getElapsedMs())
}

func (g *GameScene) surfaceOverscanTiles() references.Coordinate {
	if g.surfaceCamera.active {
		// One moving tile plus padded sprite edges at the viewport boundary.
		return 2
	}

	return 0
}

// Full daylight covers the settled viewport, but its distance cutoff clips the
// corners exposed by scrolling. Sample that presentation-only fringe using the
// existing LOS flags; never expand gameplay visibility or carried-light ranges.
func (g *GameScene) isDaylightSurfaceFringeVisible(layer *map_state.LayeredMap, pos references.Position) bool {
	if !g.surfaceCamera.active || g.gameState.MapState.PlayerLocation.Floor < 0 ||
		g.gameState.DateTime.GetVisibilityFactorWithoutTorch(0.1) < 1 {
		return false
	}
	relative := g.surfaceWorldRelativePosition(pos)
	x, y := relative.X+xCenter, relative.Y+yCenter
	margin := g.surfaceOverscanTiles()
	if x < -margin || x >= xTilesVisibleOnGameScreen+margin || y < -margin || y >= yTilesVisibleOnGameScreen+margin ||
		(x >= 0 && x < xTilesVisibleOnGameScreen && y >= 0 && y < yTilesVisibleOnGameScreen) {
		return false
	}

	return layer.GetVisibilityFlags()[pos.X][pos.Y]
}

// Source camera offset, then layout scaling/placement, then screen-space shake.
// Avatar and attached items intentionally retain the unshifted layout transform.
func (g *GameScene) worldDrawOptions() *ebiten.DrawImageOptions {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(g.surfaceCamera.x, g.surfaceCamera.y)
	layout := sprites.GetDrawOptionsFromPercentsForWholeScreen(g.mapImage, g.mapScreenPlacement())
	op.GeoM.Concat(layout.GeoM)
	op.GeoM.Translate(g.screenShakeOffsetX, g.screenShakeOffsetY)

	return op
}
