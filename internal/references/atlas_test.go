package references

import (
	"fmt"
	"os"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/atlasterrain"
	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"
	"github.com/stretchr/testify/require"
)

func TestAtlasActualImmutableTerrainAndChart(t *testing.T) {
	cfg, err := config.LoadUltimaVConfiguration("", originaldata.Directory(t, "DATA.OVL", "BRIT.DAT"))
	require.NoError(t, err)
	world, err := NewWorldLocations(cfg)
	require.NoError(t, err)
	points := SuppliedAtlasChart(world)
	require.Len(t, points, 34)
	for _, p := range points {
		require.NotEqual(t, Position{}, p.Position)
		require.NotContains(t, p.Key, "Blackthorn")
		require.NotContains(t, p.Name, "Fogsbane")
		require.NotContains(t, p.Name, "Spirituality")
	}
	require.Len(t, AtlasRegionLabels(), 8)
	terrain, err := NewLargeMapReference(cfg, OVERWORLD)
	require.NoError(t, err)
	ids := map[int]bool{}
	input, err := atlasterrain.Capture(func(x, y int) int {
		id := int(terrain.GetSpriteIndex(Coordinate(x), Coordinate(y)))
		ids[id] = true
		return id
	})
	require.NoError(t, err)
	require.Len(t, ids, 74)
	normalized, count, err := atlasterrain.Normalize(input)
	require.NoError(t, err)
	require.Equal(t, 69, count)
	// Every unmodified cell retains its exact art index; no surrounding redaction.
	for x := range 256 {
		for y := range 256 {
			if input[x][y] == 5 || input[x][y] >= 32 && input[x][y] <= 38 {
				require.Equal(t, input[x][y], normalized[x][y])
			}
		}
	}
	art, err := os.ReadFile("../sprites/assets/transparent_u5_tiles.png")
	require.NoError(t, err)
	root := t.TempDir()
	cold, err := atlasterrain.LoadOrGenerate(root, input, art)
	require.NoError(t, err)
	require.NoError(t, cold.Diagnostic)
	warm, err := atlasterrain.LoadOrGenerate(root, input, art)
	require.NoError(t, err)
	require.True(t, warm.CacheHit)
	stat, err := os.Stat(cold.Path)
	require.NoError(t, err)
	t.Logf("actual terrain: 74 IDs, 69 substitutions; PNG=%d bytes cold compose=%s encode=%s warm decode=%s", stat.Size(), cold.ComposeTime, cold.EncodeTime, warm.DecodeTime)
}

func TestAtlasPrintedSymbolsFiniteIdentityAllowlist(t *testing.T) {
	cfg, err := config.LoadUltimaVConfiguration("", originaldata.Directory(t, "DATA.OVL"))
	require.NoError(t, err)
	world, err := NewWorldLocations(cfg)
	require.NoError(t, err)
	counts := map[string]int{}
	keys := map[string]bool{}
	for _, point := range SuppliedAtlasChart(world) {
		if !IsSuppliedAtlasSymbol(point) {
			continue
		}
		counts[point.Kind]++
		keys[point.Key] = true
		if point.Kind == "moonstone" {
			require.Equal(t, ChartedMoonstoneOrigins()[point.Phase].Location.Position, point.Position)
		}
	}
	require.Len(t, keys, 27)
	require.Equal(t, map[string]int{"towne": 8, "stronghold": 3, "castle": 1, "shrine": 7, "moonstone": 8}, counts)
	for _, key := range []string{"chart-13", "chart-14", "chart-15", "chart-16", "chart-17", "chart-25", "chart-34"} {
		require.False(t, keys[key], key)
	}
	for _, site := range []Location{PalaceOfBlackthorn, BuccaneersDen, Fogsbane, Greyhaven, Stormcrow, Waveguide, IolosHut, Stonegate, Deceit} {
		require.False(t, IsSuppliedAtlasSymbol(AtlasChartPoint{SiteKey: uint8(site), Kind: "towne"}), site.String())
	}
	require.False(t, IsSuppliedAtlasSymbol(AtlasChartPoint{SiteKey: 108, Kind: "shrine"}))
}

func TestSuppliedAtlasChartPublicBaselineAndStableKeys(t *testing.T) {
	cfg, err := config.LoadUltimaVConfiguration("", originaldata.Directory(t, "DATA.OVL"))
	require.NoError(t, err)
	world, err := NewWorldLocations(cfg)
	require.NoError(t, err)
	points := SuppliedAtlasChart(world)
	require.Len(t, points, 34)
	byKey := make(map[string]AtlasChartPoint, len(points))
	bySite := make(map[uint8]AtlasChartPoint)
	for _, point := range points {
		require.NotContains(t, byKey, point.Key, "chart keys must be unique")
		byKey[point.Key] = point
		if point.SiteKey != 0 {
			require.NotContains(t, bySite, point.SiteKey)
			bySite[point.SiteKey] = point
		}
		require.NotContains(t, point.Name, "Charted")
		require.NotEqual(t, "hut", point.Kind)
		require.NotEqual(t, "keep", point.Kind)
	}
	for _, hidden := range []Location{IolosHut, SuteksHut, SinVraalsHut, GrendelsHut, Ararat, Bordermarch, Farthing, Windemere, Stonegate} {
		require.NotContains(t, bySite, uint8(hidden), "hidden sites have no supplied point")
	}
	for _, want := range []struct {
		key        string
		location   Location
		name, kind string
	}{
		{"chart-01", Moonglow, "Towne 1", "towne"},
		{"chart-02", Britain, "Towne 2", "towne"},
		{"chart-03", Jhelom, "Towne 3", "towne"},
		{"chart-04", Yew, "Towne 4", "towne"},
		{"chart-05", Minoc, "Towne 5", "towne"},
		{"chart-06", Trinsic, "Towne 6", "towne"},
		{"chart-07", SkaraBrae, "Towne 7", "towne"},
		{"chart-08", NewMagincia, "Towne 8", "towne"},
		{"chart-09", Lycaeum, "Keep 1", "stronghold"},
		{"chart-10", EmpathAbbey, "Keep 2", "stronghold"},
		{"chart-11", SerpentsHold, "Keep 3", "stronghold"},
		{"chart-12", LordBritishsCastle, "Lord British's Castle", "castle"},
		{"chart-13", PalaceOfBlackthorn, "Unnamed castle", "castle"},
		{"chart-14", Fogsbane, "Lighthouse 1", "lighthouse"},
		{"chart-15", Stormcrow, "Lighthouse 2", "lighthouse"},
		{"chart-16", Greyhaven, "Lighthouse 3", "lighthouse"},
		{"chart-17", Waveguide, "Lighthouse 4", "lighthouse"},
		{"chart-34", BuccaneersDen, "Towne 9", "towne"},
	} {
		require.Contains(t, byKey, want.key)
		point := byKey[want.key]
		position, _ := world.GetWorldLocation(want.location)
		require.Equal(t, uint8(want.location), point.SiteKey)
		require.Equal(t, want.name, point.Name)
		require.Equal(t, want.kind, point.Kind)
		require.Equal(t, position, point.Position)
		require.Equal(t, -1, point.Phase)
	}
	for index, position := range AtlasShrinePositions() {
		point := byKey[fmt.Sprintf("chart-%02d", 18+index)]
		require.Equal(t, uint8(101+index), point.SiteKey)
		require.Equal(t, fmt.Sprintf("Shrine %d", index+1), point.Name)
		require.Equal(t, "shrine", point.Kind)
		require.Equal(t, position, point.Position)
	}
	require.Equal(t, "Guardians of the Codex", byKey["chart-25"].Name)
	require.EqualValues(t, 108, byKey["chart-25"].SiteKey)
	require.Equal(t, "guardians", byKey["chart-25"].Kind)
	require.Equal(t, Position{X: 233, Y: 235}, byKey["chart-25"].Position)
	require.Equal(t, Position{X: 104, Y: 216}, byKey["chart-16"].Position, "receipt uses the installed exterior, never local room XY")
	for phase, origin := range ChartedMoonstoneOrigins() {
		key := fmt.Sprintf("chart-%02d", 26+phase)
		require.Equal(t, key, AtlasMoonstoneChartKey(phase))
		point := byKey[key]
		require.Equal(t, "moonstone", point.Kind)
		require.Equal(t, phase, point.Phase)
		require.Equal(t, origin.Location.Position, point.Position)
	}
}
