package references

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChartedMoonstoneOriginsAreDetachedFixedPhaseLandmarks(t *testing.T) {
	labels := [8]string{"Near Moonglow", "Near Britain", "Near Jhelom", "Near Yew", "Near Minoc", "Near Trinsic", "Near Skara Brae", "Near New Magincia"}
	sites := [8]Position{{X: 224, Y: 133}, {X: 96, Y: 102}, {X: 38, Y: 224}, {X: 50, Y: 37},
		{X: 166, Y: 19}, {X: 104, Y: 194}, {X: 23, Y: 126}, {X: 187, Y: 167}}
	origins := ChartedMoonstoneOrigins()
	for phase, origin := range origins {
		assert.Equal(t, labels[phase], origin.Label)
		assert.Equal(t, PlayerLocation{Location: BritanniaUnderworld, Floor: 0, Position: sites[phase]}, origin.Location)
		origins[phase].Label = "Moved"
		origins[phase].Location.Position = Position{X: 1, Y: 2}
		assert.Equal(t, origin, ChartedMoonstoneOrigins()[phase])
	}
}
