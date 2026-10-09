package save_snapshot

import "github.com/bradhannah/UltimaVRedux/internal/journal"

// saveSnapshotV15 freezes the last pre-presentation body. In particular, its
// Redux events retain the historical shape without sourceName.
type saveSnapshotV15 struct {
	Version      int                      `json:"version"`
	Party        partySnapshotV12         `json:"party"`
	Inventory    inventorySnapshotV12     `json:"inventory"`
	World        worldSnapshotV12         `json:"world"`
	Progression  progressionSnapshotV12   `json:"progression"`
	Effects      effectsSnapshotV15       `json:"effects"`
	Wind         windSnapshotV12          `json:"wind"`
	Random       randomSnapshotV12        `json:"random"`
	Continuation continuationSnapshotV14  `json:"continuation"`
	Redux        reduxSnapshotV13         `json:"redux"`
	Dungeon      *dungeonSnapshotV12      `json:"dungeon"`
	PartyVehicle *vehicleSnapshotV12      `json:"partyVehicle"`
	MapUnits     []mapUnitSnapshotV12     `json:"mapUnits"`
	ItemStacks   []itemStackSnapshotV12   `json:"itemStacks"`
	MapMutations []mapMutationSnapshotV12 `json:"mapMutations"`
}

type effectsSnapshotV15 struct {
	DurationType                 byte              `json:"durationType"`
	DurationTurns                int               `json:"durationTurns"`
	DurationToggleOnly           bool              `json:"durationToggleOnly"`
	MagicLightMinutesRemaining   int               `json:"magicLightMinutesRemaining"`
	CarriedTorchMinutesRemaining int               `json:"carriedTorchMinutesRemaining"`
	QuicknessWorldActorSkip      bool              `json:"quicknessWorldActorSkip"`
	TownMountedActorSkip         bool              `json:"townMountedActorSkip"`
	TownQuicknessActorSkip       bool              `json:"townQuicknessActorSkip"`
	OutdoorMountedActorSkip      bool              `json:"outdoorMountedActorSkip"`
	Amulet                       toggleSnapshotV12 `json:"amulet"`
	Crown                        toggleSnapshotV12 `json:"crown"`
	Badge                        toggleSnapshotV12 `json:"badge"`
	Drunk                        int               `json:"drunk"`
	Smashed                      int               `json:"smashed"`
}

func (snapshot saveSnapshotV15) validationView() SaveSnapshot {
	view := (saveSnapshotV14{
		Version: snapshot.Version, Party: snapshot.Party, Inventory: snapshot.Inventory,
		World: snapshot.World, Progression: snapshot.Progression, Wind: snapshot.Wind,
		Random: snapshot.Random, Continuation: snapshot.Continuation, Redux: snapshot.Redux,
		Dungeon: snapshot.Dungeon, PartyVehicle: snapshot.PartyVehicle, MapUnits: snapshot.MapUnits,
		ItemStacks: snapshot.ItemStacks, MapMutations: snapshot.MapMutations,
	}).validationView()
	e := snapshot.Effects
	view.Effects = EffectsSnapshot{
		DurationType: e.DurationType, DurationTurns: e.DurationTurns, DurationToggleOnly: e.DurationToggleOnly,
		MagicLightMinutesRemaining: e.MagicLightMinutesRemaining, CarriedTorchMinutesRemaining: e.CarriedTorchMinutesRemaining,
		QuicknessWorldActorSkip: e.QuicknessWorldActorSkip, TownMountedActorSkip: e.TownMountedActorSkip,
		TownQuicknessActorSkip: e.TownQuicknessActorSkip, OutdoorMountedActorSkip: e.OutdoorMountedActorSkip,
		Amulet: ToggleSnapshot(e.Amulet), Crown: ToggleSnapshot(e.Crown), Badge: ToggleSnapshot(e.Badge),
		Drunk: e.Drunk, Smashed: e.Smashed,
	}

	return view
}

func (snapshot saveSnapshotV15) promote() SaveSnapshot {
	promoted := snapshot.validationView()
	promoted.Version = CurrentVersion
	// v15 already uses the current cadence meaning; do not invert it again.
	promoted.promoteHistoricalJournal()

	return promoted
}

func (snapshot *SaveSnapshot) promoteHistoricalJournal() {
	snapshot.Presentation = journal.EmptyPresentation()
	// Historical lore is sorted by semantic key, so appended discoveries have a
	// stable order after existing defeats. Never infer a name/domain association,
	// Codex date, or lore from progress flags that did not record its provenance.
	for _, entry := range snapshot.Redux.PlayerKnowledge {
		if entry.Category == KnowledgeCategorySpell {
			continue
		}
		snapshot.Redux.QuestJournal = append(snapshot.Redux.QuestJournal, QuestJournalEventSnapshot{
			Sequence: uint64(len(snapshot.Redux.QuestJournal) + 1),
			Kind:     QuestJournalEventKnowledgeLearned, Subject: entry.Category + "." + entry.Key,
			OccurredOn: entry.DiscoveredOn, Source: entry.Source.Kind, SourceName: entry.Source.DisplayName,
			Acknowledged: true,
		})
	}
}
