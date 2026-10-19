package audio

const (
	routingTilesKey   = "tiles"
	routingDungeonKey = "dungeon"
	routingAmbientKey = "ambient"
)

// AuthoringTargets supplies consumer-owned routing labels and acoustic vocabulary
// to the terminal frontend. Empty groups expose no invented game targets.
type AuthoringTargets struct {
	Tiles         []TileTarget
	Music         map[string]string
	Dungeon       map[string]string
	Naval         map[string]string
	Families      []string
	CueCategories map[string]string // music, sfx or ambient; neutral cues default to sfx.
}

func reduxAuthoringTargets() AuthoringTargets {
	return AuthoringTargets{Tiles: TileTargets(), Music: MusicTargets(), Dungeon: DungeonTargets(),
		Naval: NavalTargets(), Families: AcousticFamilies(), CueCategories: nil}
}

// Targets returns detached descriptors; callers cannot mutate another editor's UI.
func (e *LibraryEditor) Targets() AuthoringTargets { return cloneAudioValue(e.targets) }

// AcousticFamilies returns this project's selectable family vocabulary.
func (e *LibraryEditor) AcousticFamilies() []string {
	if len(e.targets.Families) > 0 {
		return cloneAudioValue(e.targets.Families)
	}

	return sortedKeys(e.library.Taxonomy.RecordingFamilies)
}
