package debugging

const (
	// CommandQuestShardUse queues the production shard-use workflow for one stable shard identity.
	CommandQuestShardUse = "quest.shard.use"
	// CommandQuestInventoryPresetUsable applies the reviewed all-usable inventory preset.
	CommandQuestInventoryPresetUsable = "quest.inventory.preset-usable"
)

func fourthBatchCommands() []CommandDefinition {
	minimumQuantity, maximumQuantity := 0, 99
	minimumLocation, maximumLocation := 0, 255
	minimumCoordinate, maximumCoordinate := 0, 255
	maximumFloor := 8
	minimumFloor := -maximumFloor
	shards := []Choice{
		{Value: "shard_falsehood", Label: "Shard of Falsehood", Metadata: nil},
		{Value: "shard_hatred", Label: "Shard of Hatred", Metadata: nil},
		{Value: "shard_cowardice", Label: "Shard of Cowardice", Metadata: nil},
	}
	flames := []Choice{
		{Value: "none", Label: "No Flame", Metadata: nil},
		{Value: "truth", Label: "Flame of Truth", Metadata: nil},
		{Value: "love", Label: "Flame of Love", Metadata: nil},
		{Value: "courage", Label: "Flame of Courage", Metadata: nil},
	}
	shadowlords := []Choice{
		{Value: "none", Label: "No Shadowlord", Metadata: nil},
		{Value: "faulinei", Label: "Faulinei", Metadata: nil},
		{Value: "astaroth", Label: "Astaroth", Metadata: nil},
		{Value: "nosfentor", Label: "Nosfentor", Metadata: nil},
	}

	return []CommandDefinition{
		{
			ID: CommandQuestShardUse, Category: categoryQuestItems, Name: "Use Shard",
			Description: "Queue the production shard workflow for one explicitly reviewed shard and live quest state.",
			Aliases:     []string{"quest use <shard>"},
			Parameters: []ParameterDefinition{
				{ID: "shard", Name: "Shard", Kind: ParameterEnum, Required: true, Choices: shards, ControlHint: controlHintSelect, Availability: []string{contextGameplay}},
				{ID: "expectedQuantity", Name: "Reviewed Quantity", Kind: ParameterInteger, Required: true, Minimum: &minimumQuantity, Maximum: &maximumQuantity, Availability: []string{contextGameplay}},
				{ID: "expectedLocationId", Name: "Reviewed Location", Kind: ParameterInteger, Required: true, Minimum: &minimumLocation, Maximum: &maximumLocation, Availability: []string{contextGameplay}},
				{ID: "expectedX", Name: "Reviewed X", Kind: ParameterInteger, Required: true, Minimum: &minimumCoordinate, Maximum: &maximumCoordinate, Availability: []string{contextGameplay}},
				{ID: "expectedY", Name: "Reviewed Y", Kind: ParameterInteger, Required: true, Minimum: &minimumCoordinate, Maximum: &maximumCoordinate, Availability: []string{contextGameplay}},
				{ID: "expectedFloor", Name: "Reviewed Floor", Kind: ParameterInteger, Required: true, Minimum: &minimumFloor, Maximum: &maximumFloor, Availability: []string{contextGameplay}},
				{ID: "expectedFlame", Name: "Reviewed Flame", Kind: ParameterEnum, Required: true, Choices: flames, Availability: []string{contextGameplay}},
				{ID: "expectedShadowlordSummoned", Name: "Reviewed Summoning", Kind: ParameterBoolean, Required: true, Availability: []string{contextGameplay}},
				{ID: "expectedShadowlord", Name: "Reviewed Shadowlord", Kind: ParameterEnum, Required: true, Choices: shadowlords, Availability: []string{contextGameplay}},
				{ID: "expectedShadowlordAlive", Name: "Reviewed Shadowlord Life", Kind: ParameterBoolean, Required: true, Availability: []string{contextGameplay}},
			},
			ExecutionModes: []ExecutionMode{ExecutionWorkflow}, Destructive: true,
		},
		{
			ID: CommandQuestInventoryPresetUsable, Category: "Inventory Locker", Name: "Apply Usable Inventory Preset",
			Description: "Apply exact reviewed targets for every quest item, moonstone, potion, and scroll.",
			Aliases:     []string{"quest giveall"},
			Parameters: []ParameterDefinition{{
				ID: expectedFingerprintID, Name: "Review Fingerprint", Kind: ParameterString, Required: true, Availability: []string{contextGameplay},
			}},
			ExecutionModes: []ExecutionMode{ExecutionAlteration}, Destructive: true,
		},
	}
}
