package game_state

// const (
// 	ubPartyMembers StartingMemoryAddressUb = 0x2B5
// 	ubActivePlayer StartingMemoryAddressUb = 0x2D5
// )

// const (
// 	u16CurrentYear StartingMemoryAddressU16 = 0x2CE
// )

//goland:noinspection ALL

// MoonstoneStatus is a moonstone state byte in the save: buried in the ground or carried by the party.
type MoonstoneStatus byte

// Save values of MoonstoneStatus.
const (
	Buried      MoonstoneStatus = 0x00
	InInventory MoonstoneStatus = 0xFF
)
