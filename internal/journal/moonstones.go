package journal

import "github.com/bradhannah/UltimaVRedux/internal/datetime"

// MoonstoneSource is stable evidence identity, not prose or a location key.
type MoonstoneSource string

// Sources describe successful local actions. Possession supports conservative
// imports only, with no fabricated site, date or survey.
const (
	MoonstoneSourceNone       MoonstoneSource = ""
	MoonstoneSourcePossession MoonstoneSource = "possession"
	MoonstoneSourceGate       MoonstoneSource = "gate"
	MoonstoneSourceSpell      MoonstoneSource = "spell"
	MoonstoneSourceSearch     MoonstoneSource = "search"
	MoonstoneSourcePickup     MoonstoneSource = "pickup"
	MoonstoneSourceBurial     MoonstoneSource = "burial"
	MoonstoneSourceLook       MoonstoneSource = "look"
)

// String is safe for readers: no private site or raw observer payload is exposed.
func (source MoonstoneSource) String() string {
	switch source {
	case MoonstoneSourcePossession:
		return "Known from possession"
	case MoonstoneSourceGate:
		return "Reached through a moongate"
	case MoonstoneSourceSpell:
		return "Reached through Vas Rel Por"
	case MoonstoneSourceSearch:
		return "Uncovered a moonstone"
	case MoonstoneSourcePickup:
		return "Picked up a moonstone"
	case MoonstoneSourceBurial:
		return "Buried a moonstone"
	case MoonstoneSourceLook:
		return "Recognized a moongate"
	case MoonstoneSourceNone:
		return ""
	default:
		panic("invalid moonstone observation source")
	}
}

// MoonstoneContext is PRIVATE evidence at a successful action boundary. Observer
// may differ from the handled target. The accompanying timestamp owns the hour.
type MoonstoneContext struct {
	Source     MoonstoneSource `json:"source"`
	Observer   SiteLocation    `json:"observer"`
	HasSextant bool            `json:"hasSextant"`
}

// MoonstoneObservation is phase-indexed, PRIVATE GameState knowledge. Site is the
// last actually observed full site, not the live placement. Placed is retired on
// pickup; possession itself is always inventory-owned. Survey belongs only to
// the current observed placement and is cleared on pickup or site change.
// Unknown records are entirely zero. SiteKnown=false requires zero site/context/
// observation time and Placed=false. No survey requires zero SurveyContext too.
// OriginalSiteRetired is sticky evidence of handling or relocation; immutable
// chart metadata is never changed or restored by returning to its coordinates.
type MoonstoneObservation struct {
	OriginalSiteRetired bool             `json:"originalSiteRetired"`
	Known               bool             `json:"known"`
	FirstSource         MoonstoneSource  `json:"firstSource"`
	DiscoveredAt        OptionalGameTime `json:"discoveredAt"`
	SiteKnown           bool             `json:"siteKnown"`
	Site                SiteLocation     `json:"site"`
	Placed              bool             `json:"placed"`
	LastContext         MoonstoneContext `json:"lastContext"`
	ObservedAt          OptionalGameTime `json:"observedAt"`
	Survey              SurveyedLocation `json:"survey"`
	SurveyContext       MoonstoneContext `json:"surveyContext"`
}

// MoonstoneEntry is a detached public projection. Neither chart coordinates nor
// unmeasured recognition/observer sites are present. Only LastObservedSurvey is
// plottable as a remembered current placement. HistoricalSurvey remains zero.
// An earned current survey remains readable without a currently usable sextant.
type MoonstoneEntry struct {
	OriginalSiteRetired bool
	Phase               datetime.MoonPhase
	ChartedOrigin       string
	Known               bool
	Carried             bool
	Placed              bool
	FirstSource         MoonstoneSource
	DiscoveredAt        OptionalGameTime
	ObservedAt          OptionalGameTime
	LastObservedSurvey  SurveyedLocation
	HistoricalSurvey    SurveyedLocation
	SurveySource        MoonstoneSource
}
