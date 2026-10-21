package game_state

import (
	"strings"
)

// SpiritualityShrineIndex identifies the shrine reached through a moongate at midnight.
const SpiritualityShrineIndex = 6

// Quest state constants for shrine meditation
const (
	QuestStateNotAssigned     = 0 // donequests=false → assign quest
	QuestStateReadyToComplete = 1 // donequests=true, onquests=true → complete quest
	QuestStateOffering        = 2 // donequests=true, onquests=false → gold offerings
)

// ShrineData contains all information about a single shrine
type ShrineData struct {
	Index       int    // 0-7 shrine index
	VirtueName  string // Full name: "Honesty", "Compassion", etc.
	VirtueKey   string // Partial match key: "hone", "comp", etc.
	Mantra      string // "ahm", "mu", "ra", etc.
	WordOfPower string // For restoration: "FALLAX", "VILIS", etc.
	StrBonus    byte   // +1 or 0
	DexBonus    byte   // +1 or 0
	IntBonus    byte   // +1 or 0
	WorldX      byte   // Large map X coordinate
	WorldY      byte   // Large map Y coordinate
	questText   string // Opposed-vice Codex lesson from SHRINE_MSG_0 through SHRINE_MSG_7
}

// allShrines contains data for all 8 virtue shrines
// Index 6 (Spirituality) is accessed via moongate at midnight, not by walking
var allShrines = [8]ShrineData{ //nolint:gochecknoglobals // Immutable indexed shrine reference data.
	{0, "Honesty", "hone", "ahm", "FALLAX", 0, 0, 1, 0xE9, 0x42, "the failing of Dishonesty!"},
	{1, "Compassion", "comp", "mu", "VILIS", 0, 1, 0, 0x80, 0x5C, "of the heart of a cruel soul!"},
	{2, "Valour", "valo", "ra", "INOPIA", 1, 0, 0, 0x24, 0xE5, "the failing of a life without Valour!"},
	{3, "Justice", "just", "beh", "MALUM", 0, 1, 1, 0x49, 0x0B, "the weakness of the Unjust!"},
	{4, "Sacrifice", "sacr", "cah", "AVIDUS", 1, 1, 0, 0xCD, 0x2D, "the failing of unwilling Sacrifice!"},
	{5, "Honor", "hono", "summ", "INFAMA", 1, 0, 1, 0x51, 0xCF, "the darkness of Dishonor!"},
	{SpiritualityShrineIndex, "Spirituality", "spir", "om", "IGNAVUS", 1, 1, 1, 0x00, 0x00, "the neglect of one's Spirit!"},
	{7, "Humility", "humi", "lum", "VERAMOCOR", 0, 0, 0, 0xE7, 0xD8, "the weakness of a life consumed by Pride!"},
}

// shrineAnswerText contains Codex answers for each shrine quest (MSG 8-15 from MISCMSG.DAT)
// These are philosophical wisdom messages about each virtue, NOT map coordinates.
var shrineAnswerText = [8]string{ //nolint:gochecknoglobals // Immutable Codex text indexed by virtue.
	"A dishonest life brings\nunto thee temporary gain, but forsakes\nthe permanent.",
	// REDUX DEVIATION: MISCMSG.DAT SHRINE_MSG_9 (1273) hard-wraps this sentence;
	// let the wider Codex dialog wrap naturally while preserving every word.
	"Only a detested life owes its pleasures to another's pain.",
	"Those who fear to try, know\nnot their\nlimits and thus know not themselves.",
	"Those who inflict injustice upon others, cannot expect fair treatment unto themselves.",
	"None live alone, save\nthey who will\nnot share their fortune with those around them.",
	"It is the guilt, not the guillotine,\nthat\nconstitutes the shame.",
	"To forsake one's inner being is to abandon thy hopes for thyself and thy world.",
	"Pride is a vice, which Pride itself inclines one to find in others, and overlook in oneself.",
}

// CodexRevelationMessages supplies the final revelation after all eight
// shrine quests, directing the party toward Doom. ref:SHRINE-0003
var CodexRevelationMessages = [5]string{ //nolint:gochecknoglobals // Immutable ordered revelation text.
	"A STRANGE WIND CAUSES THE PAGE TO TURN!",
	"BEYOND SHAME'S\nEGRESS IN THE\nCENTRE OF THE\nUNDERWORLD THERE\nIS A PLACE OF\nDARKNESS.",
	"BEYOND THIS\nDARKNESS LIES\nTHE GATE TO THE\nCORE OF THE\nWORLD.",
	"WHEN THOU ART\nREADY THOU MUST\nCALL FORTH\nVERAMOCOR TO\nUNLOCK THE GATE\nAND VENTURE PAST\nETHEREAL WARDS\nAND STEALERS OF\nSOULS.",
	"THAT WHICH THE\nWORLD HATH LOST\nAWAITS THY COMING.",
}

// KarmaBonus returns the karma bonus for completing a shrine quest
// Humility (index 7) gives +6 karma, all others give +3
func (s *ShrineData) KarmaBonus() int {
	if s.Index == 7 {
		return 6
	}

	return 3
}

// QuestText returns the quest assignment text for this shrine
func (s *ShrineData) QuestText() string {
	return s.questText
}

// AnswerText returns the Codex answer for this shrine's quest
func (s *ShrineData) AnswerText() string {
	return shrineAnswerText[s.Index]
}

// GetShrineByPosition returns the shrine index for a world position, or -1 if not found
func GetShrineByPosition(x, y byte) int {
	for i := range allShrines {
		if allShrines[i].WorldX == x && allShrines[i].WorldY == y {
			return i
		}
	}

	return -1
}

// GetShrineByName returns the shrine data for a virtue name (case-insensitive), or nil if not found
// Matches against the full virtue name (e.g., "honesty", "compassion")
func GetShrineByName(name string) *ShrineData {
	lowerName := strings.ToLower(strings.TrimSpace(name))
	for i := range allShrines {
		if strings.ToLower(allShrines[i].VirtueName) == lowerName {
			return &allShrines[i]
		}
	}

	return nil
}

// GetShrineForWordResponse returns the first shrine whose complete canonical
// Word is contained in a normalized response.
// REDUX DEVIATION: WORD-0002: original matching can skip overlapping starts.
// Redux retains normalized substring matching for predictable accepted input.
func GetShrineForWordResponse(response string) int {
	normalized := strings.ToUpper(strings.TrimSpace(response))
	if normalized == "" {
		return -1
	}
	for i := range allShrines {
		if strings.Contains(normalized, allShrines[i].WordOfPower) {
			return i
		}
	}

	return -1
}

// GetShrineData returns the ShrineData for a given index, or nil if invalid
func GetShrineData(index int) *ShrineData {
	if index < 0 || index >= len(allShrines) {
		return nil
	}

	return &allShrines[index]
}

// ValidateVirtue checks if input contains the virtue key (case-insensitive)
// Uses the deliberately modernized substring matching described in ref:WORD-0002.
// See ref:SHRINE-0002.
func ValidateVirtue(input string, virtueKey string) bool {
	return strings.Contains(
		strings.ToLower(strings.TrimSpace(input)),
		strings.ToLower(virtueKey),
	)
}

// ValidateRestorationVirtue requires the complete canonical virtue name in a
// ruined-shrine response without changing ordinary meditation's key matching.
func ValidateRestorationVirtue(input string, virtueName string) bool {
	return strings.Contains(
		strings.ToLower(strings.TrimSpace(input)),
		strings.ToLower(virtueName),
	)
}

// ValidateMantra checks if input contains the mantra (case-insensitive)
// Uses the deliberately modernized substring matching described in ref:WORD-0002.
func ValidateMantra(input string, mantra string) bool {
	return strings.Contains(
		strings.ToLower(strings.TrimSpace(input)),
		strings.ToLower(mantra),
	)
}

// TotalShrines is the number of virtue shrines in the game
const TotalShrines = 8

// MaxStatValue is the maximum value for STR, DEX, INT stats
const MaxStatValue = 30

// MaxKarmaValue is the maximum karma value
const MaxKarmaValue = 99

// ShrineTimeCostMinutes is the game time cost for visiting a shrine
const ShrineTimeCostMinutes = 16

// MantraRepetitions is the number of times the mantra must be entered correctly
const MantraRepetitions = 3
