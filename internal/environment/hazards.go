package environment

import (
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// SystemCallbacks interface to avoid import cycle with game_state
type SystemCallbacks interface {
	AddRowStr(message string)
}

type HazardType int

const (
	NoHazard HazardType = iota
	PoisonSwamp
	LavaBurn
	FireplaceBurn
)

// DamageType represents different types of damage sources for generic damage application
type DamageType int

const (
	TrapFallDamage DamageType = iota
	ClimbingFallDamage
	// Future: CombatDamage, SpellDamage, etc.
)

// Hazard game mechanics constants
const (
	// Poison mechanics use 30 source values; the minimum depends on map context.
	PoisonRollMax = 30

	// Burning damage gives each living member a separate inclusive 1..8 draw; ref:ENV-0001.
	BurnDamageMin = 1
	BurnDamageMax = 8

	// Screen flash parameters for fire damage
	FlashColorRed   = 1.0
	FlashColorGreen = 0.2
	FlashColorBlue  = 0.1
	FlashIntensity  = 0.7
	FlashDurationMs = 300
)

type EnvironmentalHazards struct {
	rng       randomIntn
	callbacks SystemCallbacks
}

type HazardResult struct {
	Type         HazardType
	Affected     []int // Party member indices affected
	MessageShown bool
	DamageDealt  int
}

// DamageConfig specifies parameters for generic damage application
type DamageConfig struct {
	DamageType    DamageType
	FixedDamage   int // If > 0, apply this exact damage to each character
	MinDamage     int // For random damage range (ignored if FixedDamage > 0)
	MaxDamage     int // For random damage range (ignored if FixedDamage > 0)
	Message       string
	ScreenFlash   *ScreenFlashConfig
	TargetIndices []int // If specified, only affect these character indices (nil = all party)
}

// ScreenFlashConfig defines screen flash visual effect parameters
type ScreenFlashConfig struct {
	Red, Green, Blue, Intensity float64
	DurationMs                  int
}

// SkillBasedDamageResult provides detailed results for skill-based damage (e.g., climbing falls)
type SkillBasedDamageResult struct {
	HazardResult          // Embed basic result
	TotalDamage     int   // Total damage dealt across all characters
	MaxSingleDamage int   // Highest single damage dealt to any character
	InjuredMembers  []int // Indices of characters who took damage
	AllSuccessful   bool  // True if no characters took damage
}

// NewEnvironmentalHazards constructs the deterministic environmental hazard system.
func NewEnvironmentalHazards(rng randomIntn, callbacks SystemCallbacks) *EnvironmentalHazards {
	return &EnvironmentalHazards{
		rng:       rng,
		callbacks: callbacks,
	}
}

// SetCallbacks binds external effects without replacing the deterministic system.
func (h *EnvironmentalHazards) SetCallbacks(callbacks SystemCallbacks) {
	h.callbacks = callbacks
}

func (h *EnvironmentalHazards) CheckTileHazards(tile *references.Tile, party *party_state.PartyState, location references.GeneralMapType) HazardResult {
	result := h.ResolveTileHazards(tile, party, location)
	h.emitHazardResult(&result, party)

	return result
}

// damageUpToCurrentHP keeps large configured damage from wrapping through the
// character's uint16 HP field. The reported roll remains unchanged; only the
// amount passed to SubtractHp is capped to the HP that can actually be lost.
func damageUpToCurrentHP(damage int, currentHP uint16) uint16 {
	if damage < 0 {
		panic("environmental damage cannot be negative")
	}
	if damage >= int(currentHP) {
		return currentHP
	}

	return uint16(damage) //nolint:gosec // Nonnegative and strictly below current uint16 HP.
}

// ResolveTileHazards applies authoritative hazard state without presentation callbacks.
func (h *EnvironmentalHazards) ResolveTileHazards(
	tile *references.Tile,
	party *party_state.PartyState,
	location references.GeneralMapType,
) HazardResult {
	hazardType := h.detectHazardType(tile, location)

	switch hazardType {
	case PoisonSwamp:
		return h.checkSwampPoison(party, location)
	case LavaBurn, FireplaceBurn:
		return h.checkBurningHazard(party, hazardType)
	default:
		return HazardResult{Type: NoHazard}
	}
}

func (h *EnvironmentalHazards) emitHazardResult(result *HazardResult, party *party_state.PartyState) {
	if len(result.Affected) == 0 {
		return
	}
	names := make([]string, 0, len(result.Affected))
	for _, memberIndex := range result.Affected {
		name := extractCharacterName(party.GetCharacter(memberIndex).Name[:])
		if name != "" {
			names = append(names, name)
		}
	}

	switch result.Type {
	case PoisonSwamp:
		message := "Poisoned!"
		if len(names) > 0 {
			message = "Poisoned: " + joinNames(names)
		}
		h.callbacks.AddRowStr(message)
		result.MessageShown = true
	case LavaBurn, FireplaceBurn:
		message := "Burning!"
		if len(names) > 0 {
			message = "Burning: " + joinNames(names)
		}
		h.callbacks.AddRowStr(message)
		result.MessageShown = true
	case NoHazard:
	}
}

// DetectTileHazard classifies a tile without consuming RNG or mutating party state.
func (h *EnvironmentalHazards) DetectTileHazard(tile *references.Tile, location references.GeneralMapType) HazardType {
	return h.detectHazardType(tile, location)
}

func (h *EnvironmentalHazards) detectHazardType(tile *references.Tile, location references.GeneralMapType) HazardType {
	if tile == nil {
		return NoHazard
	}

	switch tile.Index {
	case indexes.Swamp:
		return PoisonSwamp
	case indexes.Lava:
		return LavaBurn
	case indexes.Fireplace:
		if location == references.SmallMapType {
			return FireplaceBurn // Only dangerous indoors
		}
	}

	return NoHazard
}

func (h *EnvironmentalHazards) checkSwampPoison(
	party *party_state.PartyState,
	location references.GeneralMapType,
) HazardResult {
	result := HazardResult{Type: PoisonSwamp}
	if location != references.SmallMapType && location != references.LargeMapType {
		return result
	}

	party.ForEachCharacter(func(index int, member *party_state.PlayerCharacter) {
		if member.IsDead() || member.IsPoisoned() || member.PartyMap != party_state.PartyMapActive {
			return // Skip dead, already poisoned, or not active party members
		}

		// Town poison checks draw 0..29 inclusive; outdoor checks draw 1..30 inclusive; ref:ENV-0002.
		poisonRoll := h.rng.Intn(PoisonRollMax)
		if location == references.LargeMapType {
			poisonRoll++
		}
		if poisonRoll > int(member.Dexterity) {
			member.SetStatus(party_state.Poisoned) // Modifies original - member is already a pointer
			result.Affected = append(result.Affected, index)
		}
	})

	return result
}

func (h *EnvironmentalHazards) checkBurningHazard(party *party_state.PartyState, hazardType HazardType) HazardResult {
	result := HazardResult{Type: hazardType}

	// Each living member takes an independent inclusive 1..8 damage draw; ref:ENV-0001.
	totalDamage := 0
	party.ForEachCharacter(func(index int, member *party_state.PlayerCharacter) {
		if !member.IsDead() && member.PartyMap == party_state.PartyMapActive {
			damage := h.rng.Intn(BurnDamageMax) + BurnDamageMin // 1..8 range
			member.SubtractHp(damageUpToCurrentHP(damage, member.GetCurrentHp()))
			totalDamage += damage
			result.Affected = append(result.Affected, index)

			if member.GetCurrentHp() == 0 {
				member.MarkAsDead()
			}
		}
	})

	result.DamageDealt = totalDamage

	return result
}

// extractCharacterName safely extracts a character name from byte array, removing null terminators
func extractCharacterName(nameBytes []byte) string {
	name := string(nameBytes)
	// Remove null terminator if present
	for i, b := range nameBytes {
		if b == 0 {
			name = name[:i]

			break
		}
	}

	return name
}

// joinNames creates a grammatically correct list of names
func joinNames(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	case 2:
		return names[0] + " and " + names[1]
	default:
		// For 3+ names: "Alice, Bob, and Charlie"
		result := ""
		for i, name := range names {
			if i == 0 {
				result = name
			} else if i == len(names)-1 {
				result += ", and " + name
			} else {
				result += ", " + name
			}
		}

		return result
	}
}

// ApplyDamageToParty applies damage to party members according to DamageConfig
// This provides a consistent damage application pattern for all damage sources
func (h *EnvironmentalHazards) ApplyDamageToParty(party *party_state.PartyState, config DamageConfig) HazardResult {
	result := HazardResult{Type: NoHazard} // Generic damage uses NoHazard type
	totalDamage := 0

	// Determine which characters to affect
	targetIndices := config.TargetIndices
	if targetIndices == nil {
		// Default: affect all party members
		targetIndices = make([]int, party.GetCharacterCount())
		for i := 0; i < party.GetCharacterCount(); i++ {
			targetIndices[i] = i
		}
	}

	for _, index := range targetIndices {
		if index < 0 || index >= party.GetCharacterCount() {
			continue // Skip invalid indices
		}

		member := party.GetCharacter(index)
		if member.IsDead() || member.PartyMap != party_state.PartyMapActive {
			continue // Skip dead or inactive characters
		}

		// Calculate damage based on config
		var damage int
		if config.FixedDamage > 0 {
			damage = config.FixedDamage
		} else {
			// Random damage in range [MinDamage, MaxDamage]
			damage = h.rng.Intn(config.MaxDamage-config.MinDamage+1) + config.MinDamage
		}

		// Apply damage with proper clamping
		member.SubtractHp(damageUpToCurrentHP(damage, member.GetCurrentHp()))
		totalDamage += damage
		result.Affected = append(result.Affected, index)

		// Update status if character dies
		if member.GetCurrentHp() == 0 {
			member.MarkAsDead()
		}
	}

	result.DamageDealt = totalDamage

	// Apply audio/visual effects if damage was dealt and message specified
	if len(result.Affected) > 0 && config.Message != "" {
		// Build message with character names (reuse existing pattern)
		message := config.Message + ": "
		names := make([]string, 0, len(result.Affected))
		for _, memberIndex := range result.Affected {
			char := party.GetCharacter(memberIndex)
			name := extractCharacterName(char.Name[:])
			if name != "" {
				names = append(names, name)
			}
		}

		if len(names) > 0 {
			message += joinNames(names)
			h.callbacks.AddRowStr(message)
			result.MessageShown = true
		} else {
			// Fallback if no names found
			h.callbacks.AddRowStr(config.Message + "!")
			result.MessageShown = true
		}
	}

	return result
}

// ApplySkillBasedDamage applies damage to specific party members with enhanced tracking
// This method is designed for skill-based actions like mountain climbing where individual
// performance determines damage outcome and sophisticated audio-visual effects are needed
func (h *EnvironmentalHazards) ApplySkillBasedDamage(party *party_state.PartyState, targetIndices []int, config DamageConfig) SkillBasedDamageResult {
	result := SkillBasedDamageResult{
		HazardResult:  HazardResult{Type: NoHazard}, // Generic damage uses NoHazard type
		AllSuccessful: true,
	}

	totalDamage := 0
	maxSingleDamage := 0
	injuredMembers := make([]int, 0)

	for _, index := range targetIndices {
		if index < 0 || index >= party.GetCharacterCount() {
			continue // Skip invalid indices
		}

		member := party.GetCharacter(index)
		if member.IsDead() || member.PartyMap != party_state.PartyMapActive {
			continue // Skip dead or inactive characters
		}

		// Calculate damage based on config
		var damage int
		if config.FixedDamage > 0 {
			damage = config.FixedDamage
		} else {
			// Random damage in range [MinDamage, MaxDamage]
			damage = h.rng.Intn(config.MaxDamage-config.MinDamage+1) + config.MinDamage
		}

		// Apply damage with proper clamping
		member.SubtractHp(damageUpToCurrentHP(damage, member.GetCurrentHp()))

		// Track damage statistics
		totalDamage += damage
		if damage > maxSingleDamage {
			maxSingleDamage = damage
		}
		injuredMembers = append(injuredMembers, index)
		result.HazardResult.Affected = append(result.HazardResult.Affected, index)
		result.AllSuccessful = false

		// Update status if character dies
		if member.GetCurrentHp() == 0 {
			member.MarkAsDead()
		}
	}

	// Set result statistics
	result.TotalDamage = totalDamage
	result.MaxSingleDamage = maxSingleDamage
	result.InjuredMembers = injuredMembers
	result.HazardResult.DamageDealt = totalDamage

	// Note: Audio and visual effects are NOT applied here
	// The calling code (ActionKlimbLargeMap) handles sophisticated graduated effects
	// based on the returned damage statistics

	return result
}
