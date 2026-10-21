package mainscreen

// MarkStatsChanged flags that the character stats display needs refresh.
// This is called by SystemCallbacks when character data changes to trigger
// display updates on the next frame.
func (c *CharacterSummary) MarkStatsChanged() {
	c.needsStatsRefresh = true
}

// IsStatsRefreshNeeded returns whether the stats display needs refresh.
// Used by rendering systems to determine if cached displays are still valid.
func (c *CharacterSummary) IsStatsRefreshNeeded() bool {
	return c.needsStatsRefresh
}

// IsInventoryRefreshNeeded returns whether inventory display needs refresh.
func (p *ProvisionSummary) IsInventoryRefreshNeeded() bool {
	return p.needsInventoryRefresh
}
