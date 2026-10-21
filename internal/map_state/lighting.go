package map_state

import (
	"math"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/pkg/helpers"
)

const (
	// TorchTileDistance is the illumination radius of a carried torch.
	TorchTileDistance = 3
	// DefaultTorchDurationMinutes is the initial fuel time of a lit torch.
	DefaultTorchDurationMinutes = 240

	// Moongate light distance
	MoongateLightDistance = 4 // Moongates emit mystical light with 4 tile radius

	// Quest item light distances
	ShardLightDistance    = 3 // Shards of Evil emit light with 3 tile radius
	ArtifactLightDistance = 4 // Lord British's Artifacts emit stronger light with 4 tile radius

	// MagicLightTileDistance is the original U5 radius for both In Lor and Vas Lor.
	MagicLightTileDistance = 5
	InLorTileDistance      = MagicLightTileDistance
	VasLorTileDistance     = MagicLightTileDistance

	// NPC light distances - magical creatures that emit light
	// All start at 2 tiles for subtle glow effect
	WispLightDistance     = 2 // Ethereal energy beings - blue-white glow
	SeahorseLightDistance = 2 // Magical sea creatures - bright blue glow
	DaemonLightDistance   = 2 // Demonic fire creatures - red glow
	GhostLightDistance    = 2 // Spectral beings - pale blue glow
)

type Lighting struct {
	torchMinutesRemaining int
	gameDimensions        GameDimensions
	baselineFactor        float32
	baselineRadius        int

	// Magical light is enabled while GameState has remaining magic_light minutes.
	hasMagicLight bool
}

// LightSource represents a tile that emits light in the game world
type LightSource struct {
	Tile *references.Tile
	Pos  references.Position
}

// LightSources is a collection of light sources
type LightSources []LightSource

type DistanceMap map[references.Position]int

func NewLighting(gameDimensions GameDimensions) Lighting {
	l := Lighting{}
	l.gameDimensions = gameDimensions
	l.baselineFactor = 0.1
	l.baselineRadius = 1

	return l
}

func (l *Lighting) HasTorchLit() bool {
	return l.torchMinutesRemaining > 0
}

// TorchMinutesRemaining returns the carried torch's authoritative fuel time.
func (l *Lighting) TorchMinutesRemaining() int {
	return l.torchMinutesRemaining
}

// SetMagicLight enables or disables player-carried magical light.
func (l *Lighting) SetMagicLight(active bool) {
	l.hasMagicLight = active
}

// HasMagicLight returns true if magic light is active.
func (l *Lighting) HasMagicLight() bool {
	return l.hasMagicLight
}

// HasAnyLight returns true if torch OR magic light is providing illumination.
func (l *Lighting) HasAnyLight() bool {
	return l.HasTorchLit() || l.hasMagicLight
}

func (l *Lighting) LightTorch() {
	l.torchMinutesRemaining = DefaultTorchDurationMinutes
}

// ExtendTorchMinutes adds carried fuel and saturates at maximum.
func (l *Lighting) ExtendTorchMinutes(minutes, maximum int) int {
	if minutes < 0 {
		panic("torch extension minutes cannot be negative")
	}
	if maximum < 0 {
		panic("torch maximum cannot be negative")
	}
	l.torchMinutesRemaining = min(l.torchMinutesRemaining+minutes, maximum)

	return l.torchMinutesRemaining
}

// RestoreTorchMinutes restores carried fuel without consuming another torch.
func (l *Lighting) RestoreTorchMinutes(minutes int) {
	if minutes < 0 {
		panic("carried torch minutes cannot be negative")
	}
	l.torchMinutesRemaining = minutes
}

// ExtinguishTorch immediately extinguishes the torch (used by death resurrection).
func (l *Lighting) ExtinguishTorch() {
	l.torchMinutesRemaining = 0
}

// AdvanceMinutes burns a carried torch by actual elapsed gameplay minutes.
func (l *Lighting) AdvanceMinutes(minutes int) {
	if minutes <= 0 {
		return
	}
	l.torchMinutesRemaining = helpers.Max(l.torchMinutesRemaining-minutes, 0)
}

func (l *Lighting) BuildGameScreenDistanceMap(centrePos references.Position) DistanceMap {
	dmm := make(DistanceMap)
	l.applyLightSource(dmm, centrePos, 0, 1)

	return dmm
}

func (l *Lighting) BuildLightSourceDistanceMap(
	lightSources LightSources,
	visibleFlags VisibilityCoords,
	playerLightType PlayerLightType,
	centerPos references.Position,
) DistanceMap {
	distanceMaskMap := make(DistanceMap)

	// Apply player-carried light based on light type (torch, magic light, ambient)
	// This uses the same PlayerLightProperties() as the tinting system for consistency
	if playerLightType != PlayerLightNone {
		props := PlayerLightProperties(playerLightType)
		if props.Radius > 0 {
			l.applyLightSource(distanceMaskMap, centerPos, props.Radius, 1)
		}
	}

	for _, ls := range lightSources {
		if visibleFlags[ls.Pos.X][ls.Pos.Y] {
			l.applyLightSource(distanceMaskMap, ls.Pos, ls.Tile.GetLightEmission(), 1)
		}
	}

	return distanceMaskMap
}

func (l *Lighting) applyLightSource(
	distanceMap DistanceMap,
	centrePos references.Position,
	nRadius int, // 0 means “use factor”
	fFactor float32, // ignored when radius > 0
) {
	// Decide radius
	if nRadius <= 0 {
		nRadius = l.radiusForFactor(fFactor)
	}
	if nRadius <= 0 {
		return // nothing to do
	}

	// (r + 0.5)² → keeps outline round
	maxDist2 := math.Pow(float64(nRadius)+0.5, 2)

	bottomRight := l.gameDimensions.GetBottomRightWithoutOverflow()
	isWrapped := l.gameDimensions.IsWrappedMap()

	// Reuse single Position struct to avoid allocations in nested loops
	var position references.Position

	for deltaY := -nRadius; deltaY <= nRadius; deltaY++ {
		yy := deltaY * deltaY
		for deltaX := -nRadius; deltaX <= nRadius; deltaX++ {
			dist2 := float64(deltaX*deltaX + yy)
			if dist2 > maxDist2 { // outside disc
				continue
			}

			// Update position in-place instead of creating new struct
			position.X = centrePos.X + references.Coordinate(deltaX)
			position.Y = centrePos.Y + references.Coordinate(deltaY)

			if isWrapped {
				position = *position.GetWrapped(bottomRight.X, bottomRight.Y)
			}

			d := int(math.Sqrt(dist2)) // Euclidean distance 0‥radius
			if cur := distanceMap[position]; cur == 0 || d < cur {
				distanceMap[position] = d
			}
		}
	}
}

func (l *Lighting) radiusForFactor(factor float32) int {
	switch {
	case factor <= l.baselineFactor:
		return l.baselineRadius
	case factor >= 1:
		return l.maxRadius()
	default:
		norm := (factor - l.baselineFactor) / (1 - l.baselineFactor) // 0‥1

		return l.baselineRadius +
			int(math.Round(float64(norm)*float64(l.maxRadius()-l.baselineRadius)))
	}
}

func (l *Lighting) maxRadius() int {
	xTilesVisibleOnGameScreen, yTilesVisibleOnGameScreen := l.gameDimensions.GetTilesVisibleOnScreen()

	return int(math.Ceil(
		math.Hypot(float64(xTilesVisibleOnGameScreen)/2, float64(yTilesVisibleOnGameScreen)/2),
	))
}
