package game_state

import (
	"math"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

const (
	lordBritishHarpsichordFloor = 2
	lordBritishSandalwoodBoxNPC = 31
	lordBritishPassageX         = 17
	lordBritishPassageY         = 13

	harpsichordEarthquakeWorkflowName = "lord_british_harpsichord_earthquake"
	harpsichordEarthquakeDurationMs   = 1000
	harpsichordEarthquakeIntensity    = 0.7
	harpsichordEarthquakePixels       = 10.0
	harpsichordEarthquakeCycles       = 15
	harpsichordMelody                 = "6789878767653"
)

// CanPlayHarpsichord reports whether a harpsichord is directly south of the Avatar.
func (g *GameState) CanPlayHarpsichord() bool {
	if g.MapState.PlayerLocation.Location.GetMapType() != references.SmallMapType {
		return false
	}

	position := references.Down.GetNewPositionInDirection(&g.MapState.PlayerLocation.Position)
	if position == nil {
		return false
	}
	tile := g.GetCurrentLayeredMap().GetTileTopMapOnlyTile(position)

	return tile != nil && tile.Is(indexes.Harpsichord)
}

// ActionPlayHarpsichordNote plays one digit note and advances the original melody matcher.
func (g *GameState) ActionPlayHarpsichordNote(digit int) bool {
	sound, validDigit := HarpsichordSoundEffectForDigit(digit)
	if !validDigit || !g.CanPlayHarpsichord() {
		return false
	}

	g.SystemCallbacks.Audio.PlaySoundEffect(sound)

	if digit == int(harpsichordMelody[g.harpsichordMelodyProgress]-'0') {
		g.harpsichordMelodyProgress++
		if g.harpsichordMelodyProgress == len(harpsichordMelody) {
			g.harpsichordMelodyProgress = 0
			g.completeHarpsichordMelody()
		}

		return true
	}

	// Preserve the original overlap-aware fallback from melody recognition.
	// See ref:QUEST-0002.
	switch {
	case g.harpsichordMelodyProgress == 10 && digit == 8:
		g.harpsichordMelodyProgress = 3
	case g.harpsichordMelodyProgress == 11 && digit == 7:
		g.harpsichordMelodyProgress = 2
	case digit == int(harpsichordMelody[0]-'0'):
		g.harpsichordMelodyProgress = 1
	default:
		g.harpsichordMelodyProgress = 0
	}

	return true
}

func (g *GameState) completeHarpsichordMelody() {
	if g.MapState.PlayerLocation.Location != references.LordBritishsCastle ||
		g.MapState.PlayerLocation.Floor != lordBritishHarpsichordFloor {
		return
	}

	// REDUX DEVIATION: Original melody recognition XORs the passage tile,
	// so repeated performances close it. Redux permanently opens the solved passage.
	// See ref:QUEST-0002.
	g.lordBritishPassageOpen = true
	g.applyLordBritishPassageState()
	g.startHarpsichordEarthquakeWorkflow()
}

func (g *GameState) startHarpsichordEarthquakeWorkflow() {
	var voice SoundVoice = SilentSoundVoice{}
	g.EventQueueManager.Workflow(harpsichordEarthquakeWorkflowName).
		Priority(event_queue.PriorityMedium).
		Sequential().
		BlockInput().
		First(func() {
			voice = g.SystemCallbacks.Audio.Play(SoundEarthquake).StartOwned()
		}).
		Animate(harpsichordEarthquakeDurationMs, func(progress float32) {
			dampen := 1 - float64(progress)
			phase := float64(progress) * harpsichordEarthquakeCycles * 2 * math.Pi
			offset := harpsichordEarthquakePixels * harpsichordEarthquakeIntensity * dampen
			g.SystemCallbacks.Visual.SetScreenShakeOffset(math.Sin(phase)*offset, math.Cos(phase)*offset)
		}).
		Finally(func() {
			g.SystemCallbacks.Visual.SetScreenShakeOffset(0, 0)
			voice.Stop(earthquakeSoundFadeOutMs)
		}).
		Start()
}

func (g *GameState) resetHarpsichordMelody() {
	// REDUX DEVIATION: Original town rules keeps function-static melody progress
	// across maps. Redux clears partial progress at floor and location boundaries.
	// See ref:QUEST-0002.
	g.harpsichordMelodyProgress = 0
}

func (g *GameState) applyLordBritishPassageState() {
	if g.PartyState.Inventory.QuestItems.HasSome(references.WoodenBox) {
		// REDUX DEVIATION: Original town rules reloads the passage as a wall and
		// tracks box removal separately. Redux treats box ownership as proof that
		// the puzzle was solved, keeping the passage open and the box NPC absent.
		// See ref:QUEST-0002.
		g.lordBritishPassageOpen = true
		g.PartyState.SetDeadNpc(references.LordBritishsCastle, lordBritishSandalwoodBoxNPC)
	}

	if !g.lordBritishPassageOpen || g.MapState.PlayerLocation.Location != references.LordBritishsCastle {
		return
	}

	layeredMap := g.MapState.LayeredMaps.GetLayeredMap(references.SmallMapType, lordBritishHarpsichordFloor)
	if layeredMap == nil {
		return
	}
	position := lordBritishPassagePosition()
	layeredMap.SetTileByLayer(map_state.MapLayer, &position, indexes.BrickFloor)
	layeredMap.InvalidateLightSourceCache()
}

func lordBritishPassagePosition() references.Position {
	return references.Position{X: lordBritishPassageX, Y: lordBritishPassageY}
}

func isLordBritishPassageSnapshotPosition(location references.Location, floor references.FloorNumber, x, y references.Coordinate) bool {
	return location == references.LordBritishsCastle && floor == lordBritishHarpsichordFloor &&
		x == lordBritishPassageX && y == lordBritishPassageY
}
