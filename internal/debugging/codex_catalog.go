package debugging

// Codex command IDs identify immutable developer-reference queries.
const (
	CommandCodexEquipmentCatalog = "codex.equipment.catalog"
	CommandCodexMonsterCatalog   = "codex.monster.catalog"
	CommandCodexSpellCatalog     = "codex.spell.catalog"
	CommandCodexTileCatalog      = "codex.tile.catalog"
	CommandCodexLocationCatalog  = "codex.location.catalog"
	CommandCodexNPCCatalog       = "codex.npc.catalog"
	CommandCodexQuestCatalog     = "codex.quest.catalog"
	CommandCodexSaveFormat       = "codex.save-format.catalog"
	CommandCodexSearchLocations  = "codex.search-location.catalog"
	CommandCodexConsumables      = "codex.consumable.catalog"
)

func codexCatalogCommands() []CommandDefinition {
	return []CommandDefinition{
		{ID: CommandCodexEquipmentCatalog, Category: "Codex", Name: "Equipment Compendium", Description: "Browse immutable original equipment identities, combat values, requirements, and abilities.", ExecutionModes: []ExecutionMode{ExecutionQuery}},
		{ID: CommandCodexMonsterCatalog, Category: "Codex", Name: "Monster Bestiary", Description: "Browse immutable monster identities, original combat data, movement flags, and abilities.", ExecutionModes: []ExecutionMode{ExecutionQuery}},
		{ID: CommandCodexSpellCatalog, Category: "Codex", Name: "Spell and Reagent Grimoire", Description: "Compare all original spells with Redux targeting, formulas, visuals, audio, events, issues, opportunities, and code references.", ExecutionModes: []ExecutionMode{ExecutionQuery}},
		{ID: CommandCodexTileCatalog, Category: "Codex", Name: "Tile Atlas", Description: "Browse immutable tile identities, rendering metadata, and derived passability rules.", ExecutionModes: []ExecutionMode{ExecutionQuery}},
		{ID: CommandCodexLocationCatalog, Category: "Codex", Name: "World Gazetteer", Description: "Browse location identities, map families, floors, and original overworld entrances.", ExecutionModes: []ExecutionMode{ExecutionQuery}},
		{ID: CommandCodexNPCCatalog, Category: "Codex", Name: "NPC Schedule Index", Description: "Browse loaded NPC identities, dialogue references, sprites, and scheduled behavior.", ExecutionModes: []ExecutionMode{ExecutionQuery}},
		{ID: CommandCodexQuestCatalog, Category: "Codex", Name: "Quest Artifact Ledger", Description: "Browse immutable quest-artifact identities, sprites, and pickup descriptions.", ExecutionModes: []ExecutionMode{ExecutionQuery}},
		{ID: CommandCodexSaveFormat, Category: "Codex", Name: "Save Format Explorer", Description: "Browse documented canonical SAVED.GAM layout and important field offsets.", ExecutionModes: []ExecutionMode{ExecutionQuery}},
		{ID: CommandCodexSearchLocations, Category: "Codex", Name: "Search Location Atlas", Description: "Decode the original search-location catalog, payloads, and repeat requirements (CMD-0007).", ExecutionModes: []ExecutionMode{ExecutionQuery}},
		{ID: CommandCodexConsumables, Category: "Codex", Name: "Consumables Codex", Description: "Compare original potion and scroll behavior with current Redux support and deviations.", ExecutionModes: []ExecutionMode{ExecutionQuery}},
	}
}
