package save_snapshot

import "github.com/bradhannah/UltimaVRedux/internal/journal"

// saveSnapshotV22 freezes the flexible-survey shape before door continuation and
// the expanded atlas authority domain. Historical schemas and sources stay closed.
type saveSnapshotV22 struct {
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
	DungeonJournal   [8]dungeonObservationV22      `json:"dungeonJournal"`
	MoonstoneJournal [8]moonstoneObservationV22    `json:"moonstoneJournal"`
	AtlasLandmarks   []atlasLandmarkObservationV22 `json:"atlasLandmarks"`
	Dungeon          *DungeonSnapshot              `json:"dungeon"`
	PartyVehicle     *vehicleSnapshotV12           `json:"partyVehicle"`
	MapUnits         []mapUnitSnapshotV23          `json:"mapUnits"`
	ItemStacks       []ItemStackSnapshot           `json:"itemStacks"`
	MapMutations     []MapMutationSnapshot         `json:"mapMutations"`
}

type surveyedLocationV22 struct {
	Measured   bool                     `json:"measured"`
	Flexible   bool                     `json:"flexible"`
	Location   journal.SiteLocation     `json:"location"`
	RecordedAt journal.OptionalGameTime `json:"recordedAt"`
}

type dungeonSiteObservationV22 struct {
	Seen       bool                     `json:"seen"`
	Location   journal.SiteLocation     `json:"location"`
	Open       bool                     `json:"open"`
	ObservedAt journal.OptionalGameTime `json:"observedAt"`
	Survey     surveyedLocationV22      `json:"survey"`
}

type dungeonObservationV22 struct {
	Known    bool                         `json:"known"`
	WordUsed bool                         `json:"wordUsed"`
	Sites    [2]dungeonSiteObservationV22 `json:"sites"`
}

type atlasLandmarkObservationV22 struct {
	SiteKey      uint8                          `json:"siteKey"`
	Site         journal.SiteLocation           `json:"site"`
	Kind         string                         `json:"kind"`
	Name         string                         `json:"name"`
	Source       journal.AtlasObservationSource `json:"source"`
	NameSource   journal.AtlasObservationSource `json:"nameSource"`
	ObservedAt   journal.OptionalGameTime       `json:"observedAt"`
	NamedAt      journal.OptionalGameTime       `json:"namedAt"`
	Survey       surveyedLocationV22            `json:"survey"`
	SurveySource journal.AtlasObservationSource `json:"surveySource"`
}

type moonstoneObservationV22 struct {
	OriginalSiteRetired bool                     `json:"originalSiteRetired"`
	Known               bool                     `json:"known"`
	FirstSource         journal.MoonstoneSource  `json:"firstSource"`
	DiscoveredAt        journal.OptionalGameTime `json:"discoveredAt"`
	SiteKnown           bool                     `json:"siteKnown"`
	Site                journal.SiteLocation     `json:"site"`
	Placed              bool                     `json:"placed"`
	LastContext         journal.MoonstoneContext `json:"lastContext"`
	ObservedAt          journal.OptionalGameTime `json:"observedAt"`
	Survey              surveyedLocationV22      `json:"survey"`
	SurveyContext       journal.MoonstoneContext `json:"surveyContext"`
}

func (previous surveyedLocationV22) promote() journal.SurveyedLocation {
	return journal.SurveyedLocation{Measured: previous.Measured, Flexible: previous.Flexible,
		Location: previous.Location, RecordedAt: previous.RecordedAt}
}

func (previous saveSnapshotV22) validationView() SaveSnapshot {
	view := SaveSnapshot{
		Version: previous.Version, Party: previous.Party, Inventory: previous.Inventory,
		World: previous.World, Progression: previous.Progression, Effects: previous.Effects,
		Wind: previous.Wind, Random: previous.Random, Continuation: previous.Continuation,
		Redux: previous.Redux, Presentation: previous.Presentation, LastConversation: previous.LastConversation,
		Dungeon: previous.Dungeon, PartyVehicle: promoteVehicleV12(previous.PartyVehicle), MapUnits: promoteMapUnitsV23(previous.MapUnits),
		ItemStacks: previous.ItemStacks, MapMutations: previous.MapMutations,
	}
	for index, old := range previous.DungeonJournal {
		view.DungeonJournal[index].Known, view.DungeonJournal[index].WordUsed = old.Known, old.WordUsed
		for slot, site := range old.Sites {
			view.DungeonJournal[index].Sites[slot] = journal.DungeonSiteObservation{Seen: site.Seen,
				Location: site.Location, Open: site.Open, ObservedAt: site.ObservedAt, Survey: site.Survey.promote()}
		}
	}
	if previous.AtlasLandmarks != nil {
		view.AtlasLandmarks = make([]journal.AtlasLandmarkObservation, len(previous.AtlasLandmarks))
		for index, old := range previous.AtlasLandmarks {
			view.AtlasLandmarks[index] = journal.AtlasLandmarkObservation{SiteKey: old.SiteKey, Site: old.Site,
				Kind: old.Kind, Name: old.Name, Source: old.Source, NameSource: old.NameSource,
				ObservedAt: old.ObservedAt, NamedAt: old.NamedAt, Survey: old.Survey.promote(), SurveySource: old.SurveySource}
		}
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
