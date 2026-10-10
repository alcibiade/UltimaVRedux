package save_snapshot

import "github.com/bradhannah/UltimaVRedux/internal/references"

// saveSnapshotV14 freezes v14 using the unchanged frozen v12/v13 nested shapes.
// Semantic chest support changed validation, not the map-unit wire fields.
type saveSnapshotV14 struct {
	Version      int                      `json:"version"`
	Party        partySnapshotV12         `json:"party"`
	Inventory    inventorySnapshotV12     `json:"inventory"`
	World        worldSnapshotV12         `json:"world"`
	Progression  progressionSnapshotV12   `json:"progression"`
	Effects      effectsSnapshotV12       `json:"effects"`
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

type continuationSnapshotV14 struct {
	ActivePlayerIndex                 int  `json:"activePlayerIndex"`
	IsCriminal                        bool `json:"isCriminal"`
	HMSCapeAutomaticWaitActorSkip     bool `json:"hmsCapeAutomaticWaitActorSkip"`
	BeggarKarmaCounter                int  `json:"beggarKarmaCounter"`
	OpeningCampaignEncounterReduction bool `json:"openingCampaignEncounterReduction"`
	CombatActionTimeCounter           int  `json:"combatActionTimeCounter"`
}

func (snapshot saveSnapshotV14) promote() SaveSnapshot {
	promoted := snapshot.validationView()
	promoted.Version = CurrentVersion
	promoted.promoteHistoricalSurfaceCadence()
	promoted.promoteHistoricalJournal()

	return promoted
}

func (snapshot saveSnapshotV14) validationView() SaveSnapshot {
	return SaveSnapshot{
		Version: snapshot.Version,
		Party:   promotePartyV12(snapshot.Party), Inventory: promoteInventoryV12(snapshot.Inventory),
		World: promoteWorldV12(snapshot.World), Progression: promoteProgressionV12(snapshot.Progression),
		Effects: promoteEffectsV12(snapshot.Effects),
		Wind:    WindSnapshot{Direction: snapshot.Wind.Direction, UpdateCounter: snapshot.Wind.UpdateCounter, NextTickMs: snapshot.Wind.NextTickMs},
		Random:  RandomSnapshot{Seed: snapshot.Random.Seed, SourceCalls: snapshot.Random.SourceCalls},
		Continuation: ContinuationSnapshot{
			ActivePlayerIndex:                 snapshot.Continuation.ActivePlayerIndex,
			IsCriminal:                        snapshot.Continuation.IsCriminal,
			HMSCapeAutomaticWaitActorSkip:     snapshot.Continuation.HMSCapeAutomaticWaitActorSkip,
			BeggarKarmaCounter:                snapshot.Continuation.BeggarKarmaCounter,
			OpeningCampaignEncounterReduction: snapshot.Continuation.OpeningCampaignEncounterReduction,
			CombatActionTimeCounter:           snapshot.Continuation.CombatActionTimeCounter,
		},
		Redux: ReduxSnapshot{
			DungeonDiscoveries:     promoteDungeonDiscoveriesV12(snapshot.Redux.DungeonDiscoveries),
			PlayerKnowledge:        promotePlayerKnowledgeV12(snapshot.Redux.PlayerKnowledge),
			QuestJournal:           promoteQuestJournalV12(snapshot.Redux.QuestJournal),
			LordBritishPassageOpen: snapshot.Redux.LordBritishPassageOpen,
		},
		Dungeon: promoteDungeonV12(snapshot.Dungeon), PartyVehicle: promoteVehicleV12(snapshot.PartyVehicle),
		MapUnits: promoteMapUnitsV12(snapshot.MapUnits), ItemStacks: promoteItemStacksV12(snapshot.ItemStacks),
		MapMutations: promoteMapMutationsV12(snapshot.MapMutations),
	}
}

func (snapshot *SaveSnapshot) promoteHistoricalSurfaceCadence() {
	oldSkipNext := snapshot.Effects.QuicknessWorldActorSkip
	// v12-v14 recorded neither riding phase nor separate context history.
	// Default unknown phases to their first qualifying gate (which skips).
	snapshot.Effects.TownMountedActorSkip = false
	snapshot.Effects.OutdoorMountedActorSkip = false
	snapshot.Effects.TownQuicknessActorSkip = false
	snapshot.Effects.QuicknessWorldActorSkip = false
	if snapshot.Effects.DurationType != 'Q' || snapshot.Effects.DurationTurns <= 0 {
		return
	}
	// The old bit meant "skip next"; v15 toggles before testing. Invert it
	// only for the active surface context whose next Quickness gate is known.
	switch references.Location(snapshot.World.Position.Location).GetMapType() {
	case references.SmallMapType:
		snapshot.Effects.TownQuicknessActorSkip = !oldSkipNext
	case references.LargeMapType:
		snapshot.Effects.QuicknessWorldActorSkip = !oldSkipNext
	case references.DungeonMapType, references.CombatMapType, references.CampingMapType:
		// Dungeon Quickness has its own already-persisted phase.
	}
}
