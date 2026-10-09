package references

// ChartedMoonstoneOrigin is a fixed printed-map landmark, never a live stone
// destination or evidence of personal recognition or sextant measurement.
type ChartedMoonstoneOrigin struct {
	Label    string
	Location PlayerLocation
}

// ChartedMoonstoneOrigins returns value-owned chart landmarks in phase order.
// Original sites and neighbouring towns follow ref:MOON-0001.
// Relocation cannot modify the supplied chart (ref:MOON-0006).
func ChartedMoonstoneOrigins() [8]ChartedMoonstoneOrigin {
	return [8]ChartedMoonstoneOrigin{
		{Label: "Near Moonglow", Location: PlayerLocation{Location: BritanniaUnderworld, Floor: 0, Position: Position{X: 224, Y: 133}}},
		{Label: "Near Britain", Location: PlayerLocation{Location: BritanniaUnderworld, Floor: 0, Position: Position{X: 96, Y: 102}}},
		{Label: "Near Jhelom", Location: PlayerLocation{Location: BritanniaUnderworld, Floor: 0, Position: Position{X: 38, Y: 224}}},
		{Label: "Near Yew", Location: PlayerLocation{Location: BritanniaUnderworld, Floor: 0, Position: Position{X: 50, Y: 37}}},
		{Label: "Near Minoc", Location: PlayerLocation{Location: BritanniaUnderworld, Floor: 0, Position: Position{X: 166, Y: 19}}},
		{Label: "Near Trinsic", Location: PlayerLocation{Location: BritanniaUnderworld, Floor: 0, Position: Position{X: 104, Y: 194}}},
		{Label: "Near Skara Brae", Location: PlayerLocation{Location: BritanniaUnderworld, Floor: 0, Position: Position{X: 23, Y: 126}}},
		{Label: "Near New Magincia", Location: PlayerLocation{Location: BritanniaUnderworld, Floor: 0, Position: Position{X: 187, Y: 167}}},
	}
}
