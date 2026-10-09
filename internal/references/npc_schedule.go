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

type NPCSchedule struct {
	Ai    [totalScheduleItemsPerNpc]byte     `json:"ai"    yaml:"ai"`
	X     [totalScheduleItemsPerNpc]byte     `json:"x"     yaml:"x"`
	Y     [totalScheduleItemsPerNpc]byte     `json:"y"     yaml:"y"`
	Floor [totalScheduleItemsPerNpc]byte     `json:"floor" yaml:"floor"`
	Time  [totalScheduleItemsPerNpc + 1]byte `json:"time"  yaml:"time"`
}

type IndividualNPCBehaviour struct {
	Ai       AiType      `json:"ai"       yaml:"ai"`
	Position Position    `json:"position" yaml:"position"`
	Floor    FloorNumber `json:"floor"    yaml:"floor"`
}

func CreateNPCSchedule(rawData []byte) NPCSchedule {
	if len(rawData) < sizeOfNPCSchedule {
		panic("not enough data to create NPCReference schedule")
	}
	npcSchedule := (*NPCSchedule)(unsafe.Pointer(&rawData[0]))

	return *npcSchedule
}

func CreateNPCScheduledFixedOneLocation(position Position, floorNumber FloorNumber) NPCSchedule {
	npcSchedule := CreateNPCSchedule(make([]byte, sizeOfNPCSchedule))
	for i := 0; i < totalScheduleItemsPerNpc; i++ {
		npcSchedule.X[i] = byte(position.X)
		npcSchedule.Y[i] = byte(position.Y)
		npcSchedule.Floor[i] = byte(floorNumber)
		npcSchedule.Ai[i] = byte(Fixed)
		npcSchedule.Time[i] = byte(i)
	}

	return npcSchedule
}

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
	const totalSchedules = totalScheduleItemsPerNpc // Alias for readability
	nHour := int(date.Hour)                         // Extract the hour from UltimaDate

	// Inline function to handle specific index logic
	getIndex := func(nOrigIndex int) int {
		if nOrigIndex == totalSchedules {
			return 1
		}

		return nOrigIndex
	}

	// If all times are zero, return 0
	if n.Time[0] == 0 && n.Time[1] == 0 && n.Time[2] == 0 && n.Time[3] == 0 {
		return 0
	}

	// Check if the hour matches any of the times
	for i := 0; i < totalSchedules+1; i++ {
		if n.Time[i] == byte(nHour) {
			return getIndex(i)
		}
	}

	// Check ranges
	if nHour > int(n.Time[3]) && nHour < int(n.Time[0]) {
		return 1
	}
	if nHour > int(n.Time[0]) && nHour < int(n.Time[1]) {
		return 0
	}
	if nHour > int(n.Time[1]) && nHour < int(n.Time[2]) {
		return 1
	}
	if nHour > int(n.Time[2]) && nHour < int(n.Time[3]) {
		return 2
	}

	// Find the index of the earliest and latest times
	nEarliestTimeIndex := n.getEarliestTimeIndex()
	var nIndexPreviousToEarliest int
	if nEarliestTimeIndex == 0 {
		nIndexPreviousToEarliest = 1
	} else {
		nIndexPreviousToEarliest = nEarliestTimeIndex - 1
	}
	nLatestTimeIndex := n.getLatestTimeIndex()

	// Handle times outside the range
	if nHour < int(n.Time[nEarliestTimeIndex]) {
		return nIndexPreviousToEarliest
	}
	if nHour > int(n.Time[nLatestTimeIndex]) {
		return getIndex(nLatestTimeIndex)
	}

	// Programming bug: schedule logic should cover all time cases
	panic("getScheduleIndex fell all the way through, which doesn't make sense.")
}

func (n *NPCSchedule) getEarliestTimeIndex() int {
	nEarliest := n.Time[0]
	nEarliestIndex := 0
	for index := 1; index < len(n.Time); index++ {
		if n.Time[index] >= nEarliest {
			continue
		}

		nEarliestIndex = index
		nEarliest = n.Time[index]
	}

	return nEarliestIndex
}

func (n *NPCSchedule) getLatestTimeIndex() int {
	nLargest := n.Time[0]
	nLargestIndex := 0

	for index := 1; index < len(n.Time); index++ {
		if n.Time[index] <= nLargest {
			continue
		}

		nLargestIndex = index
		nLargest = n.Time[index]
	}

	return nLargestIndex
}

func (n *NPCSchedule) OverrideAllAI(ai AiType) {
	for i := 0; i < len(n.Ai); i++ {
		n.Ai[i] = byte(ai)
	}
}

func (n *NPCSchedule) OverrideAllPositions(posX, posY byte) {
	for i := 0; i < len(n.X); i++ {
		n.X[i] = posX
		n.Y[i] = posY
	}
}
