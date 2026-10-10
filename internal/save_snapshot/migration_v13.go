package save_snapshot

// saveSnapshotV13 freezes the version 13 top-level wire contract while reusing
// the already-frozen nested DTOs shared with version 12.
type saveSnapshotV13 struct {
	Version      int                      `json:"version"`
	Party        partySnapshotV12         `json:"party"`
	Inventory    inventorySnapshotV12     `json:"inventory"`
	World        worldSnapshotV12         `json:"world"`
	Progression  progressionSnapshotV12   `json:"progression"`
	Effects      effectsSnapshotV12       `json:"effects"`
	Wind         windSnapshotV12          `json:"wind"`
	Random       randomSnapshotV12        `json:"random"`
	Continuation continuationSnapshotV13  `json:"continuation"`
	Redux        reduxSnapshotV13         `json:"redux"`
	Dungeon      *dungeonSnapshotV12      `json:"dungeon"`
	PartyVehicle *vehicleSnapshotV12      `json:"partyVehicle"`
	MapUnits     []mapUnitSnapshotV12     `json:"mapUnits"`
	ItemStacks   []itemStackSnapshotV12   `json:"itemStacks"`
	MapMutations []mapMutationSnapshotV12 `json:"mapMutations"`
}

type reduxSnapshotV13 struct {
	DungeonDiscoveries     []dungeonDiscoverySnapshotV12  `json:"dungeonDiscoveries"`
	PlayerKnowledge        []playerKnowledgeSnapshotV12   `json:"playerKnowledge"`
	QuestJournal           []questJournalEventSnapshotV12 `json:"questJournal"`
	LordBritishPassageOpen bool                           `json:"lordBritishPassageOpen"`
}

type continuationSnapshotV13 struct {
	ActivePlayerIndex             int  `json:"activePlayerIndex"`
	IsCriminal                    bool `json:"isCriminal"`
	HMSCapeAutomaticWaitActorSkip bool `json:"hmsCapeAutomaticWaitActorSkip"`
	BeggarKarmaCounter            int  `json:"beggarKarmaCounter"`
}

func (snapshot saveSnapshotV13) promote() SaveSnapshot {
	promoted := snapshot.toCurrent()
	promoted.Version = CurrentVersion
	promoted.Continuation = ContinuationSnapshot{
		ActivePlayerIndex:                 snapshot.Continuation.ActivePlayerIndex,
		IsCriminal:                        snapshot.Continuation.IsCriminal,
		HMSCapeAutomaticWaitActorSkip:     snapshot.Continuation.HMSCapeAutomaticWaitActorSkip,
		BeggarKarmaCounter:                snapshot.Continuation.BeggarKarmaCounter,
		OpeningCampaignEncounterReduction: snapshot.World.Calendar.Year == 139 && snapshot.World.Calendar.Month == 4,
	}
	promoted.promoteHistoricalSurfaceCadence()
	promoted.promoteHistoricalJournal()

	return promoted
}

func (snapshot saveSnapshotV13) validationView() SaveSnapshot {
	promoted := snapshot.toCurrent()
	promoted.Continuation = ContinuationSnapshot{
		ActivePlayerIndex:             snapshot.Continuation.ActivePlayerIndex,
		IsCriminal:                    snapshot.Continuation.IsCriminal,
		HMSCapeAutomaticWaitActorSkip: snapshot.Continuation.HMSCapeAutomaticWaitActorSkip,
		BeggarKarmaCounter:            snapshot.Continuation.BeggarKarmaCounter,
	}

	return promoted
}

func (snapshot saveSnapshotV13) toCurrent() SaveSnapshot {
	return SaveSnapshot{
		Version:     snapshot.Version,
		Party:       promotePartyV12(snapshot.Party),
		Inventory:   promoteInventoryV12(snapshot.Inventory),
		World:       promoteWorldV12(snapshot.World),
		Progression: promoteProgressionV12(snapshot.Progression),
		Effects:     promoteEffectsV12(snapshot.Effects),
		Wind:        WindSnapshot{Direction: snapshot.Wind.Direction, UpdateCounter: snapshot.Wind.UpdateCounter, NextTickMs: snapshot.Wind.NextTickMs},
		Random:      RandomSnapshot{Seed: snapshot.Random.Seed, SourceCalls: snapshot.Random.SourceCalls},
		Redux: ReduxSnapshot{
			DungeonDiscoveries:     promoteDungeonDiscoveriesV12(snapshot.Redux.DungeonDiscoveries),
			PlayerKnowledge:        promotePlayerKnowledgeV12(snapshot.Redux.PlayerKnowledge),
			QuestJournal:           promoteQuestJournalV12(snapshot.Redux.QuestJournal),
			LordBritishPassageOpen: snapshot.Redux.LordBritishPassageOpen,
		},
		Dungeon:      promoteDungeonV12(snapshot.Dungeon),
		PartyVehicle: promoteVehicleV12(snapshot.PartyVehicle),
		MapUnits:     promoteMapUnitsV12(snapshot.MapUnits),
		ItemStacks:   promoteItemStacksV12(snapshot.ItemStacks),
		MapMutations: promoteMapMutationsV12(snapshot.MapMutations),
	}
}
