package map_units

import (
	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// NPCFriendly is a friendly NPC that is not a player character
// It is used to represent NPCS who are either inanimate objects (ex. carpet) or non-combative NPCS
type NPCFriendly struct {
	NPCReference   references.NPCReference
	mapUnitDetails MapUnitDetails

	vehicleDetails         VehicleDetails
	surfaceChestNumber     byte
	surfaceChestAssessment references.ChestAssessment
	transientSurfaceChest  bool
	syntheticSurfaceChest  bool
}

const (
	// Authored static town chests have value 30; ref:CMD-0009.
	defaultSurfaceChestNumber byte = 30
	surfaceChestLockBit       byte = 0x80
)

func NewNPCFriendly(npcReference references.NPCReference, npcNum int) *NPCFriendly {
	var friendly NPCFriendly
	friendly.NPCReference = npcReference
	friendly.mapUnitDetails.NPCNum = npcNum
	friendly.mapUnitDetails.overriddenAiType = references.Unset
	friendly.mapUnitDetails.aiOverrideProvenance = references.AIOverrideProvenanceNone

	friendly.mapUnitDetails.CurrentPath = nil
	if friendly.IsSurfaceChest() {
		friendly.surfaceChestNumber = defaultSurfaceChestNumber
	}

	if !friendly.IsEmptyMapUnit() {
		friendly.mapUnitDetails.Visible = true
	}

	return &friendly
}

// IsSurfaceChest reports whether this authoritative surface map unit is a chest.
func (friendly *NPCFriendly) IsSurfaceChest() bool {
	return friendly != nil && friendly.NPCReference.GetSpriteIndex() == indexes.Chest
}

// SurfaceChestNumber returns the original object-number encoding for a chest.
func (friendly *NPCFriendly) SurfaceChestNumber() (byte, bool) {
	if !friendly.IsSurfaceChest() {
		return 0, false
	}

	return friendly.surfaceChestNumber, true
}

// SurfaceChestLocked reports whether the original shared lock/trap bit is set.
func (friendly *NPCFriendly) SurfaceChestLocked() bool {
	number, ok := friendly.SurfaceChestNumber()

	return ok && number&surfaceChestLockBit != 0
}

// SurfaceChestTreasureValue returns the chest value without its lock/trap bit.
func (friendly *NPCFriendly) SurfaceChestTreasureValue() int {
	number, ok := friendly.SurfaceChestNumber()
	if !ok {
		panic("surface chest value requested from non-chest NPC")
	}

	return int(number &^ surfaceChestLockBit)
}

// UnlockSurfaceChest clears the shared original lock/trap bit in place.
func (friendly *NPCFriendly) UnlockSurfaceChest() {
	if !friendly.IsSurfaceChest() {
		panic("surface chest unlock requested from non-chest NPC")
	}
	friendly.surfaceChestNumber &^= surfaceChestLockBit
	friendly.surfaceChestAssessment = references.ChestSafe
}

// RestoreSurfaceChestNumber restores physical state and resets knowledge to
// uninspected. Native hydration must restore any persisted assessment afterward.
func (friendly *NPCFriendly) RestoreSurfaceChestNumber(number byte) {
	if !friendly.IsSurfaceChest() {
		panic("surface chest state restored to non-chest NPC")
	}
	friendly.surfaceChestNumber = number
	friendly.surfaceChestAssessment = references.ChestUninspected
}

// SurfaceChestAssessment returns the party's retained knowledge of this chest.
// Non-chests always return uninspected.
func (friendly *NPCFriendly) SurfaceChestAssessment() references.ChestAssessment {
	return friendly.surfaceChestAssessment
}

// RestoreSurfaceChestAssessment sets checked knowledge after restoring physical
// state. Invalid persistence data must be rejected before calling this setter.
func (friendly *NPCFriendly) RestoreSurfaceChestAssessment(assessment references.ChestAssessment) {
	if !friendly.IsSurfaceChest() || !assessment.ValidForChest(friendly.SurfaceChestLocked()) {
		panic("invalid surface chest assessment")
	}
	friendly.surfaceChestAssessment = assessment
}

func NewNPCFriendlyVehicle(vehicleType references.VehicleType, npcRef references.NPCReference) *NPCFriendly {
	friendly := NewNPCFriendly(npcRef, int(npcRef.DialogNumber))

	friendly.vehicleDetails = NewVehicleDetails(vehicleType)
	if vehicleType == references.HorseVehicle && npcRef.GetSpriteIndex() == indexes.HorseLeft {
		friendly.vehicleDetails.SetPartyVehicleDirection(references.Left)
	}
	friendly.SetFloor(references.FloorNumber(npcRef.Schedule.Floor[0]))
	friendly.SetVisible(true)

	return friendly
}

func NewNPCFriendlyVehiceNewRef(vehicletype references.VehicleType, pos references.Position, floor references.FloorNumber) *NPCFriendly {
	npcRef := references.NewNPCReferenceForVehicle(vehicletype, pos, floor)

	return NewNPCFriendlyVehicle(vehicletype, *npcRef)
}

func NewNPCFriendlyVehiceNoVehicle() NPCFriendly {
	return *NewNPCFriendlyVehiceNewRef(references.NoPartyVehicle, references.Position{X: 0, Y: 0}, 0)
}

// NewNPCFriendlySurfaceChest creates a semantic surface chest without an
// authored small-map NPC reference.
func NewNPCFriendlySurfaceChest(
	location references.Location,
	pos references.Position,
	floor references.FloorNumber,
	npcNum int,
) *NPCFriendly {
	reference := references.NewNPCReferenceForSurfaceChest(location, pos, floor)

	chest := NewNPCFriendly(*reference, npcNum)
	chest.syntheticSurfaceChest = true

	return chest
}

// IsAuthoredSurfaceChest distinguishes source NPC references from semantic
// runtime chests, even if a dynamic slot and position happen to match a source.
func (friendly *NPCFriendly) IsAuthoredSurfaceChest() bool {
	return friendly.IsSurfaceChest() && !friendly.syntheticSurfaceChest
}

func (friendly *NPCFriendly) IsEmptyMapUnit() bool {
	return friendly.NPCReference.GetNPCType() == 0 || (friendly.NPCReference.Schedule.X[0] == 0 && friendly.NPCReference.Schedule.Y[0] == 0)
}

func (friendly *NPCFriendly) GetMapUnitType() MapUnitType {
	return NonPlayerCharacter
}

func (friendly *NPCFriendly) Pos() references.Position {
	return friendly.mapUnitDetails.Position
}

func (friendly *NPCFriendly) MapUnitDetails() *MapUnitDetails {
	return &friendly.mapUnitDetails
}

func (friendly *NPCFriendly) SetVisible(visible bool) {
	friendly.mapUnitDetails.Visible = visible
}

func (friendly *NPCFriendly) IsVisible() bool {
	return friendly.mapUnitDetails.Visible
}

func (friendly *NPCFriendly) Floor() references.FloorNumber {
	return friendly.mapUnitDetails.Floor
}

func (friendly *NPCFriendly) PosPtr() *references.Position {
	return &friendly.mapUnitDetails.Position
}

func (friendly *NPCFriendly) SetPositionByIndividualNPCBehaviour(indiv references.IndividualNPCBehaviour) {
	friendly.mapUnitDetails.Position = indiv.Position
	friendly.mapUnitDetails.Floor = indiv.Floor
	// friendly.mapUnitDetails.aiType = indiv.Ai
}

// PlaceAtScheduleSlot teleports an NPC to a schedule destination and records
// that destination as arrived. It is used for initial placement and hard
// resets as well as schedule-driven floor teleports.
func (friendly *NPCFriendly) PlaceAtScheduleSlot(slot references.NPCScheduleSlot) {
	indiv := friendly.NPCReference.Schedule.GetIndividualNPCBehaviourBySlot(slot)
	friendly.SetPositionByIndividualNPCBehaviour(indiv)
	friendly.mapUnitDetails.RestoreArrivedScheduleSlot(slot)
}

// RecordScheduleArrival records the desired slot only when the NPC is
// physically at that slot's destination and floor.
func (friendly *NPCFriendly) RecordScheduleArrival(date datetime.UltimaDate) bool {
	slot := friendly.NPCReference.Schedule.DesiredSlot(date)
	indiv := friendly.NPCReference.Schedule.GetIndividualNPCBehaviourBySlot(slot)
	if friendly.Pos() != indiv.Position || friendly.Floor() != indiv.Floor {
		return false
	}
	friendly.mapUnitDetails.RestoreArrivedScheduleSlot(slot)

	return true
}

// ArrivedScheduleSlot returns the last schedule destination reached by this
// loaded NPC.
func (friendly *NPCFriendly) ArrivedScheduleSlot() references.NPCScheduleSlot {
	return friendly.mapUnitDetails.ArrivedScheduleSlot()
}

// InferArrivedScheduleSlot reconstructs historical save state only when the
// saved position and floor exactly identify a schedule destination.
func (friendly *NPCFriendly) InferArrivedScheduleSlot() references.NPCScheduleSlot {
	for index := range friendly.NPCReference.Schedule.Ai {
		slot := references.NPCScheduleSlot(index)
		indiv := friendly.NPCReference.Schedule.GetIndividualNPCBehaviourBySlot(slot)
		if friendly.Pos() == indiv.Position && friendly.Floor() == indiv.Floor {
			return slot
		}
	}

	return 0
}

// ApplyTalk3OneShot changes TALK3 to WANDER3 in this loaded NPC's arrived
// schedule slot. NPCReference is copied into NPCFriendly, so shared references
// are not changed.
func (friendly *NPCFriendly) ApplyTalk3OneShot() bool {
	slot := friendly.ArrivedScheduleSlot()
	if !friendly.NPCReference.Schedule.ApplyTalk3OneShot(slot) {
		return false
	}
	friendly.mapUnitDetails.recordConsumedTalk3ScheduleSlot(slot)

	return true
}

// RestoreTalk3OneShot reapplies a persisted TALK3 mutation to this loaded
// schedule copy without changing shared NPC references.
func (friendly *NPCFriendly) RestoreTalk3OneShot(slot references.NPCScheduleSlot) bool {
	if !friendly.NPCReference.Schedule.ApplyTalk3OneShot(slot) {
		return false
	}
	friendly.mapUnitDetails.recordConsumedTalk3ScheduleSlot(slot)

	return true
}

func (friendly *NPCFriendly) SetDirectionBasedOnNewPos(position references.Position) {
	if friendly.GetVehicleDetails().VehicleType != references.NoPartyVehicle {
		// we change the direction of the vehicle based on the dominate direction
		friendly.SetDirection(
			friendly.MapUnitDetails().Position.GetDominateDirection(position))
	}
}

func (friendly *NPCFriendly) SetDirection(direction references.Direction) {
	friendly.GetVehicleDetails().SetPartyVehicleDirection(direction)
}

func (friendly *NPCFriendly) SetPos(position references.Position) {
	friendly.SetDirectionBasedOnNewPos(position)
	friendly.mapUnitDetails.Position = position
}

func (friendly *NPCFriendly) SetFloor(floor references.FloorNumber) {
	friendly.mapUnitDetails.Floor = floor
}

func (friendly *NPCFriendly) GetVehicleDetails() *VehicleDetails {
	if friendly.NPCReference.GetNPCType() == references.Vehicle {
		return &friendly.vehicleDetails
	}

	return &VehicleDetails{}
}

func (friendly *NPCFriendly) GetIndividualBehaviourByUltimaData(ultimaData datetime.UltimaDate) references.IndividualNPCBehaviour {
	if friendly.MapUnitDetails().overriddenAiType != references.Unset {
		// When AI is overridden (e.g., freed prisoner), use current position as "assigned"
		// This ensures the NPC takes the correct AI processing path
		return references.IndividualNPCBehaviour{
			Ai:       friendly.MapUnitDetails().overriddenAiType,
			Position: friendly.Pos(),
			Floor:    friendly.Floor(),
		}
	}

	return friendly.NPCReference.Schedule.GetIndividualNPCBehaviourByUltimaDate(ultimaData)
}

func (friendly *NPCFriendly) GetKeySpriteIndex() indexes.SpriteIndex {
	// For vehicles, use the unboarded sprite from VehicleDetails
	// This prevents vehicles from rendering as corpse/splat sprites (indices 286-287)
	// which were incorrectly calculated from Vehicle NPCType (0x1F=31) + offset (256) = 287
	if friendly.NPCReference.GetNPCType() == references.Vehicle {
		return friendly.vehicleDetails.GetUnBoardedSpriteIndex()
	}

	return friendly.NPCReference.GetSpriteIndex()
}
