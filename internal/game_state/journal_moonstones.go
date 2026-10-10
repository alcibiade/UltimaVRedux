package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func moonstoneJournalSite(location references.PlayerLocation) journal.SiteLocation {
	return journal.SiteLocation{Map: location.Location, Floor: location.Floor,
		X: int(location.Position.X), Y: int(location.Position.Y)}
}

func (g *GameState) localMoonstoneSite(position references.Position) journal.SiteLocation {
	location := g.MapState.PlayerLocation
	location.Position = position

	return moonstoneJournalSite(location)
}

// CommitMoonstoneBurial atomically owns inventory, physical placement and earned
// knowledge. The existing scene workflow still owns its normal time/feedback.
func (g *GameState) CommitMoonstoneBurial(moonstone references.Moonstone) bool {
	phase := int(moonstone)
	if phase < 0 || phase >= len(g.moonstoneJournal) || !g.PartyState.Inventory.Moonstones.HasSome(moonstone) {
		return false
	}
	location := g.MapState.PlayerLocation
	mapType := location.Location.GetMapType()
	if mapType != references.LargeMapType && mapType != references.SmallMapType {
		return false
	}
	tile := g.GetLayeredMapByCurrentLocation().GetTopTile(&location.Position)
	if tile == nil || !tile.IsValidForMoonstoneBurial() {
		return false
	}
	g.PartyState.Inventory.Moonstones.DecrementByOne(moonstone)
	//nolint:gosec // Coordinates are map-bounded; signed floors use legacy bytes.
	g.GetMoonstoneSystem().BuryMoonstone(phase, uint8(location.Position.X), uint8(location.Position.Y), uint8(location.Location), uint8(location.Floor))
	g.observeMoonstone(phase, moonstoneJournalSite(location), journal.MoonstoneSourceBurial)

	return true
}

// observeMoonstone runs only after an identified action commits successfully.
// Travel calls it immediately after TransitionToLocation, before renderer hooks.
func (g *GameState) observeMoonstone(phase int, site journal.SiteLocation, source journal.MoonstoneSource) {
	if phase < 0 || phase >= len(g.moonstoneJournal) || source == journal.MoonstoneSourceNone ||
		source == journal.MoonstoneSourcePossession {
		panic("invalid successful moonstone observation")
	}
	context := g.CurrentSextantContext()
	at := journal.OptionalGameTime{Known: true, Time: journal.GameTime{
		Year: int(g.DateTime.Year), Month: int(g.DateTime.Month), Day: int(g.DateTime.Day),
		Hour: int(g.DateTime.Hour), Minute: int(g.DateTime.Minute),
	}}
	evidence := journal.MoonstoneContext{Source: source,
		Observer: moonstoneJournalSite(g.MapState.PlayerLocation), HasSextant: context.HasSextant}
	observation := &g.moonstoneJournal[phase]
	if !observation.Known {
		observation.Known, observation.FirstSource, observation.DiscoveredAt = true, source, at
	}
	origin := moonstoneJournalSite(references.ChartedMoonstoneOrigins()[phase].Location)
	if source == journal.MoonstoneSourcePickup || source == journal.MoonstoneSourceBurial || site != origin {
		observation.OriginalSiteRetired = true
	}
	// A survey is authority for one placement, never a trail of past locations.
	if source == journal.MoonstoneSourcePickup || source == journal.MoonstoneSourceBurial || !observation.Placed || observation.Site != site {
		observation.Survey, observation.SurveyContext = journal.SurveyedLocation{}, journal.MoonstoneContext{}
	}
	observation.SiteKnown, observation.Site = true, site
	observation.Placed = source != journal.MoonstoneSourcePickup
	observation.LastContext, observation.ObservedAt = evidence, at
	// REDUX DEVIATION: original item-use rules / original spell-effect rules explicitly report the
	// Avatar's position. Redux automatically surveys a successfully identified
	// exact surface target for the journal under the sampled instrument policy,
	// without an extra command, resource cost, time advance or RNG draw.
	// See ref:MOON-0008.
	if observation.Placed && site.Map == references.BritanniaUnderworld && site.Floor == 0 &&
		site.Map == context.Location && site.Floor == context.Floor &&
		EvaluateSextantEligibility(context) == SextantReady {
		observation.Survey = journal.SurveyedLocation{Measured: true, Flexible: context.Flexible, Location: site, RecordedAt: at}
		observation.SurveyContext = evidence
	}
}

// moonstoneSourceAt identifies a sole physical source by its full current site,
// never by the supplied chart, remembered placement or travel destination phase.
func (g *GameState) moonstoneSourceAt(position references.Position) (int, bool) {
	site := g.localMoonstoneSite(position)
	matched := -1
	for phase, stone := range g.moonstoneSystem.Moonstones {
		if !stone.CanCreateMoongate() || stone.Map != uint8(site.Map) ||
			stone.Level != uint8(site.Floor) || int(stone.X) != site.X || int(stone.Y) != site.Y { //nolint:gosec // Signed floors encode as legacy bytes.
			continue
		}
		if matched >= 0 {
			return 0, false
		}
		matched = phase
	}
	if matched < 0 {
		return 0, false
	}

	return matched, !g.PartyState.Inventory.Moonstones.HasSome(references.Moonstone(matched))
}

// observeMoongateLook records only the source actually described by live Look.
// Redux policy: an unambiguous active source identifies its moonstone phase;
// the existing observation authority owns sextant eligibility and retirement.
func (g *GameState) observeMoongateLook(phase int, position references.Position) {
	before := g.moonstoneJournal[phase]
	g.observeMoonstone(phase, g.localMoonstoneSite(position), journal.MoonstoneSourceLook)
	after := g.moonstoneJournal[phase]
	if after.Site.Map != references.BritanniaUnderworld || after.Site.Floor != 0 {
		return
	}
	change := journal.AtlasChange{
		EntryKey: references.AtlasMoonstoneChartKey(phase), Kind: "moonstone", Name: datetime.MoonPhase(phase).String(),
		Discovered: !before.SiteKnown,
		Surveyed:   after.Survey.Measured && (!before.Survey.Measured || before.Survey.Location != after.Survey.Location),
		Measured:   after.Survey.Measured,
	}
	if after.OriginalSiteRetired {
		change.Name += " (last seen)"
	}
	if change.Discovered || change.Surveyed {
		g.SystemCallbacks.UI.AtlasUpdated(change)
	}
}

func (g *GameState) journalMoonstones() [8]journal.MoonstoneEntry {
	var entries [8]journal.MoonstoneEntry
	for phase, origin := range references.ChartedMoonstoneOrigins() {
		observed := g.moonstoneJournal[phase]
		carried := g.PartyState.Inventory.Moonstones.HasSome(references.Moonstone(phase))
		entry := journal.MoonstoneEntry{Phase: datetime.MoonPhase(phase), ChartedOrigin: origin.Label,
			Known: observed.Known || carried, Carried: carried, OriginalSiteRetired: observed.OriginalSiteRetired || carried}
		if entry.OriginalSiteRetired {
			entry.ChartedOrigin = ""
		}
		if observed.Known {
			entry.FirstSource, entry.DiscoveredAt, entry.ObservedAt = observed.FirstSource, observed.DiscoveredAt, observed.ObservedAt
			entry.Placed = observed.SiteKnown && observed.Placed && !carried
			if entry.Placed && observed.Survey.Measured && observed.Site == observed.Survey.Location {
				entry.SurveySource = observed.SurveyContext.Source
				entry.LastObservedSurvey = observed.Survey
			}
		} else if carried {
			entry.FirstSource = journal.MoonstoneSourcePossession
		}
		entries[phase] = entry
	}

	return entries
}
