package environment

import (
	"math"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// WindTickIntervalMs is the deterministic approximation of the original 18 Hz tick.
const WindTickIntervalMs int64 = 55

const (
	windCandidateCount      = 5
	windCalmRollRange       = 256
	windCalmAcceptThreshold = 192
)

type randomIntn interface {
	Intn(n int) int
}

// WindSystemCallbacks interface to avoid import cycles with game_state
type WindSystemCallbacks interface {
	ShowWindUI(direction WindDirection)
	AddRowStr(message string)
}

type WindDirection int

const (
	Calm  WindDirection = 0
	North WindDirection = 1
	South WindDirection = 2
	East  WindDirection = 3
	West  WindDirection = 4
)

func (w WindDirection) String() string {
	switch w {
	case Calm:
		return "Calm"
	case North:
		return "North"
	case South:
		return "South"
	case East:
		return "East"
	case West:
		return "West"
	default:
		return "Unknown"
	}
}

func (w WindDirection) GetUIString() string {
	return w.String() + " Winds"
}

type WindSystem struct {
	rng           randomIntn
	callbacks     WindSystemCallbacks
	currentWind   WindDirection
	updateCounter int
	nextTickMs    int64
}

// WindSnapshot is a detached deterministic wind state.
type WindSnapshot struct {
	Direction     WindDirection
	UpdateCounter int
	NextTickMs    int64
}

// Snapshot returns the serializable wind state.
func (w *WindSystem) Snapshot() WindSnapshot {
	return WindSnapshot{Direction: w.currentWind, UpdateCounter: w.updateCounter, NextTickMs: w.nextTickMs}
}

// Restore replaces the wind state from a serialized snapshot.
func (w *WindSystem) Restore(snapshot WindSnapshot) {
	w.currentWind = snapshot.Direction
	w.updateCounter = snapshot.UpdateCounter
	w.nextTickMs = snapshot.NextTickMs
}

// NewWindSystem constructs the deterministic wind system.
func NewWindSystem(rng randomIntn, callbacks WindSystemCallbacks) *WindSystem {
	return &WindSystem{
		rng:           rng,
		callbacks:     callbacks,
		currentWind:   Calm, // Start with calm winds
		updateCounter: 0,
		nextTickMs:    WindTickIntervalMs,
	}
}

// SetCallbacks binds presentation callbacks without replacing deterministic wind state.
func (w *WindSystem) SetCallbacks(callbacks WindSystemCallbacks) {
	w.callbacks = callbacks
}

// AdvanceThrough processes every wind tick due at elapsedMs and preserves the next tick phase.
func (w *WindSystem) AdvanceThrough(elapsedMs int64) {
	// Reject impossible runtime clocks before consuming RNG or wrapping a deadline.
	if elapsedMs < 0 || elapsedMs > math.MaxInt64-WindTickIntervalMs || w.nextTickMs <= 0 {
		panic("wind clock is outside the safe deadline domain")
	}
	for w.nextTickMs <= elapsedMs {
		w.UpdateWindTick()
		w.nextTickMs += WindTickIntervalMs
	}
}

// UpdateWindTick processes automatic wind changes (1/64 chance per tick)
func (w *WindSystem) UpdateWindTick() {
	// 1/64 chance per tick to change wind
	if w.rng.Intn(64) == 0 {
		w.generateRandomWindChange()
	}
}

// SetWind manually sets wind direction (used by spells, debug commands, etc.)
func (w *WindSystem) SetWind(direction WindDirection, immediate bool) {
	if direction >= Calm && direction <= West {
		w.currentWind = direction
		w.updateCounter = 0
		if immediate {
			w.callbacks.ShowWindUI(direction)
		}
	}
}

// GetCurrentWind returns the current wind direction
func (w *WindSystem) GetCurrentWind() WindDirection {
	return w.currentWind
}

// SetWindByDirection sets wind using references.Direction (for spell integration)
func (w *WindSystem) SetWindByDirection(direction references.Direction, immediate bool) bool {
	var windDir WindDirection
	switch direction {
	case references.Up:
		windDir = North
	case references.Down:
		windDir = South
	case references.Right:
		windDir = East
	case references.Left:
		windDir = West
	default:
		return false // Invalid direction (including references.NoneDirection)
	}

	w.SetWind(windDir, immediate)

	return true
}

// generateRandomWindChange implements the original Ultima V wind change algorithm
func (w *WindSystem) generateRandomWindChange() {
	var newWind WindDirection

	// Keep rolling until we get a valid wind (with calm bias)
	for {
		newWind = WindDirection(w.rng.Intn(windCandidateCount)) // 0..4 inclusive

		// Calm bias: when calm is picked, only accept byte rolls 192 through 255.
		if newWind == Calm && w.rng.Intn(windCalmRollRange) < windCalmAcceptThreshold {
			continue // Reject this calm and try again
		}

		break // Accept this wind direction
	}

	w.SetWind(newWind, true) // Show UI update immediately
}

// GetFrigateSailingSpeed calculates frigate movement speed based on wind direction
// Returns the number of ticks required for movement (authentic Ultima V algorithm)
func (w *WindSystem) GetFrigateSailingSpeed(direction references.Direction) int {
	// If no wind (calm), ship movement is very slow - return high tick count
	if w.currentWind == Calm {
		return 4 // Stalled in calm conditions - matches test expectations
	}

	// Wind vectors in order are south, north, west, east; ref:SHIP-0001.
	// Index by winds-1: [North-1, South-1, East-1, West-1] = [0, 1, 2, 3]
	// Note: 0xff represents -1 in C's signed byte interpretation
	// Wind represents direction wind is blowing FROM (not TOWARDS)
	windVectors := map[WindDirection][2]int{
		North: {0, 1},  // Wind from north blows south.
		South: {0, -1}, // Wind from south blows north.
		East:  {-1, 0}, // Wind from east blows west.
		West:  {1, 0},  // Wind from west blows east.
	}

	// Convert direction to movement vector (Ultima V coordinate system)
	// Note: In original C code, 0xff is used as -1 in comparisons
	var dx, dy int
	switch direction {
	case references.Up: // North (DIR_NORTH: dy = 0xff)
		dx, dy = 0, -1 // 0xff interpreted as -1 for comparison
	case references.Down: // South (DIR_SOUTH: dy = 1)
		dx, dy = 0, 1
	case references.Right: // East (DIR_EAST: dx = 1)
		dx, dy = 1, 0
	case references.Left: // West (DIR_WEST: dx = 0xff)
		dx, dy = -1, 0 // 0xff interpreted as -1 for comparison
	default:
		return 1
	}

	// Start at one wait; add one for each movement component differing from wind; ref:SHIP-0001.
	windVec := windVectors[w.currentWind]
	num := 1
	if windVec[0] != dx {
		num++
	}
	if windVec[1] != dy {
		num++
	}

	return num
}

// IncrementUpdateCount increments the internal update counter for sailing timing
func (w *WindSystem) IncrementUpdateCount() {
	w.updateCounter++
}

// ShouldMoveShip returns true if the ship should move based on wind timing
// Used in sailing mode: updtcnt >= num % 3
func (w *WindSystem) ShouldMoveShip(direction references.Direction) bool {
	speed := w.GetFrigateSailingSpeed(direction)

	return w.updateCounter >= (speed % 3)
}

// IsWindCalm returns true if current wind is calm
func (w *WindSystem) IsWindCalm() bool {
	return w.currentWind == Calm
}

// GetPirateShipSpeedMultiplier maintains compatibility with existing game state integration
// This is an alias for GetFrigateSailingSpeed to match existing interface expectations
func (w *WindSystem) GetPirateShipSpeedMultiplier(shipDirection references.Direction) int {
	return w.GetFrigateSailingSpeed(shipDirection)
}
