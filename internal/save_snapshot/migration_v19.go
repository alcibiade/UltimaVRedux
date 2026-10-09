package save_snapshot

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// saveSnapshotV19 freezes the shared v17-v19 shape. In particular, historical
// dungeon observations must never decode through evolving runtime journal types.
// Map-unit and physical dungeon records retain the already frozen v12 shape.
type saveSnapshotV19 struct {
	Version          int                      `json:"version"`
	Party            PartySnapshot            `json:"party"`
	Inventory        InventorySnapshot        `json:"inventory"`
	World            WorldSnapshot            `json:"world"`
	Progression      ProgressionSnapshot      `json:"progression"`
	Effects          EffectsSnapshot          `json:"effects"`
	Wind             WindSnapshot             `json:"wind"`
	Random           RandomSnapshot           `json:"random"`
	Continuation     ContinuationSnapshot     `json:"continuation"`
	Redux            ReduxSnapshot            `json:"redux"`
	Presentation     journal.Presentation     `json:"presentation"`
	LastConversation journal.Conversation     `json:"lastConversation"`
	DungeonJournal   [8]dungeonObservationV19 `json:"dungeonJournal"`
	Dungeon          *dungeonSnapshotV12      `json:"dungeon"`
	PartyVehicle     *vehicleSnapshotV12      `json:"partyVehicle"`
	MapUnits         []mapUnitSnapshotV12     `json:"mapUnits"`
	ItemStacks       []ItemStackSnapshot      `json:"itemStacks"`
	MapMutations     []MapMutationSnapshot    `json:"mapMutations"`
}

type dungeonObservationV19 struct {
	Known         bool `json:"known"`
	WordUsed      bool `json:"wordUsed"`
	EntranceSeen  bool `json:"entranceSeen"`
	EntranceX     int  `json:"entranceX"`
	EntranceY     int  `json:"entranceY"`
	EntranceFloor int  `json:"entranceFloor"`
	Open          bool `json:"open"`
}

func (previous saveSnapshotV19) validationView() SaveSnapshot {
	view := SaveSnapshot{
		Version: previous.Version, Party: previous.Party, Inventory: previous.Inventory,
		World: previous.World, Progression: previous.Progression, Effects: previous.Effects,
		Wind: previous.Wind, Random: previous.Random, Continuation: previous.Continuation,
		Redux: previous.Redux, Presentation: previous.Presentation, LastConversation: previous.LastConversation,
		Dungeon: promoteDungeonV12(previous.Dungeon), PartyVehicle: promoteVehicleV12(previous.PartyVehicle),
		MapUnits: promoteMapUnitsV12(previous.MapUnits), ItemStacks: previous.ItemStacks, MapMutations: previous.MapMutations,
	}
	for index, old := range previous.DungeonJournal {
		observation := journal.DungeonObservation{Known: old.Known, WordUsed: old.WordUsed}
		if old.EntranceSeen {
			slot := journal.DungeonSurfaceSite
			if old.EntranceFloor == -1 {
				slot = journal.DungeonUnderworldSite
			}
			observation.Sites[slot] = journal.DungeonSiteObservation{
				Seen: true, Open: old.Open,
				Location: journal.SiteLocation{Map: references.BritanniaUnderworld,
					Floor: references.FloorNumber(old.EntranceFloor), X: old.EntranceX, Y: old.EntranceY},
			}
		}
		view.DungeonJournal[index] = observation
	}

	return view
}

func (previous saveSnapshotV19) validateDungeonJournal() []Issue {
	var issues []Issue
	for index, old := range previous.DungeonJournal {
		path := fmt.Sprintf("dungeonJournal[%d]", index)
		if old.EntranceSeen && !old.Known || !old.EntranceSeen && (old.Open || old.WordUsed) {
			issues = append(issues, Issue{Path: path, Code: "invalid", Message: "unobserved entrance cannot disclose seal or word"})
		}
		// Dormant historical coordinates are intentionally ignored. Observed
		// noncanonical in-bounds coordinates are private evidence, not surveys.
		if old.EntranceSeen && (old.EntranceX < 0 || old.EntranceX > 255 || old.EntranceY < 0 || old.EntranceY > 255 ||
			(old.EntranceFloor != 0 && old.EntranceFloor != -1)) {
			issues = append(issues, Issue{Path: path, Code: "invalid", Message: "observed entrance requires valid world coordinates"})
		}
	}

	return issues
}

// BackfillMoonstonePossession adds only inventory-proven recognition for native
// historical promotion and legacy import. Hidden placement arrays are not lore.
func (snapshot *SaveSnapshot) BackfillMoonstonePossession() {
	for phase, quantity := range snapshot.Inventory.Moonstones {
		if quantity > 0 {
			if !snapshot.MoonstoneJournal[phase].Known {
				snapshot.MoonstoneJournal[phase] = journal.MoonstoneObservation{Known: true, FirstSource: journal.MoonstoneSourcePossession}
			}
			snapshot.MoonstoneJournal[phase].OriginalSiteRetired = true
			snapshot.MoonstoneJournal[phase].Survey = journal.SurveyedLocation{}
			snapshot.MoonstoneJournal[phase].SurveyContext = journal.MoonstoneContext{}
		}
	}
}
