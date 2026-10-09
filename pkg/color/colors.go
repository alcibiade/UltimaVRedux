package color

import "image/color"

// Alpha Constants for consistent transparency values
const (
	AlphaOpaque          = 255
	AlphaSemiTransparent = 128
	AlphaVeryTransparent = 60
	AlphaSubtle          = 40
	AlphaHeavy           = 200
	AlphaLight           = 100
	AlphaMedium          = 180
)

// Color conversion constants
const (
	// maxUint16Float is the maximum value of a uint16 as float64.
	// Used for converting color.RGBA() output (0-65535 range) to 0.0-1.0 range.
	maxUint16Float = 65535.0
)

// ============================================================================
// EGA COLOR TYPE
// ============================================================================

// EGAColor represents an EGA palette color with pixel matching capabilities.
// Implements color.Color interface for seamless compatibility with existing code.
type EGAColor struct {
	R, G, B, A uint8
}

// RGBA implements color.Color interface.
// Returns 16-bit premultiplied values as required by the interface.
func (c EGAColor) RGBA() (r, g, b, a uint32) {
	return uint32(c.R) * 0x101, uint32(c.G) * 0x101, uint32(c.B) * 0x101, uint32(c.A) * 0x101
}

// Matches returns true if the given RGB values exactly match this EGA color.
func (c EGAColor) Matches(r, g, b uint8) bool {
	return c.R == r && c.G == g && c.B == b
}

// MatchesWithTolerance returns true if RGB values match within the given tolerance.
func (c EGAColor) MatchesWithTolerance(r, g, b uint8, tolerance int) bool {
	return intAbs(int(c.R)-int(r)) <= tolerance &&
		intAbs(int(c.G)-int(g)) <= tolerance &&
		intAbs(int(c.B)-int(b)) <= tolerance
}

// MatchesRGBA returns true if the given color.RGBA exactly matches this EGA color.
// Ignores alpha channel - only compares RGB components.
func (c EGAColor) MatchesRGBA(other color.RGBA) bool {
	return c.R == other.R && c.G == other.G && c.B == other.B
}

// ToRGBA returns this color as a standard color.RGBA for explicit conversion.
func (c EGAColor) ToRGBA() color.RGBA {
	return color.RGBA{R: c.R, G: c.G, B: c.B, A: c.A}
}

// intAbs returns the absolute value of an integer.
// Used for color tolerance comparisons.
func intAbs(x int) int {
	if x < 0 {
		return -x
	}

	return x
}

// Standard EGA 16-Color Palette
// These are the exact colors available in the original EGA standard
//
//goland:noinspection GoUnusedGlobalVariable
var (
	// Primary EGA Colors (0-7)
	EGABlack     = EGAColor{0, 0, 0, AlphaOpaque}       // 0: Black
	EGABlue      = EGAColor{0, 0, 170, AlphaOpaque}     // 1: Blue
	EGAGreen     = EGAColor{0, 170, 0, AlphaOpaque}     // 2: Green
	EGACyan      = EGAColor{0, 170, 170, AlphaOpaque}   // 3: Cyan
	EGARed       = EGAColor{170, 0, 0, AlphaOpaque}     // 4: Red
	EGAMagenta   = EGAColor{170, 0, 170, AlphaOpaque}   // 5: Magenta
	EGABrown     = EGAColor{170, 85, 0, AlphaOpaque}    // 6: Brown/Dark Yellow
	EGALightGray = EGAColor{170, 170, 170, AlphaOpaque} // 7: Light Gray

	// Bright EGA Colors (8-15)
	EGADarkGray     = EGAColor{85, 85, 85, AlphaOpaque}    // 8: Dark Gray
	EGALightBlue    = EGAColor{85, 85, 255, AlphaOpaque}   // 9: Light Blue
	EGALightGreen   = EGAColor{85, 255, 85, AlphaOpaque}   // 10: Light Green
	EGALightCyan    = EGAColor{85, 255, 255, AlphaOpaque}  // 11: Light Cyan
	EGALightRed     = EGAColor{255, 85, 85, AlphaOpaque}   // 12: Light Red
	EGALightMagenta = EGAColor{255, 85, 255, AlphaOpaque}  // 13: Light Magenta
	EGAYellow       = EGAColor{255, 255, 85, AlphaOpaque}  // 14: Yellow
	EGAWhite        = EGAColor{255, 255, 255, AlphaOpaque} // 15: White

	// Special
	EGATransparent = EGAColor{0, 0, 0, 0} // Fully transparent
)

// Legacy aliases for backward compatibility
var (
	Gray                   = EGADarkGray
	Black                  = EGABlack
	White                  = EGAWhite
	Green                  = color.RGBA{R: 0, G: 255, B: 0, A: 255}   // Non-EGA green kept for compatibility
	Red                    = color.RGBA{R: 255, G: 0, B: 0, A: 255}   // Non-EGA red kept for compatibility
	Yellow                 = color.RGBA{R: 0, G: 255, B: 255, A: 255} // This appears to be cyan, kept for compatibility
	UltimaBlue             = color.RGBA{R: 22, G: 7, B: 178, A: 255}  // Custom Ultima blue
	BlackSemi              = color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x77}
	LighterBlackSemi       = color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xAA}
	LighterBlueSemi        = color.RGBA{R: 0x00, G: 0x00, B: 0xFF, A: 0x77}
	DisabledTextBackground = color.RGBA{R: 16, G: 16, B: 16, A: 0x77}
	TextInputColor         = EGALightGray
	TextPlaceHolderColor   = EGALightGray
	DialogInputTextColor   = EGALightGreen // Light green for InputBox dialog text
)

// ExtractRGBA safely extracts RGBA components from any color.Color
// Returns R, G, B, A values as uint8 (0-255 range)
func ExtractRGBA(c color.Color) (uint8, uint8, uint8, uint8) {
	r, g, b, a := c.RGBA()

	return uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)
}

// ToRGBFloats extracts RGB components from a color.Color as float32 values (0.0-1.0).
// This is useful for bitmap font rendering with Ebitengine's ColorScale.
// Alpha channel is ignored; use ExtractRGBA if you need all four components.
func ToRGBFloats(c color.Color) (r, g, b float32) {
	r16, g16, b16, _ := c.RGBA()

	return float32(r16) / maxUint16Float, float32(g16) / maxUint16Float, float32(b16) / maxUint16Float
}

// clampUint32 clamps a uint32 value to a maximum
func clampUint32(value uint32, max uint32) uint32 {
	if value > max {
		return max
	}

	return value
}

// ColorBuilder provides a fluent interface for color transformations
// Allows chaining of operations like alpha adjustment, intensity changes, etc.
type ColorBuilder struct {
	r, g, b, a uint8
}

// From creates a ColorBuilder from any color.Color
func From(c color.Color) *ColorBuilder {
	r, g, b, a := ExtractRGBA(c)

	return &ColorBuilder{r: r, g: g, b: b, a: a}
}

// Alpha sets the alpha (transparency) value (0-255)
func (cb *ColorBuilder) Alpha(alpha uint8) *ColorBuilder {
	return &ColorBuilder{r: cb.r, g: cb.g, b: cb.b, a: alpha}
}

// SemiTransparent sets alpha to 128 (semi-transparent)
func (cb *ColorBuilder) SemiTransparent() *ColorBuilder {
	return cb.Alpha(AlphaSemiTransparent)
}

// VeryTransparent sets alpha to 60 (very transparent)
func (cb *ColorBuilder) VeryTransparent() *ColorBuilder {
	return cb.Alpha(AlphaVeryTransparent)
}

// Subtle sets alpha to 40 (subtle transparency)
func (cb *ColorBuilder) Subtle() *ColorBuilder {
	return cb.Alpha(AlphaSubtle)
}

// Heavy sets alpha to 200 (heavy/strong transparency)
func (cb *ColorBuilder) Heavy() *ColorBuilder {
	return cb.Alpha(AlphaHeavy)
}

// Light sets alpha to 100 (light transparency)
func (cb *ColorBuilder) Light() *ColorBuilder {
	return cb.Alpha(AlphaLight)
}

// Medium sets alpha to 180 (medium transparency)
func (cb *ColorBuilder) Medium() *ColorBuilder {
	return cb.Alpha(AlphaMedium)
}

// Opaque sets alpha to 255 (fully opaque)
func (cb *ColorBuilder) Opaque() *ColorBuilder {
	return cb.Alpha(AlphaOpaque)
}

// Intensity adjusts color brightness by multiplying RGB components
// factor: 0.0 = black, 1.0 = original color, >1.0 = brighter (clamped to 255)
func (cb *ColorBuilder) Intensity(factor float32) *ColorBuilder {
	newR := uint8(clampUint32(uint32(float32(cb.r)*factor), 255))
	newG := uint8(clampUint32(uint32(float32(cb.g)*factor), 255))
	newB := uint8(clampUint32(uint32(float32(cb.b)*factor), 255))

	return &ColorBuilder{r: newR, g: newG, b: newB, a: cb.a}
}

// Brighten increases brightness (factor > 1.0 makes brighter)
func (cb *ColorBuilder) Brighten(factor float32) *ColorBuilder {
	if factor < 1.0 {
		factor = 1.0 // Ensure brightening
	}

	return cb.Intensity(factor)
}

// Darken decreases brightness (factor < 1.0 makes darker)
func (cb *ColorBuilder) Darken(factor float32) *ColorBuilder {
	if factor > 1.0 {
		factor = 1.0 // Ensure darkening
	}

	return cb.Intensity(factor)
}

// RGBA returns the final color.RGBA result
func (cb *ColorBuilder) RGBA() color.RGBA {
	return color.RGBA{R: cb.r, G: cb.g, B: cb.b, A: cb.a}
}

// Semantic Color Categories - ColorBuilder Variables
// UI Colors - Common interface elements (ColorBuilder instances for chaining)
var (
	// General UI
	UIBackground     = From(EGABlack)
	UIBackgroundSemi = From(EGABlack).SemiTransparent()
	UIBorder         = From(EGALightGray)
	UIBorderBright   = From(EGAWhite)
	UITextPrimary    = From(EGAWhite)
	UITextSecondary  = From(EGALightGray)
	UITextDisabled   = From(EGADarkGray)
	// Low stock remains readable on a dark supplies panel.
	SupplyLow = EGAYellow //nolint:gochecknoglobals // Semantic palette values are immutable after package initialization.
	// Empty stock, paired with explicit OUT text.
	SupplyEmpty = EGALightRed //nolint:gochecknoglobals // Immutable semantic palette value.

	// Button States
	ButtonDefault      = From(EGADarkGray)
	ButtonActive       = From(EGALightBlue)
	ButtonDisabled     = From(EGADarkGray).Light()
	ButtonText         = From(EGAWhite)
	ButtonTextActive   = From(EGAWhite)
	ButtonTextDisabled = From(EGADarkGray).Opaque()

	// Focus/Selection
	FocusIndicator     = From(EGAYellow)
	FocusIndicatorSemi = From(EGAYellow).VeryTransparent()
	SelectionHighlight = From(EGALightBlue).Light()
	SelectionBorder    = From(EGALightBlue)

	// Tab Interface
	TabActive     = From(EGALightBlue).Heavy()
	TabInactive   = From(EGADarkGray).Heavy()
	TabText       = From(EGAWhite)
	TabBackground = From(EGADarkGray)

	// Dialog Colors
	DialogBackground    = From(EGABlack)        // Black dialog background
	DialogTabDefault    = From(EGABlue).Heavy() // Dark EGA blue tab with transparency
	DialogTabActive     = From(EGALightBlue)    // EGA light blue for active tab
	DialogTextPrimary   = From(EGAWhite)        // White text for primary labels
	DialogTextSecondary = From(EGALightGray)    // Light gray for secondary text
)

// Visual Effect Colors
var (
	// Explosion/Combat Effects
	MissileYellow = EGAYellow  // Yellow projectile base
	MagicPurple   = EGAMagenta // Magic effects

	// Map Fade/Transition
	MapFadeBlack = EGABlack // Base color for fades

	// Moongate Colors
	MoongateCenter = color.RGBA{150, 100, 255, AlphaOpaque} // Custom purple center
	MoongateRing   = color.RGBA{120, 80, 220, AlphaOpaque}  // Custom purple ring
)

// Overlay Effect Colors - ColorBuilder Variables for pulsing glows and auras
//
//nolint:gochecknoglobals // Semantic palette values are immutable after package initialization.
var (
	// MoongateGlow is the pulsing blue glow for active moongates.
	// Uses EGA Light Blue with semi-transparency for mystical effect.
	MoongateGlow = From(EGALightBlue).SemiTransparent() //nolint:gochecknoglobals

	// Creature Glow Colors - subtle magical auras for NPCs (additive blending)
	// Using non-light EGA base colors with low intensity to avoid washed-out look

	// WispGlow is an ethereal cyan glow for energy beings (fast flicker)
	WispGlow = From(EGACyan).Intensity(0.4).Alpha(50) //nolint:gochecknoglobals,mnd

	// DaemonGlow is a fiery red glow for fire creatures (fire flicker)
	DaemonGlow = From(EGARed).Intensity(0.4).Alpha(50) //nolint:gochecknoglobals,mnd

	// GhostGlow is a pale spectral glow for undead spirits (slow waver)
	GhostGlow = From(EGALightGray).Intensity(0.3).Alpha(40) //nolint:gochecknoglobals,mnd

	// SeahorseGlow is a subtle aqua glow for mystical sea creatures
	SeahorseGlow = From(EGACyan).Intensity(0.4).Alpha(50) //nolint:gochecknoglobals,mnd

	// Quest Item Glow Colors - subtle but noticeable for collectible objects
	// ShardGlow is a sinister purple glow for corrupted Shards of Evil
	ShardGlow = From(EGAMagenta).Intensity(0.4).Alpha(50) //nolint:gochecknoglobals,mnd

	// ArtifactGlow is a noble golden glow for Lord British's Artifacts
	ArtifactGlow = From(EGABrown).Intensity(0.5).Alpha(60) //nolint:gochecknoglobals,mnd

	// Combat Overlay Colors
	// These semantic colors are intentionally centralized so combat readability
	// can be tuned without changing rendering code.
	// Empty targeting deliberately uses interpolated VGA-like yellows so its
	// sine pulse changes luminance without shifting into white or brown.
	CombatAimEmptyBright    = EGAColor{255, 255, 0, AlphaOpaque}
	CombatAimEmptyDim       = EGAColor{170, 170, 0, AlphaOpaque}
	CombatAimFriendlyBright = EGALightCyan
	CombatAimFriendlyDim    = EGACyan
	CombatAimEnemyBright    = EGALightRed
	CombatAimEnemyDim       = EGARed

	CombatActivePCOutline          = EGALightGreen
	CombatInactivePCOutline        = EGABlack
	CombatActiveMonsterOutline     = EGAYellow
	CombatThreatenedPCOutline      = EGALightRed
	CombatPossessed                = EGALightBlue
	CombatUnderground              = EGABrown
	CombatUndergroundText          = EGAYellow
	CombatMaterialization          = EGALightMagenta
	CombatMaterializationHighlight = EGAWhite
	// Live combat side belongs to the exterior HP-frame outline and explicit
	// Ally/Enemy text, never to the HP fill or an upper-left actor symbol.
	CombatAllegianceAlly       = EGALightCyan
	CombatAllegianceEnemy      = EGALightRed
	CombatTeleportArrival      = EGALightCyan
	CombatAttackEligibleFace   = EGALightBlue
	CombatAttackEligibleBorder = EGABlack
	CombatDamageText           = EGALightRed
	CombatDamageShadow         = EGABlack
	CombatMissText             = EGAWhite
	CombatGrazeText            = EGAYellow
	CombatReportLoot           = EGALightGreen
	CombatReportWarning        = EGALightRed
)

// Debug Overlay Colors - ColorBuilder Variables
var (
	// Debug Overlays
	DebugLightOverlay     = From(EGAYellow).Subtle()         // Yellow for light areas
	DebugSearchOverlay    = From(EGAGreen).VeryTransparent() // Green for searchable
	DebugMoonstoneOverlay = From(EGAMagenta).Light()         // Purple for moonstones
	DebugPathOverlay      = From(EGABlue).Subtle()           // Blue for pathfinding
	DebugCollisionOverlay = From(EGARed).Subtle()            // Red for collision

	// Search Overlay States
	DebugSearchUnsearched       = From(EGAGreen).VeryTransparent() // Green for unsearched locations
	DebugSearchAlreadyFound     = From(EGARed).VeryTransparent()   // Red for already searched locations
	DebugSearchBorderUnsearched = From(EGAGreen).Light()           // Green border for unsearched
	DebugSearchBorderFound      = From(EGARed).Light()             // Red border for searched

	// Moonstone Debug States
	MoonstoneActiveBright = From(EGALightMagenta).Alpha(120) // Bright purple for active moonstone
	MoonstoneFoundGreen   = From(EGALightGreen).Alpha(80)    // Green for found moonstone
	MoonstoneActiveBorder = From(EGAWhite).Alpha(180)        // White border for active moonstone

	// Debug UI
	DebugBackground = From(EGABlack).Opaque()
	DebugText       = From(EGAWhite)
	DebugBorder     = From(EGAWhite)
)

// Widget-Specific Colors - ColorBuilder Variables
var (
	// Button
	ButtonTextDisabledGray = From(EGADarkGray).Heavy() // Gray text for disabled buttons

	// Slider
	SliderTrack      = From(EGABlue) // EGA blue track
	SliderHandle     = From(EGAWhite)
	SliderLabel      = From(EGAWhite)
	SliderBackground = From(EGADarkGray)
	SliderFocus      = From(EGAWhite) // Bright white for focus

	// List Widget
	ListHighlight   = From(EGABlue).Light()
	ListScrollTrack = From(EGADarkGray)
	ListScrollThumb = From(EGALightGray)

	// Loading Indicator
	LoadingBackground = From(EGADarkGray).Heavy()
	LoadingSpinner    = From(EGALightBlue)
	LoadingText       = From(EGAWhite)

	// Tabbed Interface
	TabActiveBlue     = From(EGALightBlue).Heavy()        // Light blue for active tabs
	TabInactiveGray   = From(EGADarkGray)                 // Gray for inactive tabs
	TabDisabledDim    = From(EGADarkGray).Intensity(0.4)  // Dim gray for empty/disabled tabs
	TabTextWhite      = From(EGAWhite)                    // White tab text
	TabTextDim        = From(EGALightGray).Intensity(0.6) // Dim text for disabled tabs
	TabBackgroundDark = From(EGADarkGray)                 // Dark gray tab background
	TabBorderDefault  = From(EGALightGray)                // Default tab border
	TabBorderActive   = From(EGAWhite)                    // Active tab border

	// Focus Indicator
	FocusOutline            = From(EGAWhite)             // White focus outline
	FocusIndicatorBlue      = From(EGALightBlue)         // Blue focus indicator
	FocusIndicatorSemiBlue  = From(EGALightBlue).Light() // Semi-transparent blue
	FocusHighContrastYellow = From(EGAYellow)            // High contrast yellow
	FocusBlackOutline       = From(EGABlack)             // Black outline for high contrast

	// NewOrder Dialog specific
	NewOrderAvatarBackground = From(EGABlue).Intensity(0.25).Alpha(128) // Very dark blue for Avatar
	NewOrderLockedBackground = From(EGABlue).Intensity(0.4).Alpha(200)  // Very dark blue for locked state (chosen)
	NewOrderNormalBackground = From(EGABlack).Alpha(64)                 // Semi-transparent for normal
)
