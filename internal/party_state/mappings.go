package party_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/game_state/util"
)

/*
enum <ubyte> PartyMap { Active = 0x00, Martyr = 0x7F, Unrecruited = 0xFF };
enum <ubyte> Location { };
enum <ubyte> Boolean { False = 0, True = 1 };
enum <ubyte> BritOrUnderworld { Britannia = 0, Underworld = 0xFF };
enum <ubyte> MoonstoneStatus { Buried = 0, InInventory = 0xFF };
enum <ubyte> DungeonOrientation { North = 0, East = 1, South = 2, West = 2 };
enum <ubyte> DungeonType { Cave = 1, Mine = 2, Dungeon = 3 };
*/

/*
CharacterClass

enum <ubyte> Class { Avatar = 'A', Fighter = 'F', Bard = 'B', Wizard = 'M' };
*/

// MaxCharactersInParty is the original limit of simultaneous party members.
const MaxCharactersInParty = 6

// CharacterClass is the original class byte, an ASCII letter.
type CharacterClass byte

// Avatar, Fighter, Bard and Wizard are the class letters stored in a character record.
const (
	Avatar  CharacterClass = 'A'
	Fighter CharacterClass = 'F'
	Bard    CharacterClass = 'B'
	Wizard  CharacterClass = 'M'
)

// String returns the friendly name for a character class
func (c CharacterClass) String() string {
	switch c {
	case Avatar:
		return "Avatar"
	case Fighter:
		return "Fighter"
	case Bard:
		return "Bard"
	case Wizard:
		return "Wizard"
	}

	return "Unknown"
}

// CharacterStatus is the original status byte, an ASCII letter.
type CharacterStatus byte

// Good, Poisoned, Charmed, Sleep and Dead are the status letters stored in a character record.
const (
	Good     CharacterStatus = 'G'
	Poisoned CharacterStatus = 'P'
	Charmed  CharacterStatus = 'C'
	Sleep    CharacterStatus = 'S'
	Dead     CharacterStatus = 'D'
)

// CharacterStatuses lists every status with its display name, in save order.
var CharacterStatuses = util.OrderedMapping[CharacterStatus]{
	{Id: Good, FriendlyName: "Good"},
	{Id: Poisoned, FriendlyName: "Poisoned"},
	{Id: Charmed, FriendlyName: "Charmed"},
	{Id: Sleep, FriendlyName: "Sleep"},
	{Id: Dead, FriendlyName: "Dead"},
}

// Gender
// enum <ubyte> Gender { Male = 0x0B, Female = 0x0C };

// CharacterGender is the original gender byte, a code of 0x0B or 0x0C.
type CharacterGender byte

// Male and Female are the gender codes stored in a character record.
const (
	Male   CharacterGender = 0x0B
	Female CharacterGender = 0x0C
)

// CharacterGenders lists every gender with its display name, in save order.
var CharacterGenders = util.OrderedMapping[CharacterGender]{
	{Id: Male, FriendlyName: "Male"},
	{Id: Female, FriendlyName: "Female"},
}

/*
Other smaller definitions
*/
// type BritOrUnderworld byte
//
// const (
//	Britannia  BritOrUnderworld = 0xff
//	Underworld BritOrUnderworld = 0x00
// )
