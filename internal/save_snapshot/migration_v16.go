package save_snapshot

import "github.com/bradhannah/UltimaVRedux/internal/journal"

// saveSnapshotV16 freezes the body before conversation retention was added.
// All referenced member DTOs retain their v16 wire shapes.
type saveSnapshotV16 struct {
	Version      int                   `json:"version"`
	Party        PartySnapshot         `json:"party"`
	Inventory    InventorySnapshot     `json:"inventory"`
	World        WorldSnapshot         `json:"world"`
	Progression  ProgressionSnapshot   `json:"progression"`
	Effects      EffectsSnapshot       `json:"effects"`
	Wind         WindSnapshot          `json:"wind"`
	Random       RandomSnapshot        `json:"random"`
	Continuation ContinuationSnapshot  `json:"continuation"`
	Redux        ReduxSnapshot         `json:"redux"`
	Presentation journal.Presentation  `json:"presentation"`
	Dungeon      *dungeonSnapshotV12   `json:"dungeon"`
	PartyVehicle *vehicleSnapshotV12   `json:"partyVehicle"`
	MapUnits     []mapUnitSnapshotV12  `json:"mapUnits"`
	ItemStacks   []ItemStackSnapshot   `json:"itemStacks"`
	MapMutations []MapMutationSnapshot `json:"mapMutations"`
}

func (previous saveSnapshotV16) validationView() SaveSnapshot {
	return SaveSnapshot{
		Version: previous.Version, Party: previous.Party, Inventory: previous.Inventory,
		World: previous.World, Progression: previous.Progression, Effects: previous.Effects,
		Wind: previous.Wind, Random: previous.Random, Continuation: previous.Continuation,
		Redux: previous.Redux, Presentation: previous.Presentation,
		LastConversation: journal.Conversation{Exchanges: nil, Truncated: false},
		DungeonJournal:   [8]journal.DungeonObservation{},
		Dungeon:          promoteDungeonV12(previous.Dungeon), PartyVehicle: promoteVehicleV12(previous.PartyVehicle), MapUnits: promoteMapUnitsV12(previous.MapUnits),
		ItemStacks: previous.ItemStacks, MapMutations: previous.MapMutations, reconstructMapUnits: false,
	}
}
