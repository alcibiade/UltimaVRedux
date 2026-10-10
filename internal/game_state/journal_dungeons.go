package game_state

import (
	"fmt"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func (g *GameState) journalDungeonIdentityKnown(dungeon references.Location) bool {
	if g.dungeonJournal[DungeonToIndexMap[dungeon]].Known {
		return true
	}
	if dungeon == references.Shame || dungeon == references.Doom {
		return false
	}
	_, learned := g.WordOfPowerKnowledge(strings.ToLower(dungeon.String()))

	return learned
}

// dungeonObservationSample retains the accepted source context across Enter's fade.
type dungeonObservationSample struct {
	sextant SextantContext
	at      journal.OptionalGameTime
}

func (g *GameState) sampleDungeonObservation() dungeonObservationSample {
	return dungeonObservationSample{
		sextant: g.CurrentSextantContext(),
		at: journal.OptionalGameTime{Known: true, Time: journal.GameTime{
			Year: int(g.DateTime.Year), Month: int(g.DateTime.Month), Day: int(g.DateTime.Day),
			Hour: int(g.DateTime.Hour), Minute: int(g.DateTime.Minute),
		}},
	}
}

func (g *GameState) observeDungeonEntrance(
	dungeon references.Location,
	position references.Position,
	sample dungeonObservationSample,
	wordUsed bool,
) {
	index, ok := DungeonToIndexMap[dungeon]
	if !ok || sample.sextant.Location != references.BritanniaUnderworld ||
		!DungeonEntranceExistsOnFloor(dungeon, sample.sextant.Floor) ||
		position != g.GetDungeonEntranceCoordinates()[dungeon] {
		panic("invalid observed dungeon exterior site")
	}
	siteIndex := journal.DungeonSurfaceSite
	if sample.sextant.Floor == references.FloorNumber(references.UNDERWORLD) {
		siteIndex = journal.DungeonUnderworldSite
	}
	observation := &g.dungeonJournal[index]
	location := journal.SiteLocation{
		Map: sample.sextant.Location, Floor: sample.sextant.Floor, X: int(position.X), Y: int(position.Y),
	}
	site := &observation.Sites[siteIndex]
	if site.Location != location {
		// Historical private coordinates need not have been canonical. A real
		// encounter replaces that evidence; it cannot transfer a survey to this site.
		*site = journal.DungeonSiteObservation{}
	}
	before := *site
	site.Seen, site.Location, site.Open, site.ObservedAt = true, location, g.IsDungeonOpen(dungeon), sample.at
	// REDUX DEVIATION: original item-use rules and original spell-effect rules report only the
	// Avatar's current position on explicit Use. Redux remembers an encountered
	// dungeon target and automatically surveys it with the same instrument gates
	// for navigation convenience, without time, RNG or resource costs.
	// See ref:ASTRO-0006.
	if !site.Survey.Measured && EvaluateSextantEligibility(sample.sextant) == SextantReady {
		site.Survey = journal.SurveyedLocation{Measured: true, Flexible: sample.sextant.Flexible, Location: location, RecordedAt: sample.at}
	}
	observation.Known, observation.WordUsed = true, observation.WordUsed || wordUsed
	if siteIndex == journal.DungeonSurfaceSite && dungeon != references.Doom &&
		(!before.Seen || !before.Survey.Measured && site.Survey.Measured) {
		g.SystemCallbacks.UI.AtlasUpdated(journal.AtlasChange{EntryKey: "dungeon-" + dungeon.String(),
			Name: dungeon.String(), Kind: "dungeon", Discovered: !before.Seen,
			Surveyed: !before.Survey.Measured && site.Survey.Measured, Measured: site.Survey.Measured})
	}
}

func dungeonEntranceJournalLocation(site journal.DungeonSiteEntry) string {
	if !site.Survey.Measured {
		return site.World + " Not measured"
	}

	return fmt.Sprintf(
		"%s %s, %s",
		site.World,
		encodeSextantCoordinate(references.Coordinate(site.Survey.Location.Y)),
		encodeSextantCoordinate(references.Coordinate(site.Survey.Location.X)),
	)
}

func projectDungeonSites(entry *journal.DungeonEntry, observation journal.DungeonObservation) {
	var locations, entrances []string
	for index, site := range observation.Sites {
		if !site.Seen {
			continue
		}
		world := "Britannia"
		if index == journal.DungeonUnderworldSite {
			world = "Underworld"
		}
		public := journal.DungeonSiteEntry{Inspected: true, World: world, Open: site.Open, ObservedAt: site.ObservedAt}
		if site.Survey.Measured && site.Survey.Location == site.Location && index == journal.DungeonSurfaceSite {
			public.Survey = site.Survey
		}
		entry.Sites[index] = public
		entry.Entrance = "Sealed"
		if site.Open {
			entry.Entrance = "Open"
		}
		entrances = append(entrances, world+": "+entry.Entrance)
		locations = append(locations, dungeonEntranceJournalLocation(public))
	}
	if len(entrances) > 1 {
		entry.Entrance = strings.Join(entrances, "; ")
	}
	entry.Location = strings.Join(locations, "; ")
}

func (g *GameState) journalDungeons() []journal.DungeonEntry {
	var entries []journal.DungeonEntry
	for index, candidate := range canonicalDungeonWords {
		observed := g.dungeonJournal[index]
		knowledge, learned := g.WordOfPowerKnowledge(strings.ToLower(candidate.dungeon.String()))
		// Goeth's question_01/drat answer establishes Destard before the reveal.
		// Shame and Doom disclosures still lack an established association.
		explicit := learned && candidate.dungeon != references.Shame && candidate.dungeon != references.Doom
		if !observed.Known && !explicit {
			continue
		}
		entry := journal.DungeonEntry{Name: candidate.dungeon.String(), Entrance: "Not inspected", Location: "", Word: "", Source: ""}
		projectDungeonSites(&entry, observed)
		if explicit || observed.WordUsed {
			entry.Word = candidate.word
			if observed.WordUsed {
				entry.Source = "Successful use at the entrance"
			} else {
				entry.Source = knowledge.Source.DisplayName
				if candidate.dungeon == references.Destard {
					entry.Word = "AIPONI"
					entry.WordIsClue = true
				}
			}
		}
		entries = append(entries, entry)
	}

	return entries
}
