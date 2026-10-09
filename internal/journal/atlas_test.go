package journal

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAtlasPlaceFilterKeepsAllKnownSourcesAndPrivateSitesHidden(t *testing.T) {
	point := SiteLocation{X: 10, Y: 20}
	entries := []AtlasEntry{
		{Key: "merged", Category: AtlasCastles, Charted: true,
			Survey: SurveyedLocation{Measured: true, Location: point}, LastObservedDestination: true},
		{Key: "unmeasured", Category: AtlasSettlements},
		{Key: "survey", Category: AtlasDungeons, Survey: SurveyedLocation{Measured: true, Location: point}},
	}
	filters := DefaultAtlasFilters()
	got := FilterAtlasEntries(entries, filters)
	require.Len(t, got, 3)
	filters.Places = AtlasCastlePlaces
	got = FilterAtlasEntries(entries, filters)
	require.Len(t, got, 1)
	require.Equal(t, entries[0], got[0], "filtering preserves both supplied and surveyed facts")
	filters.Places = AtlasDungeonPlaces
	got = FilterAtlasEntries(entries, filters)
	require.Equal(t, []AtlasEntry{entries[2]}, got)
	filters.Places = AtlasTownes
	got = FilterAtlasEntries(entries, filters)
	require.Len(t, got, 1)
	require.Equal(t, "unmeasured", got[0].Key)
	_, ok := got[0].PlotPosition()
	require.False(t, ok)
	filters.Places = AtlasMoonstoneSites
	require.Empty(t, FilterAtlasEntries(entries, filters))
	filters.Places = AtlasAllPlaces
	require.Len(t, FilterAtlasEntries(entries, filters), 3)
	require.True(t, entries[0].Charted)
	require.True(t, entries[0].Survey.Measured, "filtering cannot mutate source evidence")
}

func TestAtlasPlaceFilterCategoriesAndLabels(t *testing.T) {
	filters := []AtlasPlaceFilter{AtlasAllPlaces, AtlasTownes, AtlasKeepPlaces, AtlasCastlePlaces,
		AtlasShrinePlaces, AtlasLighthousePlaces, AtlasDungeonPlaces, AtlasMoonstoneSites, AtlasVesselPlaces, AtlasNotePlaces}
	categories := []AtlasCategory{AtlasSettlements, AtlasKeeps, AtlasCastles, AtlasShrines,
		AtlasLighthouses, AtlasDungeons, AtlasMoonstones, AtlasVessels, AtlasNotes}
	labels := []string{"All", "Settlements", "Keeps", "Castles", "Shrines", "Lighthouses", "Dungeons", "Moonstones", "Vessels", "My Notes"}
	var entries []AtlasEntry
	for index, category := range categories {
		require.EqualValues(t, index, category)
		entries = append(entries, AtlasEntry{Key: labels[index+1], Category: category})
	}
	require.EqualValues(t, len(filters), AtlasPlaceFilterCount)
	require.Equal(t, AtlasFilters{Places: AtlasAllPlaces}, DefaultAtlasFilters())
	for index, filter := range filters {
		require.EqualValues(t, index, filter)
		require.Equal(t, labels[index], filter.String())
		got := FilterAtlasEntries(entries, AtlasFilters{Places: filter})
		if filter == AtlasAllPlaces {
			require.Equal(t, entries, got)
		} else {
			require.Equal(t, []AtlasEntry{entries[index-1]}, got)
		}
		for _, entry := range got {
			position, plotted := entry.PlotPosition()
			require.False(t, plotted)
			require.Equal(t, SiteLocation{}, position)
		}
	}
	require.Empty(t, FilterAtlasEntries(entries, AtlasFilters{Places: AtlasPlaceFilterCount}))
	require.Empty(t, FilterAtlasEntries([]AtlasEntry{{Category: AtlasNotes + 1}}, DefaultAtlasFilters()))
}
