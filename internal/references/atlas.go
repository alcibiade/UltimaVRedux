package references

import "fmt"

// AtlasChartPoint is supplied printed knowledge, never a visit or measurement.
// Key is neutral: unnamed sites must not disclose hidden names in reader metadata.
type AtlasChartPoint struct {
	Key      string
	SiteKey  uint8
	Name     string
	Kind     string
	Position Position
	Phase    int
}

// IsSuppliedAtlasSymbol is the finite approved printed-artwork allowlist. The
// larger supplied directory is deliberately not itself plotting permission.
func IsSuppliedAtlasSymbol(point AtlasChartPoint) bool {
	if point.Kind == "moonstone" {
		return point.Phase >= 0 && point.Phase < 8 && point.Key == AtlasMoonstoneChartKey(point.Phase)
	}
	switch point.SiteKey {
	case uint8(Moonglow), uint8(Britain), uint8(Jhelom), uint8(Yew), uint8(Minoc), uint8(Trinsic), uint8(SkaraBrae), uint8(NewMagincia),
		uint8(Lycaeum), uint8(EmpathAbbey), uint8(SerpentsHold), uint8(LordBritishsCastle),
		101, 102, 103, 104, 105, 106, 107:
		return true
	default:
		return false
	}
}

// AtlasShrinePositions excludes Spirituality's non-spatial sentinel.
func AtlasShrinePositions() [7]Position {
	return [7]Position{{X: 233, Y: 66}, {X: 128, Y: 92}, {X: 36, Y: 229}, {X: 73, Y: 11}, {X: 205, Y: 45}, {X: 81, Y: 207}, {X: 231, Y: 216}}
}

// AtlasMoonstoneChartKey preserves the original neutral chart identity per phase,
// independent of the number or ordering of other supplied points.
func AtlasMoonstoneChartKey(phase int) string {
	return fmt.Sprintf("chart-%02d", 26+phase)
}

// SuppliedAtlasChart deliberately looks up only the approved settlement allowlist.
// REDUX DEVIATION: the physical supplied map becomes a readable, static chart;
// these reference labels do not imply personal discoveries or authorize list-linked
// geometry. IsSuppliedAtlasSymbol separately permits a finite decorative subset.
// See ref:VIEW-0005.
func SuppliedAtlasChart(world *WorldLocations) []AtlasChartPoint {
	points := make([]AtlasChartPoint, 0, 34)
	for _, item := range []struct {
		loc        Location
		name, kind string
	}{
		{Moonglow, "Towne 1", "towne"}, {Britain, "Towne 2", "towne"}, {Jhelom, "Towne 3", "towne"}, {Yew, "Towne 4", "towne"},
		{Minoc, "Towne 5", "towne"}, {Trinsic, "Towne 6", "towne"}, {SkaraBrae, "Towne 7", "towne"}, {NewMagincia, "Towne 8", "towne"},
		{Lycaeum, "Keep 1", "stronghold"}, {EmpathAbbey, "Keep 2", "stronghold"}, {SerpentsHold, "Keep 3", "stronghold"},
		{LordBritishsCastle, "Lord British's Castle", "castle"}, {PalaceOfBlackthorn, "Unnamed castle", "castle"},
		{Fogsbane, "Lighthouse 1", "lighthouse"}, {Stormcrow, "Lighthouse 2", "lighthouse"},
		{Greyhaven, "Lighthouse 3", "lighthouse"}, {Waveguide, "Lighthouse 4", "lighthouse"},
	} {
		position, _ := world.GetWorldLocation(item.loc)
		points = append(points, AtlasChartPoint{Key: fmt.Sprintf("chart-%02d", len(points)+1), SiteKey: uint8(item.loc), Name: item.name, Kind: item.kind, Position: position, Phase: -1})
	}
	for i, p := range AtlasShrinePositions() {
		points = append(points, AtlasChartPoint{Key: fmt.Sprintf("chart-%02d", len(points)+1), SiteKey: uint8(101 + i), Name: fmt.Sprintf("Shrine %d", i+1), Kind: "shrine", Position: p, Phase: -1})
	}
	points = append(points, AtlasChartPoint{Key: "chart-25", SiteKey: 108, Name: "Guardians of the Codex", Kind: "guardians", Position: Position{X: 233, Y: 235}, Phase: -1})
	for phase, origin := range ChartedMoonstoneOrigins() {
		points = append(points, AtlasChartPoint{Key: AtlasMoonstoneChartKey(phase), Name: origin.Label, Kind: "moonstone", Position: origin.Location.Position, Phase: phase})
	}
	// Append with a fixed key so the original chart identities never shift.
	position, _ := world.GetWorldLocation(BuccaneersDen)
	points = append(points, AtlasChartPoint{Key: "chart-34", SiteKey: uint8(BuccaneersDen), Name: "Towne 9", Kind: "towne", Position: position, Phase: -1})

	return points
}

// AtlasRegionLabel anchors are decorative layout coordinates, not site facts.
type AtlasRegionLabel struct {
	Name string
	X, Y int
}

// AtlasRegionLabels returns detached decoration, never focusable directory points.
func AtlasRegionLabels() [8]AtlasRegionLabel {
	return [8]AtlasRegionLabel{{"Spiritwood", 53, 141}, {"Serpent's Spine", 82, 74}, {"The Deep Forest", 60, 51}, {"The Drylands", 207, 40}, {"Dagger Isle", 239, 56}, {"Verity Isle", 223, 120}, {"Valorian Isles", 37, 220}, {"Isle of the Avatar", 231, 225}}
}
