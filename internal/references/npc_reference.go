package references

import (
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// NPCType general type of NPC.
type NPCType byte

const (
	Blacksmith  NPCType = 0x81
	Barkeeper   NPCType = 0x82
	HorseSeller NPCType = 0x83
	Shipwright  NPCType = 0x84
	Healer      NPCType = 0x87
	InnKeeper   NPCType = 0x88
	MagicSeller NPCType = 0x85
	GuildMaster NPCType = 0x86
	NoStatedNpc NPCType = 0xFF
	Guard       NPCType = 0xFE
	WishingWell NPCType = 0xFD
	Vehicle     NPCType = 0x1F
	// unknowns may be crown and sandlewood box.
)

type NPCReference struct {
	Position     Position    `json:"position"      yaml:"position"`
	Location     Location    `json:"location"      yaml:"location"`
	DialogNumber byte        `json:"dialog_number" yaml:"dialog_number"`
	Schedule     NPCSchedule `json:"schedule"      yaml:"schedule"`
	NpcType      NPCType     `json:"npc_type"      yaml:"npc_type"`
}

const secondHalfSpriteTableIndex = 0x100

// LordBritishMagicCarpetNPC is the authored Castle Britannia carpet object slot.
const LordBritishMagicCarpetNPC = 22

func isValidType(nType int) bool {
	switch NPCType(nType) {
	case Blacksmith:
	case Barkeeper:
	case HorseSeller:
	case Shipwright:
	case Healer:
	case InnKeeper:
	case MagicSeller:
	case GuildMaster:
	case NoStatedNpc:
	case Guard:
	case WishingWell:
	case Vehicle:
		return true
	}

	return false
}

func NewNPCReferenceForVehicle(vehicle VehicleType, position Position, floorNumber FloorNumber) *NPCReference {
	npcRef := &NPCReference{}
	npcRef.Position = position
	npcRef.NpcType = Vehicle
	npcRef.SetKeyIndex(vehicle.GetBoardedSpriteByDirection(Right, Right))
	// SetKeyIndex overwrites NpcType based on sprite index, but for vehicles we need it to remain Vehicle
	npcRef.NpcType = Vehicle
	npcRef.Schedule = CreateNPCScheduledFixedOneLocation(position, floorNumber)

	return npcRef
}

// NewNPCReferenceForSurfaceChest creates a semantic chest reference without
// requiring a small-map authored NPC table entry.
func NewNPCReferenceForSurfaceChest(location Location, position Position, floorNumber FloorNumber) *NPCReference {
	npcRef := &NPCReference{Location: location, Position: position}
	npcRef.SetKeyIndex(indexes.Chest)
	npcRef.Schedule = CreateNPCScheduledFixedOneLocation(position, floorNumber)

	return npcRef
}

func (n *NPCReference) GetSpriteIndex() indexes.SpriteIndex {
	return indexes.SpriteIndex(int(n.NpcType) + secondHalfSpriteTableIndex)
}

func (n *NPCReference) SetKeyIndex(index indexes.SpriteIndex) {
	n.NpcType = NPCType(index - secondHalfSpriteTableIndex)
}

// SmallMapCombatMonsterType classifies authored NPC shapes independently of
// location, slot, dialog, and scheduled action. Four
// animation frames map to one creature; combat admission excludes quest-object
// crawler shapes and the field/whirlpool families from town Attack targets.
func (n *NPCReference) SmallMapCombatMonsterType() (MonsterType, bool) {
	if n == nil {
		return MonsterNone, false
	}
	delta := int(n.GetSpriteIndex()) - int(indexes.MageKeyIndex)
	if delta < 0 || delta >= TotalMonsterTypes*indexes.StandardNumberOfAnimationFrames {
		return MonsterNone, false
	}
	monster := MonsterType(delta / indexes.StandardNumberOfAnimationFrames)
	if monster == MonsterCrawler || monster.IsEnvironmentalHazard() {
		return MonsterNone, false
	}

	return monster, true
}

func (n *NPCReference) GetVehicleType() VehicleType {
	spriteIndex := n.GetSpriteIndex()
	if spriteIndex.IsSkiff() {
		return SkiffVehicle
	}
	if spriteIndex.IsHorseUnBoarded() {
		return HorseVehicle
	}
	if spriteIndex.IsFrigateFurled() {
		return FrigateVehicle
	}
	if spriteIndex.IsMagicCarpetUnboarded() {
		return CarpetVehicle
	}

	return NoPartyVehicle
}

func (n *NPCReference) IsEmptyNPC() bool {
	return false
}

func (n *NPCReference) GetNPCType() NPCType {
	tileIndex := n.GetSpriteIndex()

	if tileIndex.IsPartOfAnimation(indexes.GuardKeyIndex) {
		return Guard
	}
	if isDaemonSprite(tileIndex) {
		return NoStatedNpc
	}
	const avatarTileIndexInNPCRefes = 256
	if tileIndex == avatarTileIndexInNPCRefes { // it's the Avatar
		return NoStatedNpc
	}

	// Vehicle is a synthetic shape marker, not a dialogue discriminator.
	// NPC records store appearance separately from Talk number (ref:ENEMY-0003); dialogue
	// 0x1f belongs to ordinary speakers such as Woolfe, not vehicle objects.
	if n.DialogNumber != byte(Vehicle) && isValidType(int(n.DialogNumber)) {
		return NPCType(n.DialogNumber)
	}

	if isValidType(int(n.NpcType)) {
		return n.NpcType
	}

	if tileIndex.IsHorseUnBoarded() || tileIndex.IsMagicCarpetUnboarded() || tileIndex.IsFrigateFurled() || tileIndex.IsSkiff() {
		return Vehicle
	}

	return NoStatedNpc
}

func isDaemonSprite(sprite indexes.SpriteIndex) bool {
	return sprite >= indexes.DaemonKeyIndex &&
		sprite < indexes.DaemonKeyIndex+indexes.StandardNumberOfAnimationFrames
}

func (n *NPCReference) WantsToAttackAvatarWhenBadStuffGoesDown() bool {
	spriteIndex := n.GetSpriteIndex()

	return spriteIndex.IsPartOfAnimation(indexes.GuardKeyIndex) ||
		spriteIndex.IsPartOfAnimation(indexes.DaemonKeyIndex) ||
		spriteIndex.IsPartOfAnimation(indexes.ShadowLordKeyIndex)
}

// IsBeggar returns true if this NPC is a beggar based on their sprite.
// Used for karma bonus detection when giving gold to beggars in conversations.
// The Beggar family occupies raw tiles 0x6c..0x6f; ref:ENEMY-0001.
func (n *NPCReference) IsBeggar() bool {
	return n.GetSpriteIndex().IsBeggar()
}
