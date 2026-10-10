package save_snapshot

import (
	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// saveSnapshotV20 freezes the accepted v20 wire shape before the retirement
// field was introduced. v17-v19 continue to use their separate historical DTO.
type saveSnapshotV20 struct {
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
	MoonstoneJournal [8]moonstoneObservationV20    `json:"moonstoneJournal"`
	AtlasLandmarks   []atlasLandmarkObservationV21 `json:"atlasLandmarks"`
	Dungeon          *DungeonSnapshot              `json:"dungeon"`
	PartyVehicle     *vehicleSnapshotV12           `json:"partyVehicle"`
	MapUnits         []mapUnitSnapshotV23          `json:"mapUnits"`
	ItemStacks       []ItemStackSnapshot           `json:"itemStacks"`
	MapMutations     []MapMutationSnapshot         `json:"mapMutations"`
}

type moonstoneObservationV20 struct {
	Known         bool                     `json:"known"`
	FirstSource   journal.MoonstoneSource  `json:"firstSource"`
	DiscoveredAt  journal.OptionalGameTime `json:"discoveredAt"`
	SiteKnown     bool                     `json:"siteKnown"`
	Site          journal.SiteLocation     `json:"site"`
	Placed        bool                     `json:"placed"`
	LastContext   journal.MoonstoneContext `json:"lastContext"`
	ObservedAt    journal.OptionalGameTime `json:"observedAt"`
	Survey        surveyedLocationV21      `json:"survey"`
	SurveyContext journal.MoonstoneContext `json:"surveyContext"`
}

func (previous saveSnapshotV20) validationView() SaveSnapshot {
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
			Known: old.Known, FirstSource: old.FirstSource, DiscoveredAt: old.DiscoveredAt,
			SiteKnown: old.SiteKnown, Site: old.Site, Placed: old.Placed,
			LastContext: old.LastContext, ObservedAt: old.ObservedAt,
			Survey: old.Survey.promote(), SurveyContext: old.SurveyContext,
		}
	}

	return view
}

// retireHistoricalMoonstoneOrigins uses only persisted personal evidence or
// inventory. Neither live placement bytes nor current Avatar position prove a visit.
func (snapshot *SaveSnapshot) retireHistoricalMoonstoneOrigins() {
	origins := references.ChartedMoonstoneOrigins()
	for phase := range snapshot.MoonstoneJournal {
		observation := &snapshot.MoonstoneJournal[phase]
		if !observation.Known {
			continue
		}
		origin := origins[phase]
		canonical := journal.SiteLocation{Map: origin.Location.Location, Floor: origin.Location.Floor,
			X: int(origin.Location.Position.X), Y: int(origin.Location.Position.Y)}
		observation.OriginalSiteRetired = snapshot.Inventory.Moonstones[phase] > 0 ||
			observation.FirstSource == journal.MoonstoneSourcePickup || observation.FirstSource == journal.MoonstoneSourceBurial ||
			observation.LastContext.Source == journal.MoonstoneSourcePickup || observation.LastContext.Source == journal.MoonstoneSourceBurial ||
			observation.SurveyContext.Source == journal.MoonstoneSourcePickup || observation.SurveyContext.Source == journal.MoonstoneSourceBurial ||
			observation.SiteKnown && observation.Site != canonical ||
			observation.Survey.Measured && observation.Survey.Location != observation.Site
		if snapshot.Inventory.Moonstones[phase] > 0 || !observation.Placed || !observation.SiteKnown ||
			observation.Survey.Location != observation.Site || observation.SurveyContext.Source == journal.MoonstoneSourcePickup ||
			observation.LastContext.Source == journal.MoonstoneSourceBurial && observation.Survey.RecordedAt != observation.ObservedAt {
			// Coordinate equality cannot turn a pickup measurement or a survey
			// predating the latest burial into a measurement of that placement.
			observation.Survey, observation.SurveyContext = journal.SurveyedLocation{}, journal.MoonstoneContext{}
		}
	}
}
