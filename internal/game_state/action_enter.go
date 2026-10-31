package game_state

import (
	"fmt"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

const (
	smallMapStartingPositionX     = 15
	smallMapStartingPositionY     = 30
	smallMapStartingPositionFloor = 0
)

// Enter location transition timing constants
const (
	EnterFadeOutMs = 500 // Fade to black duration
	EnterFadeInMs  = 500 // Fade from black duration
)

// EnterLargeMapOutcome identifies the authoritative result of one Enter command.
type EnterLargeMapOutcome uint8

const (
	// EnterLargeMapRejected means Enter did not start an interaction or transition.
	EnterLargeMapRejected EnterLargeMapOutcome = iota
	// EnterLargeMapInteraction means Enter started a non-location interaction such as a shrine.
	EnterLargeMapInteraction
	// EnterLargeMapTransition means Enter started an ordinary location transition.
	EnterLargeMapTransition
	// EnterLargeMapDoomTransition means the party was admitted to Doom.
	EnterLargeMapDoomTransition
	// EnterLargeMapDoomCombat means living Shadowlords intercepted the party at Doom's entrance.
	EnterLargeMapDoomCombat
)

// Accepted reports whether Enter started authoritative work.
func (outcome EnterLargeMapOutcome) Accepted() bool {
	return outcome != EnterLargeMapRejected
}

// ResolveEnterLargeMap handles Enter and exposes its exact authoritative outcome.
func (g *GameState) ResolveEnterLargeMap() EnterLargeMapOutcome {
	g.SystemCallbacks.Message.ShowCommandPrompt("Enter ")

	// Check for shrine or codex tile first (before location check)
	currentMap := g.GetLayeredMapByCurrentLocation()
	pos := &g.MapState.PlayerLocation.Position

	topTile := currentMap.GetTopTile(pos)
	if topTile != nil {
		// Check for shrine tile
		if topTile.Is(indexes.Shrine) {
			shrineIndex := GetShrineByPosition(
				byte(g.MapState.PlayerLocation.Position.X),
				byte(g.MapState.PlayerLocation.Position.Y),
			)
			if shrineIndex >= 0 {
				if g.IsShrineRuined(shrineIndex) {
					g.SystemCallbacks.Message.AppendToCurrentRowStr("ruins")

					return EnterLargeMapRejected
				}
				g.SystemCallbacks.Message.AppendToCurrentRowStr(
					"the shrine of\n" + allShrines[shrineIndex].VirtueName,
				)
				g.observeAtlasLandmark(*pos, "shrine", "the shrine of "+allShrines[shrineIndex].VirtueName,
					journal.AtlasSourceShrineIntroduction, g.sampleDungeonObservation())

				g.SystemCallbacks.Flow.SacredSites.StartShrineWorkflow(shrineIndex)

				return EnterLargeMapInteraction
			}
		}

		// Check for broken/ruined shrine tile (fallback if tile is already BrokenShrine)
		if topTile.Is(indexes.BrokenShrine) {
			g.SystemCallbacks.Message.AppendToCurrentRowStr("ruins")

			return EnterLargeMapRejected
		}

		// Check for codex tile
		if topTile.Is(indexes.Codex) {
			g.SystemCallbacks.Message.AppendToCurrentRowStr("the Shrine of the Codex!")
			g.SystemCallbacks.Flow.SacredSites.StartCodexWorkflow()

			return EnterLargeMapInteraction
		}
	}

	// A deed square is never a world location; its hut leads to the deed map (ref:CRAFT-0004).
	if g.ResolveEnterDeed() {
		return EnterLargeMapTransition
	}

	newLocation := g.GameReferences.LocationReferences.WorldLocations.GetLocationByPosition(
		g.MapState.PlayerLocation.Position)

	if newLocation != references.EmptyLocation {
		slr := g.GameReferences.LocationReferences.GetLocationReference(newLocation)
		if slr.Location != references.EmptyLocation {
			if newLocation.GetMapType() == references.DungeonMapType {
				effectiveTile := &references.Tile{Index: g.GetEffectiveTile(pos)}
				if !g.canEnterDungeonAtCurrentLocation(newLocation, effectiveTile.Index) {
					g.SystemCallbacks.Message.AppendToCurrentRowStr("What?")

					return EnterLargeMapRejected
				}

				g.SystemCallbacks.Message.AppendToCurrentRowStr(enterLocationDescription(effectiveTile, slr))
				if g.PartyVehicle.GetVehicleDetails().VehicleType != references.NoPartyVehicle {
					g.SystemCallbacks.Message.AddRowStr("On foot!")

					return EnterLargeMapRejected
				}
				if newLocation == references.Doom && !g.ShadowlordState.AllDefeated() {
					g.SystemCallbacks.Message.AddRowStr("Attacked at entrance!")
					if _, err := g.startDoomEntranceCombatEncounter(); err != nil {
						logging.Errorf(logging.CategoryGameState, "Unable to start Doom entrance combat: %v", err)
						g.SystemCallbacks.Message.AddRowStr("Combat could not begin.")

						return EnterLargeMapRejected
					}

					return EnterLargeMapDoomCombat
				}
			} else {
				g.SystemCallbacks.Message.AppendToCurrentRowStr(enterLocationDescription(topTile, slr))
			}
			g.StartEnterLocationWorkflow(slr)

			if newLocation == references.Doom {
				return EnterLargeMapDoomTransition
			}

			return EnterLargeMapTransition
		}
	}

	g.SystemCallbacks.Message.AppendToCurrentRowStr("What?")

	return EnterLargeMapRejected
}

// IsDoomEnterAttempt reports whether the current world coordinate names Doom,
// including admission attempts that authoritative terrain or state will reject.
func (g *GameState) IsDoomEnterAttempt() bool {
	return g.GameReferences.LocationReferences.WorldLocations.GetLocationByPosition(
		g.MapState.PlayerLocation.Position,
	) == references.Doom
}

func (g *GameState) canEnterDungeonAtCurrentLocation(
	dungeon references.Location,
	effectiveTile indexes.SpriteIndex,
) bool {
	if !g.IsDungeonOpen(dungeon) {
		return false
	}
	if effectiveTile != indexes.CaveEntrance &&
		effectiveTile != indexes.MineEntrance &&
		effectiveTile != indexes.DoomEntrance {
		return false
	}
	entrance, exists := g.GetDungeonEntranceCoordinates()[dungeon]
	if !exists || entrance != g.MapState.PlayerLocation.Position {
		return false
	}
	floor := g.MapState.PlayerLocation.Floor
	if dungeon == references.Doom {
		return floor == references.FloorNumber(references.UNDERWORLD)
	}

	return floor == references.FloorNumber(references.OVERWORLD) ||
		floor == references.FloorNumber(references.UNDERWORLD)
}

func enterLocationDescription(tile *references.Tile, location *references.SmallLocationReference) string {
	if tile == nil {
		return location.FriendlyLocationName
	}
	descriptions := map[indexes.SpriteIndex]string{
		indexes.Hut:                  "hut",
		indexes.Keep:                 "keep",
		indexes.Village:              "village",
		indexes.SmallCastle:          "towne",
		indexes.LargeCastle:          "castle",
		indexes.CaveEntrance:         "cave",
		indexes.MineEntrance:         "mine",
		indexes.DoomEntrance:         "dungeon",
		indexes.Lighthouse:           "lighthouse",
		indexes.EvilCastleEntrance:   "the palace of Blackthorn!",
		indexes.CastleBritianEntrace: "the Castle of Lord British!",
	}
	if description, exists := descriptions[tile.Index]; exists {
		return description
	}

	return location.FriendlyLocationName
}

// StartEnterLocationWorkflow creates event queue for location entry with fade transition
// Sequence: fade out → transition → show popup → fade in → Shadowlord presence effects (if any)
func (g *GameState) StartEnterLocationWorkflow(slr *references.SmallLocationReference) {
	routeOwnsCycle := g.surfaceCycleOwned
	if g.EventQueueManager == nil {
		panic("GameState.StartEnterLocationWorkflow: EventQueueManager is nil")
	}

	if g.SystemCallbacks == nil {
		panic("GameState.StartEnterLocationWorkflow: SystemCallbacks is nil")
	}

	// Check if enter sequence is already running
	if g.EventQueueManager.HasWorkflow("enter_location") {
		return
	}

	// Capture transition result for post-transition effects (Shadowlord presence)
	var transitionResult TransitionResult
	entranceLocation := g.MapState.PlayerLocation
	entranceObservation := g.sampleDungeonObservation()
	entranceKind := atlasKind(g.GetEffectiveTile(&entranceLocation.Position))
	entryFloor := references.FloorNumber(smallMapStartingPositionFloor)
	entryPosition := references.Position{X: smallMapStartingPositionX, Y: smallMapStartingPositionY}
	var preparedDungeon *DungeonState
	if slr.Location.GetMapType() == references.DungeonMapType {
		var facing references.DungeonFacing
		var lastDirection references.DungeonLastDirection
		entryFloor, entryPosition, facing, lastDirection = DungeonEntrySetup(
			slr.Location,
			g.MapState.PlayerLocation.Floor,
		)
		var err error
		preparedDungeon, err = g.PrepareDungeonState(slr.Location, facing, lastDirection)
		if err != nil {
			panic(fmt.Errorf("prepare dungeon entry for %s: %w", slr.Location, err))
		}
	}

	g.EventQueueManager.Workflow("enter_location").
		Priority(event_queue.PriorityCritical).
		Sequential().BlockInput().
		First(func() {

		}).
		Animate(EnterFadeOutMs, func(progress float32) {
			g.SystemCallbacks.Visual.SetMapFadeLevel(float64(progress))
		}).
		Then(func() {
			if preparedDungeon != nil {
				g.installPreparedDungeonState(preparedDungeon)
			}
			transitionResult = g.TransitionToLocation(
				slr.Location,
				entryFloor,
				entryPosition,
				TransitionFullLocation,
				true,
			)
			if preparedDungeon != nil && slr.Location != references.Doom {
				g.observeDungeonEntrance(slr.Location, entranceLocation.Position, entranceObservation, false)
			}
			// REDUX DEVIATION: location entry changes maps without a dedicated
			// entry sound. Redux exposes the completed transition as an empty cue slot.
			// See ref:CMD-0001.
			g.SystemCallbacks.Audio.PlaySoundEffect(SoundLocationEnter)
			if preparedDungeon != nil {
				g.generateDungeonWandererAfterFreshEntry()
			}

			g.SystemCallbacks.UI.ShowActionFeedback(slr.EnteringText, true)
			// The Atlas already carries the deed's own pin; a deed is not a public landmark.
			if preparedDungeon == nil && !slr.Location.IsDeedLocation() {
				g.observeAtlasLandmark(entranceLocation.Position, entranceKind,
					atlasPresentedEntryName(slr.EnteringText), journal.AtlasSourceEnter, entranceObservation)
			}
		}).
		Animate(EnterFadeInMs, func(progress float32) {
			g.SystemCallbacks.Visual.SetMapFadeLevel(1.0 - float64(progress))
		}).
		Then(func() {
			if !routeOwnsCycle && preparedDungeon == nil {
				g.AdvanceGameplayMinutes(1)
			}
		}).
		ThenSequence(

			func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
				if len(transitionResult.ShadowlordsPresent) > 0 {
					g.SystemCallbacks.Flow.Shadowlord.ThenShadowlordPresenceEffect(transitionResult.ShadowlordsPresent)(q)
				}

				return q
			}).
		Finally(

			func() {
				g.SystemCallbacks.Visual.SetMapFadeLevel(0.0)
				if !routeOwnsCycle {
					g.ProcessPendingProvisionHours()
				}
			}).
		Start()
}

// DungeonEntrySetup resolves the original entry coordinate and orientation for
// a dungeon approached from Britannia or the Underworld.
func DungeonEntrySetup(
	location references.Location,
	sourceFloor references.FloorNumber,
) (
	references.FloorNumber,
	references.Position,
	references.DungeonFacing,
	references.DungeonLastDirection,
) {
	if sourceFloor == references.FloorNumber(references.UNDERWORLD) && location != references.Doom {
		return references.DungeonFloorCount - 1,
			references.Position{X: references.XDungeonMapTiles - 1, Y: references.YDungeonMapTiles - 1},
			references.DungeonFacingWest,
			references.DungeonArrivedFromBelow
	}

	return 0,
		references.Position{X: 1, Y: 1},
		references.DungeonFacingEast,
		references.DungeonArrivedFromAbove
}

// DungeonUnderworldExitSetup resolves the down-ladder endpoint that exits a
// non-Doom dungeon to the Underworld. Doom has no such endpoint.
func DungeonUnderworldExitSetup(
	location references.Location,
) (
	references.FloorNumber,
	references.Position,
	references.DungeonFacing,
	references.DungeonLastDirection,
	bool,
) {
	if location < references.Deceit || location >= references.Doom {
		return 0, references.Position{}, 0, 0, false
	}

	return references.DungeonFloorCount - 1,
		references.Position{X: references.XDungeonMapTiles - 1, Y: references.YDungeonMapTiles - 1},
		references.DungeonFacingWest,
		references.DungeonLastWest,
		true
}

// ThenExitLocationTransition returns a modifier function for use with ThenSequence
// that adds exit location transition effects (fade out → transition → popup → fade in)
// to an existing event queue builder chain.
//
// This avoids the nested event queue anti-pattern by returning a modifier
// instead of starting a new queue.
//
// Usage:
//
//	yesQ.ThenSequence(g.gameState.ThenExitLocationTransition())
func (g *GameState) ThenExitLocationTransition() func(
	*event_queue.SequentialEventBuilder,
) *event_queue.SequentialEventBuilder {
	routeOwnsCycle := g.surfaceCycleOwned

	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		// Capture exiting text and exit coordinates at call time (before transition changes location)
		var exitingText string
		var exitPosition references.Position
		var exitFloor references.FloorNumber

		return q.Then(func() {
			currentLocation := g.MapState.PlayerLocation.Location
			slr := g.locationReference(currentLocation)
			exitingText = "Leaving..."
			if slr != nil {
				exitingText = slr.GetExitingText()
			}

			// A deed map is not in DATA.OVL's world table; it leads back to its hut (ref:CRAFT-0004).
			if currentLocation.IsDeedLocation() {
				exitPosition, exitFloor = g.deedExit()

				return
			}
			exitPosition, exitFloor = g.GameReferences.LocationReferences.WorldLocations.GetWorldLocation(currentLocation)
		}).
			Animate(EnterFadeOutMs, func(progress float32) {
				g.SystemCallbacks.Visual.SetMapFadeLevel(float64(progress))
			}).
			Then(func() {
				g.TransitionToLocation(
					references.BritanniaUnderworld,
					exitFloor,
					exitPosition,
					TransitionFullLocation,
					false,
				)

				g.SystemCallbacks.UI.ShowActionFeedback(strings.TrimSpace(exitingText), true)
			}).
			Animate(EnterFadeInMs, func(progress float32) {
				g.SystemCallbacks.Visual.SetMapFadeLevel(1.0 - float64(progress))
			}).
			Then(func() {
				g.SystemCallbacks.Visual.SetMapFadeLevel(0.0)
				if !routeOwnsCycle {
					g.AdvanceGameplayMinutes(1)
					g.ProcessPendingProvisionHours()
				}
			})
	}
}
