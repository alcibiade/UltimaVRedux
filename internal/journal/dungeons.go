package journal

import "github.com/bradhannah/UltimaVRedux/internal/references"

// Dungeon exterior slots are fixed so neither side can replace the other.
const (
	DungeonSurfaceSite = iota
	DungeonUnderworldSite
	DungeonExteriorSiteCount
)

// SiteLocation includes semantic map identity: matching X/Y alone is not a site.
type SiteLocation struct {
	Map   references.Location    `json:"map"`
	Floor references.FloorNumber `json:"floor"`
	X     int                    `json:"x"`
	Y     int                    `json:"y"`
}

// GameTime is a concrete in-world timestamp, independent of wall time and turns.
type GameTime struct {
	Year   int `json:"year"`
	Month  int `json:"month"`
	Day    int `json:"day"`
	Hour   int `json:"hour"`
	Minute int `json:"minute"`
}

// OptionalGameTime preserves absent historical dates without inventing provenance.
// When Known is false, Time must be zero. All fields have value ownership.
type OptionalGameTime struct {
	Known bool     `json:"known"`
	Time  GameTime `json:"time"`
}

// SurveyedLocation is optional public coordinate authority for this exact site.
// When Measured is false the entire remaining value must be zero, including dates.
type SurveyedLocation struct {
	Measured   bool             `json:"measured"`
	Flexible   bool             `json:"flexible"`
	Location   SiteLocation     `json:"location"`
	RecordedAt OptionalGameTime `json:"recordedAt"`
}

// DungeonSiteObservation is PRIVATE recognition state, never a player DTO.
// A site is populated only after an encounter. The survey is independently optional.
// Open and ObservedAt remember the last observation, not the live seal state.
type DungeonSiteObservation struct {
	Seen       bool             `json:"seen"`
	Location   SiteLocation     `json:"location"`
	Open       bool             `json:"open"`
	ObservedAt OptionalGameTime `json:"observedAt"`
	Survey     SurveyedLocation `json:"survey"`
}

// DungeonObservation is private GameState-owned knowledge, not seal authority.
// Fixed slots are surface then Underworld; unused slots are entirely zero.
type DungeonObservation struct {
	Known    bool                                             `json:"known"`
	WordUsed bool                                             `json:"wordUsed"`
	Sites    [DungeonExteriorSiteCount]DungeonSiteObservation `json:"sites"`
}

// DungeonEntry is a spoiler-safe reader projection, not a hidden dungeon index.
// A Gate entry is a learned clue rather than an observed dungeon: it has no
// sites, entrance or location, and Description holds only disclosed clue text.
type DungeonEntry struct {
	Sites       [DungeonExteriorSiteCount]DungeonSiteEntry
	Name        string
	Entrance    string
	Location    string
	Word        string
	Source      string
	Description string
	WordIsClue  bool
	Gate        bool
}

// DungeonSiteEntry contains only disclosed facts. An uninspected slot is zero;
// an inspected but unmeasured site exposes its world label, never private X/Y.
// Arrays and concrete optional values make reader copies fully detached.
type DungeonSiteEntry struct {
	Inspected  bool
	World      string
	Open       bool
	ObservedAt OptionalGameTime
	Survey     SurveyedLocation
}
