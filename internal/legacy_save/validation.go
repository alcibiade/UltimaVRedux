package legacy_save

import (
	"errors"
	"fmt"
)

const (
	unavailableOffset              = -1
	gamValidationPath              = "gam"
	oolValidationPath              = "ool"
	invalidLengthCode              = "invalid_length"
	maxActiveParty                 = 6
	maxCalendarMonth               = 13
	maxCalendarDay                 = 28
	maxCalendarHour                = 23
	maxCalendarMinute              = 59
	maxMapID                 MapID = 40
	maxSmallMapID            MapID = 32
	maxSmallCoordinate             = 31
	maxDungeonCoordinate           = 7
	maxFloor                       = 7
	originalCombatTimeModulo       = 10
	maxDungeonLastDirection        = 5
)

// ErrInvalidBundle is the sentinel wrapped by ValidationError.
var ErrInvalidBundle = errors.New("invalid original save bundle")

// ValidationIssue is one structural or semantic failure. Offset is an absolute
// byte offset in the file named by Path's gam or ool prefix, or -1 when no
// source byte exists (for example, a missing file).
type ValidationIssue struct {
	Path    string `json:"path"`
	Code    string `json:"code"`
	Offset  int    `json:"offset"`
	Message string `json:"message"`
}

// ValidationError preserves all issues in deterministic source order.
type ValidationError struct {
	Issues []ValidationIssue
}

func (validationError *ValidationError) Error() string {
	if len(validationError.Issues) == 0 {
		return ErrInvalidBundle.Error()
	}
	first := validationError.Issues[0]

	return fmt.Sprintf(
		"%v: %d issue(s); first: %s [%s] offset %#x: %s",
		ErrInvalidBundle,
		len(validationError.Issues),
		first.Path,
		first.Code,
		first.Offset,
		first.Message,
	)
}

func (validationError *ValidationError) Unwrap() error {
	return ErrInvalidBundle
}

func newValidationError(issues []ValidationIssue) *ValidationError {
	return &ValidationError{Issues: append([]ValidationIssue(nil), issues...)}
}

func validateLengths(gam, ool []byte) []ValidationIssue {
	issues := make([]ValidationIssue, 0)
	if len(gam) != SavedGameSize {
		issues = append(issues, ValidationIssue{
			Path:    gamValidationPath,
			Code:    invalidLengthCode,
			Offset:  unavailableOffset,
			Message: fmt.Sprintf("SAVED.GAM must be exactly %d bytes; got %d", SavedGameSize, len(gam)),
		})
	}
	if len(ool) != SavedObjectsSize {
		issues = append(issues, ValidationIssue{
			Path:    oolValidationPath,
			Code:    invalidLengthCode,
			Offset:  unavailableOffset,
			Message: fmt.Sprintf("SAVED.OOL must be exactly %d bytes; got %d", SavedObjectsSize, len(ool)),
		})
	}

	return issues
}

func validateBundle(bundle Bundle) []ValidationIssue {
	issues := make([]ValidationIssue, 0)
	validateCharacters(bundle.Game.Characters, &issues)
	if bundle.Game.PartyCount == 0 {
		addValidationIssue(
			&issues,
			"gam.partyCount",
			"out_of_range",
			partyCountOffset,
			"party count must include at least the Avatar",
		)
	}
	if bundle.Game.PartyCount > maxActiveParty {
		addValidationIssue(
			&issues,
			"gam.partyCount",
			"out_of_range",
			partyCountOffset,
			"party count cannot exceed the original six-character active-party limit",
		)
	}
	for index := range min(int(bundle.Game.PartyCount), maxActiveParty) {
		if bundle.Game.Characters[index].Occupied {
			continue
		}
		addValidationIssue(
			&issues,
			fmt.Sprintf("gam.characters[%d].name", index),
			"missing_active_character",
			charactersOffset+index*characterRecordSize,
			"the original party count requires a named character in this roster-prefix slot",
		)
	}
	validateCalendar(bundle.Game.Calendar, &issues)
	if bundle.Game.CombatActionTimeCount >= originalCombatTimeModulo {
		addValidationIssue(
			&issues,
			"gam.combatActionTimeCount",
			"out_of_range",
			combatTimeCountOffset,
			"combat action time counter must be from 0 through 9",
		)
	}
	validatePosition(bundle.Game.Position, &issues)
	validateDungeon(bundle.Game.Dungeon, &issues)

	return issues
}

func validateCharacters(characters [characterCount]Character, issues *[]ValidationIssue) {
	for index, character := range characters {
		if !character.Occupied {
			continue
		}
		base := charactersOffset + index*characterRecordSize
		path := fmt.Sprintf("gam.characters[%d]", index)
		validateCharacter(character, path, base, issues)
	}
}

func validateCharacter(character Character, path string, base int, issues *[]ValidationIssue) {
	validateCharacterName(character.NameBytes, path, base, issues)
	validateCharacterValues(character, path, base, issues)
}

func validateCharacterName(name [9]byte, path string, base int, issues *[]ValidationIssue) {
	terminator := firstZero(name[:])
	if terminator < 0 {
		terminator = len(name)
	}
	for nameIndex := range terminator {
		if name[nameIndex] < 0x20 || name[nameIndex] > 0x7e {
			addValidationIssue(
				issues,
				path+".name",
				"non_printable",
				base+nameIndex,
				"occupied character name must contain printable ASCII before its terminator",
			)

			break
		}
	}
}

func validateCharacterValues(character Character, path string, base int, issues *[]ValidationIssue) {
	if !knownGender(character.Gender) {
		addValidationIssue(
			issues,
			path+".gender",
			"unknown_value",
			base+genderFieldOffset,
			fmt.Sprintf("unknown occupied-character gender %#x", byte(character.Gender)),
		)
	}
	if !knownClass(character.Class) {
		addValidationIssue(
			issues,
			path+".class",
			"unknown_value",
			base+classFieldOffset,
			fmt.Sprintf("unknown occupied-character class %#x", byte(character.Class)),
		)
	}
	if !knownStatus(character.Status) {
		addValidationIssue(
			issues,
			path+".status",
			"unknown_value",
			base+statusFieldOffset,
			fmt.Sprintf("unknown occupied-character status %#x", byte(character.Status)),
		)
	}
	if character.CurrentHP > character.MaximumHP {
		addValidationIssue(
			issues,
			path+".currentHP",
			"inconsistent",
			base+currentHPFieldOffset,
			"current HP cannot exceed maximum HP",
		)
	}
	if !knownPartyMap(character.PartyMap) {
		addValidationIssue(
			issues,
			path+".partyMap",
			"unknown_value",
			base+partyMapFieldOffset,
			fmt.Sprintf("unknown pmap state %#x", byte(character.PartyMap)),
		)
	}
}

func validateCalendar(calendar Calendar, issues *[]ValidationIssue) {
	if calendar.Year == 0 {
		addValidationIssue(issues, "gam.calendar.year", "out_of_range", yearOffset, "year must be at least 1")
	}
	if calendar.Month < 1 || calendar.Month > maxCalendarMonth {
		addValidationIssue(
			issues,
			"gam.calendar.month",
			"out_of_range",
			monthOffset,
			"month must be between 1 and 13",
		)
	}
	if calendar.Day < 1 || calendar.Day > maxCalendarDay {
		addValidationIssue(
			issues,
			"gam.calendar.day",
			"out_of_range",
			dayOffset,
			"day must be between 1 and 28",
		)
	}
	if calendar.Hour > maxCalendarHour {
		addValidationIssue(
			issues,
			"gam.calendar.hour",
			"out_of_range",
			hourOffset,
			"hour must be between 0 and 23",
		)
	}
	if calendar.Minute > maxCalendarMinute {
		addValidationIssue(
			issues,
			"gam.calendar.minute",
			"out_of_range",
			minuteOffset,
			"minute must be between 0 and 59",
		)
	}
}

func validatePosition(position WorldPosition, issues *[]ValidationIssue) {
	if position.Location > maxMapID {
		addValidationIssue(
			issues,
			"gam.position.location",
			"out_of_range",
			locationOffset,
			"location must be an original large, town, or dungeon map ID from 0 through 40",
		)

		return
	}
	switch {
	case position.Location == 0:
		validateLargeMapPosition(position, issues)
	case position.Location <= maxSmallMapID:
		validateSmallMapPosition(position, issues)
	default:
		validateDungeonPosition(position, issues)
	}
}

func validateLargeMapPosition(position WorldPosition, issues *[]ValidationIssue) {
	if position.Floor != 0 && position.Floor != 0xff {
		addValidationIssue(
			issues,
			"gam.position.floor",
			"out_of_range",
			floorOffset,
			"large-map floor must identify Britannia or the Underworld",
		)
	}
}

func validateSmallMapPosition(position WorldPosition, issues *[]ValidationIssue) {
	if position.Floor > maxFloor && position.Floor != 0xff {
		addValidationIssue(
			issues,
			"gam.position.floor",
			"out_of_range",
			floorOffset,
			"small-map floor is outside the broad original level range",
		)
	}
	if position.X > maxSmallCoordinate {
		addValidationIssue(
			issues,
			"gam.position.x",
			"out_of_range",
			partyXOffset,
			"small-map X coordinate must be between 0 and 31",
		)
	}
	if position.Y > maxSmallCoordinate {
		addValidationIssue(
			issues,
			"gam.position.y",
			"out_of_range",
			partyYOffset,
			"small-map Y coordinate must be between 0 and 31",
		)
	}
}

func validateDungeonPosition(position WorldPosition, issues *[]ValidationIssue) {
	if position.Floor > maxFloor {
		addValidationIssue(
			issues,
			"gam.position.floor",
			"out_of_range",
			floorOffset,
			"dungeon floor must be between 0 and 7",
		)
	}
	if position.X > maxDungeonCoordinate {
		addValidationIssue(
			issues,
			"gam.position.x",
			"out_of_range",
			partyXOffset,
			"dungeon X coordinate must be between 0 and 7",
		)
	}
	if position.Y > maxDungeonCoordinate {
		addValidationIssue(
			issues,
			"gam.position.y",
			"out_of_range",
			partyYOffset,
			"dungeon Y coordinate must be between 0 and 7",
		)
	}
}

func validateDungeon(dungeon DungeonState, issues *[]ValidationIssue) {
	if !dungeon.Active {
		return
	}
	if dungeon.Facing > DungeonWest {
		addValidationIssue(
			issues,
			"gam.dungeon.facing",
			"unknown_value",
			dungeonFacingOffset,
			"dungeon facing must be north, east, south, or west",
		)
	}
	if dungeon.LastDirection > maxDungeonLastDirection {
		addValidationIssue(
			issues,
			"gam.dungeon.lastDirection",
			"out_of_range",
			dungeonLastDirOffset,
			"active dungeon last direction must be cardinal or vertical",
		)
	}
	if dungeon.Type < DungeonTypeCave || dungeon.Type > DungeonTypeDungeon {
		addValidationIssue(
			issues,
			"gam.dungeon.type",
			"unknown_value",
			dungeonTypeOffset,
			"dungeon type must be cave, mine, or dungeon",
		)
	}
}

func addValidationIssue(issues *[]ValidationIssue, path, code string, offset int, message string) {
	*issues = append(*issues, ValidationIssue{
		Path:    path,
		Code:    code,
		Offset:  offset,
		Message: message,
	})
}

func knownGender(gender Gender) bool {
	return gender == GenderMale || gender == GenderFemale
}

func knownClass(class CharacterClass) bool {
	switch class {
	case ClassAvatar, ClassFighter, ClassBard, ClassWizard:
		return true
	default:
		return false
	}
}

func knownStatus(status CharacterStatus) bool {
	switch status {
	case StatusGood, StatusPoisoned, StatusCharmed, StatusAsleep, StatusDead:
		return true
	default:
		return false
	}
}

func knownPartyMap(partyMap PartyMap) bool {
	if partyMap == PartyMapActive || partyMap == PartyMapMartyr || partyMap == PartyMapUnrecruited {
		return true
	}

	return partyMap.IsInnLocation()
}
