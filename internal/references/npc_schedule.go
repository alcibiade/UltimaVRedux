package references

import (
	"unsafe"

	"github.com/bradhannah/UltimaVRedux/internal/datetime"
)

const (
	totalScheduleItemsPerNpc = 3

	sizeOfNPCSchedule = int(unsafe.Sizeof(NPCSchedule{}))
)

// NPCScheduleSlot identifies one of the three destination/action slots in an
// NPC schedule. The fourth schedule time maps back to slot one.
type NPCScheduleSlot int

// IsValid reports whether the slot can index an NPC schedule destination.
func (s NPCScheduleSlot) IsValid() bool {
	return s >= 0 && s < totalScheduleItemsPerNpc
}

// IsMerchantSlot reports whether the original merchant-hours test considers
// this slot open. Original merchant availability requires both arrived and desired slots to
// be odd.
func (s NPCScheduleSlot) IsMerchantSlot() bool {
	return s.IsValid() && s&1 != 0
}

// NPCSchedule stores three destinations/actions and four unsigned time markers.
type NPCSchedule struct {
	Ai    [totalScheduleItemsPerNpc]byte     `json:"ai"    yaml:"ai"`
	X     [totalScheduleItemsPerNpc]byte     `json:"x"     yaml:"x"`
	Y     [totalScheduleItemsPerNpc]byte     `json:"y"     yaml:"y"`
	Floor [totalScheduleItemsPerNpc]byte     `json:"floor" yaml:"floor"`
	Time  [totalScheduleItemsPerNpc + 1]byte `json:"time"  yaml:"time"`
}

// IndividualNPCBehaviour is the selected action and destination.
type IndividualNPCBehaviour struct {
	Ai       AiType      `json:"ai"       yaml:"ai"`
	Position Position    `json:"position" yaml:"position"`
	Floor    FloorNumber `json:"floor"    yaml:"floor"`
}

// CreateNPCSchedule decodes one complete fixed-width schedule record.
func CreateNPCSchedule(rawData []byte) NPCSchedule {
	if len(rawData) < sizeOfNPCSchedule {
		panic("not enough data to create NPCReference schedule")
	}
	npcSchedule := (*NPCSchedule)(unsafe.Pointer(&rawData[0]))

	return *npcSchedule
}

// CreateNPCScheduledFixedOneLocation constructs a stationary schedule on one floor.
func CreateNPCScheduledFixedOneLocation(position Position, floorNumber FloorNumber) NPCSchedule {
	npcSchedule := CreateNPCSchedule(make([]byte, sizeOfNPCSchedule))
	for i := range totalScheduleItemsPerNpc {
		npcSchedule.X[i] = byte(position.X)
		npcSchedule.Y[i] = byte(position.Y)
		npcSchedule.Floor[i] = byte(floorNumber)
		npcSchedule.Ai[i] = byte(Fixed)
		npcSchedule.Time[i] = byte(i)
	}

	return npcSchedule
}

// GetPreviousIndividualNPCBehaviourByUltimaDate returns the preceding destination slot.
func (n *NPCSchedule) GetPreviousIndividualNPCBehaviourByUltimaDate(ud datetime.UltimaDate) IndividualNPCBehaviour {
	index := int(n.DesiredSlot(ud))
	if index == 0 {
		index = len(n.Ai) - 1
	} else {
		index--
	}

	return IndividualNPCBehaviour{
		Ai:       AiType(n.Ai[index]),
		Position: Position{X: Coordinate(n.X[index]), Y: Coordinate(n.Y[index])},
		Floor:    FloorNumber(n.Floor[index]),
	}
}

// GetIndividualNPCBehaviourByUltimaDate resolves the selected action and destination.
func (n *NPCSchedule) GetIndividualNPCBehaviourByUltimaDate(ud datetime.UltimaDate) IndividualNPCBehaviour {
	return n.GetIndividualNPCBehaviourBySlot(n.DesiredSlot(ud))
}

// GetIndividualNPCBehaviourBySlot returns the action and destination assigned
// to one schedule slot.
func (n *NPCSchedule) GetIndividualNPCBehaviourBySlot(slot NPCScheduleSlot) IndividualNPCBehaviour {
	if !slot.IsValid() {
		panic("invalid NPC schedule slot")
	}
	index := int(slot)
	ai := AiType(n.Ai[index])

	return IndividualNPCBehaviour{
		Ai:       ai,
		Position: Position{X: Coordinate(n.X[index]), Y: Coordinate(n.Y[index])},
		Floor:    FloorNumber(n.Floor[index]),
	}
}

// DesiredSlot returns the schedule slot selected for the supplied game time.
// This is Redux's selected schedule destination (ref:NPC-0001); it does
// not imply that the NPC has reached that destination.
func (n *NPCSchedule) DesiredSlot(ud datetime.UltimaDate) NPCScheduleSlot {
	return NPCScheduleSlot(n.getScheduleIndex(ud))
}

// ApplyTalk3OneShot changes TALK3 to WANDER3 only in the supplied current
// slot. It returns whether the loaded schedule was changed.
func (n *NPCSchedule) ApplyTalk3OneShot(slot NPCScheduleSlot) bool {
	if !slot.IsValid() {
		panic("invalid NPC schedule slot")
	}
	if AiType(n.Ai[slot]) != Talk3 {
		return false
	}
	n.Ai[slot] = byte(Wander3)

	return true
}

func (n *NPCSchedule) getScheduleIndex(date datetime.UltimaDate) int {
	// Original unsigned-byte subtraction retains the first marker on ties.
	// The fourth marker selects slot 1, not a fourth destination. ref:NPC-0001.
	best := 0
	difference := date.Hour - n.Time[0]
	for index := 1; index < len(n.Time); index++ {
		candidate := date.Hour - n.Time[index]
		if candidate < difference {
			best, difference = index, candidate
		}
	}
	if best == totalScheduleItemsPerNpc {
		return 1
	}

	return best
}

// OverrideAllAI replaces the action of each destination slot.
func (n *NPCSchedule) OverrideAllAI(ai AiType) {
	for i := 0; i < len(n.Ai); i++ {
		n.Ai[i] = byte(ai)
	}
}

// OverrideAllPositions replaces the coordinates of each destination slot.
func (n *NPCSchedule) OverrideAllPositions(posX, posY byte) {
	for i := 0; i < len(n.X); i++ {
		n.X[i] = posX
		n.Y[i] = posY
	}
}
