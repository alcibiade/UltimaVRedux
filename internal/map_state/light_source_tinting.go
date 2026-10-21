// Package map_state provides light source tinting for visual atmosphere.
//
// This implements colored light tinting for torches, braziers, fireplaces, etc.
// Light sources tint nearby tiles with warm or cool colors at night.
//
// Original light-source behavior is documented in docs/gamefacts/LIGHTING.md.
package map_state

import (
	"math"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// ============================================================================
// TUNABLE PARAMETERS - Adjust these to change light tinting behavior
// ============================================================================

// Breathing animation parameters (slow, gentle oscillation)
const (
	// BreathingPeriodMs is the time for one complete breathing cycle.
	// Higher = slower breathing. 2000-3000ms recommended for visible effect.
	BreathingPeriodMs = 2250

	// BreathingAmplitude controls how much the light intensity varies.
	// 0.05 = ±5% variation (very subtle), 0.10 = ±10% (more noticeable)
	BreathingAmplitude           = 0.065
	breathingPhaseHashMultiplier = 17
)

// Moongate breathing animation (faster, more intense than fire-based sources)
const (
	// MoongateBreathingPeriodMs is the moongate pulse cycle (matches sprite glow)
	MoongateBreathingPeriodMs = 1500

	// MoongateBreathingAmplitude is the moongate intensity variation (±35%)
	MoongateBreathingAmplitude = 0.35
)

// Color tinting intensity parameters
const (
	// TintStrength controls how strongly the light color affects tiles.
	// 0.0 = no tinting, 1.0 = full color. 0.15-0.25 recommended for subtle warmth.
	TintStrength          = 0.25
	halfDayTintMultiplier = 0.5

	// BaseBrightness is the minimum brightness multiplier at the edge of light.
	// 1.0 = no darkening. Lower values = darker edges.
	// 0.55 = 45% darker at edges for moderate falloff
	BaseBrightness = 0.55

	// CenterBrightness is the brightness at the light source itself.
	// Values > 1.0 make the center slightly brighter.
	CenterBrightness = 1.05
)

// Blending parameters for overlapping light sources
const (
	// FalloffExponent controls how quickly light fades with distance.
	// 1.0 = linear, 2.0 = inverse square (realistic), 1.5 = balanced
	FalloffExponent = 1.5

	// MaxSourcesPerTile limits how many light sources affect a single tile.
	// More sources = smoother blending but more computation.
	MaxSourcesPerTile = 4
)

// Unlit area parameters
const (
	// UnlitDarkeningFactor darkens tiles outside any light source radius.
	// Applied when there are active light sources to create contrast.
	// 1.0 = no darkening, 0.75 = 25% darker, 0.40 = 60% darker.
	// Previous value: 0.55 (45% darker)
	UnlitDarkeningFactor float32 = 0.40
)

// ============================================================================
// Ambient Light Configuration
// ============================================================================

// ambientLightRadius is the configurable radius for PlayerLightAmbient.
// Set via SetAmbientLightRadius() during game initialization.
// Default: 2 tiles. Range: 1-5.
//
// To change: Call map_state.SetAmbientLightRadius() from gamescene_initialization.go
// or modify config.AmbientLightRadius in the config file.
var ambientLightRadius = 2

// SetAmbientLightRadius configures the ambient light radius for night vision.
// Called during game initialization with value from config.
// Falls back to 2 if radius < 1.
func SetAmbientLightRadius(radius int) {
	if radius < 1 {
		radius = 2
	}
	ambientLightRadius = radius
}

// ============================================================================
// Math helpers
// ============================================================================

const pi = math.Pi

func sin(x float64) float64    { return math.Sin(x) }
func sqrt(x float64) float64   { return math.Sqrt(x) }
func pow(x, y float64) float64 { return math.Pow(x, y) }

func clampFloat(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}

	return v
}

// ============================================================================
// Player Light Types
// ============================================================================

// PlayerLightType identifies the source of player-carried light
type PlayerLightType int

const (
	PlayerLightNone    PlayerLightType = iota
	PlayerLightAmbient                 // Minimal night vision, neutral gray, 2 tile radius
	PlayerLightTorch                   // Orange light, 3 tile radius
	PlayerLightInLor                   // White light, 5 tile radius (In Lor spell)
	PlayerLightVasLor                  // Bright white, 6 tile radius (Vas Lor spell)
)

// ============================================================================
// Light Source Properties
// ============================================================================

// TintMode controls when a light source applies color tinting to nearby tiles.
type TintMode int

const (
	// TintNightOnly applies tinting only at night (default for fire-based sources)
	TintNightOnly TintMode = iota
	// TintHalfDuringDay applies full tint at night, 50% tint during day (subtle magical glow)
	TintHalfDuringDay
	// TintAlways applies full tint at all times (strong magical sources)
	TintAlways
)

// LightSourceProperties defines visual characteristics of a light source.
// These properties control how nearby tiles are tinted at night.
type LightSourceProperties struct {
	// TintColor is the color this light adds to tiles (as RGB scale factors > 1.0)
	// Values > 1.0 brighten that channel, < 1.0 darken it
	TintColor LightTintColor

	// HasBreathing enables slow breathing animation for fire-based sources
	HasBreathing bool

	// BreathingPeriodMs is the custom breathing period in milliseconds.
	// 0 = use default BreathingPeriodMs constant (2250ms)
	BreathingPeriodMs int

	// BreathingAmplitude is the custom breathing intensity variation for emitted light.
	// 0 = use default BreathingAmplitude constant (0.065)
	BreathingAmplitude float64

	// IndependentBreathing varies the breathing period per-source based on position.
	// When true, each instance breathes at a slightly different rate (±15% variation).
	// When false (default), all sources of this type breathe at the same rate.
	// Use for fire sources to create organic flickering; leave false for magical sources.
	IndependentBreathing bool

	// Radius overrides the tile's LightEmission if non-zero
	Radius int

	// TintMode controls when this light source applies color tinting.
	// TintNightOnly (default): tint only at night
	// TintHalfDuringDay: full tint at night, 50% during day
	// TintAlways: full tint at all times
	TintMode TintMode
}

// LightTintColor represents RGB scale factors for light tinting.
// Values around 1.0 are neutral. > 1.0 brightens, < 1.0 darkens.
// These are NOT raw colors but multipliers applied to existing tile colors.
type LightTintColor struct {
	R, G, B float32
}

// Warm light tint colors (fire-based sources)
// These add warmth by boosting red/green and reducing blue for orange/amber glow
var (
	// TorchWarm adds a warm orange tint for torches and sconces
	TorchWarm = LightTintColor{R: 1.47, G: 0.93, B: 0.17}

	// BrazierWarm adds a deeper orange tint (hot coals effect)
	BrazierWarm = LightTintColor{R: 1.47, G: 0.93, B: 0.17}

	// FireplaceWarm adds a reddish-orange tint (burning logs)
	FireplaceWarm = LightTintColor{R: 1.47, G: 0.93, B: 0.17}

	// CandleWarm adds a soft amber tint (gentler than torch)
	CandleWarm = LightTintColor{R: 1.47, G: 0.93, B: 0.17}

	// CookStoveRed adds a redder tint for cooking fires
	CookStoveRed = LightTintColor{R: 1.55, G: 0.75, B: 0.15}

	// LampPostWhite adds a whiter/cooler tint for street lamps
	LampPostWhite = LightTintColor{R: 1.25, G: 1.15, B: 0.90}
)

// Cool light tint colors (magical sources)
var (
	// BlueFlameCool is a strong distinct blue tint for magical blue flames
	// Reduces red/green and boosts blue for a clear magical blue glow
	BlueFlameCool = LightTintColor{R: 0.70, G: 0.85, B: 1.45}

	// MagicWhite is near-neutral (slight blue for magical feel)
	MagicWhite = LightTintColor{R: 1.00, G: 1.00, B: 1.02}

	// MoongatePurpleTint is a mystical blue tint for moongates
	// Bright, slightly white-shifted blue for a glowing magical feel
	MoongatePurpleTint = LightTintColor{R: 0.82, G: 0.90, B: 1.50}
)

// Lighthouse tint color (warm red for the rotating beam)
var (
	// LighthouseRed is the warm red tint for lighthouse beams.
	// Creates a distinctive red glow for the rotating lighthouse cone.
	LighthouseRed = LightTintColor{R: 1.15, G: 0.80, B: 0.80}
)

// LighthouseMarkerIndex is a special sprite index marker for lighthouse light sources.
// Used to identify lighthouse contributions in the lit tiles map.
const LighthouseMarkerIndex indexes.SpriteIndex = 0xFFFE

// MoongateMarkerIndex is a special sprite index marker for moongate light sources.
// Used to identify moongate contributions in the lit tiles map for purple/blue tinting.
const MoongateMarkerIndex indexes.SpriteIndex = 0xFFFD

// ============================================================================
// NPC Light Source Marker Indexes
// ============================================================================
// These marker indexes identify magical NPCs that emit light.
// Used for tint color lookup in LightSourcePropertiesMap.
// Allocation: 0xFFFC and below (0xFFFE=Lighthouse, 0xFFFD=Moongate)

const (
	WispMarkerIndex     indexes.SpriteIndex = 0xFFFC // Ethereal energy beings
	SeahorseMarkerIndex indexes.SpriteIndex = 0xFFFB // Magical sea creatures
	DaemonMarkerIndex   indexes.SpriteIndex = 0xFFFA // Demonic fire creatures
	GhostMarkerIndex    indexes.SpriteIndex = 0xFFF9 // Spectral beings
)

// ============================================================================
// Quest Item Light Source Marker Indexes
// ============================================================================
// These marker indexes identify quest items that emit light.
// Used for tint color lookup in LightSourcePropertiesMap.
// Allocation: 0xFFF8 and below

const (
	ShardMarkerIndex    indexes.SpriteIndex = 0xFFF8 // Shards of Evil (corrupted virtue gems)
	ArtifactMarkerIndex indexes.SpriteIndex = 0xFFF7 // Lord British's Artifacts
)

// ============================================================================
// NPC Light Tint Colors
// ============================================================================
// Colors for magical creatures that emit light.

var (
	// WispGlow is an ethereal blue-white glow for energy beings
	WispGlow = LightTintColor{R: 0.90, G: 0.95, B: 1.40}

	// SeahorseGlow is a bright white-cyan magical glow for mystical sea creatures
	// Uses high brightness with slight cyan tint to stand out against blue water
	SeahorseGlow = LightTintColor{R: 1.30, G: 1.45, B: 1.50}

	// DaemonGlow is a red demonic glow for fire creatures
	DaemonGlow = LightTintColor{R: 1.40, G: 0.85, B: 0.75}

	// GhostGlow is a pale blue spectral glow for undead spirits
	GhostGlow = LightTintColor{R: 0.95, G: 0.98, B: 1.25}
)

// ============================================================================
// Quest Item Tint Colors
// ============================================================================
// Colors for quest items that emit light.

var (
	// AmbientSightDim provides minimal visibility in darkness without a torch.
	// Neutral gray, slightly brighter than complete darkness but much dimmer than torch.
	// Used for PlayerLightAmbient to give the avatar basic night vision.
	AmbientSightDim = LightTintColor{R: 0.55, G: 0.55, B: 0.55}

	// ShardGlow is a sinister red-purple glow for the corrupted Shards of Evil
	// These represent corrupted virtue gems and should look ominous
	ShardGlow = LightTintColor{R: 1.30, G: 0.80, B: 1.20}

	// ArtifactGlow is a noble golden-white glow for Lord British's Artifacts
	// Crown, Sceptre, and Amulet should have a regal, pure light
	ArtifactGlow = LightTintColor{R: 1.22, G: 1.18, B: 0.85}
)

// NeutralTint applies no color change (for reference)
var NeutralTint = LightTintColor{R: 1.0, G: 1.0, B: 1.0}

// LightSourcePropertiesMap maps sprite indexes to their light properties.
var LightSourcePropertiesMap = map[indexes.SpriteIndex]LightSourceProperties{
	// Wall-mounted torches/sconces (independent breathing for organic fire)
	indexes.RightSconce: {TintColor: TorchWarm, HasBreathing: true, BreathingAmplitude: 0.6, IndependentBreathing: true, TintMode: TintHalfDuringDay},
	indexes.LeftSconce:  {TintColor: TorchWarm, HasBreathing: true, BreathingAmplitude: 0.6, IndependentBreathing: true, TintMode: TintHalfDuringDay},

	// Braziers - standing fire bowls (independent breathing for organic fire)
	indexes.Brazier: {TintColor: BrazierWarm, HasBreathing: true, IndependentBreathing: true, TintMode: TintHalfDuringDay},

	// Fireplaces and burning logs (independent breathing for organic fire)
	indexes.Fireplace: {TintColor: FireplaceWarm, HasBreathing: true, IndependentBreathing: true, TintMode: TintHalfDuringDay},
	indexes.FireLogs:  {TintColor: FireplaceWarm, HasBreathing: true, IndependentBreathing: true, TintMode: TintHalfDuringDay},

	// Campfire (independent breathing for organic fire)
	indexes.CampFire: {TintColor: TorchWarm, HasBreathing: true, IndependentBreathing: true, TintMode: TintHalfDuringDay},

	// Candles (independent breathing for organic fire)
	indexes.CandleOnTable: {TintColor: CandleWarm, HasBreathing: true, IndependentBreathing: true, TintMode: TintHalfDuringDay},

	// Street lamps (steady - no breathing, whiter light)
	indexes.LampPost: {TintColor: LampPostWhite, HasBreathing: false, TintMode: TintHalfDuringDay},

	// Blue magical flame (synchronized - magical source)
	indexes.BlueFlame: {TintColor: BlueFlameCool, HasBreathing: true, BreathingAmplitude: 0.4, TintMode: TintAlways},

	// Fire field spell effect (independent breathing for organic fire)
	indexes.FireField: {TintColor: TorchWarm, HasBreathing: true, IndependentBreathing: true, TintMode: TintHalfDuringDay},

	// Cook stove (steady, redder flame)
	indexes.CookStove: {TintColor: CookStoveRed, HasBreathing: false, TintMode: TintHalfDuringDay},

	// Ground torch item (independent breathing for organic fire)
	indexes.ItemTorch: {TintColor: TorchWarm, HasBreathing: true, IndependentBreathing: true, TintMode: TintHalfDuringDay},

	// Lighthouse beam (steady mechanical light - no breathing)
	LighthouseMarkerIndex: {TintColor: LighthouseRed, HasBreathing: false, TintMode: TintHalfDuringDay},

	// Moongate mystical glow (pulsing purple/blue light with faster, stronger pulse)
	MoongateMarkerIndex: {
		TintColor:          MoongatePurpleTint,
		HasBreathing:       true,
		BreathingPeriodMs:  MoongateBreathingPeriodMs,
		BreathingAmplitude: MoongateBreathingAmplitude,
		TintMode:           TintAlways,
	},

	// ============================================================================
	// Magical NPC light sources
	// ============================================================================

	// Wisps - ethereal energy beings with flickering blue-white glow
	WispMarkerIndex: {
		TintColor:          WispGlow,
		HasBreathing:       true,
		BreathingPeriodMs:  1200, // Fast flicker for energy beings
		BreathingAmplitude: 0.25,
		TintMode:           TintAlways,
	},

	// Seahorses - mystical sea creatures with pulsing white-cyan glow
	SeahorseMarkerIndex: {
		TintColor:          SeahorseGlow,
		HasBreathing:       true,
		BreathingPeriodMs:  1800, // Gentle magical pulse
		BreathingAmplitude: 0.45, // Light emission amplitude
		TintMode:           TintHalfDuringDay,
	},

	// Daemons - demonic fire creatures with flickering red glow
	DaemonMarkerIndex: {
		TintColor:          DaemonGlow,
		HasBreathing:       true,
		BreathingPeriodMs:  1400, // Fire flicker
		BreathingAmplitude: 0.50, // Very visible pulse for testing
		TintMode:           TintAlways,
	},

	// Ghosts - spectral beings with wavering pale blue glow
	GhostMarkerIndex: {
		TintColor:          GhostGlow,
		HasBreathing:       true,
		BreathingPeriodMs:  2000, // Slow spectral waver
		BreathingAmplitude: 0.15,
		TintMode:           TintHalfDuringDay,
	},

	// ============================================================================
	// Quest item light sources
	// ============================================================================

	// Shards of Evil - corrupted virtue gems with sinister red-purple pulsing glow
	ShardMarkerIndex: {
		TintColor:          ShardGlow,
		HasBreathing:       true,
		BreathingPeriodMs:  2500, // Slow, ominous pulse
		BreathingAmplitude: 0.20,
		TintMode:           TintAlways,
	},

	// Lord British's Artifacts - noble golden-white steady glow
	// Crown, Sceptre, and Amulet have a regal, pure light
	ArtifactMarkerIndex: {
		TintColor:          ArtifactGlow,
		HasBreathing:       true,
		BreathingPeriodMs:  2500, // Slow, majestic pulse
		BreathingAmplitude: 0.35,
		TintMode:           TintHalfDuringDay,
	},
}

// PlayerLightProperties returns light properties for player-carried light sources.
func PlayerLightProperties(lightType PlayerLightType) LightSourceProperties {
	switch lightType {
	case PlayerLightAmbient:
		return LightSourceProperties{
			TintColor:    AmbientSightDim,
			HasBreathing: false,
			Radius:       ambientLightRadius, // Configurable via SetAmbientLightRadius()
		}
	case PlayerLightTorch:
		return LightSourceProperties{
			TintColor:    TorchWarm,
			HasBreathing: true,
			Radius:       TorchTileDistance,
		}
	case PlayerLightInLor:
		return LightSourceProperties{
			TintColor:    MagicWhite,
			HasBreathing: false,
			Radius:       InLorTileDistance,
		}
	case PlayerLightVasLor:
		return LightSourceProperties{
			TintColor:    MagicWhite,
			HasBreathing: false,
			Radius:       VasLorTileDistance,
		}
	default:
		return LightSourceProperties{}
	}
}

// GetLightSourceProperties returns the light properties for a sprite index.
func GetLightSourceProperties(spriteIndex indexes.SpriteIndex) LightSourceProperties {
	if props, ok := LightSourcePropertiesMap[spriteIndex]; ok {
		return props
	}

	return LightSourceProperties{TintColor: NeutralTint}
}

// ============================================================================
// Light Contribution - tracks a single light source's effect on a tile
// ============================================================================

// LightContribution represents one light source's contribution to a tile.
type LightContribution struct {
	SourceIndex indexes.SpriteIndex
	SourcePos   references.Position
	Distance    float64 // Actual distance (not squared)
	Radius      int     // Max radius of this source
	Weight      float64 // Computed weight based on distance
}

// ============================================================================
// Light Source Tinting State
// ============================================================================

// LightSourceTintingState tracks which tiles are lit by light sources for tinting.
// Supports blending multiple overlapping light sources smoothly.
type LightSourceTintingState struct {
	// litTilesMap maps positions to all light sources affecting that tile
	litTilesMap map[references.Position][]LightContribution

	// isLargeMap indicates if operating on wrapping 256x256 map
	isLargeMap bool

	// playerLightType tracks the current player light source
	playerLightType PlayerLightType

	// playerPos tracks the player's current position
	playerPos references.Position
}

// NewLightSourceTintingState creates a new light source tinting state.
func NewLightSourceTintingState() *LightSourceTintingState {
	return &LightSourceTintingState{
		litTilesMap:     make(map[references.Position][]LightContribution),
		playerLightType: PlayerLightNone,
	}
}

// SetLargeMapMode sets whether the tinting operates on a wrapping large map.
func (ls *LightSourceTintingState) SetLargeMapMode(isLargeMap bool) {
	ls.isLargeMap = isLargeMap
}

// BuildLitTilesMap recalculates which tiles are lit by light sources.
// Only light sources visible to the player (determined by visibleFlags) will contribute.
// This ensures that light from behind walls doesn't bleed through.
// Pass nil for visibleFlags to include all light sources (useful for tests).
func (ls *LightSourceTintingState) BuildLitTilesMap(
	lightSources LightSources,
	visibleFlags VisibilityCoords,
) {
	ls.litTilesMap = make(map[references.Position][]LightContribution)

	for _, source := range lightSources {
		// Skip light sources that aren't visible to the player
		// This prevents light from bleeding through walls
		if visibleFlags != nil && !visibleFlags[source.Pos.X][source.Pos.Y] {
			continue
		}

		spriteIndex := source.Tile.Index
		radius := source.Tile.GetLightEmission()
		if radius <= 0 {
			continue
		}

		ls.addLitTilesForSource(source.Pos, spriteIndex, radius)
	}
}

// addLitTilesForSource adds light contributions for a single source.
func (ls *LightSourceTintingState) addLitTilesForSource(
	sourcePos references.Position,
	spriteIndex indexes.SpriteIndex,
	radius int,
) {
	// Use radius + 0.5 to match lighting.go's applyLightSource calculation.
	// This ensures tinting covers the exact same tiles that visibility does.
	// Without this, edge tiles would be lit but not tinted.
	maxDist := float64(radius) + 0.5

	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			dist := sqrt(float64(dx*dx + dy*dy))
			if dist > maxDist {
				continue // Outside circle
			}

			pos := references.Position{
				X: sourcePos.X + references.Coordinate(dx),
				Y: sourcePos.Y + references.Coordinate(dy),
			}

			if ls.isLargeMap {
				pos = *pos.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
			}

			// Calculate weight using inverse distance with falloff exponent
			// Weight = 1 / (distance + 1)^exponent
			weight := 1.0 / pow(dist+1.0, FalloffExponent)

			contribution := LightContribution{
				SourceIndex: spriteIndex,
				SourcePos:   sourcePos,
				Distance:    dist,
				Radius:      radius,
				Weight:      weight,
			}

			// Add to existing contributions for this tile
			existing := ls.litTilesMap[pos]
			if len(existing) < MaxSourcesPerTile {
				ls.litTilesMap[pos] = append(existing, contribution)
			} else {
				// Replace weakest contribution if this one is stronger
				minIdx := 0
				minWeight := existing[0].Weight
				for i, c := range existing {
					if c.Weight < minWeight {
						minWeight = c.Weight
						minIdx = i
					}
				}
				if weight > minWeight {
					ls.litTilesMap[pos][minIdx] = contribution
				}
			}
		}
	}
}

// UpdatePlayerLight updates the lit tiles map with the player's current light source.
func (ls *LightSourceTintingState) UpdatePlayerLight(
	playerPos references.Position,
	lightType PlayerLightType,
) {
	// Clear previous player light contribution if position or type changed
	if ls.playerLightType != PlayerLightNone && (lightType != ls.playerLightType || playerPos != ls.playerPos) {
		ls.clearPlayerLight()
	}

	ls.playerLightType = lightType
	ls.playerPos = playerPos

	if lightType == PlayerLightNone {
		return
	}

	props := PlayerLightProperties(lightType)
	if props.Radius <= 0 {
		return
	}

	var spriteIndex indexes.SpriteIndex
	switch lightType {
	case PlayerLightAmbient:
		spriteIndex = indexes.SpriteIndex(0) // Ambient sight has no specific sprite
	case PlayerLightTorch:
		spriteIndex = indexes.ItemTorch
	case PlayerLightInLor, PlayerLightVasLor:
		spriteIndex = indexes.SpriteIndex(0) // Magic light marker
	}

	ls.addLitTilesForSource(playerPos, spriteIndex, props.Radius)
}

// clearPlayerLight removes the player's light contribution from tiles.
func (ls *LightSourceTintingState) clearPlayerLight() {
	if ls.playerLightType == PlayerLightNone {
		return
	}

	props := PlayerLightProperties(ls.playerLightType)
	radius := props.Radius

	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			pos := references.Position{
				X: ls.playerPos.X + references.Coordinate(dx),
				Y: ls.playerPos.Y + references.Coordinate(dy),
			}

			if ls.isLargeMap {
				pos = *pos.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
			}

			// Remove contributions from this player position
			if contributions, ok := ls.litTilesMap[pos]; ok {
				filtered := contributions[:0]
				for _, c := range contributions {
					if c.SourcePos != ls.playerPos {
						filtered = append(filtered, c)
					}
				}
				if len(filtered) > 0 {
					ls.litTilesMap[pos] = filtered
				} else {
					delete(ls.litTilesMap, pos)
				}
			}
		}
	}
}

// GetTintScaleFactors returns the RGB scale factors for a tile at the given position.
// Returns nil if the tile is not lit by any light source (or all sources are filtered out).
// The elapsedMs parameter is used for breathing animation.
// The ambientDarkness parameter controls TintMode filtering and smoothly ranges
// from 0 during full daylight to 1 at night:
//   - TintNightOnly sources scale with ambient darkness
//   - TintHalfDuringDay sources scale from 50% intensity to full intensity
//   - TintAlways sources apply at full intensity always
//
// Color tint and brightness falloff are handled separately:
// - Color tint (warm/cool) stays consistent across the lit area
// - Brightness falls off with distance (darker at edges)
// This creates the effect of "same color, just dimmer" at edges.
func (ls *LightSourceTintingState) GetTintScaleFactors(
	pos references.Position,
	elapsedMs int64,
	ambientDarkness float32,
) *LightTintColor {
	return ls.getTintScaleFactors(pos, elapsedMs, ambientDarkness, 1)
}

// GetTintScaleFactorsWithDaylight composes source tint with a local daylight target.
// daylightTarget must be finite and in [0.3, 1]; invalid targets panic.
// ambientDarkness remains global clock darkness for TintMode and color intensity.
// Only sources eligible in full daylight can raise daytime brightness above the
// target, up to 1, before adding color deltas. Target 1 and full night preserve
// GetTintScaleFactors exactly.
// Unlit tiles and fully filtered contributions return nil for caller ambient fallback.
func (ls *LightSourceTintingState) GetTintScaleFactorsWithDaylight(
	pos references.Position,
	elapsedMs int64,
	ambientDarkness float32,
	daylightTarget float32,
) *LightTintColor {
	const minimumDaylightTarget float32 = 0.3
	if !(daylightTarget >= minimumDaylightTarget && daylightTarget <= 1) {
		panic("daylight target must be finite and in [0.3, 1]")
	}

	return ls.getTintScaleFactors(pos, elapsedMs, ambientDarkness, daylightTarget)
}

//nolint:cyclop,funlen // Keep source filtering and daylight composition together to preserve legacy arithmetic.
func (ls *LightSourceTintingState) getTintScaleFactors(
	pos references.Position,
	elapsedMs int64,
	ambientDarkness float32,
	daylightTarget float32,
) *LightTintColor {
	contributions, ok := ls.litTilesMap[pos]
	if !ok || len(contributions) == 0 {
		return nil
	}
	darkness := clampFloat(float64(ambientDarkness), 0, 1)

	// Blend all contributions weighted by inverse distance
	// We accumulate color DELTAS (deviation from neutral 1.0), not absolute values
	var totalDeltaR, totalDeltaG, totalDeltaB float64
	var totalWeight float64
	var closestDist float64 = 1000
	var closestRadius int
	closestDayDist := math.Inf(1)
	var closestDayRadius int

	for _, c := range contributions {
		props := GetLightSourceProperties(c.SourceIndex)

		// Determine tint multiplier based on TintMode and ambient darkness.
		var tintMultiplier = 1.0
		switch props.TintMode {
		case TintNightOnly:
			if darkness == 0 {
				continue
			}
			tintMultiplier = darkness
		case TintHalfDuringDay:
			tintMultiplier = halfDayTintMultiplier + halfDayTintMultiplier*darkness
		case TintAlways:
			tintMultiplier = 1.0
		}

		// Get base tint color deltas (how much each channel deviates from 1.0)
		deltaR := (float64(props.TintColor.R) - 1.0) * tintMultiplier
		deltaG := (float64(props.TintColor.G) - 1.0) * tintMultiplier
		deltaB := (float64(props.TintColor.B) - 1.0) * tintMultiplier

		// Apply breathing modulation if enabled (affects color intensity)
		if props.HasBreathing {
			// Use custom breathing parameters if specified, otherwise use defaults
			periodMs := props.BreathingPeriodMs
			if periodMs == 0 {
				periodMs = BreathingPeriodMs
			}
			amplitude := props.BreathingAmplitude
			if amplitude == 0 {
				amplitude = BreathingAmplitude
			}

			breathMod := ls.calculateBreathingModulationWithParams(c.SourcePos, elapsedMs, periodMs, amplitude, props.IndependentBreathing)
			deltaR *= breathMod
			deltaG *= breathMod
			deltaB *= breathMod
		}

		// Apply tint strength to dampen the color effect
		deltaR *= TintStrength
		deltaG *= TintStrength
		deltaB *= TintStrength

		// Accumulate weighted color deltas
		totalDeltaR += deltaR * c.Weight
		totalDeltaG += deltaG * c.Weight
		totalDeltaB += deltaB * c.Weight
		totalWeight += c.Weight

		// Track closest source for brightness falloff
		if c.Distance < closestDist {
			closestDist = c.Distance
			closestRadius = c.Radius
		}
		if props.TintMode != TintNightOnly && c.Distance < closestDayDist {
			closestDayDist = c.Distance
			closestDayRadius = c.Radius
		}
	}

	if totalWeight == 0 {
		return nil
	}

	// Calculate weighted average of color deltas
	avgDeltaR := totalDeltaR / totalWeight
	avgDeltaG := totalDeltaG / totalWeight
	avgDeltaB := totalDeltaB / totalWeight

	// Preserve the legacy nighttime falloff and outdoor interpolation arithmetic.
	const radiusPadding = 0.5
	var brightnessMult = 1.0
	nightBrightness := 1.0
	if darkness > 0 && closestRadius > 0 {
		// Use radius + 0.5 to match the extended lit area (same as addLitTilesForSource)
		maxDist := float64(closestRadius) + radiusPadding
		distRatio := clampFloat(closestDist/maxDist, 0, 1)
		nightBrightness = CenterBrightness + (BaseBrightness-CenterBrightness)*distRatio
		brightnessMult = 1 + (nightBrightness-1)*darkness
	}
	if daylightTarget < 1 && darkness < 1 {
		// Night-only sources must not lift the daytime endpoint when dawn/dusk starts.
		dayBrightness := float64(daylightTarget)
		if closestDayRadius > 0 {
			maxDist := float64(closestDayRadius) + radiusPadding
			distRatio := clampFloat(closestDayDist/maxDist, 0, 1)
			daySourceBrightness := CenterBrightness + (BaseBrightness-CenterBrightness)*distRatio
			dayBrightness = max(dayBrightness, min(1.0, daySourceBrightness))
		} else if !math.IsInf(closestDayDist, 1) {
			// An eligible source without a positive radius retains the default scalar of 1.
			dayBrightness = 1
		}
		brightnessMult = dayBrightness + (nightBrightness-dayBrightness)*darkness
	}

	// Combine brightness falloff with color tint:
	// - Full daylight: source brightness has a daylightTarget floor and a ceiling of 1
	// - Dawn/dusk: brightness and night-specific color deltas fade smoothly
	// - Nighttime: brightnessMult varies (1.05 at center, 0.55 at edge), plus color deltas
	finalR := float32(clampFloat(brightnessMult+avgDeltaR, 0.3, 1.5))
	finalG := float32(clampFloat(brightnessMult+avgDeltaG, 0.3, 1.5))
	finalB := float32(clampFloat(brightnessMult+avgDeltaB, 0.3, 1.5))

	return &LightTintColor{R: finalR, G: finalG, B: finalB}
}

// calculateBreathingModulationWithParams returns breathing modulation with custom parameters.
// periodMs controls how long one complete breathing cycle takes.
// amplitude controls how much the intensity varies (0.10 = ±10% variation).
// independentBreathing varies the period by ±15% based on position for organic fire effects.
// Uses position-based phase offset so different sources breathe at different phases.
func (ls *LightSourceTintingState) calculateBreathingModulationWithParams(
	sourcePos references.Position,
	elapsedMs int64,
	periodMs int,
	amplitude float64,
	independentBreathing bool,
) float64 {
	// Apply period variation for independent breathing (organic fire effect)
	// Each source gets a slightly different period based on its position
	posHash := sourcePos.AnimationHash()
	if independentBreathing {
		// Hash to get 0.85 to 1.15 multiplier (±15% variation)
		variation := 0.85 + 0.30*float64(posHash%100)/100.0
		periodMs = int(float64(periodMs) * variation)
	}

	// Position-based phase offset (different sources out of sync)
	phaseOffset := int64(posHash) * breathingPhaseHashMultiplier
	adjustedTime := elapsedMs + phaseOffset

	// Calculate breathing phase (0 to 2π)
	phase := float64(adjustedTime%int64(periodMs)) / float64(periodMs) * 2 * pi

	// Sine wave oscillation: returns -1 to 1, then scaled to (1-amp) to (1+amp)
	// When result is 1.0, we get full tint. When less, tint is reduced.
	breathValue := 1.0 + sin(phase)*amplitude

	return breathValue
}

// IsTileLit returns true if the tile at the given position is lit by any light source.
func (ls *LightSourceTintingState) IsTileLit(pos references.Position) bool {
	contributions, ok := ls.litTilesMap[pos]

	return ok && len(contributions) > 0
}

// GetLitTileCount returns the number of tiles currently lit (for debugging).
func (ls *LightSourceTintingState) GetLitTileCount() int {
	return len(ls.litTilesMap)
}

// Clear removes all lit tiles (used when leaving a map).
func (ls *LightSourceTintingState) Clear() {
	ls.litTilesMap = make(map[references.Position][]LightContribution)
	ls.playerLightType = PlayerLightNone
}

// ============================================================================
// Lighthouse Integration
// ============================================================================

// ClearLighthouseTiles removes all lighthouse contributions from the lit tiles map.
// Called before re-injecting lighthouse tiles after rotation.
func (ls *LightSourceTintingState) ClearLighthouseTiles() {
	for pos, contributions := range ls.litTilesMap {
		// Filter out lighthouse contributions
		filtered := contributions[:0]
		for _, c := range contributions {
			if c.SourceIndex != LighthouseMarkerIndex {
				filtered = append(filtered, c)
			}
		}

		if len(filtered) > 0 {
			ls.litTilesMap[pos] = filtered
		} else {
			delete(ls.litTilesMap, pos)
		}
	}
}

// InjectLighthouseTiles adds lighthouse-lit tiles to the tinting state.
// This integrates the lighthouse's rotating beam with the unified light source system.
// Each lit tile gets distance-based brightness falloff from the nearest beacon.
func (ls *LightSourceTintingState) InjectLighthouseTiles(lighthouseState *LighthouseState) {
	if lighthouseState == nil || !lighthouseState.HasLighthouses() {
		return
	}

	litTiles := lighthouseState.GetLitTiles()
	beaconPositions := lighthouseState.GetPositions()

	if len(litTiles) == 0 || len(beaconPositions) == 0 {
		return
	}

	// Estimate max distance for lighthouse beams (based on cone shapes, ~7 tiles max)
	const lighthouseMaxRadius = 7

	for tilePos := range litTiles {
		// Find distance to nearest beacon for brightness falloff
		minDist := float64(1000)
		for _, beaconPos := range beaconPositions {
			dx := float64(tilePos.X - beaconPos.X)
			dy := float64(tilePos.Y - beaconPos.Y)
			dist := sqrt(dx*dx + dy*dy)
			if dist < minDist {
				minDist = dist
			}
		}

		// Calculate weight based on distance (same formula as other lights)
		weight := 1.0 / pow(minDist+1.0, FalloffExponent)

		contribution := LightContribution{
			SourceIndex: LighthouseMarkerIndex,
			SourcePos:   beaconPositions[0], // Use first beacon as source pos
			Distance:    minDist,
			Radius:      lighthouseMaxRadius,
			Weight:      weight,
		}

		// Add to existing contributions for this tile
		existing := ls.litTilesMap[tilePos]
		if len(existing) < MaxSourcesPerTile {
			ls.litTilesMap[tilePos] = append(existing, contribution)
		} else {
			// Replace weakest contribution if lighthouse is stronger
			minIdx := 0
			minWeight := existing[0].Weight
			for i, c := range existing {
				if c.Weight < minWeight {
					minWeight = c.Weight
					minIdx = i
				}
			}
			if weight > minWeight {
				ls.litTilesMap[tilePos][minIdx] = contribution
			}
		}
	}

	// Also add beacon positions themselves (they glow too)
	for _, beaconPos := range beaconPositions {
		contribution := LightContribution{
			SourceIndex: LighthouseMarkerIndex,
			SourcePos:   beaconPos,
			Distance:    0, // Beacon is at distance 0 from itself
			Radius:      lighthouseMaxRadius,
			Weight:      1.0, // Full weight at source
		}

		existing := ls.litTilesMap[beaconPos]
		if len(existing) < MaxSourcesPerTile {
			ls.litTilesMap[beaconPos] = append(existing, contribution)
		}
	}
}
