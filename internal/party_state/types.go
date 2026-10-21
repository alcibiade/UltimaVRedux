package party_state

import (
	"encoding/binary"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// NMaxPlayerNameSize is the byte length of the fixed name field in a character record.
const NMaxPlayerNameSize = 9

// PartyMap preserves the original pmap byte. Zero identifies the packed active
// roster, map IDs identify companions left at inns, and the two sentinels
// represent Blackthorn's victims and unrecruited companions.
type PartyMap byte

const (
	// PartyMapActive identifies a character in the packed active roster.
	PartyMapActive PartyMap = 0x00
	// PartyMapMartyr identifies a character executed by Blackthorn.
	PartyMapMartyr PartyMap = 0x7f
	// PartyMapUnrecruited identifies a companion who has not joined the party.
	PartyMapUnrecruited PartyMap = 0xff
)

// PartyMapAtInn returns the original map ID representation for a companion at an inn.
func PartyMapAtInn(location references.Location) PartyMap {
	if location < references.Moonglow || location > references.Doom {
		panic("inn party map requires a persistent map location")
	}

	return PartyMap(location)
}

// IsMartyr reports whether Blackthorn executed the character.
func (p PartyMap) IsMartyr() bool { return p == PartyMapMartyr }

// PlayerCharacter mirrors one 32-byte character record of the original SAVED.GAM roster.
type PlayerCharacter struct {
	Name         [NMaxPlayerNameSize]byte
	Gender       CharacterGender
	Class        CharacterClass
	status       CharacterStatus // private - use GetStatus() and setter methods
	Strength     byte
	Dexterity    byte
	Intelligence byte
	currentMp    byte   // private - use GetCurrentMp(), AddMp(), SubtractMp(), SetMp()
	currentHp    uint16 // private - use GetCurrentHp(), AddHp(), SubtractHp(), SetHp()
	maxHp        uint16 // private - use GetMaxHp(), SetMaxHp()
	Exp          uint16
	Level        byte
	MonthsAtInn  byte
	ArmorClass   byte // Legacy persisted value; combat derives defense from equipped items.
	Helmet       byte
	Armor        byte
	Weapon       byte
	Shield       byte
	Ring         byte
	Amulet       byte
	PartyMap     PartyMap
}

// characterRecordSize is the byte length of one packed original character record.
const characterRecordSize = 32

// uint16Size is the byte width of a packed 16-bit record field.
const uint16Size = 2

// recordReader consumes consecutive fields of a packed little-endian record.
type recordReader struct {
	data []byte
}

func (r *recordReader) bytes(count int) []byte {
	field := r.data[:count]
	r.data = r.data[count:]

	return field
}

func (r *recordReader) byte() byte {
	return r.bytes(1)[0]
}

func (r *recordReader) uint16() uint16 {
	return binary.LittleEndian.Uint16(r.bytes(uint16Size))
}

// decodeCharacter reads one packed original character record in field order.
func decodeCharacter(record []byte) PlayerCharacter {
	var character PlayerCharacter
	reader := recordReader{data: record}
	copy(character.Name[:], reader.bytes(NMaxPlayerNameSize))
	character.Gender = CharacterGender(reader.byte())
	character.Class = CharacterClass(reader.byte())
	character.status = CharacterStatus(reader.byte())
	character.Strength = reader.byte()
	character.Dexterity = reader.byte()
	character.Intelligence = reader.byte()
	character.currentMp = reader.byte()
	character.currentHp = reader.uint16()
	character.maxHp = reader.uint16()
	character.Exp = reader.uint16()
	character.Level = reader.byte()
	character.MonthsAtInn = reader.byte()
	character.ArmorClass = reader.byte()
	character.Helmet = reader.byte()
	character.Armor = reader.byte()
	character.Weapon = reader.byte()
	character.Shield = reader.byte()
	character.Ring = reader.byte()
	character.Amulet = reader.byte()
	character.PartyMap = PartyMap(reader.byte())

	return character
}
