package journal

import (
	"encoding/json"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/require"
)

func TestAtlasPlotRequiresSurveyForEveryCategoryAndSpecialSite(t *testing.T) {
	for _, kind := range []string{"towne", "village", "hut", "keep", "castle", "shrine", "lighthouse", "dungeon", "moonstone", "guardians"} {
		entry := AtlasEntry{Kind: kind, Charted: true, NameKnown: true, Discovered: true,
			Survey: SurveyedLocation{Location: SiteLocation{X: 11, Y: 12}}}
		point, plotted := entry.PlotPosition()
		require.False(t, plotted, kind)
		require.Zero(t, point)
		encoded, err := json.Marshal(AtlasEntry{Kind: kind, Charted: true})
		require.NoError(t, err)
		require.NotContains(t, string(encoded), "ChartPosition")
		entry.Survey.Measured = true
		point, plotted = entry.PlotPosition()
		require.True(t, plotted)
		require.Equal(t, entry.Survey.Location, point)
	}
}

func TestAtlasReceiptValidatorNarrowAuthority(t *testing.T) {
	at := OptionalGameTime{Known: true, Time: GameTime{Year: 139, Month: 4, Day: 5, Hour: 12, Minute: 3}}
	site := SiteLocation{Map: references.BritanniaUnderworld, X: 104, Y: 216}
	valid := AtlasLandmarkObservation{SiteKey: uint8(references.Greyhaven), Site: site, Kind: "lighthouse",
		Name: "GREYHAVEN", NameSource: AtlasSourceEnter, NamedAt: at,
		Source: AtlasSourceSextantReceipt, ObservedAt: at, SurveySource: AtlasSourceSextantReceipt,
		Survey: SurveyedLocation{Measured: true, Location: site, RecordedAt: at}}
	require.NoError(t, ValidateAtlasLandmarks([]AtlasLandmarkObservation{valid}))
	for _, mutate := range []func(*AtlasLandmarkObservation){
		func(o *AtlasLandmarkObservation) { o.SiteKey = uint8(references.Fogsbane) },
		func(o *AtlasLandmarkObservation) { o.Kind = "hut" },
		func(o *AtlasLandmarkObservation) { o.Name = ""; o.NameSource = ""; o.NamedAt = OptionalGameTime{} },
		func(o *AtlasLandmarkObservation) { o.NameSource = AtlasSourceLook },
		func(o *AtlasLandmarkObservation) { o.SurveySource = AtlasSourceEnter },
		func(o *AtlasLandmarkObservation) { o.Site.X++; o.Survey.Location = o.Site },
		func(o *AtlasLandmarkObservation) { o.NamedAt.Known = false },
		func(o *AtlasLandmarkObservation) { o.NamedAt.Time.Hour++ },
		func(o *AtlasLandmarkObservation) { o.Survey.RecordedAt.Time.Hour++ },
		func(o *AtlasLandmarkObservation) { o.ObservedAt.Time.Minute++ },
		func(o *AtlasLandmarkObservation) { o.Survey.RecordedAt.Time.Year = 0 },
		func(o *AtlasLandmarkObservation) { o.Survey.Measured = false },
	} {
		invalid := valid
		mutate(&invalid)
		require.ErrorIs(t, ValidateAtlasLandmarks([]AtlasLandmarkObservation{invalid}), ErrInvalidAtlasObservation)
	}
	valid.Source = AtlasSourceLook
	valid.ObservedAt.Time.Hour++
	require.NoError(t, ValidateAtlasLandmarks([]AtlasLandmarkObservation{valid}), "later Look preserves original receipt survey")
	valid.NamedAt = OptionalGameTime{}
	require.NoError(t, ValidateAtlasLandmarks([]AtlasLandmarkObservation{valid}), "legitimate old Enter naming can be undated")
}

func TestAtlasGuardiansValidatorOnlyLocalLookWithoutLearnedName(t *testing.T) {
	valid := AtlasLandmarkObservation{SiteKey: AtlasGuardiansSiteKey, Site: SiteLocation{X: 233, Y: 235}, Kind: "guardians", Source: AtlasSourceLook}
	require.NoError(t, ValidateAtlasLandmarks([]AtlasLandmarkObservation{valid}))
	for _, mutate := range []func(*AtlasLandmarkObservation){
		func(o *AtlasLandmarkObservation) { o.Kind = "shrine" },
		func(o *AtlasLandmarkObservation) { o.Site.X-- },
		func(o *AtlasLandmarkObservation) { o.Source = AtlasSourceEnter },
		func(o *AtlasLandmarkObservation) { o.Source = AtlasSourceShrineIntroduction },
		func(o *AtlasLandmarkObservation) { o.Name = "Guardians"; o.NameSource = AtlasSourceEnter },
		func(o *AtlasLandmarkObservation) { o.SiteKey = 109 },
	} {
		invalid := valid
		mutate(&invalid)
		require.Error(t, ValidateAtlasLandmarks([]AtlasLandmarkObservation{invalid}))
	}
}
