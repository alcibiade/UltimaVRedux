package save_snapshot

import "github.com/bradhannah/UltimaVRedux/internal/journal"

// saveSnapshotV21 freezes the accepted retirement-era shape. All nested surveys
// are concrete historical DTOs, never aliases of evolving runtime observations.
type saveSnapshotV21 struct {
	Version          int                           `json:"version"`
	Party            PartySnapshot                 `json:"party"`
	Inventory        InventorySnapshot             `json:"inventory"`
	World            WorldSnapshot                 `json:"world"`
	Progression      ProgressionSnapshot           `json:"progression"`
	Effects          EffectsSnapshot               `json:"effects"`
	Wind             WindSnapshot                  `json:"wind"`
	Random           RandomSnapshot                `json:"random"`
	Continuation     ContinuationSnapshot          `json:"continuation"`
	Redux            ReduxSnapshot                 `json:"redux"`
	Presentation     journal.Presentation          `json:"presentation"`
	LastConversation journal.Conversation          `json:"lastConversation"`
	DungeonJournal   [8]dungeonObservationV21      `json:"dungeonJournal"`
	MoonstoneJournal [8]moonstoneObservationV21    `json:"moonstoneJournal"`
	AtlasLandmarks   []atlasLandmarkObservationV21 `json:"atlasLandmarks"`
	Dungeon          *DungeonSnapshot              `json:"dungeon"`
	PartyVehicle     *vehicleSnapshotV12           `json:"partyVehicle"`
	MapUnits         []mapUnitSnapshotV23          `json:"mapUnits"`
	ItemStacks       []ItemStackSnapshot           `json:"itemStacks"`
	MapMutations     []MapMutationSnapshot         `json:"mapMutations"`
}

// surveyedLocationV21 is the identical v20/v21 original-policy wire shape.
type surveyedLocationV21 struct {
	Measured   bool                     `json:"measured"`
	Location   journal.SiteLocation     `json:"location"`
	RecordedAt journal.OptionalGameTime `json:"recordedAt"`
}

type dungeonSiteObservationV21 struct {
	Seen       bool                     `json:"seen"`
	Location   journal.SiteLocation     `json:"location"`
	Open       bool                     `json:"open"`
	ObservedAt journal.OptionalGameTime `json:"observedAt"`
	Survey     surveyedLocationV21      `json:"survey"`
}

type dungeonObservationV21 struct {
	Known    bool                         `json:"known"`
	WordUsed bool                         `json:"wordUsed"`
	Sites    [2]dungeonSiteObservationV21 `json:"sites"`
}

type atlasLandmarkObservationV21 struct {
	SiteKey      uint8                          `json:"siteKey"`
	Site         journal.SiteLocation           `json:"site"`
	Kind         string                         `json:"kind"`
	Name         string                         `json:"name"`
	Source       journal.AtlasObservationSource `json:"source"`
	NameSource   journal.AtlasObservationSource `json:"nameSource"`
	ObservedAt   journal.OptionalGameTime       `json:"observedAt"`
	NamedAt      journal.OptionalGameTime       `json:"namedAt"`
	Survey       surveyedLocationV21            `json:"survey"`
	SurveySource journal.AtlasObservationSource `json:"surveySource"`
}

type moonstoneObservationV21 struct {
	OriginalSiteRetired bool                     `json:"originalSiteRetired"`
	Known               bool                     `json:"known"`
	FirstSource         journal.MoonstoneSource  `json:"firstSource"`
	DiscoveredAt        journal.OptionalGameTime `json:"discoveredAt"`
	SiteKnown           bool                     `json:"siteKnown"`
	Site                journal.SiteLocation     `json:"site"`
	Placed              bool                     `json:"placed"`
	LastContext         journal.MoonstoneContext `json:"lastContext"`
	ObservedAt          journal.OptionalGameTime `json:"observedAt"`
	Survey              surveyedLocationV21      `json:"survey"`
	SurveyContext       journal.MoonstoneContext `json:"surveyContext"`
}

func (previous surveyedLocationV21) promote() journal.SurveyedLocation {
	// Historical measurements were acquired under the original nighttime rule.
	// A current user preference cannot grant them flexible provenance.
	return journal.SurveyedLocation{Measured: previous.Measured, Flexible: false,
		Location: previous.Location, RecordedAt: previous.RecordedAt}
}

func promoteDungeonJournalV21(previous [8]dungeonObservationV21) [8]journal.DungeonObservation {
	var current [8]journal.DungeonObservation
	for index, old := range previous {
		current[index].Known, current[index].WordUsed = old.Known, old.WordUsed
		for slot, site := range old.Sites {
			current[index].Sites[slot] = journal.DungeonSiteObservation{Seen: site.Seen,
				Location: site.Location, Open: site.Open, ObservedAt: site.ObservedAt, Survey: site.Survey.promote()}
		}
	}

	return current
}

func promoteAtlasLandmarksV21(previous []atlasLandmarkObservationV21) []journal.AtlasLandmarkObservation {
	if previous == nil {
		return nil
	}
	current := make([]journal.AtlasLandmarkObservation, len(previous))
	for index, old := range previous {
		current[index] = journal.AtlasLandmarkObservation{SiteKey: old.SiteKey, Site: old.Site,
			Kind: old.Kind, Name: old.Name, Source: old.Source, NameSource: old.NameSource,
			ObservedAt: old.ObservedAt, NamedAt: old.NamedAt, Survey: old.Survey.promote(), SurveySource: old.SurveySource}
	}

	return current
}

func (previous saveSnapshotV21) validationView() SaveSnapshot {
	view := SaveSnapshot{
		Version: previous.Version, Party: previous.Party, Inventory: previous.Inventory,
		World: previous.World, Progression: previous.Progression, Effects: previous.Effects,
		Wind: previous.Wind, Random: previous.Random, Continuation: previous.Continuation,
		Redux: previous.Redux, Presentation: previous.Presentation, LastConversation: previous.LastConversation,
		DungeonJournal: promoteDungeonJournalV21(previous.DungeonJournal), AtlasLandmarks: promoteAtlasLandmarksV21(previous.AtlasLandmarks),
		Dungeon: previous.Dungeon, PartyVehicle: promoteVehicleV12(previous.PartyVehicle), MapUnits: promoteMapUnitsV23(previous.MapUnits),
		ItemStacks: previous.ItemStacks, MapMutations: previous.MapMutations,
	}
	for phase, old := range previous.MoonstoneJournal {
		view.MoonstoneJournal[phase] = journal.MoonstoneObservation{
			OriginalSiteRetired: old.OriginalSiteRetired, Known: old.Known, FirstSource: old.FirstSource, DiscoveredAt: old.DiscoveredAt,
			SiteKnown: old.SiteKnown, Site: old.Site, Placed: old.Placed,
			LastContext: old.LastContext, ObservedAt: old.ObservedAt,
			Survey: old.Survey.promote(), SurveyContext: old.SurveyContext,
		}
	}

	return view
}
