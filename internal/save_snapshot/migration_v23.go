package save_snapshot

import "github.com/bradhannah/UltimaVRedux/internal/journal"

// saveSnapshotV23 freezes the complete accepted door/atlas contract. In
// particular its vehicles and map units must not acquire V24 identity fields.
type saveSnapshotV23 struct {
	Version          int                                `json:"version"`
	Party            PartySnapshot                      `json:"party"`
	Inventory        InventorySnapshot                  `json:"inventory"`
	World            WorldSnapshot                      `json:"world"`
	Progression      ProgressionSnapshot                `json:"progression"`
	Effects          EffectsSnapshot                    `json:"effects"`
	Wind             WindSnapshot                       `json:"wind"`
	Random           RandomSnapshot                     `json:"random"`
	Continuation     ContinuationSnapshot               `json:"continuation"`
	Redux            ReduxSnapshot                      `json:"redux"`
	Presentation     journal.Presentation               `json:"presentation"`
	LastConversation journal.Conversation               `json:"lastConversation"`
	DungeonJournal   [8]journal.DungeonObservation      `json:"dungeonJournal"`
	MoonstoneJournal [8]journal.MoonstoneObservation    `json:"moonstoneJournal"`
	AtlasLandmarks   []journal.AtlasLandmarkObservation `json:"atlasLandmarks"`
	Dungeon          *DungeonSnapshot                   `json:"dungeon"`
	PartyVehicle     *vehicleSnapshotV12                `json:"partyVehicle"`
	MapUnits         []mapUnitSnapshotV23               `json:"mapUnits"`
	ItemStacks       []ItemStackSnapshot                `json:"itemStacks"`
	MapMutations     []MapMutationSnapshot              `json:"mapMutations"`
	OpenDoor         *OpenDoorSnapshot                  `json:"openDoor"`
}

// mapUnitSnapshotV23 is also the frozen V20-V22 map-unit wire shape.
type mapUnitSnapshotV23 struct {
	World                     int                 `json:"world"`
	ControllerOrder           int                 `json:"controllerOrder"`
	Kind                      string              `json:"kind"`
	ReferenceID               int                 `json:"referenceId"`
	NPCNum                    int                 `json:"npcNum"`
	Position                  PositionSnapshot    `json:"position"`
	Visible                   bool                `json:"visible"`
	SkipNextTurn              bool                `json:"skipNextTurn"`
	LegacyWantsToAutoTalk     bool                `json:"wantsToAutoTalk"`
	OverriddenAI              int                 `json:"overriddenAi"`
	AIOverrideProvenance      string              `json:"aiOverrideProvenance"`
	ArrivedScheduleSlot       *int                `json:"arrivedScheduleSlot,omitempty"`
	ConsumedTalk3ScheduleSlot *int                `json:"consumedTalk3ScheduleSlot,omitempty"`
	CurrentPath               []PositionSnapshot  `json:"currentPath"`
	Vehicle                   *vehicleSnapshotV12 `json:"vehicle"`
	EnemyVariant              string              `json:"enemyVariant"`
	EnemyCurrentHP            int                 `json:"enemyCurrentHp"`
	EnemyFacing               int                 `json:"enemyFacing"`
	Shadowlord                *int                `json:"shadowlord"`
	SurfaceChestNumber        *int                `json:"surfaceChestNumber,omitempty"`
	SurfaceChestAssessment    int                 `json:"surfaceChestAssessment"`
}

func promoteMapUnitsV23(previous []mapUnitSnapshotV23) []MapUnitSnapshot {
	if previous == nil {
		return nil
	}
	units := make([]MapUnitSnapshot, len(previous))
	for i, old := range previous {
		units[i] = MapUnitSnapshot{
			World: old.World, ControllerOrder: old.ControllerOrder, Kind: old.Kind,
			ReferenceID: old.ReferenceID, NPCNum: old.NPCNum, Position: old.Position,
			Visible: old.Visible, SkipNextTurn: old.SkipNextTurn, LegacyWantsToAutoTalk: old.LegacyWantsToAutoTalk,
			OverriddenAI: old.OverriddenAI, AIOverrideProvenance: old.AIOverrideProvenance,
			ArrivedScheduleSlot: old.ArrivedScheduleSlot, ConsumedTalk3ScheduleSlot: old.ConsumedTalk3ScheduleSlot,
			CurrentPath: old.CurrentPath, Vehicle: promoteVehicleV12(old.Vehicle),
			EnemyVariant: old.EnemyVariant, EnemyCurrentHP: old.EnemyCurrentHP, EnemyFacing: old.EnemyFacing,
			Shadowlord: old.Shadowlord, SurfaceChestNumber: old.SurfaceChestNumber, SurfaceChestAssessment: old.SurfaceChestAssessment,
		}
	}

	return units
}

func (previous saveSnapshotV23) validationView() SaveSnapshot {
	return SaveSnapshot{Version: previous.Version, Party: previous.Party, Inventory: previous.Inventory,
		World: previous.World, Progression: previous.Progression, Effects: previous.Effects, Wind: previous.Wind,
		Random: previous.Random, Continuation: previous.Continuation, Redux: previous.Redux,
		Presentation: previous.Presentation, LastConversation: previous.LastConversation,
		DungeonJournal: previous.DungeonJournal, MoonstoneJournal: previous.MoonstoneJournal, AtlasLandmarks: previous.AtlasLandmarks,
		Dungeon: previous.Dungeon, PartyVehicle: promoteVehicleV12(previous.PartyVehicle), MapUnits: promoteMapUnitsV23(previous.MapUnits),
		ItemStacks: previous.ItemStacks, MapMutations: previous.MapMutations, OpenDoor: previous.OpenDoor}
}
