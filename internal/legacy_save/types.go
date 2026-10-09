package legacy_save

// SavedGameSize and SavedObjectsSize are the exact original bundle file sizes.
const (
	SavedGameSize    = 4192
	SavedObjectsSize = 512
)

const (
	characterCount   = 16
	objectCount      = 32
	objectRecordSize = 8
)

// Gender is the original character gender byte.
type Gender byte

// GenderMale and GenderFemale are the original character gender values.
const (
	GenderMale   Gender = 0x0b
	GenderFemale Gender = 0x0c
)

// CharacterClass is the original character class letter.
type CharacterClass byte

// ClassAvatar, ClassFighter, ClassBard, and ClassWizard are the original class values.
const (
	ClassAvatar  CharacterClass = 'A'
	ClassFighter CharacterClass = 'F'
	ClassBard    CharacterClass = 'B'
	ClassWizard  CharacterClass = 'M'
)

// CharacterStatus is the original character status letter.
type CharacterStatus byte

// StatusGood through StatusDead are the original character status values.
const (
	StatusGood     CharacterStatus = 'G'
	StatusPoisoned CharacterStatus = 'P'
	StatusCharmed  CharacterStatus = 'C'
	StatusAsleep   CharacterStatus = 'S'
	StatusDead     CharacterStatus = 'D'
)

// PartyMap is the original pmap byte from a character record.
type PartyMap byte

// PartyMapActive, PartyMapMartyr, and PartyMapUnrecruited are original pmap sentinels.
const (
	PartyMapActive      PartyMap = 0x00
	PartyMapMartyr      PartyMap = 0x7f
	PartyMapUnrecruited PartyMap = 0xff
)

// IsInnLocation reports whether the value is an original map ID. Original
// pmap values use map IDs for companions left at inns.
func (partyMap PartyMap) IsInnLocation() bool {
	return partyMap >= 1 && partyMap <= 40
}

// EquippedItems contains the six readied-item bytes in source order.
type EquippedItems struct {
	Helmet byte
	Armor  byte
	Hands  [2]byte
	Ring   byte
	Amulet byte
}

// Character is one decoded 32-byte original roster record.
type Character struct {
	Occupied     bool
	Name         string
	NameBytes    [9]byte
	Gender       Gender
	Class        CharacterClass
	Status       CharacterStatus
	Strength     byte
	Dexterity    byte
	Intelligence byte
	CurrentMP    byte
	CurrentHP    uint16
	MaximumHP    uint16
	Experience   uint16
	Level        byte
	MonthsAtInn  byte
	ArmorClass   byte
	Equipment    EquippedItems
	PartyMap     PartyMap
}

// ShardSourceRegion preserves the four-byte shard allocation in SAVED.GAM; ref:SAVE-0003.
// Indexes 0, 1, and 2 represent Falsehood, Hatred, and Cowardice.
// Source[3] has no verified semantic meaning; ref:SAVE-0003.
type ShardSourceRegion struct {
	Source    [4]byte
	Falsehood byte
	Hatred    byte
	Cowardice byte
}

// SpecialItems contains the six contiguous special-item bytes at 0x0214.
type SpecialItems struct {
	Spyglass     byte
	HMSCapePlans byte
	Sextant      byte
	PocketWatch  byte
	BlackBadge   byte
	WoodenBox    byte
}

// Inventory contains all explicitly decoded party inventory regions.
type Inventory struct {
	Food           uint16
	Gold           uint16
	Keys           byte
	Gems           byte
	Torches        byte
	Grapples       byte
	Carpets        byte
	SkullKeys      byte
	Amulet         byte
	Crown          byte
	Sceptre        byte
	Shards         ShardSourceRegion
	SpecialItems   SpecialItems
	Equipment      [48]byte
	PreparedSpells [48]byte
	Scrolls        [8]byte
	Potions        [8]byte
	Reagents       [8]byte
}

// Moonstone combines one index from the four parallel moonstone arrays.
type Moonstone struct {
	X     byte
	Y     byte
	Map   byte
	Level byte
}

// Calendar is the saved Britannian date and time.
type Calendar struct {
	Year   uint16
	Month  byte
	Day    byte
	Hour   byte
	Minute byte
}

// MapID is the original onmap byte.
type MapID byte

// WorldPosition is the saved map, floor, and party coordinates.
type WorldPosition struct {
	Location MapID
	Floor    byte
	X        byte
	Y        byte
}

// ShrineProgress contains the virtue-indexed shrine quest bits; ref:SHRINE-0002.
type ShrineProgress struct {
	ActiveBits    byte
	CompletedBits byte
	Closed        [8]byte
}

// ShipState contains the verified ship-shop and sailing bytes. ShipX and ShipY
// identify the pending ship-shop placement, not the boarded party vehicle.
type ShipState struct {
	X       byte
	Y       byte
	Sailing byte
}

// DungeonFacing is the original cardinal dungeon orientation.
type DungeonFacing byte

// DungeonNorth through DungeonWest are the original dungeon-facing values.
const (
	DungeonNorth DungeonFacing = iota
	DungeonEast
	DungeonSouth
	DungeonWest
)

// DungeonType is the original cave, mine, or dungeon presentation type.
type DungeonType byte

// DungeonTypeCave through DungeonTypeDungeon are the original dungeon-type values.
const (
	DungeonTypeCave DungeonType = iota + 1
	DungeonTypeMine
	DungeonTypeDungeon
)

// DungeonState contains the decoded dungeon-context marker and tail fields.
type DungeonState struct {
	Active        bool
	Cells         [512]byte
	LastDirection byte
	Facing        DungeonFacing
	Type          DungeonType
	Skiffs        byte
}

// ObjectRecord is the original eight-byte world-object record used by SAVED.GAM and
// both halves of SAVED.OOL.
type ObjectRecord struct {
	Tile      byte
	Shape     byte
	X         byte
	Y         byte
	Level     byte
	Number    byte
	Animation byte
	Misc      byte
}

// SavedGame contains the verified typed regions decoded from SAVED.GAM.
type SavedGame struct {
	Characters            [characterCount]Character
	Inventory             Inventory
	Moonstones            [8]Moonstone
	SkullKeysFoundDay     byte
	ReagentCacheDays      [3]byte
	PartyCount            byte
	SearchFlags           [24]byte
	Calendar              Calendar
	ActiveSpell           byte
	ActivePlayer          byte
	PlayerShape           byte
	OldHour               byte
	CombatActionTimeCount byte
	WindUpdateCount       byte
	Karma                 byte
	LastGave              byte
	LastCamped            byte
	LastWandererMonth     byte
	SpellDuration         byte
	Spell                 byte
	Magic                 byte
	Wind                  byte
	Position              WorldPosition
	MagicLight            byte
	TorchTurns            byte
	Shrines               ShrineProgress
	OpenDungeons          [8]byte
	DungeonRooms          [14]byte
	ShadowlordLords       [3]byte
	LastCalledShadowlord  byte
	Ship                  ShipState
	Smashed               byte
	CurrentShadowlord     byte
	Easy                  byte
	DeadNPCs              [128]byte
	MetNPCs               [128]byte
	ActiveObjects         [objectCount]ObjectRecord
	Dungeon               DungeonState
}

// SavedObjects contains the two ordered SAVED.OOL object-list halves.
type SavedObjects struct {
	Britannia  [objectCount]ObjectRecord
	Underworld [objectCount]ObjectRecord
}

// Bundle is detached from its input buffers: every source region is copied
// into values or fixed-size arrays.
type Bundle struct {
	Game    SavedGame
	Objects SavedObjects

	sourceDigest [32]byte
}

// IsDungeon reports whether SAVED.GAM was captured on map 33 through 40.
func (bundle Bundle) IsDungeon() bool {
	return bundle.Game.Dungeon.Active
}
