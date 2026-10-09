package main

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/bradhannah/UltimaVRedux/internal/conversation/semantic"
	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

type codexEquipmentEntry struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	Slot        string   `json:"slot"`
	Ammo        string   `json:"ammo"`
	Attack      int      `json:"attack"`
	Defense     int      `json:"defense"`
	Strength    int      `json:"strength"`
	Range       int      `json:"range"`
	Price       int      `json:"price"`
	TwoHanded   bool     `json:"twoHanded"`
	Bludgeoning bool     `json:"bludgeoning"`
	Abilities   []string `json:"abilities"`
}

type codexMonsterEntry struct {
	ID            int      `json:"id"`
	Name          string   `json:"name"`
	Strength      int      `json:"strength"`
	Dexterity     int      `json:"dexterity"`
	Intelligence  int      `json:"intelligence"`
	Armour        int      `json:"armour"`
	Damage        int      `json:"damage"`
	HitPoints     int      `json:"hitPoints"`
	MaximumPerMap int      `json:"maximumPerMap"`
	Treasure      int      `json:"treasure"`
	AttackRange   int      `json:"attackRange"`
	SpriteIndex   int      `json:"spriteIndex"`
	Abilities     []string `json:"abilities"`
	Water         bool     `json:"water"`
	Sand          bool     `json:"sand"`
	Flies         bool     `json:"flies"`
	WallPass      bool     `json:"wallPass"`
	Immobile      bool     `json:"immobile"`
}

type codexSpellEntry struct {
	ID                 int      `json:"id"`
	Circle             int      `json:"circle"`
	Name               string   `json:"name"`
	Targeting          string   `json:"targeting"`
	Context            string   `json:"context"`
	Description        string   `json:"description"`
	Reagents           []string `json:"reagents"`
	Status             string   `json:"status"`
	OriginalBehavior   string   `json:"originalBehavior"`
	ReduxBehavior      string   `json:"reduxBehavior"`
	OriginalTargeting  string   `json:"originalTargeting"`
	ReduxTargeting     string   `json:"reduxTargeting"`
	ActorVisual        string   `json:"actorVisual"`
	ScreenVisual       string   `json:"screenVisual"`
	StatusIndicator    string   `json:"statusIndicator"`
	TurnOrder          string   `json:"turnOrder"`
	CombatEvents       string   `json:"combatEvents"`
	AreaEffect         string   `json:"areaEffect"`
	Failure            string   `json:"failure"`
	Formula            string   `json:"formula"`
	Audio              string   `json:"audio"`
	KnownIssues        []string `json:"knownIssues"`
	Opportunities      []string `json:"opportunities"`
	CodeReferences     []string `json:"codeReferences"`
	OriginalReferences []string `json:"originalReferences"`
}

type codexTileEntry struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	LookDescription string `json:"lookDescription"`
	Speed           int    `json:"speed"`
	Light           int    `json:"light"`
	AnimationFrames int    `json:"animationFrames"`
	Animated        bool   `json:"animated"`
	BlocksLight     bool   `json:"blocksLight"`
	Enemy           bool   `json:"enemy"`
	NPC             bool   `json:"npc"`
	Building        bool   `json:"building"`
	Walk            bool   `json:"walk"`
	Horse           bool   `json:"horse"`
	Skiff           bool   `json:"skiff"`
	Frigate         bool   `json:"frigate"`
	Carpet          bool   `json:"carpet"`
	NPCPassable     bool   `json:"npcPassable"`
}

type codexLocationEntry struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	MapType string `json:"mapType"`
	WorldX  int    `json:"worldX"`
	WorldY  int    `json:"worldY"`
	Floor   int    `json:"floor"`
}

type codexNPCEntry struct {
	LocationID              int      `json:"locationID"`
	DialogNumber            int      `json:"dialogNumber"`
	SpriteIndex             int      `json:"spriteIndex"`
	Location                string   `json:"location"`
	Type                    string   `json:"type"`
	Times                   []string `json:"times"`
	AI                      []string `json:"ai"`
	Positions               []string `json:"positions"`
	ConversationKind        string   `json:"conversationKind"`
	ConversationSource      string   `json:"conversationSource"`
	ConversationName        string   `json:"conversationName"`
	ConversationDescription string   `json:"conversationDescription"`
	ConversationIndex       int      `json:"conversationIndex"`
	HasConversation         bool     `json:"hasConversation"`
}

type codexQuestEntry struct {
	ID            int    `json:"id"`
	SpriteIndex   int    `json:"spriteIndex"`
	Name          string `json:"name"`
	PickupMessage string `json:"pickupMessage"`
	Description   string `json:"description"`
}

type codexSaveField struct {
	Offset      string `json:"offset"`
	Size        string `json:"size"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type codexSearchLocationEntry struct {
	Index             int      `json:"index"`
	LocationID        int      `json:"locationID"` //nolint:tagliatelle // Public workbench protocol uses ID initialism.
	LocationName      string   `json:"locationName"`
	MapType           string   `json:"mapType"`
	Floor             int      `json:"floor"`
	X                 int      `json:"x"`
	Y                 int      `json:"y"`
	RawObjectType     int      `json:"rawObjectType"`
	RawPayload        int      `json:"rawPayload"`
	Classification    string   `json:"classification"`
	Category          string   `json:"category"`
	ItemID            int      `json:"itemID"` //nolint:tagliatelle // Public workbench protocol uses ID initialism.
	ItemName          string   `json:"itemName"`
	PickupQuantity    int      `json:"pickupQuantity"`
	QuantitySemantics string   `json:"quantitySemantics"`
	RepeatPolicy      string   `json:"repeatPolicy"`
	Requirements      []string `json:"requirements"`
	Suspicious        bool     `json:"suspicious"`
	EncodingNote      string   `json:"encodingNote"`
}

type codexConsumableEntry struct {
	Kind                      string `json:"kind"`
	ID                        int    `json:"id"`
	Name                      string `json:"name"`
	OriginalBehavior          string `json:"originalBehavior"`
	ReduxBehavior             string `json:"reduxBehavior"`
	OriginalContext           string `json:"originalContext"`
	ReduxContext              string `json:"reduxContext"`
	OriginalTarget            string `json:"originalTarget"`
	ReduxTarget               string `json:"reduxTarget"`
	OriginalConsumptionTiming string `json:"originalConsumptionTiming"`
	ReduxConsumptionTiming    string `json:"reduxConsumptionTiming"`
	OriginalDuration          string `json:"originalDuration"`
	ReduxDuration             string `json:"reduxDuration"`
	Support                   string `json:"support"`
	Deviation                 string `json:"deviation"`
}

func (g *Game) executeCodexReferenceDebugCommand(request debugging.Request) (debugging.Result, bool) {
	if !isCodexReferenceCommand(request.CommandID) {
		return debugging.Result{}, false
	}
	if g.debugGameReferences == nil {
		loaded, err := g.debugReferencesLoader(g.currentScene.GetUltimaConfiguration())
		if err != nil {
			return debugging.FailureResult(request, "codex_reference_load_failed", err.Error()), true
		}
		g.debugGameReferences = loaded
	}
	result, handled := executeCodexDebugCommand(request, g.debugGameReferences)
	if !handled {
		panic("recognized Codex command was not handled: " + request.CommandID)
	}

	return result, true
}

func isCodexReferenceCommand(commandID string) bool {
	switch commandID {
	case debugging.CommandCodexEquipmentCatalog, debugging.CommandCodexMonsterCatalog, debugging.CommandCodexSpellCatalog,
		debugging.CommandCodexTileCatalog, debugging.CommandCodexLocationCatalog, debugging.CommandCodexNPCCatalog,
		debugging.CommandCodexQuestCatalog, debugging.CommandCodexSaveFormat, debugging.CommandCodexSearchLocations,
		debugging.CommandCodexConsumables:
		return true
	default:
		return false
	}
}

//nolint:cyclop // The catalog dispatcher intentionally enumerates each codex command.
func executeCodexDebugCommand(request debugging.Request, gameReferences *references.GameReferences) (debugging.Result, bool) {
	switch request.CommandID {
	case debugging.CommandCodexEquipmentCatalog:
		return debugSuccessResult(request, "Equipment compendium loaded.", newCodexEquipmentCatalog()), true
	case debugging.CommandCodexMonsterCatalog:
		return debugSuccessResult(request, "Monster bestiary loaded.", newCodexMonsterCatalog(gameReferences.EnemyReferences)), true
	case debugging.CommandCodexSpellCatalog:
		return debugSuccessResult(request, "Spell and reagent grimoire loaded.", newCodexSpellCatalog()), true
	case debugging.CommandCodexTileCatalog:
		return debugSuccessResult(request, "Tile atlas loaded.", newCodexTileCatalog(gameReferences.TileReferences, gameReferences.LookReferences)), true
	case debugging.CommandCodexLocationCatalog:
		return debugSuccessResult(request, "World gazetteer loaded.", newCodexLocationCatalog(gameReferences)), true
	case debugging.CommandCodexNPCCatalog:
		entries, err := newCodexNPCCatalog(gameReferences)
		if err != nil {
			return debugging.FailureResult(request, "codex_reference_query_failed", err.Error()), true
		}

		return debugSuccessResult(request, "NPC schedule index loaded.", entries), true
	case debugging.CommandCodexQuestCatalog:
		return debugSuccessResult(request, "Quest artifact ledger loaded.", newCodexQuestCatalog(gameReferences.InventoryItemReferences)), true
	case debugging.CommandCodexSaveFormat:
		return debugSuccessResult(request, "Save format explorer loaded.", newCodexSaveFormat()), true
	case debugging.CommandCodexSearchLocations:
		return debugSuccessResult(request, "Search location atlas loaded.", newCodexSearchLocationCatalog()), true
	case debugging.CommandCodexConsumables:
		return debugSuccessResult(request, "Consumables codex loaded.", newCodexConsumableCatalog()), true
	default:
		var empty debugging.Result

		return empty, false
	}
}

func newCodexEquipmentCatalog() []codexEquipmentEntry {
	entries := make([]codexEquipmentEntry, 0, 48)
	for equipment := references.LeatherHelm; equipment <= references.Ankh; equipment++ {
		abilities := equipment.GetAbilities()
		abilityNames := make([]string, 0, len(abilities))
		for _, ability := range abilities {
			abilityNames = append(abilityNames, ability.String())
		}
		ammo := equipment.GetRequiredAmmo()
		ammoName := ""
		if ammo != references.NoEquipment {
			ammoName = ammo.String()
		}
		entries = append(entries, codexEquipmentEntry{ID: equipment.ID(), Name: equipment.String(), Category: equipment.GetCategory().String(), Slot: equipment.GetSlotType().String(), Ammo: ammoName, Attack: equipment.GetAttackValue(), Defense: equipment.GetDefenseValue(), Strength: equipment.GetStrengthRequired(), Range: equipment.GetAttackRange(), Price: equipment.GetBasePrice(), TwoHanded: equipment.IsTwoHanded(), Bludgeoning: equipment.IsBludgeoning(), Abilities: abilityNames})
	}

	return entries
}

func newCodexMonsterCatalog(enemies *references.EnemyReferences) []codexMonsterEntry {
	entries := make([]codexMonsterEntry, 0, len(*enemies))
	for _, enemy := range *enemies {
		abilities := make([]string, 0, len(enemy.EnemyAbilities))
		for ability, enabled := range enemy.EnemyAbilities {
			if enabled {
				abilities = append(abilities, references.EnemyAbilityToString(ability))
			}
		}
		sort.Strings(abilities)
		entries = append(entries, codexMonsterEntry{ID: enemy.MonsterType.Index(), Name: enemy.Name, Strength: enemy.Strength, Dexterity: enemy.Dexterity, Intelligence: enemy.Intelligence, Armour: enemy.Armour, Damage: enemy.Damage, HitPoints: enemy.HitPoints, MaximumPerMap: enemy.MaxPerMap, Treasure: enemy.TreasureNumber, AttackRange: enemy.AttackRange, SpriteIndex: int(enemy.KeyFrameTile.Index), Abilities: abilities, Water: enemy.IsWaterEnemy, Sand: enemy.IsSandEnemy, Flies: enemy.CanFlyOverWater, WallPass: enemy.CanPassThroughWalls, Immobile: enemy.DoNotMove})
	}

	return entries
}

func newCodexSpellCatalog() []codexSpellEntry {
	entries := make([]codexSpellEntry, 0, references.TotalSpellCount)
	for spell := references.Spell(0); spell < references.Spell(references.TotalSpellCount); spell++ {
		info := references.GetSpellInfo(spell)
		reagents := make([]string, 0, 8)
		for reagent := references.SulfurAsh; reagent <= references.MandrakeRoot; reagent++ {
			if references.GetSpellReagents(spell)&(1<<reagent) != 0 {
				reagents = append(reagents, reagentName(reagent))
			}
		}
		audit := spellAuditCatalog[spell]
		entries = append(entries, codexSpellEntry{
			ID: int(spell), Circle: info.Circle, Name: info.Name, Targeting: info.Targeting.String(), Context: info.Context.String(), Description: info.Description, Reagents: reagents,
			Status: audit.Status, OriginalBehavior: audit.OriginalBehavior, ReduxBehavior: audit.ReduxBehavior,
			OriginalTargeting: audit.OriginalTargeting, ReduxTargeting: audit.ReduxTargeting,
			ActorVisual: audit.ActorVisual, ScreenVisual: audit.ScreenVisual, StatusIndicator: audit.StatusIndicator,
			TurnOrder: audit.TurnOrder, CombatEvents: audit.CombatEvents, AreaEffect: audit.AreaEffect,
			Failure: audit.Failure, Formula: audit.Formula, Audio: audit.Audio,
			KnownIssues: audit.KnownIssues, Opportunities: audit.Opportunities,
			CodeReferences: audit.CodeReferences, OriginalReferences: audit.OriginalReferences,
		})
	}

	return entries
}

func newCodexTileCatalog(tiles *references.Tiles, looks *references.LookReferences) []codexTileEntry {
	entries := make([]codexTileEntry, 0, len(*tiles))
	for index, tile := range *tiles {
		annotation := debugging.TileAtlasAnnotation(index)
		entries = append(entries, codexTileEntry{ID: int(index), Name: tile.Name, Description: annotation.Description, LookDescription: looks.GetTileLookDescription(index), Speed: tile.SpeedFactor, Light: tile.LightEmission, AnimationFrames: annotation.TotalAnimationFrames, Animated: annotation.IsPartOfAnimation, BlocksLight: tile.BlocksLight, Enemy: annotation.IsEnemy, NPC: annotation.IsNPC, Building: annotation.IsBuilding, Walk: tile.IsPassable(references.NoPartyVehicle), Horse: tile.IsPassable(references.HorseVehicle), Skiff: tile.IsPassable(references.SkiffVehicle), Frigate: tile.IsPassable(references.FrigateVehicle), Carpet: tile.IsPassable(references.CarpetVehicle), NPCPassable: tile.IsPassable(references.NPC)})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })

	return entries
}

func newCodexLocationCatalog(gameReferences *references.GameReferences) []codexLocationEntry {
	entries := make([]codexLocationEntry, 0, 40)
	worldLocations := gameReferences.LocationReferences.WorldLocations
	for location := references.Moonglow; location <= references.Doom; location++ {
		world, exists := worldLocations.LargeMapLocationPositions[location]
		entry := codexLocationEntry{ID: int(location), Name: location.String(), MapType: mapTypeName(location.GetMapType()), Floor: 0}
		if exists {
			entry.WorldX, entry.WorldY, entry.Floor = int(world.Position.X), int(world.Position.Y), int(world.Floor)
		}
		entries = append(entries, entry)
	}

	return entries
}

func newCodexNPCCatalog(gameReferences *references.GameReferences) ([]codexNPCEntry, error) {
	entries := make([]codexNPCEntry, 0, len(gameReferences.NPCReferences.Npcs))
	for _, npc := range gameReferences.NPCReferences.Npcs {
		times, ai, positions := make([]string, 0, 3), make([]string, 0, 3), make([]string, 0, 3)
		for index := range npc.Schedule.Ai {
			times = append(times, strconv.Itoa(int(npc.Schedule.Time[index])))
			ai = append(ai, strconv.Itoa(int(npc.Schedule.Ai[index])))
			positions = append(positions, fmt.Sprintf("%d,%d,%d", npc.Schedule.X[index], npc.Schedule.Y[index], npc.Schedule.Floor[index]))
		}
		entry := codexNPCEntry{LocationID: int(npc.Location), Location: npc.Location.String(), Type: fmt.Sprintf("0x%02X", npc.GetNPCType()), DialogNumber: int(npc.DialogNumber), SpriteIndex: int(npc.GetSpriteIndex()), Times: times, AI: ai, Positions: positions, ConversationKind: "No response", ConversationIndex: -1}
		dialog := references.DialogNumber(npc.DialogNumber)
		if dialog.IsRegularNPC() {
			entry.ConversationKind = "Semantic conversation"
			entry.ConversationSource = conversationSourceForLocation(npc.Location)
			entry.ConversationIndex = int(dialog) - 1
			program, exists := gameReferences.ConversationCatalog.Original(semantic.OriginalSource(entry.ConversationSource), int(dialog))
			if exists {
				definition, err := program.Definition()
				if err != nil {
					return nil, fmt.Errorf("loading NPC conversation %s/%d: %w", entry.ConversationSource, dialog, err)
				}
				entry.HasConversation = true
				entry.ConversationName = semanticBodyText(definition.Name)
				entry.ConversationDescription = semanticBodyText(definition.Description)
			} else {
				entry.ConversationKind = "Missing semantic conversation"
			}
		} else if dialog.IsSpecial() {
			entry.ConversationKind = dialog.GetMerchantType()
			if entry.ConversationKind == "" {
				entry.ConversationKind = "Special dialogue"
			}
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func conversationSourceForLocation(location references.Location) string {
	switch references.GetMapMasterFromLocation(location) {
	case references.Castle:
		return "castle"
	case references.Towne:
		return "towne"
	case references.Dwelling:
		return "dwelling"
	case references.Keep:
		return "keep"
	default:
		return ""
	}
}

func newCodexQuestCatalog(items *references.InventoryItemReferences) []codexQuestEntry {
	entries := make([]codexQuestEntry, 0, references.NumberOfQuestItems)
	for _, item := range references.QuestItemCatalog() {
		itemReference := items.GetReferenceByItem(item)
		entries = append(entries, codexQuestEntry{ID: item.ID(), Name: items.GetQuestItemDisplayName(item), PickupMessage: items.GetQuestItemPickupMessage(item), Description: itemReference.ItemDescription, SpriteIndex: int(items.GetQuestItemSpriteIndex(item))})
	}

	return entries
}

func newCodexSearchLocationCatalog() []codexSearchLocationEntry {
	locations := references.GetAllSearchLocations()
	entries := make([]codexSearchLocationEntry, 0, len(locations))
	for _, location := range locations {
		decoded := references.DecodeSearchLocationItem(location)
		repeatPolicy := "one_time"
		requirements := make([]string, 0, 2)
		switch {
		case location.Requirements.RequiresNoKeys:
			repeatPolicy = "conditional"
			requirements = append(requirements, "no_keys", "no_object_at_location")
		case location.Requirements.IsSkullKeys:
			repeatPolicy = "daily"
			requirements = append(requirements, "different_day_from_last_skull_keys")
		case location.Requirements.RequiresNoGlassSwords:
			repeatPolicy = "conditional"
			requirements = append(requirements, "no_glass_sword", "no_object_at_location")
		case location.Requirements.IsRepeatable:
			repeatPolicy = "repeatable"
		}
		locationID := references.Location(location.MapNumber)
		entries = append(entries, codexSearchLocationEntry{
			Index: location.Index, LocationID: location.MapNumber, LocationName: locationID.String(),
			MapType: mapTypeName(locationID.GetMapType()), Floor: location.Level, X: location.X, Y: location.Y,
			RawObjectType: int(decoded.RawObjectType), RawPayload: int(decoded.RawPayload), Classification: decoded.Classification,
			Category: decoded.Category, ItemID: decoded.ItemID, ItemName: decoded.ItemName, PickupQuantity: decoded.PickupQuantity,
			QuantitySemantics: decoded.QuantitySemantics, RepeatPolicy: repeatPolicy, Requirements: requirements,
			Suspicious: decoded.Suspicious, EncodingNote: decoded.EncodingNote,
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Index < entries[j].Index })

	return entries
}

func newCodexConsumableCatalog() []codexConsumableEntry {
	entries := []codexConsumableEntry{
		{Kind: "potion", ID: 0, Name: "Blue", OriginalBehavior: "Awakens one sleeping party member.", ReduxBehavior: "Awakens the confirmed field target or active combatant.", OriginalContext: "Field and combat", ReduxContext: "Field and combat", OriginalTarget: "Selected member in the field; current acting member in combat", ReduxTarget: "Confirmed field party member; current combat actor", OriginalConsumptionTiming: "Before target selection, random roll, and validation", ReduxConsumptionTiming: "Field: after target confirmation; combat: before random roll and effect", Support: "complete", Deviation: "Field cancellation preserves the potion rather than consuming before the target prompt."},
		{Kind: "potion", ID: 1, Name: "Yellow", OriginalBehavior: "Heals 1d30 HP.", ReduxBehavior: "Heals the confirmed field target or active combatant by 1d30 HP.", OriginalContext: "Field and combat", ReduxContext: "Field and combat", OriginalTarget: "Selected member in the field; current acting member in combat", ReduxTarget: "Confirmed field party member; current combat actor", OriginalConsumptionTiming: "Before target selection, random roll, and validation", ReduxConsumptionTiming: "Field: after target confirmation; combat: before random roll and effect", Support: "complete", Deviation: "Field cancellation preserves the potion rather than consuming before the target prompt."},
		{Kind: "potion", ID: 2, Name: "Red", OriginalBehavior: "Cures poison.", ReduxBehavior: "Cures the confirmed field target or active combatant.", OriginalContext: "Field and combat", ReduxContext: "Field and combat", OriginalTarget: "Selected member in the field; current acting member in combat", ReduxTarget: "Confirmed field party member; current combat actor", OriginalConsumptionTiming: "Before target selection, random roll, and validation", ReduxConsumptionTiming: "Field: after target confirmation; combat: before random roll and effect", Support: "complete", Deviation: "Field cancellation preserves the potion rather than consuming before the target prompt."},
		{Kind: "potion", ID: 3, Name: "Green", OriginalBehavior: "Poisons a good-status party member.", ReduxBehavior: "Poisons the confirmed field target or active combatant when effective.", OriginalContext: "Field and combat", ReduxContext: "Field and combat", OriginalTarget: "Selected member in the field; current acting member in combat", ReduxTarget: "Confirmed field party member; current combat actor", OriginalConsumptionTiming: "Before target selection, random roll, and validation", ReduxConsumptionTiming: "Field: after target confirmation; combat: before random roll and effect", Support: "complete", Deviation: "Field cancellation preserves the potion rather than consuming before the target prompt."},
		{Kind: "potion", ID: 4, Name: "Orange", OriginalBehavior: "Causes sleep; random roll 0 always forces this Orange effect, while roll 1 chooses a random potion ID 0-7.", ReduxBehavior: "Puts the confirmed field target or active combatant to sleep when effective.", OriginalContext: "Field and combat", ReduxContext: "Field and combat", OriginalTarget: "Selected/current party member", ReduxTarget: "Confirmed field party member; current combat actor", OriginalConsumptionTiming: "Before target selection, random roll, and validation", ReduxConsumptionTiming: "Field: after target confirmation; combat: before random roll and effect", Support: "complete", Deviation: "Field cancellation preserves the potion rather than consuming before the target prompt."},
		{Kind: "potion", ID: 5, Name: "Purple", OriginalBehavior: "Transforms the current combatant into a rat; no noticeable field effect.", ReduxBehavior: "Transforms the active combatant into a rat; field use reports no noticeable effect.", OriginalContext: "Combat effect; field has no noticeable effect", ReduxContext: "Field and combat", OriginalTarget: "Current acting combatant", ReduxTarget: "Confirmed field party member; current combat actor", OriginalConsumptionTiming: "Before target selection, random roll, and validation", ReduxConsumptionTiming: "Field: after target confirmation; combat: before random roll and effect", Support: "complete", Deviation: "Field cancellation preserves the potion rather than consuming before the target prompt."},
		{Kind: "potion", ID: 6, Name: "Black", OriginalBehavior: "Makes the current combatant invisible; no noticeable field effect.", ReduxBehavior: "Makes the active combatant invisible; field use reports no noticeable effect.", OriginalContext: "Combat effect; field has no noticeable effect", ReduxContext: "Field and combat", OriginalTarget: "Current acting combatant", ReduxTarget: "Confirmed field party member; current combat actor", OriginalConsumptionTiming: "Before target selection, random roll, and validation", ReduxConsumptionTiming: "Field: after target confirmation; combat: before random roll and effect", Support: "complete", Deviation: "Field cancellation preserves the potion rather than consuming before the target prompt."},
		{Kind: "potion", ID: 7, Name: "White", OriginalBehavior: "Provides X-ray vision outside dungeons.", ReduxBehavior: "Provides X-ray vision in the field; combat use reports no noticeable effect.", OriginalContext: "Non-dungeon field maps", ReduxContext: "Field and combat", OriginalTarget: "Party view", ReduxTarget: "Confirmed field party member; current combat actor", OriginalConsumptionTiming: "Before target selection, random roll, and validation", ReduxConsumptionTiming: "Field: after target confirmation; combat: before random roll and effect", Support: "complete", Deviation: "Field cancellation preserves the potion rather than consuming before the target prompt."},
		{Kind: "scroll", ID: 0, Name: "Vas Lor", OriginalBehavior: "Creates magical light for 240 game minutes.", ReduxBehavior: "Activates independent magical light for 240 gameplay minutes without replacing activeSpell.", OriginalContext: "Any map", ReduxContext: "Any map", OriginalTarget: "Party surroundings", ReduxTarget: "Party surroundings", OriginalConsumptionTiming: "Before all validation and effect execution", ReduxConsumptionTiming: "After effect dispatch succeeds", OriginalDuration: "240 game minutes", ReduxDuration: "240 gameplay minutes", Support: "complete", Deviation: "Redux delays consumption until effect dispatch succeeds."},
		{Kind: "scroll", ID: 1, Name: "Rel Hur", OriginalBehavior: "Prompts for direction, then changes wind on any non-dungeon map, including towns.", ReduxBehavior: "Changes wind only on the large map after direction selection.", OriginalContext: "Non-dungeon maps, including towns", ReduxContext: "Large map only", OriginalTarget: "Chosen direction", ReduxTarget: "Chosen direction", OriginalConsumptionTiming: "Before direction prompt and location validation", ReduxConsumptionTiming: "After direction prompt and successful wind change", Support: "partial", Deviation: "Redux excludes towns and delays consumption until success."},
		{Kind: "scroll", ID: 2, Name: "In Sanct", OriginalBehavior: "Adds +3 protection for 100 duration units in combat.", ReduxBehavior: "Activates the shared 100-turn Protection spell and applies +3 combat defense.", OriginalContext: "Combat", ReduxContext: "Any map; combat applies the defense bonus", OriginalTarget: "Party", ReduxTarget: "Party", OriginalConsumptionTiming: "Before all validation and effect execution", ReduxConsumptionTiming: "After effect dispatch succeeds", OriginalDuration: "+3 protection / 100", ReduxDuration: "+3 protection / 100 turns", Support: "complete", Deviation: "Redux permits field activation and delays consumption until effect dispatch succeeds."},
		{Kind: "scroll", ID: 3, Name: "In An", OriginalBehavior: "Negates combat magic for 20 duration units.", ReduxBehavior: "Activates a 20-turn Negate Magic effect enforced by combat casting and monster magic paths.", OriginalContext: "Combat", ReduxContext: "Any map; combat enforcement is active", OriginalTarget: "Combat", ReduxTarget: "Global activeSpell effect", OriginalConsumptionTiming: "Before all validation and effect execution", ReduxConsumptionTiming: "After effect dispatch succeeds", OriginalDuration: "20", ReduxDuration: "20 turns", Support: "complete", Deviation: "Redux permits field activation and delays consumption until effect dispatch succeeds."},
		{Kind: "scroll", ID: 4, Name: "In Quas Wis", OriginalBehavior: "Shows an overhead view on non-combat maps, including towns and dungeons.", ReduxBehavior: "Shows the aligned gem-style surface view on overworld/town maps and the shared symbolic flood-fill in dungeons.", OriginalContext: "All non-combat maps", ReduxContext: "All non-combat maps", OriginalTarget: "Current area", ReduxTarget: "Current area", OriginalConsumptionTiming: "Before all validation and effect execution", ReduxConsumptionTiming: "Surface after location validation; dungeon after selection validation and before the modal opens", Support: "complete", Deviation: "Consumption follows successful validation; the dungeon shell adds responsive presentation aids without revealing extra state."},
		{Kind: "scroll", ID: 5, Name: "Kal Xen Corp", OriginalBehavior: "Summons a safe charmed daemon in combat.", ReduxBehavior: "Combat Use summons a guaranteed-loyal daemon with placement and queue integration.", OriginalContext: "Combat", ReduxContext: "Combat", OriginalTarget: "Open combat position", ReduxTarget: "Random legal combat position", OriginalConsumptionTiming: "Before all validation and summon attempt", ReduxConsumptionTiming: "Before placement; consumed even when no room exists", Support: "complete", Deviation: "Prepared casting retains an INT binding check; the scroll guarantees loyalty."},
		{Kind: "scroll", ID: 6, Name: "In Mani Corp", OriginalBehavior: "Resurrects a selected dead member, then recalculates MP, XP, level, and stats.", ReduxBehavior: "Confirmed dead targets receive the shared original progression recalculation and return at 1 HP; confirmed living targets consume the scroll with no effect.", OriginalContext: "Non-combat maps", ReduxContext: "Non-combat maps", OriginalTarget: "Selected dead party member", ReduxTarget: "Any confirmed active party member", OriginalConsumptionTiming: "Up front, before target and validation", ReduxConsumptionTiming: "After target confirmation and before target validation", Support: "complete", Deviation: "Cancellation preserves the scroll rather than consuming before the target prompt."},
		{Kind: "scroll", ID: 7, Name: "An Tym", OriginalBehavior: "Negates time for 20 duration units; direct use in Stonegate or Doom consumes the scroll with no effect.", ReduxBehavior: "Tracks time stop for 20 turns; direct use in Stonegate or Doom consumes the scroll and reports no effect, while combat use resolves normally regardless of origin.", OriginalContext: "Any map, with Stonegate/Doom no-effect exceptions outside combat", ReduxContext: "Any map, with Stonegate/Doom no-effect exceptions outside combat", OriginalTarget: "World time", ReduxTarget: "Tracked world effect", OriginalConsumptionTiming: "Before Stonegate/Doom validation, so both consume", ReduxConsumptionTiming: "Surface use consumes after effect dispatch starts; dungeon and combat use consume before the effect; Stonegate/Doom no-effect routes still consume", OriginalDuration: "20", ReduxDuration: "20 turns tracked", Support: "partial", Deviation: "Time stop is tracking-only, and surface consumption remains delayed until effect dispatch starts."},
	}

	return entries
}

func newCodexSaveFormat() []codexSaveField {
	return []codexSaveField{{Offset: "0x0000", Size: "2", Name: "Version marker", Description: "Save file version or marker."}, {Offset: "0x0002", Size: "512", Name: "Character records", Description: "16 fixed 32-byte character records."}, {Offset: "0x0202", Size: "2", Name: "Food", Description: "Party food supply."}, {Offset: "0x0204", Size: "2", Name: "Gold", Description: "Party gold."}, {Offset: "0x021A", Size: "48", Name: "Equipment inventory", Description: "Equipment quantities in stable item-ID order."}, {Offset: "0x024A", Size: "48", Name: "Prepared spells", Description: "Spell quantities in stable spell-ID order."}, {Offset: "0x028A", Size: "32", Name: "Moonstones", Description: "Four coordinate arrays for eight moonstones."}, {Offset: "0x02CE", Size: "52", Name: "Time and map state", Description: "Calendar, moon, map, position, and transient world fields."}, {Offset: "0x0326", Size: "4", Name: "Quest flags", Description: "In-progress and completed quest bitfields."}}
}

func reagentName(reagent references.Reagent) string {
	return []string{"Sulfurous Ash", "Ginseng", "Garlic", "Spider Silk", "Blood Moss", "Black Pearl", "Nightshade", "Mandrake Root"}[reagent]
}

func mapTypeName(mapType references.GeneralMapType) string {
	return []string{"Large Map", "Small Map", "Combat", "Dungeon", "Camping"}[mapType]
}
