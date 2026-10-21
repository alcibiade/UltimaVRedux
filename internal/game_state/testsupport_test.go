package game_state

import (
	"fmt"
	"slices"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// ValidateAndConsumePartyTargetSpellResources commits a prepared character
// spell after final target confirmation, with the same validation as other routes.
func (g *GameState) ValidateAndConsumePartyTargetSpellResources(request CastRequest) SpellValidationResult {
	profile := references.GetSpellTargetingProfile(request.Spell, g.GetCurrentSpellContext())
	if profile.Selection != references.TargetPartyMember {
		panic("party-target resource commit called for another targeting mode")
	}

	return g.ValidateAndConsumeSpellResources(request)
}

// ActionFireCombatMap rejects Fire during combat, as required by the command context.
func (g *GameState) ActionFireCombatMap(_ references.Direction) bool {
	g.SystemCallbacks.Message.AddRowStr("What?")

	return false
}

// ActionFireDungeonMap rejects Fire in dungeon corridors.
func (g *GameState) ActionFireDungeonMap(_ references.Direction) bool {
	g.SystemCallbacks.Message.AddRowStr("What?")

	return false
}

// ActionGetCombatMap delegates collection and reports whether combat handled the request.
func (g *GameState) ActionGetCombatMap(direction references.Direction) bool {
	return g.GetCombatItem(direction).Handled
}

// ProcessBedRestProvisionHour consumes a meal or applies one starvation consequence.
func (g *GameState) ProcessBedRestProvisionHour(hour int) BedRestProvisionResult {
	return g.processBedRestProvisionHour(hour, g.cachePlayerUpdateStatuses().Diners)
}

// ActionJimmySmallMap attempts a town unlock with the eligible selected character.
func (g *GameState) ActionJimmySmallMap(direction references.Direction) bool {
	if !g.PartyState.Inventory.Provisions.Keys.HasSome() {
		return g.actionJimmySurface(direction, nil)
	}
	selectedCharacter := g.SelectCharacterForJimmy()
	if selectedCharacter == nil {
		g.SystemCallbacks.Message.AddRowStr("No characters available!")

		return false
	}

	return g.ActionJimmySmallMapWithCharacter(direction, selectedCharacter)
}

// ActionJimmyLargeMap attempts an outdoor unlock with the eligible selected character.
func (g *GameState) ActionJimmyLargeMap(direction references.Direction) bool {
	if !g.PartyState.Inventory.Provisions.Keys.HasSome() {
		return g.actionJimmySurface(direction, nil)
	}
	selectedCharacter := g.SelectCharacterForJimmy()
	if selectedCharacter == nil {
		g.SystemCallbacks.Message.AddRowStr("No characters available!")

		return false
	}

	return g.actionJimmySurface(direction, selectedCharacter)
}

// ActionJimmyCombatMap reports whether the target was handled, including failed attempts.
func (g *GameState) ActionJimmyCombatMap(direction references.Direction) bool {
	handled, _ := g.ActionJimmyCombatMapWithDisarm(direction)

	return handled
}

func (g *GameState) ActionReadySmallMap() bool {
	g.SystemCallbacks.Message.ShowCommandPrompt("Ready...")
	request, outcome := g.BeginReadyEquipment(false)
	if outcome == ReadyBeginDialog {
		g.SystemCallbacks.Flow.Items.ShowReadyDialog(request)
	}

	return outcome != ReadyBeginRejected
}

// ConsumeDungeonViewScroll commits one In Quas Wis scroll and returns the detached symbolic view.
func (g *GameState) ConsumeDungeonViewScroll() (DungeonViewModel, bool) {
	g.requireActiveDungeon()
	if !g.PartyState.Inventory.Scrolls.HasSome(references.ScrollInQuasWis) {
		g.SystemCallbacks.UI.ShowActionFeedback("You have none!", true)

		return DungeonViewModel{}, false
	}
	model, err := g.CurrentDungeonViewModel()
	if err != nil {
		panic(err)
	}
	g.PartyState.Inventory.Scrolls.DecrementByOne(references.ScrollInQuasWis)
	g.SystemCallbacks.Screen.RefreshInventoryDisplay()

	return model, true
}

// ActionUseDungeonMap is a legacy direction handler, unused by the dungeon item picker.
func (g *GameState) ActionUseDungeonMap(_ references.Direction) bool {
	g.SystemCallbacks.Message.AddRowStr("Nothing to use!")

	return false
}

// CommitSkullKeyUnlock atomically revalidates and unlocks the reviewed door.
func (g *GameState) CommitSkullKeyUnlock(review SkullKeyUnlockReview) SkullKeyUnlockResult {
	return g.commitSkullKeyUnlock(review, true)
}

func (g *GameState) ActionViewSmallMap() bool {
	if !g.ActionViewSurface() {
		return false
	}

	// Trigger the presentation-only gem view workflow.
	g.SystemCallbacks.Flow.Observation.ShowGemViewInteraction(nil)

	return true
}

func (g *GameState) ActionViewLargeMap() bool {
	if !g.ActionViewSurface() {
		return false
	}

	// Trigger the presentation-only gem view workflow.
	g.SystemCallbacks.Flow.Observation.ShowGemViewInteraction(nil)

	return true
}

// GetAllShadowlordPositions returns a map of town locations to their assigned Shadowlords.
// Only includes towns that have a living Shadowlord assigned.
func (g *GameState) GetAllShadowlordPositions() map[references.Location]references.Shadowlord {
	positions := make(map[references.Location]references.Shadowlord)

	for loc := ShadowlordTownMinLocation; loc <= ShadowlordTownMaxLocation; loc++ {
		lord := g.GetShadowlordForLocation(references.Location(loc))
		if lord != references.NoShadowlord {
			positions[references.Location(loc)] = lord
		}
	}

	return positions
}

// GetTotalSearchLocations returns the total number of search locations in the system
func (aslm *AuthenticSearchLocationManager) GetTotalSearchLocations() int {
	return len(references.GetAllSearchLocations())
}

// GetFoundLocationsByMap returns a map of found locations grouped by map location
func (aslm *AuthenticSearchLocationManager) GetFoundLocationsByMap() map[references.Location][]int {
	result := make(map[references.Location][]int)
	allLocations := references.GetAllSearchLocations()

	for _, loc := range allLocations {
		if aslm.isAlreadyFound(loc.Index) {
			mapLoc := references.Location(loc.MapNumber)
			result[mapLoc] = append(result[mapLoc], loc.Index)
		}
	}

	return result
}

// CastCombatInFlamHur applies the original flame-wind cone. Each occupied tile
// takes direct random damage from 0 through 30 without an armor reduction.
// original spell projection rules and original command rules establish the inclusive source contract;
// the external RNG implementation is unavailable for binary confirmation.
// See ref:SPELL-0003.
func (g *GameState) CastCombatInFlamHur(casterIdx int, direction references.Direction) []references.Position {
	return g.castCommittedCombatCone(references.InFlamHur, casterIdx, direction)
}

// CastCombatInVasGravCorp applies the original energy-vortex cone. Targets
// first receive an intelligence save; spell-immune and immortal monsters
// cannot be killed. A failed save deals the original instant-death value.
func (g *GameState) CastCombatInVasGravCorp(casterIdx int, direction references.Direction) []references.Position {
	return g.castCommittedCombatCone(references.InVasGravCorp, casterIdx, direction)
}

// CastCombatInVasPorYlem applies the original earthquake to hostile combatants
// whose DEX does not exceed their individual D30 roll.
func (g *GameState) CastCombatInVasPorYlem(casterIdx int) []references.Position {
	return g.CastCombatEarthquakeResult(casterIdx).processedPositions()
}

// TurnDelay returns the turn delay (36 - DEX for initiative order).
func (m *CombatMonster) TurnDelay() int { return m.turnDelay }

// SetHasActed marks whether the monster has acted this round.
func (m *CombatMonster) SetHasActed(acted bool) {
	m.hasActed = acted
}

// GetActiveHostileMonsters returns active monsters loyal to the opposing side.
func (m *CombatMonsterManager) GetActiveHostileMonsters() []*CombatMonster {
	hostile := make([]*CombatMonster, 0)
	for _, monster := range m.monsters {
		if monster.IsActive() && !monster.IsDead() && monster.IsHostile() {
			hostile = append(hostile, monster)
		}
	}

	return hostile
}

// GetActivePartyLoyalMonsters returns active monsters that support the party.
func (m *CombatMonsterManager) GetActivePartyLoyalMonsters() []*CombatMonster {
	partyLoyal := make([]*CombatMonster, 0)
	for _, monster := range m.monsters {
		if monster.IsActive() && !monster.IsDead() && monster.IsPartyLoyal() {
			partyLoyal = append(partyLoyal, monster)
		}
	}

	return partyLoyal
}

// CountActivePartyLoyal returns the number of active monsters that support the party.
func (m *CombatMonsterManager) CountActivePartyLoyal() int {
	count := 0
	for _, monster := range m.monsters {
		if monster.IsActive() && !monster.IsDead() && monster.IsPartyLoyal() {
			count++
		}
	}

	return count
}

// GetMonstersAtPosition returns every active monster at pos in stable manager order.
func (m *CombatMonsterManager) GetMonstersAtPosition(pos references.Position) []*CombatMonster {
	monsters := make([]*CombatMonster, 0)
	for _, monster := range m.monsters {
		if monster.IsActive() && !monster.IsDead() && monster.Position() == pos {
			monsters = append(monsters, monster)
		}
	}

	return monsters
}

// GetMonsterIndicesAtPosition returns stable manager indices for every active monster at pos.
func (m *CombatMonsterManager) GetMonsterIndicesAtPosition(pos references.Position) []int {
	indices := make([]int, 0)
	for index, monster := range m.monsters {
		if monster.IsActive() && !monster.IsDead() && monster.Position() == pos {
			indices = append(indices, index)
		}
	}

	return indices
}

// GetEntry returns the queue entry for a specific combatant.
// Returns nil if not found.
// GetEntry returns the normal-slot entry retained for existing PC/monster callers.
func (q *CombatTurnQueue) GetEntry(entityIndex int, isPC bool) *TurnQueueEntry {
	return q.GetEntryByIdentity(CombatTurnIdentity{IsPartyMember: isPC, EntityIndex: entityIndex})
}

// ActiveEntryCount returns the number of entries that are still active.
func (q *CombatTurnQueue) ActiveEntryCount() int {
	count := 0
	for _, entry := range q.entries {
		if entry.status == turnQueueActive {
			count++
		}
	}

	return count
}

// IsPlayerNearDungeonEntrance checks if the player is within proximity of a dungeon entrance
// Returns the dungeon location and true if near, or zero location and false if not
func (gs *GameState) IsPlayerNearDungeonEntrance(proximityRange int) (references.Location, bool) {
	playerPos := gs.MapState.PlayerLocation.Position

	// Get current dungeon entrance coordinates from authoritative source
	dungeonCoords := gs.GetDungeonEntranceCoordinates()

	for dungeon, entrancePos := range dungeonCoords {
		if !DungeonEntranceExistsOnFloor(dungeon, gs.MapState.PlayerLocation.Floor) {
			continue
		}

		// Calculate distance using Manhattan distance
		dx := absInt(int(playerPos.X) - int(entrancePos.X))
		dy := absInt(int(playerPos.Y) - int(entrancePos.Y))

		if dx <= proximityRange && dy <= proximityRange {
			return dungeon, true
		}
	}

	return references.Location(0), false
}

// TriggerEarthquakeManually commits earthquake damage without the random eligibility roll.
func (e *EarthquakeSystem) TriggerEarthquakeManually(partyState *party_state.PartyState) EarthquakeResult {
	return e.triggerEarthquake(partyState)
}

// ProcessWindOnTurnAdvancement should be called during turn processing for automatic wind changes
func (g *GameState) ProcessWindOnTurnAdvancement() {
	if g.windSystem == nil {
		// System corruption: wind system should be initialized in GameState constructor
		panic("ProcessWindOnTurnAdvancement: windSystem is nil")
	}

	g.windSystem.UpdateWindTick()
}

// AlterEquipmentLoadout validates and atomically applies a partial character loadout.
func (g *GameState) AlterEquipmentLoadout(
	input EquipmentLoadoutAlteration,
) (EquipmentLoadoutAlterationResult, error) {
	result := EquipmentLoadoutAlterationResult{
		CharacterSlot: input.CharacterSlot,
		CharacterName: "",
		Changes:       nil,
	}
	batch, err := g.AlterEquipmentLoadouts(EquipmentLoadoutBatchAlteration{Targets: []EquipmentLoadoutAlteration{input}})
	if err != nil {
		return result, err
	}
	batch.Targets[0].Changes = batch.Changes

	return batch.Targets[0], nil
}

// SetPartyMemberInvisible updates encounter-local party invisibility.
func (combatState *CombatState) SetPartyMemberInvisible(memberIndex int, invisible bool) {
	if memberIndex < 0 || memberIndex >= party_state.MaxActiveParty {
		panic("combat party invisibility index out of range")
	}
	combatState.PartyInvisible[memberIndex] = invisible
}

// QuoteHorsePurchase returns the current canonical price and delivery fingerprint.
func (g *GameState) QuoteHorsePurchase(session MerchantSession) (HorsePurchaseQuote, error) {
	state, err := g.GetHorseShopState(session)
	if err != nil {
		return HorsePurchaseQuote{}, err
	}

	return state.Quote, nil
}

// QuoteInnPickup returns one current stored-guest bill and projection.
func (g *GameState) QuoteInnPickup(
	session MerchantSession,
	character InnCharacterIdentity,
) (InnGuestRow, error) {
	state, err := g.GetInnShopState(session)
	if err != nil {
		return InnGuestRow{}, err
	}
	for _, row := range state.GuestRows {
		if row.Identity == character {
			return row, nil
		}
	}

	return InnGuestRow{}, fmt.Errorf("%w: guest %q is unavailable", ErrInvalidInnRequest, character.Name)
}

// MerchantID returns the merchant's stable canonical identity.
func (s MerchantSession) MerchantID() references.MerchantID { return s.merchant.ID }

// PricingCharacterName returns the captured pricing character's stable display identity.
func (s MerchantSession) PricingCharacterName() string {
	return s.buyerName
}

// BeginMerchantSessionByID resolves a stable merchant identity to the exact
// loaded NPC, then delegates all production validation to BeginMerchantSession.
func (g *GameState) BeginMerchantSessionByID(id references.MerchantID) (MerchantSession, error) {
	_, npc, err := g.resolveLoadedMerchantByID(id)
	if err != nil {
		return MerchantSession{}, err
	}

	return g.BeginMerchantSession(npc)
}

// PrepareMoongateTravel checks moongate status and prepares for travel.
// Returns a captured travel route. Does NOT start any event queues.
// The caller is responsible for orchestrating the visual sequence.
func (g *GameState) PrepareMoongateTravel() MoongateTravelResult {
	result := g.ReviewMoongateTravel()
	if result.Success || result.FailedButActivated {
		g.ApplyMoongateTravelEffects(result.Success)
	}

	return result
}

// GetMoonstoneAtPosition finds moonstone buried at specific coordinates
// Returns moonstone index if found, -1 if no moonstone at position
// Only returns unfound moonstones (for search discovery)
// NOTE: This does NOT check map - use GetMoonstoneAtPositionOnMap for proper search
func (ms *MoonstoneSystem) GetMoonstoneAtPosition(x, y uint8) int {
	for i, moonstone := range ms.Moonstones {
		if moonstone.X == x && moonstone.Y == y && !moonstone.IsFound && moonstone.IsBuried() {
			return i
		}
	}

	return -1
}

// GetFoundMoonstoneAtPosition finds a moonstone in FoundNotCollected state at specific coordinates
// Returns moonstone index if found, -1 if no found moonstone at position
func (ms *MoonstoneSystem) GetFoundMoonstoneAtPosition(x, y uint8) int {
	for i, moonstone := range ms.Moonstones {
		if moonstone.X == x && moonstone.Y == y && moonstone.IsFoundNotCollected() {
			return i
		}
	}

	return -1
}

// GetMoonstoneIndexForMoonPhase returns the moonstone index for current moon phase - delegation to MoonstoneSystem
func (g *GameState) GetMoonstoneIndexForMoonPhase(phase int) int {
	if g.moonstoneSystem == nil {
		g.moonstoneSystem = NewMoonstoneSystem()
	}

	return g.moonstoneSystem.GetMoonstoneIndexForMoonPhase(phase)
}

// GetMoongateVisibilityLevel converts the authoritative stage to 0.0 through 1.0.
func (g *GameState) GetMoongateVisibilityLevel() float32 {
	return float32(g.moongateStage) / float32(MoongateStageFull)
}

// HasActiveMoongateAtPosition checks stage and placed moonstone source state.
func (g *GameState) HasActiveMoongateAtPosition(pos *references.Position) bool {
	if g.moonstoneSystem == nil {
		g.moonstoneSystem = NewMoonstoneSystem()
	}

	location := g.MapState.PlayerLocation

	return g.moongateStage > MoongateStageHidden &&
		g.moonstoneSystem.CanCreateMoongateAt(location.Location, location.Floor, *pos)
}

// CheckPirateAttack handles aligned pirate fire at a distance below
// four tiles. A misaligned pirate retains one warning/rotation phase.
// Already aligned pirates fire in descending object-slot order within
// the phase; cardinal contact has its separate earlier route. ref:SHIP-0005
func (g *GameState) CheckPirateAttack() bool {
	// Only check on large maps
	if g.MapState.PlayerLocation.Location.GetMapType() != references.LargeMapType {
		return false
	}

	// Get all pirate ships using generic lookup
	npcs := g.CurrentNPCAIController.GetNpcs()
	pirates := npcs.GetEnemiesMatching(isPiratePredicate())
	playerPos := g.MapState.PlayerLocation.Position
	floor := g.MapState.PlayerLocation.Floor
	slices.SortFunc(pirates, func(left, right *map_units.NPCEnemy) int {
		return right.MapUnitDetails().NPCNum - left.MapUnitDetails().NPCNum
	})

	acted := false
	for _, pirate := range pirates {
		if pirate.IsVisible() && pirate.Floor() == floor && g.checkPirateActorAttack(pirate, playerPos) {
			acted = true
		}
	}

	return acted
}

// FirePirateCannon fires a cannon from pirate ship to player position
// See ref:SHIP-0005.
func (g *GameState) FirePirateCannon(pirate *map_units.NPCEnemy, targetPos references.Position) {
	result := g.resolvePirateCannon(pirate, targetPos)
	g.pirateAttackQueue = append(g.pirateAttackQueue, pirateAttackRequest{
		result: result,
	})
	g.startPirateAttackWorkflow()
}

// MarkKnowledgeRead authoritatively clears unread state for an existing key.
func (g *GameState) MarkKnowledgeRead(key KnowledgeKey) bool {
	g.requirePlayerKnowledge()
	entry, found := g.playerKnowledge.entries[key]
	if !found || !entry.Unread {
		return false
	}
	entry.Unread = false
	g.playerKnowledge.entries[key] = entry

	return true
}

// GetQuestItemLightRadius returns the light emission radius for a quest item at a given position
// Returns 0 if no quest item is at the position or if the item is in inventory (not visible)
func (gs *GameState) GetQuestItemLightRadius(position references.Position) int {
	hasQuestItem, questItem := gs.IsQuestItemAtPosition(position)
	if !hasQuestItem {
		return 0
	}

	// Check if quest item is visible (not in inventory)
	if !gs.IsQuestItemVisible(position, questItem) {
		return 0
	}

	// Return appropriate light radius based on item type
	if IsLordBritishArtifact(questItem) {
		return map_state.ArtifactLightDistance // Artifacts emit stronger light
	} else if IsShard(questItem) {
		return map_state.ShardLightDistance // Shards emit moderate light
	}

	return 0 // Default: no light
}

// ExecuteSearch performs a search operation with clear priority ordering
// The ORDER of method calls defines the search priority
func (ssm *SearchSystemManager) ExecuteSearch(direction references.Direction) SearchResponse {
	return ssm.ExecuteSearchWithCharacter(direction, ssm.gameState.defaultSurfaceSearchCharacterIndex())
}

// QuoteShipPurchase returns one current canonical vessel quote.
func (g *GameState) QuoteShipPurchase(
	session MerchantSession,
	vehicleType references.VehicleType,
) (ShipPurchaseQuote, error) {
	state, err := g.GetShipShopState(session)
	if err != nil {
		return ShipPurchaseQuote{}, err
	}

	switch vehicleType {
	case references.FrigateVehicle:
		return state.FrigateQuote, nil
	case references.SkiffVehicle:
		return state.SkiffQuote, nil
	case references.NoPartyVehicle, references.CarpetVehicle, references.HorseVehicle, references.NPC:
		return ShipPurchaseQuote{}, fmt.Errorf("%w: unsupported vehicle type %d", ErrInvalidShipPurchase, vehicleType)
	default:
		return ShipPurchaseQuote{}, fmt.Errorf("%w: unsupported vehicle type %d", ErrInvalidShipPurchase, vehicleType)
	}
}

// SkiffDeepWaterDamageSequence lets a workflow own a qualifying hazard inline.
// An owner that must await damage uses ThenSequence, never a child workflow plus
// a wait for its name; that remains safe even for high/critical-priority callers.
// The owner must clear transient visuals in Finally via ClearSkiffDeepWaterEffects.
func (g *GameState) SkiffDeepWaterDamageSequence(position references.Position) event_queue.Sequence {
	return func(queue *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		if !g.isSkiffDeepWaterDamageEligible(&position) {
			return queue
		}

		return queue.ThenSequence(g.skiffDeepWaterDamageSequence(func() {}))
	}
}

// OneShot configures the sound to play once and stop (default behavior)
func (b *SoundBuilder) OneShot() *SoundBuilder {
	b.mode = SoundPlaybackOneShot

	return b
}

// Volume configures the playback volume (0.0 to 1.0)
func (b *SoundBuilder) Volume(v float64) *SoundBuilder {
	if v < 0.0 {
		v = 0.0
	} else if v > 1.0 {
		v = 1.0
	}
	b.volume = v

	return b
}

// QuoteTavernMeal returns the current trusted meal charge.
func (g *GameState) QuoteTavernMeal(session MerchantSession) (TavernMealQuote, error) {
	state, err := g.GetTavernShopState(session)
	if err != nil {
		return TavernMealQuote{}, err
	}

	return state.Meal, nil
}

// QuoteTavernDrink returns the current ordinary drink quote.
func (g *GameState) QuoteTavernDrink(session MerchantSession) (TavernDrinkQuote, error) {
	state, err := g.GetTavernShopState(session)
	if err != nil {
		return TavernDrinkQuote{}, err
	}
	if state.Menu.Drink.IsWineList {
		return TavernDrinkQuote{}, fmt.Errorf("%w: merchant %q offers a wine list", ErrInvalidTavernRequest, state.Merchant.ID)
	}

	return state.Drink, nil
}

// QuoteTavernWine returns one fixed-price canonical wine quote.
func (g *GameState) QuoteTavernWine(session MerchantSession, wineIndex int) (TavernDrinkQuote, error) {
	state, err := g.GetTavernShopState(session)
	if err != nil {
		return TavernDrinkQuote{}, err
	}
	if !state.Menu.Drink.IsWineList || wineIndex < 0 || wineIndex >= len(state.Wines) {
		return TavernDrinkQuote{}, fmt.Errorf("%w: wine index %d is unavailable", ErrInvalidTavernRequest, wineIndex)
	}

	return state.Wines[wineIndex], nil
}

// QuoteTavernProvisions returns the active menu's canonical per-pack quote.
func (g *GameState) QuoteTavernProvisions(session MerchantSession) (TavernProvisionQuote, error) {
	state, err := g.GetTavernShopState(session)
	if err != nil {
		return TavernProvisionQuote{}, err
	}
	if !state.Menu.Provision.Available {
		return TavernProvisionQuote{}, fmt.Errorf("%w: provisions are unavailable", ErrInvalidTavernRequest)
	}

	return state.Provision, nil
}

// TrySpawnWhirlpool is the probabilistic single-whirlpool integration fixture
// helper. Ordinary world generation is owned by the large-map AI controller;
// its source terrain/region gates and multiple-whirlpool support are independent.
func (g *GameState) TrySpawnWhirlpool(pos references.Position) bool {
	// Only spawn on large maps
	if g.MapState.PlayerLocation.Location.GetMapType() != references.LargeMapType {
		return false
	}

	// Only allow one whirlpool at a time on the map
	if g.HasAnyWhirlpool() {
		return false
	}

	// Check if position already has a whirlpool (redundant with above but kept for safety)
	if g.hasWhirlpoolAt(pos) {
		return false
	}

	// Get tile at position
	layeredMap := g.GetCurrentLayeredMap()
	tile := layeredMap.GetTopTile(&pos)

	// Whirlpool generation requires deep-water terrain (raw tile 1) and
	// the upper-boundary result of the 0..7 spawn draw. ref:SHIP-0006
	if !tile.Is(indexes.Water1) {
		return false
	}

	// 1 in 8 spawn probability using GameState RNG for determinism
	if !g.OneInXOdds(WhirlpoolSpawnProbability) {
		return false
	}

	// Create whirlpool enemy at position
	return g.spawnWhirlpoolAt(pos)
}

// CheckWhirlpoolAdjacency checks if whirlpool is adjacent to player and triggers collision
// Called before monster movement in turn.go.
//
// Original Ultima V : outdoor contact and ranged attack checks adjacency BEFORE world creature movement
// outdoor contact and ranged attack at original world-creature rules check: (dis_x == 1 && dis_y == 0) || (dis_x == 0 && dis_y == 1)
// If adjacent, outdoor contact resolution is called which triggers whirlpool transportation
//
// Returns true if the whirlpool collision workflow was triggered.
// See ref:SHIP-0003.
func (g *GameState) CheckWhirlpoolAdjacency() bool {
	// Only check on large maps where whirlpools exist
	if g.MapState.PlayerLocation.Location.GetMapType() != references.LargeMapType {
		return false
	}

	if g.EventQueueManager.HasWorkflow("whirlpool_collision") {
		return true
	}
	playerPos := g.MapState.PlayerLocation.Position
	playerFloor := g.MapState.PlayerLocation.Floor
	// Canonical monster generation can create multiple whirlpools. Scan in
	// stable controller order rather than letting a distant first one mask contact.
	whirlpool := g.CurrentNPCAIController.GetNpcs().GetFirstEnemyMatching(func(enemy *map_units.NPCEnemy) bool {
		return enemy.Variant() == map_units.EnemyVariantWhirlpool && enemy.IsVisible() &&
			playerPos.IsNextTo(enemy.Pos(), playerFloor, enemy.Floor())
	})
	if whirlpool == nil {
		return false
	}

	return g.checkWhirlpoolActorContact(whirlpool)
}

// hasWhirlpoolAt checks if a whirlpool already exists at the given position
func (g *GameState) hasWhirlpoolAt(pos references.Position) bool {
	return g.CurrentNPCAIController.GetNpcs().HasEnemyMatching(isWhirlpoolAtPosPredicate(pos))
}

func (g *GameState) checkWhirlpoolActorContact(whirlpool *map_units.NPCEnemy) bool {
	if g.EventQueueManager.HasWorkflow("whirlpool_collision") {
		return true
	}
	whirlpoolPos := whirlpool.Pos()

	// Original outdoor contact resolution leaves an adjacent whirlpool in place when it
	// hits a party on foot. Every vehicle path consumes it before transport.
	// See ref:SHIP-0003.
	if g.PartyVehicle.GetVehicleDetails().VehicleType != references.NoPartyVehicle {
		g.RemoveWhirlpoolAt(whirlpoolPos)
	}

	// Build and start transportation queue
	q := g.EventQueueManager.Workflow("whirlpool_collision").
		Priority(event_queue.PriorityHigh).
		Sequential()
	g.BuildWhirlpoolTransportationQueue(q)
	q.Start()

	return true
}
