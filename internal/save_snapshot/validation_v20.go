package save_snapshot

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func validateChestAssessments(snapshot SaveSnapshot, add func(string, string, string)) {
	for index, unit := range snapshot.MapUnits {
		path := fmt.Sprintf("mapUnits[%d].surfaceChestAssessment", index)
		value := unit.SurfaceChestAssessment
		// Check the wire integer before narrowing to the uint8 runtime enum.
		if value < 0 || value > 3 {
			add(path, "out_of_range", "chest assessment must be between 0 and 3")

			continue
		}
		if unit.SurfaceChestNumber == nil || (unit.Kind != MapUnitKindChest && unit.Kind != MapUnitKindNPC) {
			if value != 0 {
				add(path, "inconsistent", "non-chest units must be uninspected")
			}

			continue
		}
		if !references.ChestAssessment(value).ValidForChest(*unit.SurfaceChestNumber&0x80 != 0) {
			add(path, "inconsistent", "assessment contradicts physical chest state")
		}
	}
	if snapshot.Dungeon == nil {
		return
	}
	for index, value := range snapshot.Dungeon.ChestAssessments {
		path := fmt.Sprintf("dungeon.chestAssessments[%d]", index)
		if value < 0 || value > 3 {
			add(path, "out_of_range", "chest assessment must be between 0 and 3")

			continue
		}
		if index >= len(snapshot.Dungeon.Cells) {
			continue
		}
		raw := snapshot.Dungeon.Cells[index]
		if raw&0xf0 != 0x40 {
			if value != 0 {
				add(path, "inconsistent", "only closed chest cells may retain an assessment")
			}
		} else if !references.ChestAssessment(value).ValidForChest(raw&7 != 0) {
			add(path, "inconsistent", "assessment contradicts physical chest cell")
		}
	}
}

func validateWorldJournal(snapshot SaveSnapshot, add func(string, string, string)) {
	validateAtlasLandmarks(snapshot.AtlasLandmarks, add)
	for index, observation := range snapshot.DungeonJournal {
		path := fmt.Sprintf("dungeonJournal[%d]", index)
		seen := false
		for slot, site := range observation.Sites {
			sitePath := fmt.Sprintf("%s.sites[%d]", path, slot)
			if !site.Seen {
				if site != (journal.DungeonSiteObservation{}) {
					add(sitePath, "inconsistent", "unseen site must be entirely zero")
				}

				continue
			}
			seen = true
			floor := references.FloorNumber(0)
			if slot == journal.DungeonUnderworldSite {
				floor = -1
			}
			if !observation.Known || site.Location.Map != references.BritanniaUnderworld || site.Location.Floor != floor ||
				site.Location.X < 0 || site.Location.X > 255 || site.Location.Y < 0 || site.Location.Y > 255 {
				add(sitePath, "invalid", "observed site requires a known dungeon and coordinates on its assigned world plane")
			}
			validateOptionalGameTime(sitePath+".observedAt", site.ObservedAt, add)
			validateSurvey(sitePath+".survey", site.Survey, add)
			if site.Survey.Measured {
				if !site.ObservedAt.Known {
					add(sitePath+".observedAt", "required", "a surveyed site requires an observation date")
				}
				if slot != journal.DungeonSurfaceSite || index == 7 || site.Survey.Location != site.Location {
					add(sitePath+".survey", "inconsistent", "only an ordinary surface entrance can have a matching survey")
				}
				// The frozen survey domain uses the original town-coordinate
				// entrances. Unmeasured historical private sites need not match.
				entrances := [8][2]int{{240, 73}, {91, 67}, {72, 168}, {126, 20}, {156, 27}, {58, 102}, {239, 240}, {128, 128}}
				if site.Location.X != entrances[index][0] || site.Location.Y != entrances[index][1] {
					add(sitePath+".survey", "inconsistent", "survey must identify the canonical dungeon entrance")
				}
				validateTimeOrder(sitePath+".survey.recordedAt", site.Survey.RecordedAt, site.ObservedAt, add)
			}
		}
		if observation.WordUsed && !seen {
			add(path, "inconsistent", "a used word requires an observed entrance")
		}
	}
	for phase, observation := range snapshot.MoonstoneJournal {
		path := fmt.Sprintf("moonstoneJournal[%d]", phase)
		validateMoonstoneObservation(path, observation, add)
		validateCurrentMoonstonePlacement(path, observation, snapshot.Inventory.Moonstones[phase] > 0, add)
	}
}

func validateCurrentMoonstonePlacement(path string, observation journal.MoonstoneObservation, carried bool, add func(string, string, string)) {
	if observation.Known && (carried || observation.FirstSource == journal.MoonstoneSourcePickup ||
		observation.FirstSource == journal.MoonstoneSourceBurial || observation.LastContext.Source == journal.MoonstoneSourcePickup ||
		observation.LastContext.Source == journal.MoonstoneSourceBurial) && !observation.OriginalSiteRetired {
		add(path+".originalSiteRetired", "inconsistent", "handling evidence must retire the original site")
	}
	if observation.Survey.Measured && (carried || !observation.Placed || !observation.SiteKnown || observation.Survey.Location != observation.Site ||
		observation.SurveyContext.Source == journal.MoonstoneSourcePickup ||
		observation.LastContext.Source == journal.MoonstoneSourceBurial && observation.Survey.RecordedAt != observation.ObservedAt) {
		add(path+".survey", "inconsistent", "survey belongs only to the current observed placement, never a carried stone or historical site")
	}
}

func validateAtlasLandmarks(records []journal.AtlasLandmarkObservation, add func(string, string, string)) {
	if err := journal.ValidateAtlasLandmarks(records); err != nil {
		add("atlasLandmarks", "invalid", err.Error())
	}
	for index, record := range records {
		path := fmt.Sprintf("atlasLandmarks[%d]", index)
		validateOptionalGameTime(path+".observedAt", record.ObservedAt, add)
		validateOptionalGameTime(path+".namedAt", record.NamedAt, add)
		validateTimeOrder(path+".namedAt", record.NamedAt, record.ObservedAt, add)
		receipt := record.SurveySource == journal.AtlasSourceSextantReceipt
		validateSurveyPolicy(path+".survey", record.Survey, receipt, add)
		validateTimeOrder(path+".survey.recordedAt", record.Survey.RecordedAt, record.ObservedAt, add)
		if record.SiteKey >= 101 && record.SiteKey <= 107 {
			position := references.AtlasShrinePositions()[record.SiteKey-101]
			if record.Site.X != int(position.X) || record.Site.Y != int(position.Y) {
				add(path+".site", "inconsistent", "shrine observation must identify its canonical exterior")
			}
		} else if record.Kind == "shrine" || record.Kind == "ruins" {
			add(path+".kind", "inconsistent", "ordinary landmark cannot be a shrine or shrine ruins")
		}
		if (record.Source == journal.AtlasSourceEnter || record.Source == journal.AtlasSourceShrineIntroduction) && record.Name == "" {
			add(path+".name", "required", "accepted entry must retain its presented name")
		}
	}
}

func validateOptionalGameTime(path string, value journal.OptionalGameTime, add func(string, string, string)) {
	if !value.Known {
		if value.Time != (journal.GameTime{}) {
			add(path, "inconsistent", "absent timestamp must be zero")
		}

		return
	}
	time := value.Time
	if time.Year < 1 || time.Year > 65535 || time.Month < 1 || time.Month > 13 || time.Day < 1 || time.Day > 28 ||
		time.Hour < 0 || time.Hour > 23 || time.Minute < 0 || time.Minute > 59 {
		add(path, "out_of_range", "timestamp is outside Britannian calendar bounds")
	}
}

func validateTimeOrder(path string, earlier, later journal.OptionalGameTime, add func(string, string, string)) {
	if !earlier.Known || !later.Known {
		return
	}
	a, b := earlier.Time, later.Time
	left := [5]int{a.Year, a.Month, a.Day, a.Hour, a.Minute}
	right := [5]int{b.Year, b.Month, b.Day, b.Hour, b.Minute}
	for index, value := range left {
		if value < right[index] {
			return
		}
		if value > right[index] {
			add(path, "inconsistent", "timestamps are not chronological")

			return
		}
	}
}

func validateSurvey(path string, survey journal.SurveyedLocation, add func(string, string, string)) {
	validateSurveyPolicy(path, survey, false, add)
}

func validateSurveyPolicy(path string, survey journal.SurveyedLocation, receipt bool, add func(string, string, string)) {
	if !survey.Measured {
		if survey != (journal.SurveyedLocation{}) {
			add(path, "inconsistent", "absent survey must be entirely zero")
		}

		return
	}
	validateJournalSite(path+".location", survey.Location, add)
	validateOptionalGameTime(path+".recordedAt", survey.RecordedAt, add)
	if survey.Location.Map != references.BritanniaUnderworld || survey.Location.Floor != 0 ||
		!survey.RecordedAt.Known {
		add(path, "inconsistent", "survey requires Britannia surface and a measurement date")
	}
	if !receipt && !survey.Flexible && survey.RecordedAt.Time.Hour >= 6 && survey.RecordedAt.Time.Hour < 19 {
		add(path, "inconsistent", "original-policy survey requires a nighttime measurement")
	}
}

func validateJournalSite(path string, site journal.SiteLocation, add func(string, string, string)) {
	limit := 0
	validFloor := false
	switch {
	case site.Map == references.BritanniaUnderworld:
		limit, validFloor = 256, site.Floor == 0 || site.Floor == -1
	case site.Map >= references.Moonglow && site.Map <= references.SerpentsHold:
		limit, validFloor = 32, site.Floor >= -1 && site.Floor <= 3
	case site.Map >= references.Deceit && site.Map <= references.Doom:
		limit, validFloor = 8, site.Floor >= 0 && site.Floor < 8
	}
	if !validFloor || site.X < 0 || site.Y < 0 || site.X >= limit || site.Y >= limit {
		add(path, "out_of_range", "site must be on a persistent map within its coordinate bounds")
	}
}

func isMoonstoneAction(source journal.MoonstoneSource) bool {
	switch source {
	case journal.MoonstoneSourceGate, journal.MoonstoneSourceSpell, journal.MoonstoneSourceSearch,
		journal.MoonstoneSourcePickup, journal.MoonstoneSourceBurial, journal.MoonstoneSourceLook:
		return true
	case journal.MoonstoneSourceNone, journal.MoonstoneSourcePossession:
		return false
	default:
		return false
	}
}

func validateMoonstoneContext(path string, context journal.MoonstoneContext, site journal.SiteLocation, add func(string, string, string)) {
	validateJournalSite(path+".observer", context.Observer, add)
	if !isMoonstoneAction(context.Source) || context.Observer.Map != site.Map || context.Observer.Floor != site.Floor {
		add(path, "inconsistent", "successful observation requires an action and observer on the same map and floor")
	}
	if (context.Source == journal.MoonstoneSourceGate || context.Source == journal.MoonstoneSourceSpell ||
		context.Source == journal.MoonstoneSourceBurial) && context.Observer != site {
		add(path, "inconsistent", "arrival or burial requires observer at the target site")
	}
}

func validateMoonstoneObservation(path string, observation journal.MoonstoneObservation, add func(string, string, string)) {
	if !observation.Known {
		if observation != (journal.MoonstoneObservation{}) {
			add(path, "inconsistent", "unknown moonstone must be entirely zero")
		}

		return
	}
	if !isMoonstoneAction(observation.FirstSource) && observation.FirstSource != journal.MoonstoneSourcePossession {
		add(path+".firstSource", "unsupported", "first evidence must identify a successful action or possession")
	}
	validateOptionalGameTime(path+".discoveredAt", observation.DiscoveredAt, add)
	if observation.FirstSource != journal.MoonstoneSourcePossession && !observation.DiscoveredAt.Known {
		add(path+".discoveredAt", "required", "action evidence requires a discovery date")
	}
	if observation.FirstSource == journal.MoonstoneSourcePossession && observation.DiscoveredAt != (journal.OptionalGameTime{}) {
		add(path+".discoveredAt", "inconsistent", "possession backfill cannot invent a discovery date")
	}
	if !observation.SiteKnown {
		if observation.Site != (journal.SiteLocation{}) || observation.Placed || observation.LastContext != (journal.MoonstoneContext{}) ||
			observation.ObservedAt != (journal.OptionalGameTime{}) || observation.Survey != (journal.SurveyedLocation{}) ||
			observation.SurveyContext != (journal.MoonstoneContext{}) || observation.FirstSource != journal.MoonstoneSourcePossession {
			add(path, "inconsistent", "site-unknown possession cannot contain site, date, context or survey evidence")
		}

		return
	}
	validateJournalSite(path+".site", observation.Site, add)
	validateMoonstoneContext(path+".lastContext", observation.LastContext, observation.Site, add)
	validateOptionalGameTime(path+".observedAt", observation.ObservedAt, add)
	if !observation.ObservedAt.Known {
		add(path+".observedAt", "required", "site evidence requires an observation date")
	}
	validateTimeOrder(path+".observedAt", observation.DiscoveredAt, observation.ObservedAt, add)
	if observation.Placed == (observation.LastContext.Source == journal.MoonstoneSourcePickup) {
		add(path+".placed", "inconsistent", "pickup retires placement; other successful observations record placement")
	}
	validateSurvey(path+".survey", observation.Survey, add)
	if !observation.Survey.Measured {
		if observation.SurveyContext != (journal.MoonstoneContext{}) {
			add(path+".surveyContext", "inconsistent", "absent survey context must be zero")
		}

		return
	}
	validateMoonstoneContext(path+".surveyContext", observation.SurveyContext, observation.Survey.Location, add)
	if !observation.SurveyContext.HasSextant {
		add(path+".surveyContext", "inconsistent", "survey requires evidence of a held sextant")
	}
	validateTimeOrder(path+".survey.recordedAt", observation.DiscoveredAt, observation.Survey.RecordedAt, add)
	validateTimeOrder(path+".survey.recordedAt", observation.Survey.RecordedAt, observation.ObservedAt, add)
}
