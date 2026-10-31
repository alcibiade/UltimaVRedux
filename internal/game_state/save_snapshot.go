package game_state

import (
	"errors"
	"fmt"
	"slices"
	"sort"

	"github.com/bradhannah/UltimaVRedux/internal/ai"
	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/environment"
	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	rng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

// ErrSaveSnapshotNotRepresentable indicates that runtime state cannot be serialized safely.
var ErrSaveSnapshotNotRepresentable = errors.New("game state is not representable as a save snapshot")

// ValidateSaveSnapshotCompatibility verifies the snapshot against this state's
// runtime references without mutating gameplay state.
func (g *GameState) ValidateSaveSnapshotCompatibility(snapshot save_snapshot.SaveSnapshot) error {
	if g == nil {
		return errors.New("active runtime references are unavailable")
	}

	return ValidateSaveSnapshotCompatibility(snapshot, g.GameReferences)
}

// ValidateSaveSnapshotCompatibility verifies that a structurally valid
// snapshot can resolve every runtime identity and map represented by the
// supplied game data without constructing a live GameState.
func ValidateSaveSnapshotCompatibility(snapshot save_snapshot.SaveSnapshot, gameReferences *references.GameReferences) error {
	if issues := save_snapshot.Validate(snapshot); len(issues) != 0 {
		return &save_snapshot.ValidationError{Issues: issues}
	}
	if err := validateSnapshotActivePlayer(snapshot); err != nil {
		return err
	}
	if gameReferences == nil || gameReferences.LocationReferences == nil || gameReferences.EnemyReferences == nil {
		return errors.New("active runtime references are unavailable")
	}
	if err := validateJournalSiteCompatibility(snapshot, gameReferences); err != nil {
		return err
	}
	if err := validatePersonalSnapshotCompatibility(snapshot, gameReferences); err != nil {
		return err
	}
	sourceFor := func(location references.Location) *references.SmallLocationReference {
		if location.IsDeedLocation() {
			return deedLocationReferenceFromSnapshot(snapshot)
		}

		return gameReferences.LocationReferences.GetLocationReference(location)
	}
	if err := validateOpenDoorSnapshotSource(snapshot.OpenDoor, snapshot.World.Position, snapshot.MapMutations, gameReferences, sourceFor); err != nil {
		return err
	}
	if err := validateSnapshotWorldCompatibility(snapshot.World, snapshot.Wind, gameReferences); err != nil {
		return err
	}
	if err := validateSnapshotMapUnits(snapshot.MapUnits, snapshot.World.Position.Location, gameReferences); err != nil {
		return err
	}
	if err := validateSnapshotItemStacks(snapshot.ItemStacks, snapshot.World.Position); err != nil {
		return err
	}
	if err := validateSnapshotMapMutations(snapshot.MapMutations, snapshot.World.Position.Location, gameReferences); err != nil {
		return err
	}
	if err := validateSnapshotDungeonDiscoveries(snapshot.Redux.DungeonDiscoveries, gameReferences); err != nil {
		return err
	}
	if snapshot.Dungeon != nil && snapshot.Dungeon.Wanderer != nil &&
		enemyReference(gameReferences.EnemyReferences, references.MonsterType(snapshot.Dungeon.Wanderer.Monster)) == nil {
		return fmt.Errorf("dungeon.wanderer: enemy reference %d is unavailable", snapshot.Dungeon.Wanderer.Monster)
	}

	return nil
}

func validateSnapshotWorldCompatibility(
	world save_snapshot.WorldSnapshot,
	wind save_snapshot.WindSnapshot,
	gameReferences *references.GameReferences,
) error {
	if err := validateSnapshotPosition(gameReferences, "world.position", world.Position); err != nil {
		return err
	}
	if err := validateSnapshotPosition(gameReferences, "world.lastLargeMapPosition", world.LastLargeMapPosition); err != nil {
		return err
	}
	lastLargeMapLocation := references.Location(world.LastLargeMapPosition.Location)
	if lastLargeMapLocation != references.BritanniaUnderworld || lastLargeMapLocation.GetMapType() != references.LargeMapType {
		return errors.New("world.lastLargeMapPosition: position is not on the large world map")
	}
	if world.ElapsedMs > save_snapshot.MaxElapsedMs || wind.NextTickMs > save_snapshot.MaxElapsedMs+environment.WindTickIntervalMs {
		return errors.New("world.elapsedMs/wind.nextTickMs: elapsed time exceeds the supported 100-year save domain")
	}
	windPhase := wind.NextTickMs - world.ElapsedMs
	if windPhase <= 0 || windPhase > environment.WindTickIntervalMs {
		return fmt.Errorf("wind.nextTickMs: tick phase must be within %d ms after world.elapsedMs", environment.WindTickIntervalMs)
	}

	return nil
}

func validateSnapshotMapUnits(
	units []save_snapshot.MapUnitSnapshot,
	activeLocation int,
	gameReferences *references.GameReferences,
) error {
	for index, unit := range units {
		if err := validateSnapshotPosition(gameReferences, fmt.Sprintf("mapUnits[%d].position", index), unit.Position); err != nil {
			return err
		}
		if unit.Vehicle != nil && unit.Vehicle.PendingDelivery && !isPendingDeliveryDockPosition(gameReferences.DockReferences, unit.Position) {
			return fmt.Errorf("mapUnits[%d].vehicle.pendingDelivery: position is not one of the four canonical docks", index)
		}
		if references.Location(unit.Position.Location).GetMapType() == references.DungeonMapType {
			return fmt.Errorf("mapUnits[%d]: dungeon occupancy must use dungeon.wanderer", index)
		}
		if references.Location(unit.Position.Location).GetMapType() == references.LargeMapType {
			if unit.World != unit.Position.Floor {
				return fmt.Errorf("mapUnits[%d]: large-map controller world must match the unit floor", index)
			}
		} else if unit.Position.Location != activeLocation || unit.World != 0 {
			return fmt.Errorf("mapUnits[%d]: only the active small-map controller (world 0) is loaded during hydration", index)
		}
		if err := validateSnapshotMapUnitIdentity(index, unit, gameReferences); err != nil {
			return err
		}
	}

	return nil
}

func validateSnapshotMapUnitIdentity(index int, unit save_snapshot.MapUnitSnapshot, gameReferences *references.GameReferences) error {
	switch unit.Kind {
	case "enemy":
		if enemyReference(gameReferences.EnemyReferences, references.MonsterType(unit.ReferenceID)) == nil {
			return fmt.Errorf("mapUnits[%d]: enemy reference %d is unavailable", index, unit.ReferenceID)
		}
		if err := validateSnapshotEnemyIdentity(index, unit); err != nil {
			return err
		}
	case "npc":
		location := gameReferences.LocationReferences.GetLocationReference(references.Location(unit.Position.Location))
		if location == nil || unit.NPCNum < 0 || unit.NPCNum >= len(*location.GetNPCReferences()) {
			return fmt.Errorf("mapUnits[%d]: NPC reference %d is unavailable", index, unit.NPCNum)
		}
		if unit.SurfaceChestNumber != nil && (*unit.SurfaceChestNumber < 0 || *unit.SurfaceChestNumber > 0xff) {
			return fmt.Errorf("mapUnits[%d]: surface chest number is outside byte range", index)
		}
		if unit.SurfaceChestNumber != nil && (*location.GetNPCReferences())[unit.NPCNum].GetSpriteIndex() != indexes.Chest {
			return fmt.Errorf("mapUnits[%d]: surface chest number belongs to a non-chest NPC", index)
		}
	case "chest":
		if unit.SurfaceChestNumber == nil {
			return fmt.Errorf("mapUnits[%d]: chest state is unavailable", index)
		}
	case "vehicle":
		if unit.SurfaceChestNumber != nil {
			return fmt.Errorf("mapUnits[%d]: vehicle cannot carry surface chest state", index)
		}
	}
	if unit.Kind == "enemy" && unit.SurfaceChestNumber != nil {
		return fmt.Errorf("mapUnits[%d]: enemy cannot carry surface chest state", index)
	}

	return nil
}

func validateSnapshotItemStacks(stacks []save_snapshot.ItemStackSnapshot, activePosition save_snapshot.PositionSnapshot) error {
	for index, stack := range stacks {
		if stack.Position != snapshotPositionAt(activePosition, stack.Position.X, stack.Position.Y) {
			return fmt.Errorf("itemStacks[%d]: stack is not on the active snapshot map", index)
		}
		for itemIndex, item := range stack.Items {
			if _, ok := references.ItemFromIdentity(references.ItemType(item.Type), item.ID); !ok {
				return fmt.Errorf("itemStacks[%d].items[%d]: item reference is unavailable", index, itemIndex)
			}
		}
	}

	return nil
}

func validateSnapshotMapMutations(
	mutations []save_snapshot.MapMutationSnapshot,
	activeLocation int,
	gameReferences *references.GameReferences,
) error {
	for index, mutation := range mutations {
		if err := validateSnapshotPosition(gameReferences, fmt.Sprintf("mapMutations[%d].position", index), mutation.Position); err != nil {
			return err
		}
		mapType := references.Location(mutation.Position.Location).GetMapType()
		if mapType == references.DungeonMapType {
			return fmt.Errorf("mapMutations[%d]: derived dungeon projections cannot be restored", index)
		}
		if mapType != references.LargeMapType && mutation.Position.Location != activeLocation {
			return fmt.Errorf("mapMutations[%d]: non-active small map is not loaded during hydration", index)
		}
		// Historical object overrides use both -1 and 0 for an empty layer.
		// Base-layer capture preserves these values, so they must resolve like
		// other installed tiles (NoSprite and Explosion in the runtime registry).
		if mutation.Kind == save_snapshot.MapMutationSetObject && (mutation.Value == -1 || mutation.Value == 0) {
			continue
		}
		if gameReferences.TileReferences == nil || gameReferences.TileReferences.GetTile(indexes.SpriteIndex(mutation.Value)) == nil {
			return fmt.Errorf("mapMutations[%d].value: tile reference %d is unavailable", index, mutation.Value)
		}
	}

	return nil
}

func validateSnapshotDungeonDiscoveries(discoveries []save_snapshot.DungeonDiscoverySnapshot, gameReferences *references.GameReferences) error {
	for index, discovery := range discoveries {
		location := references.Location(discovery.Location)
		locationReference := gameReferences.LocationReferences.GetLocationReference(location)
		if locationReference == nil {
			return fmt.Errorf("redux.dungeonDiscoveries[%d]: dungeon reference is unavailable", index)
		}
		position := references.Position{
			X: references.Coordinate(discovery.X),
			Y: references.Coordinate(discovery.Y),
		}
		source, ok := locationReference.GetDungeonCell(references.FloorNumber(discovery.Floor), position)
		if !ok {
			return fmt.Errorf("redux.dungeonDiscoveries[%d]: source cell is unavailable", index)
		}
		switch discovery.Kind {
		case save_snapshot.DungeonDiscoveryHiddenDoor:
			if source.Kind() != references.DungeonHiddenDoor {
				return fmt.Errorf("redux.dungeonDiscoveries[%d]: source cell is not a hidden door", index)
			}
		case save_snapshot.DungeonDiscoveryHiddenPit:
			if !isDungeonHiddenPit(source) {
				return fmt.Errorf("redux.dungeonDiscoveries[%d]: source cell is not a hidden pit", index)
			}
		default:
			return fmt.Errorf("redux.dungeonDiscoveries[%d]: unsupported discovery kind", index)
		}
	}

	return nil
}

func validateSnapshotActivePlayer(snapshot save_snapshot.SaveSnapshot) error {
	index := snapshot.Continuation.ActivePlayerIndex
	if index == -1 {
		return nil
	}
	if index < 0 || index >= party_state.MaxActiveParty {
		return errors.New("continuation.activePlayerIndex: index is outside the active party")
	}
	character := snapshot.Party.Roster[index]
	if !character.Occupied || party_state.PartyMap(character.PartyMap) != party_state.PartyMapActive {
		return errors.New("continuation.activePlayerIndex: slot is not occupied by an active party member")
	}
	status := party_state.CharacterStatus(character.Status)
	if character.CurrentHP <= 0 || status == party_state.Dead || status == party_state.Sleep {
		return errors.New("continuation.activePlayerIndex: selected party member is not living and awake")
	}

	return nil
}

func validateJournalSiteCompatibility(snapshot save_snapshot.SaveSnapshot, gameReferences *references.GameReferences) error {
	if err := validateAtlasLandmarkCompatibility(snapshot.AtlasLandmarks, gameReferences); err != nil {
		return err
	}
	for phase, observation := range snapshot.MoonstoneJournal {
		if !observation.SiteKnown {
			continue
		}
		sites := []journal.SiteLocation{observation.Site, observation.LastContext.Observer}
		if observation.Survey.Measured {
			sites = append(sites, observation.Survey.Location, observation.SurveyContext.Observer)
		}
		for index, site := range sites {
			position := save_snapshot.PositionSnapshot{Location: int(site.Map), Floor: int(site.Floor), X: site.X, Y: site.Y}
			if err := validateSnapshotPosition(gameReferences, fmt.Sprintf("moonstoneJournal[%d].sites[%d]", phase, index), position); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateSnapshotEnemyIdentity(index int, unit save_snapshot.MapUnitSnapshot) error {
	monsterType := references.MonsterType(unit.ReferenceID)
	valid := false
	switch map_units.EnemyVariant(unit.EnemyVariant) {
	case map_units.EnemyVariantStandard:
		valid = monsterType != references.MonsterStocks && monsterType != references.MonsterWhirlpool && monsterType != references.MonsterShadowlord
	case map_units.EnemyVariantPirate:
		valid = monsterType == references.MonsterStocks
	case map_units.EnemyVariantWhirlpool:
		valid = monsterType == references.MonsterWhirlpool
	case map_units.EnemyVariantShadowlord:
		valid = monsterType == references.MonsterShadowlord
	}
	if !valid {
		return fmt.Errorf("mapUnits[%d]: enemy variant %q is inconsistent with monster type %d", index, unit.EnemyVariant, unit.ReferenceID)
	}

	return nil
}

func validateSnapshotPosition(gameReferences *references.GameReferences, path string, position save_snapshot.PositionSnapshot) error {
	location := references.Location(position.Location)
	switch location.GetMapType() {
	case references.LargeMapType:
		if location != references.BritanniaUnderworld || (position.Floor != int(references.OVERWORLD) && position.Floor != int(references.UNDERWORLD)) {
			return fmt.Errorf("%s: large-map world reference is unavailable", path)
		}
	case references.SmallMapType:
		if location.IsDeedLocation() {
			// A deed map is generated from the snapshot's own deed; structural validation ties
			// world.position to crafting.activeDeedId (ref:CRAFT-0004).
			if position.Floor != 0 || position.X >= int(references.XSmallMapTiles) || position.Y >= int(references.YSmallMapTiles) {
				return fmt.Errorf("%s: coordinates are outside the deed map", path)
			}

			return nil
		}
		locationReference := gameReferences.LocationReferences.GetLocationReference(location)
		if locationReference == nil {
			return fmt.Errorf("%s: location reference %d is unavailable", path, position.Location)
		}
		minimum, maximum := locationReference.GetFloorMinMax()
		if position.Floor < int(minimum) || position.Floor > int(maximum) {
			return fmt.Errorf("%s: floor %d is unavailable", path, position.Floor)
		}
		limit := int(references.XSmallMapTiles)
		if position.X >= limit || position.Y >= limit {
			return fmt.Errorf("%s: coordinates are outside the runtime map", path)
		}
	case references.DungeonMapType:
		if locationReference := gameReferences.LocationReferences.GetLocationReference(location); locationReference == nil {
			return fmt.Errorf("%s: dungeon reference %d is unavailable", path, position.Location)
		}
		if !references.IsDungeonCoordinate(
			references.FloorNumber(position.Floor),
			references.Position{X: references.Coordinate(position.X), Y: references.Coordinate(position.Y)},
		) {
			return fmt.Errorf("%s: coordinates are outside the runtime dungeon", path)
		}
	case references.CombatMapType, references.CampingMapType:
		return fmt.Errorf("%s: transient combat or camping map cannot be hydrated", path)
	default:
		return fmt.Errorf("%s: transient location %d cannot be hydrated", path, position.Location)
	}

	return nil
}

func snapshotPositionAt(mapPosition save_snapshot.PositionSnapshot, x, y int) save_snapshot.PositionSnapshot {
	mapPosition.X, mapPosition.Y = x, y

	return mapPosition
}

func isPendingDeliveryDockPosition(docks *references.DockReferences, position save_snapshot.PositionSnapshot) bool {
	if docks == nil {
		return false
	}
	for _, dockLocation := range references.GetListOfAllLocationsWithDocks() {
		dock, ok := docks.GetDockPositionChecked(dockLocation)
		if ok && int(dock.X) == position.X && int(dock.Y) == position.Y {
			return true
		}
	}

	return false
}

// CaptureSaveSnapshot returns a complete detached native save DTO. It is
// update-thread-neutral: the caller is responsible for excluding concurrent mutations.
func (g *GameState) CaptureSaveSnapshot() (save_snapshot.SaveSnapshot, error) {
	g.clearUnavailableActivePlayer()
	if err := g.ValidateSaveSnapshotEligibility(); err != nil {
		return save_snapshot.SaveSnapshot{}, err
	}

	snapshot := save_snapshot.SaveSnapshot{Version: save_snapshot.CurrentVersion, Presentation: journal.EmptyPresentation()}
	snapshot.DungeonJournal = g.dungeonJournal
	snapshot.MoonstoneJournal = g.moonstoneJournal
	snapshot.AtlasLandmarks = slices.Clone(g.atlasLandmarks)
	g.capturePersonalSnapshot(&snapshot)
	g.captureParty(&snapshot)
	g.captureInventory(&snapshot)
	g.captureWorld(&snapshot)
	g.captureProgression(&snapshot)
	g.captureEffects(&snapshot)
	g.captureContinuation(&snapshot)
	g.captureRedux(&snapshot)
	g.captureCrafting(&snapshot)
	g.captureDungeon(&snapshot)
	wind := g.windSystem.Snapshot()
	snapshot.Wind = save_snapshot.WindSnapshot{Direction: int(wind.Direction), UpdateCounter: wind.UpdateCounter, NextTickMs: wind.NextTickMs}
	snapshot.Random = save_snapshot.RandomSnapshot{Seed: g.rng.Seed(), SourceCalls: g.rng.Calls()}
	if vehicle := captureVehicle(&g.PartyVehicle); vehicle != nil {
		snapshot.PartyVehicle = vehicle
	}

	var err error
	snapshot.MapUnits, err = g.captureMapUnits()
	if err != nil {
		return save_snapshot.SaveSnapshot{}, err
	}
	snapshot.ItemStacks, err = g.captureItemStacks()
	if err != nil {
		return save_snapshot.SaveSnapshot{}, err
	}
	snapshot.OpenDoor = g.captureOpenDoor()
	snapshot.MapMutations = g.canonicalizeOpenDoorMutations(g.captureMapMutations())
	if issues := save_snapshot.Validate(snapshot); len(issues) != 0 {
		return save_snapshot.SaveSnapshot{}, &save_snapshot.ValidationError{Issues: issues}
	}

	return snapshot, nil
}

func (g *GameState) captureContinuation(snapshot *save_snapshot.SaveSnapshot) {
	snapshot.Continuation = save_snapshot.ContinuationSnapshot{
		ActivePlayerIndex:                 g.activePlayerIndex,
		IsCriminal:                        g.isCriminal,
		HMSCapeAutomaticWaitActorSkip:     g.hmsCapeAutomaticWaitActorSkip,
		BeggarKarmaCounter:                g.beggarKarmaCounter,
		OpeningCampaignEncounterReduction: g.openingCampaignEncounterReduction,
		CombatActionTimeCounter:           g.combatActionTimeCounter,
	}
}

func (g *GameState) captureRedux(snapshot *save_snapshot.SaveSnapshot) {
	snapshot.Redux.LordBritishPassageOpen = g.lordBritishPassageOpen
	discoveries := g.sortedDungeonDiscoveries()
	snapshot.Redux.DungeonDiscoveries = make([]save_snapshot.DungeonDiscoverySnapshot, 0, len(discoveries))
	for _, discovery := range discoveries {
		snapshot.Redux.DungeonDiscoveries = append(
			snapshot.Redux.DungeonDiscoveries,
			save_snapshot.DungeonDiscoverySnapshot{
				Location: int(discovery.location),
				Floor:    int(discovery.floor),
				X:        int(discovery.position.X),
				Y:        int(discovery.position.Y),
				Kind:     string(discovery.kind),
			},
		)
	}
	knowledge := g.KnowledgeSnapshot()
	snapshot.Redux.PlayerKnowledge = make([]save_snapshot.PlayerKnowledgeSnapshot, 0, len(knowledge))
	for _, entry := range knowledge {
		saved := save_snapshot.PlayerKnowledgeSnapshot{
			Category: string(entry.Key.Category),
			Key:      entry.Key.ID,
			Source: save_snapshot.KnowledgeSourceSnapshot{
				Kind:        string(entry.Source.Kind),
				DisplayName: entry.Source.DisplayName,
			},
			Unread: entry.Unread,
		}
		if entry.DiscoveredOn != nil {
			saved.DiscoveredOn = &save_snapshot.BritannianDateSnapshot{
				Year: int(entry.DiscoveredOn.Year), Month: int(entry.DiscoveredOn.Month), Day: int(entry.DiscoveredOn.Day),
			}
		}
		if entry.SpellFormula != nil {
			saved.SpellFormula = &save_snapshot.SpellFormulaKnowledgeSnapshot{
				State:          string(entry.SpellFormula.State),
				ConfirmedMask:  entry.SpellFormula.ConfirmedMask,
				CandidateMasks: append(make([]uint16, 0, len(entry.SpellFormula.CandidateMasks)), entry.SpellFormula.CandidateMasks...),
			}
		}
		snapshot.Redux.PlayerKnowledge = append(snapshot.Redux.PlayerKnowledge, saved)
	}
	events := g.QuestJournalEvents()
	snapshot.Redux.QuestJournal = make([]save_snapshot.QuestJournalEventSnapshot, 0, len(events))
	for _, event := range events {
		saved := save_snapshot.QuestJournalEventSnapshot{
			Sequence: event.Sequence, Kind: string(event.Kind), Subject: string(event.Subject),
			Source: string(event.Source), SourceName: event.SourceName, Acknowledged: event.Acknowledged,
		}
		if event.OccurredOn != nil {
			saved.OccurredOn = &save_snapshot.BritannianDateSnapshot{
				Year: int(event.OccurredOn.Year), Month: int(event.OccurredOn.Month), Day: int(event.OccurredOn.Day),
			}
		}
		snapshot.Redux.QuestJournal = append(snapshot.Redux.QuestJournal, saved)
	}
}

// ValidateSaveSnapshotEligibility reports whether authoritative state is at a
// stable boundary that the native snapshot format can represent.
func (g *GameState) ValidateSaveSnapshotEligibility() error {
	switch {
	case g.GameReferences == nil || g.GameConfig == nil:
		return fmt.Errorf("%w: game references and configuration are required", ErrSaveSnapshotNotRepresentable)
	case g.finaleHandoffPending || g.lordBritishFinale != nil:
		return fmt.Errorf("%w: terminal Lord British finale", ErrSaveSnapshotNotRepresentable)
	case g.rng == nil || g.windSystem == nil || g.SearchManager == nil || g.EventQueueManager == nil:
		return fmt.Errorf("%w: core deterministic managers are missing", ErrSaveSnapshotNotRepresentable)
	case g.CombatScene != nil:
		return fmt.Errorf("%w: active combat", ErrSaveSnapshotNotRepresentable)
	case g.CampingScene != nil || g.ShrineScene != nil:
		return fmt.Errorf("%w: active camping or shrine scene", ErrSaveSnapshotNotRepresentable)
	case g.EventQueueManager.HasActiveWorkflows():
		return fmt.Errorf("%w: active event queue work", ErrSaveSnapshotNotRepresentable)
	case g.surfaceCycleOwned || g.completingSurfaceCycle:
		return fmt.Errorf("%w: active surface command cycle", ErrSaveSnapshotNotRepresentable)
	case g.blackthornSession != nil:
		return fmt.Errorf("%w: active Blackthorn interrogation", ErrSaveSnapshotNotRepresentable)
	case len(g.pirateAttackQueue) != 0:
		return fmt.Errorf("%w: pending pirate attack presentation", ErrSaveSnapshotNotRepresentable)
	case g.tavernCustomerSessionSequence != 0:
		return fmt.Errorf("%w: active tavern customer transaction", ErrSaveSnapshotNotRepresentable)
	case g.moongateStage != g.desiredMoongateStage():
		return fmt.Errorf("%w: moongate materialization transition", ErrSaveSnapshotNotRepresentable)
	case g.isSailing:
		return fmt.Errorf("%w: automatic sailing workflow state", ErrSaveSnapshotNotRepresentable)
	case g.outdoorActorContinuation != nil || g.pendingOverworldEncounter != nil || g.pendingCombatEncounter != nil ||
		g.pendingSmallMapEncounter != nil || g.pendingCampingCombat != nil ||
		g.pendingDirectMonsterContact != OverworldContactNone:
		return fmt.Errorf("%w: pending encounter", ErrSaveSnapshotNotRepresentable)
	case g.activeShipRepair != nil:
		return fmt.Errorf("%w: active ship repair", ErrSaveSnapshotNotRepresentable)
	case g.activeSlowMovement != nil:
		return fmt.Errorf("%w: pending slow movement completion", ErrSaveSnapshotNotRepresentable)
	case g.activeWaterfall != nil:
		return fmt.Errorf("%w: pending waterfall continuation", ErrSaveSnapshotNotRepresentable)
	case !g.isValidActivePlayerIndex(g.activePlayerIndex):
		return fmt.Errorf("%w: invalid active player selection", ErrSaveSnapshotNotRepresentable)
	case len(g.pendingProvisionHours) != 0:
		return fmt.Errorf("%w: pending provision-hour processing", ErrSaveSnapshotNotRepresentable)
	case g.activeDungeonCommand != nil:
		return fmt.Errorf("%w: active dungeon command cycle", ErrSaveSnapshotNotRepresentable)
	case g.activeDungeon.hasPendingRoomContinuation():
		return fmt.Errorf("%w: pending dungeon room continuation", ErrSaveSnapshotNotRepresentable)
	case g.MapState.PlayerLocation.Location.GetMapType() == references.CombatMapType:
		return fmt.Errorf("%w: combat or mini-map location", ErrSaveSnapshotNotRepresentable)
	case g.MapState.PlayerLocation.Location.GetMapType() == references.DungeonMapType:
		if g.activeDungeon == nil || g.activeDungeon.location != g.MapState.PlayerLocation.Location {
			return fmt.Errorf("%w: active dungeon state does not match player location", ErrSaveSnapshotNotRepresentable)
		}
	}

	return nil
}

func (g *GameState) captureDungeon(snapshot *save_snapshot.SaveSnapshot) {
	if g.MapState.PlayerLocation.Location.GetMapType() != references.DungeonMapType {
		return
	}

	dungeon := g.requireActiveDungeon().Snapshot()
	saved := &save_snapshot.DungeonSnapshot{
		Location:          int(dungeon.Location),
		Cells:             make([]int, 0, save_snapshot.DungeonCellCount),
		Facing:            int(dungeon.Facing),
		LastDirection:     int(dungeon.LastDirection),
		QuicknessSkipNext: dungeon.QuicknessSkipNext,
	}
	for floor := range references.DungeonFloorCount {
		for y := range int(references.YDungeonMapTiles) {
			for x := range int(references.XDungeonMapTiles) {
				saved.Cells = append(saved.Cells, int(dungeon.Cells[floor][y][x].Raw()))
				saved.ChestAssessments[floor*64+y*8+x] = int(dungeon.ChestAssessments[floor][y][x])
			}
		}
	}
	if dungeon.Wanderer != nil {
		saved.Wanderer = &save_snapshot.DungeonWandererSnapshot{
			Monster: int(dungeon.Wanderer.Monster), Floor: int(dungeon.Wanderer.Floor),
			X: int(dungeon.Wanderer.Position.X), Y: int(dungeon.Wanderer.Position.Y),
			PreviousX: int(dungeon.Wanderer.PreviousPosition.X), PreviousY: int(dungeon.Wanderer.PreviousPosition.Y),
			OnCeiling: dungeon.Wanderer.OnCeiling,
		}
	}
	snapshot.Dungeon = saved
}

func (g *GameState) captureParty(snapshot *save_snapshot.SaveSnapshot) {
	g.PartyState.ForEachCharacter(func(index int, character *party_state.PlayerCharacter) {
		snapshot.Party.Roster[index] = save_snapshot.CharacterSnapshot{
			Occupied: true, Name: character.GetNameAsString(), Gender: int(character.Gender), Class: int(character.Class),
			Status: int(character.GetStatus()), Strength: int(character.Strength), Dexterity: int(character.Dexterity),
			Intelligence: int(character.Intelligence), CurrentMP: int(character.GetCurrentMp()), CurrentHP: int(character.GetCurrentHp()),
			MaximumHP: int(character.GetMaxHp()), Experience: int(character.Exp), Level: int(character.Level),
			MonthsAtInn: int(character.MonthsAtInn), ArmorClass: int(character.ArmorClass), Helmet: int(character.Helmet),
			Armor: int(character.Armor), Weapon: int(character.Weapon), Shield: int(character.Shield), Ring: int(character.Ring),
			Amulet: int(character.Amulet), PartyMap: int(character.PartyMap),
		}
	})
	snapshot.Party.Karma = g.PartyState.Karma.Value
	met, dead := g.PartyState.NPCFlagsSnapshot()
	snapshot.Party.MetNPCs = flattenNPCFlags(met)
	snapshot.Party.DeadNPCs = flattenNPCFlags(dead)
}

func flattenNPCFlags(flags map[references.Location][]bool) []save_snapshot.NPCFlagSnapshot {
	result := make([]save_snapshot.NPCFlagSnapshot, 0)
	for location, npcFlags := range flags {
		for npc, set := range npcFlags {
			if set {
				// Native v5 stored NPC absence by zero-based location index.
				// Runtime state uses canonical map IDs 1 through 32. ref:SAVE-0005
				result = append(result, save_snapshot.NPCFlagSnapshot{
					Location: int(location - references.Moonglow),
					NPC:      npc,
				})
			}
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Location != result[j].Location {
			return result[i].Location < result[j].Location
		}

		return result[i].NPC < result[j].NPC
	})

	return result
}

func (g *GameState) captureInventory(snapshot *save_snapshot.SaveSnapshot) {
	inventory := &g.PartyState.Inventory
	snapshot.Inventory.Food = inventory.Provisions.Food.Get()
	snapshot.Inventory.Gold = inventory.Gold.Get()
	snapshot.Inventory.Gems = inventory.Provisions.Gems.Get()
	snapshot.Inventory.Torches = inventory.Provisions.Torches.Get()
	snapshot.Inventory.Keys = inventory.Provisions.Keys.Get()
	snapshot.Inventory.SkullKeys = inventory.Provisions.SkullKeys.Get()
	for index := range save_snapshot.EquipmentCount {
		snapshot.Inventory.Equipment[index] = inventory.Equipment.Get(references.Equipment(index))
	}
	for index := range save_snapshot.SpellCount {
		snapshot.Inventory.Spells[index] = inventory.Spells.Get(references.Spell(index))
	}
	for index := range save_snapshot.ConsumableCount {
		snapshot.Inventory.Scrolls[index] = inventory.Scrolls.Get(references.Scroll(index))
		snapshot.Inventory.Potions[index] = inventory.Potions.Get(references.Potion(index))
		snapshot.Inventory.Reagents[index] = inventory.Reagent.Get(references.Reagent(index))
	}
	for index := range save_snapshot.QuestItemCount {
		snapshot.Inventory.QuestItems[index] = inventory.QuestItems.Get(references.QuestItem(index))
	}
	for index := range save_snapshot.MoonstoneCount {
		snapshot.Inventory.Moonstones[index] = inventory.Moonstones.Get(references.Moonstone(index))
	}
}

func (g *GameState) captureWorld(snapshot *save_snapshot.SaveSnapshot) {
	snapshot.World = save_snapshot.WorldSnapshot{
		Position:             positionSnapshot(g.MapState.PlayerLocation),
		LastLargeMapPosition: save_snapshot.PositionSnapshot{Location: int(references.BritanniaUnderworld), X: int(g.LastLargeMapPosition.X), Y: int(g.LastLargeMapPosition.Y), Floor: int(g.LastLargeMapFloor)},
		Calendar:             save_snapshot.CalendarSnapshot{Year: int(g.DateTime.Year), Month: int(g.DateTime.Month), Day: int(g.DateTime.Day), Hour: int(g.DateTime.Hour), Minute: int(g.DateTime.Minute), Turns: g.DateTime.Turn},
		ElapsedMs:            g.ElapsedMs, LastProcessedHour: g.lastProcessedHour, LastCamped: g.LastCamped,
		LastWandererMonth: g.LastWandererMonth, CombatSequence: g.combatEncounterSequence,
	}
}

func (g *GameState) captureProgression(snapshot *save_snapshot.SaveSnapshot) {
	found, skullDay, reagentDays := g.SearchManager.snapshotState()
	snapshot.Progression.SearchFound = found
	snapshot.Progression.SkullKeysFoundDay = skullDay
	snapshot.Progression.ReagentCachesFoundDay = reagentDays
	for index, stone := range g.moonstoneSystem.Moonstones {
		snapshot.Progression.Moonstones[index] = save_snapshot.MoonstoneSnapshot{X: stone.X, Y: stone.Y, Map: stone.Map, Level: stone.Level, State: byte(stone.State), IsFound: stone.IsFound}
	}
	for index := range TotalShrines {
		snapshot.Progression.Shrines[index] = save_snapshot.ShrineSnapshot{Active: g.shrineQuestsActive[index], Completed: g.shrineQuestsCompleted[index], Ruined: g.closedShrines[index]}
	}
	snapshot.Progression.Dungeons = g.DungeonStates
	snapshot.Progression.DungeonRooms = g.deferredDungeonRoomProgress
	townAssignments := [save_snapshot.ShadowlordCount]int{}
	for index, assignment := range g.ShadowlordState.TownAssignments {
		townAssignments[index] = int(assignment)
	}
	snapshot.Progression.Shadowlords = save_snapshot.ShadowlordsSnapshot{Alive: g.ShadowlordState.Alive, TownAssignments: townAssignments, LastCalled: int(g.ShadowlordState.LastCalled), Summoned: g.ShadowlordState.Summoned, AutomaticPresence: int(g.ShadowlordState.AutomaticPresence)}
	for spell, mask := range g.MagicState.KnownRecipes {
		if spell >= references.InLor && spell <= references.AnTym {
			snapshot.Progression.Recipes[spell] = save_snapshot.RecipeSnapshot{Known: true, Mask: mask}
		}
	}
	snapshot.Progression.HMSCapeActivated = g.hmsCapeActivated
}

func (g *GameState) captureEffects(snapshot *save_snapshot.SaveSnapshot) {
	snapshot.Effects = save_snapshot.EffectsSnapshot{
		DurationType: byte(g.ActiveDurationEffect.EffectType), DurationTurns: g.ActiveDurationEffect.TurnsRemaining,
		DurationToggleOnly: g.ActiveDurationEffect.IsToggleItemOnly, MagicLightMinutesRemaining: g.MagicLightMinutesRemaining,
		CarriedTorchMinutesRemaining: g.MapState.Lighting.TorchMinutesRemaining(), QuicknessWorldActorSkip: g.quicknessWorldActorSkip,
		TownMountedActorSkip: g.townMountedActorSkip, TownQuicknessActorSkip: g.townQuicknessActorSkip,
		OutdoorMountedActorSkip: g.outdoorMountedActorSkip,
		Amulet:                  save_snapshot.ToggleSnapshot{Active: g.AmuletState.IsActive, Turns: g.AmuletState.TurnsRemaining},
		Crown:                   save_snapshot.ToggleSnapshot{Active: g.CrownState.IsActive, Turns: g.CrownState.TurnsRemaining},
		Badge:                   save_snapshot.ToggleSnapshot{Active: g.BadgeState.IsActive, Turns: g.BadgeState.TurnsRemaining},
		Drunk:                   g.drunk, Smashed: g.smashed,
	}
}

func captureVehicle(friendly *map_units.NPCFriendly) *save_snapshot.VehicleSnapshot {
	if friendly == nil || friendly.GetVehicleDetails().VehicleType == references.NoPartyVehicle {
		return nil
	}
	vehicle := friendly.GetVehicleDetails().Snapshot()

	return &save_snapshot.VehicleSnapshot{VesselID: vehicle.VesselID, Type: int(vehicle.VehicleType), CurrentDirection: int(vehicle.CurrentDirection), PreviousDirection: int(vehicle.PreviousDirection), Skiffs: vehicle.SkiffQuantity, OwnedByNPC: vehicle.IsOwnedByNPC, Hull: vehicle.HullHP, SailState: int(vehicle.SailState), HasHMSCape: friendly.GetVehicleDetails().HasHMSCape, PendingDelivery: vehicle.PendingDelivery}
}

func (g *GameState) captureMapUnits() ([]save_snapshot.MapUnitSnapshot, error) {
	result := make([]save_snapshot.MapUnitSnapshot, 0)
	for _, world := range []references.World{references.UNDERWORLD, references.OVERWORLD} {
		controller := g.LargeMapNPCAIController[world]
		units, err := captureControllerUnits(controller.GetNpcs(), int(world), references.BritanniaUnderworld)
		if err != nil {
			return nil, err
		}
		result = append(result, units...)
	}
	if g.MapState.PlayerLocation.Location.GetMapType() == references.SmallMapType {
		location := g.MapState.PlayerLocation.Location
		loaded := g.CurrentNPCAIController.GetNpcs()
		representable := make(map_units.MapUnits, 0, len(*loaded))
		for _, unit := range *loaded {
			friendly, isFriendly := unit.(*map_units.NPCFriendly)
			if isFriendly && friendly.GetVehicleDetails().VehicleType == references.NoPartyVehicle &&
				!friendly.IsTransientSurfaceChest() &&
				g.PartyState.IsNPCDead(location, friendly.MapUnitDetails().NPCNum) {
				continue
			}
			representable = append(representable, unit)
		}
		units, err := captureControllerUnits(&representable, 0, location)
		if err != nil {
			return nil, err
		}
		result = append(result, units...)
	}
	sort.Slice(result, func(i, j int) bool {
		a, b := result[i], result[j]
		if a.Position.Location != b.Position.Location {
			return a.Position.Location < b.Position.Location
		}
		if a.World != b.World {
			return a.World < b.World
		}

		return a.NPCNum < b.NPCNum
	})

	return result, nil
}

func captureControllerUnits(units *map_units.MapUnits, world int, location references.Location) ([]save_snapshot.MapUnitSnapshot, error) {
	result := make([]save_snapshot.MapUnitSnapshot, 0, len(*units))
	for controllerOrder, unit := range *units {
		details := unit.MapUnitDetails()
		arrivedScheduleSlot := int(details.ArrivedScheduleSlot())
		snapshot := save_snapshot.MapUnitSnapshot{World: world, ControllerOrder: controllerOrder, NPCNum: details.NPCNum, Position: save_snapshot.PositionSnapshot{Location: int(location), X: int(unit.Pos().X), Y: int(unit.Pos().Y), Floor: int(unit.Floor())}, Visible: unit.IsVisible(), SkipNextTurn: details.SkipNextTurn, LegacyWantsToAutoTalk: details.PendingInteraction != map_units.PendingNPCInteractionNone, OverriddenAI: int(details.GetOverriddenAiType()), AIOverrideProvenance: string(details.AIOverrideProvenance()), ArrivedScheduleSlot: &arrivedScheduleSlot}
		if consumedSlot, ok := details.ConsumedTalk3ScheduleSlot(); ok {
			value := int(consumedSlot)
			snapshot.ConsumedTalk3ScheduleSlot = &value
		}
		for _, position := range details.CurrentPath {
			snapshot.CurrentPath = append(snapshot.CurrentPath, save_snapshot.PositionSnapshot{Location: int(location), X: int(position.X), Y: int(position.Y), Floor: int(unit.Floor())})
		}
		switch typed := unit.(type) {
		case *map_units.NPCEnemy:
			snapshot.Kind, snapshot.ReferenceID = "enemy", int(typed.EnemyReference.MonsterType)
			snapshot.EnemyVariant, snapshot.EnemyCurrentHP, snapshot.EnemyFacing = string(typed.Variant()), typed.CurrentHP(), int(typed.Facing())
			if lord, ok := typed.ShadowlordIdentity(); ok {
				identity := int(lord)
				snapshot.Shadowlord = &identity
			}
		case *map_units.NPCFriendly:
			if vehicle := captureVehicle(typed); vehicle != nil {
				snapshot.Kind, snapshot.Vehicle = "vehicle", vehicle
			} else if number, ok := typed.SurfaceChestNumber(); ok {
				value := int(number)
				snapshot.Kind, snapshot.SurfaceChestNumber = "chest", &value
				snapshot.SurfaceChestAssessment = int(typed.SurfaceChestAssessment())
				snapshot.SurfaceChestAuthored = typed.IsAuthoredSurfaceChest()
				snapshot.SurfaceChestTransient = typed.IsTransientSurfaceChest()
			} else {
				snapshot.Kind, snapshot.ReferenceID = "npc", details.NPCNum
			}
		default:
			return nil, fmt.Errorf("%w: map unit type %T", ErrSaveSnapshotNotRepresentable, unit)
		}
		result = append(result, snapshot)
	}

	return result, nil
}

func (g *GameState) captureItemStacks() ([]save_snapshot.ItemStackSnapshot, error) {
	entries := g.ItemStacksMap.Entries()
	result := make([]save_snapshot.ItemStackSnapshot, 0, len(entries))
	for _, entry := range entries {
		stack := save_snapshot.ItemStackSnapshot{Position: save_snapshot.PositionSnapshot{Location: int(g.MapState.PlayerLocation.Location), X: int(entry.Position.X), Y: int(entry.Position.Y), Floor: int(g.MapState.PlayerLocation.Floor)}}
		for _, item := range entry.Stack.Items {
			if item.Item == nil {
				return nil, fmt.Errorf("%w: nil item in world stack", ErrSaveSnapshotNotRepresentable)
			}
			stack.Items = append(stack.Items, save_snapshot.ItemSnapshot{Type: int(item.Item.Type()), ID: item.Item.ID(), Quantity: item.Quantity})
		}
		result = append(result, stack)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Position.Y != result[j].Position.Y {
			return result[i].Position.Y < result[j].Position.Y
		}

		return result[i].Position.X < result[j].Position.X
	})

	return result, nil
}

func (g *GameState) captureMapMutations() []save_snapshot.MapMutationSnapshot {
	mutations := make([]save_snapshot.MapMutationSnapshot, 0)
	for _, loaded := range g.MapState.LayeredMaps.Loaded() {
		if loaded.MapType == references.CombatMapType || loaded.MapType == references.DungeonMapType {
			continue
		}
		if loaded.MapType == references.SmallMapType && loaded.Location != g.MapState.PlayerLocation.Location {
			continue
		}
		for y := references.Coordinate(0); y < loaded.Map.YMaxTilesPerMap; y++ {
			for x := references.Coordinate(0); x < loaded.Map.XMaxTilesPerMap; x++ {
				position := references.Position{X: x, Y: y}
				actual := loaded.Map.SpriteIndexByLayer(map_state.MapLayer, position)
				actual = g.persistentTileWithoutTransientLargeMapPush(
					loaded.Location,
					loaded.Floor,
					position,
					actual,
				)
				base := g.baseTile(loaded.Location, loaded.Floor, position)
				pos := save_snapshot.PositionSnapshot{Location: int(loaded.Location), X: int(x), Y: int(y), Floor: int(loaded.Floor)}
				if actual != base && !isLordBritishPassageSnapshotPosition(loaded.Location, loaded.Floor, x, y) {
					mutations = append(mutations, save_snapshot.MapMutationSnapshot{Position: pos, Kind: save_snapshot.MapMutationSetTile, Value: int(actual)})
				}
				override := loaded.Map.SpriteIndexByLayer(map_state.MapOverrideLayer, position)
				if override > 0 {
					mutations = append(mutations, save_snapshot.MapMutationSnapshot{Position: pos, Kind: save_snapshot.MapMutationSetObject, Value: int(override)})
				}
			}
		}
	}
	sort.Slice(mutations, func(i, j int) bool {
		a, b := mutations[i], mutations[j]
		if a.Position.Location != b.Position.Location {
			return a.Position.Location < b.Position.Location
		}
		if a.Position.Floor != b.Position.Floor {
			return a.Position.Floor < b.Position.Floor
		}
		if a.Position.Y != b.Position.Y {
			return a.Position.Y < b.Position.Y
		}
		if a.Position.X != b.Position.X {
			return a.Position.X < b.Position.X
		}

		return a.Kind < b.Kind
	})

	return mutations
}

func (g *GameState) baseTile(location references.Location, floor references.FloorNumber, position references.Position) indexes.SpriteIndex {
	if location.GetMapType() == references.LargeMapType {
		if floor == references.FloorNumber(references.UNDERWORLD) {
			return g.GameReferences.UnderworldLargeMapReference.GetSpriteIndex(position.X, position.Y)
		}

		return g.GameReferences.OverworldLargeMapReference.GetSpriteIndex(position.X, position.Y)
	}

	return g.locationReference(location).GetTileNumber(position, floor)
}

func positionSnapshot(location references.PlayerLocation) save_snapshot.PositionSnapshot {
	return save_snapshot.PositionSnapshot{Location: int(location.Location), X: int(location.Position.X), Y: int(location.Position.Y), Floor: int(location.Floor)}
}

// NewGameStateFromSaveSnapshot hydrates a detached state without running ordinary
// location-entry actions, dialogs, transitions, or progression effects.
func NewGameStateFromSaveSnapshot(snapshot save_snapshot.SaveSnapshot, gameConfig *config.UltimaVConfiguration, gameReferences *references.GameReferences, xTilesVisibleOnGameScreen, yTilesVisibleOnGameScreen int) (*GameState, error) {
	if issues := save_snapshot.Validate(snapshot); len(issues) != 0 {
		return nil, &save_snapshot.ValidationError{Issues: issues}
	}
	if gameConfig == nil || gameReferences == nil {
		return nil, errors.New("hydrating save snapshot: configuration and references are required")
	}
	if err := ValidateSaveSnapshotCompatibility(snapshot, gameReferences); err != nil {
		return nil, fmt.Errorf("hydrating save snapshot: %w", err)
	}
	g := initBlankGameState(gameReferences, gameConfig, xTilesVisibleOnGameScreen, yTilesVisibleOnGameScreen)
	if err := g.restoreParty(snapshot); err != nil {
		return nil, err
	}
	g.restoreContinuation(snapshot.Continuation)
	g.restoreWorld(snapshot)
	// Deeds precede map hydration: a save on a deed map resolves its location through them.
	g.restoreCrafting(snapshot.Crafting)
	g.restoreBasementChestHistory(snapshot)
	g.initializeMoongateStage()
	g.lordBritishPassageOpen = snapshot.Redux.LordBritishPassageOpen
	g.dungeonJournal = snapshot.DungeonJournal
	g.moonstoneJournal = snapshot.MoonstoneJournal
	g.atlasLandmarks = slices.Clone(snapshot.AtlasLandmarks)
	g.rng = rng.Restore(snapshot.Random.Seed, snapshot.Random.SourceCalls)
	g.initializeEnvironmentalSystems()
	g.windSystem.Restore(environment.WindSnapshot{Direction: environment.WindDirection(snapshot.Wind.Direction), UpdateCounter: snapshot.Wind.UpdateCounter, NextTickMs: snapshot.Wind.NextTickMs})
	g.initializeSnapshotMapsAndControllers()
	g.restoreDungeonDiscoveries(snapshot.Redux.DungeonDiscoveries)
	if err := g.restoreDungeon(snapshot.Dungeon); err != nil {
		return nil, err
	}
	g.restoreProgression(snapshot)
	g.restorePlayerKnowledge(snapshot.Redux.PlayerKnowledge)
	g.restoreQuestJournal(snapshot.Redux.QuestJournal)
	g.restoreEffects(snapshot)
	if err := g.restoreMapUnits(snapshot.MapUnits, snapshot.NeedsMapUnitReconstruction()); err != nil {
		return nil, err
	}
	if err := g.restoreItemStacks(snapshot.ItemStacks); err != nil {
		return nil, err
	}
	if err := g.restoreMapMutations(snapshot.MapMutations); err != nil {
		return nil, err
	}
	if err := g.restoreOpenDoor(snapshot.OpenDoor); err != nil {
		return nil, err
	}
	switch g.MapState.PlayerLocation.Location.GetMapType() {
	case references.LargeMapType:
		g.UpdateLighthouseForLargeMap()
	case references.SmallMapType:
		g.UpdateLighthouseForSmallMap()
	case references.DungeonMapType:
		// Dungeon hydration installs its exact semantic cells and needs no world projection updates.
	case references.CombatMapType, references.CampingMapType:
		panic("transient map passed snapshot compatibility validation")
	}
	if snapshot.PartyVehicle != nil {
		g.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(references.VehicleType(snapshot.PartyVehicle.Type), g.MapState.PlayerLocation.Position, g.MapState.PlayerLocation.Floor)
		restoreVehicle(g.PartyVehicle.GetVehicleDetails(), *snapshot.PartyVehicle)
		g.PartyVehicle.MapUnitDetails().Position = g.MapState.PlayerLocation.Position
		g.PartyVehicle.MapUnitDetails().Floor = g.MapState.PlayerLocation.Floor
	} else {
		g.PartyVehicle = map_units.NewNPCFriendlyVehiceNoVehicle()
	}
	if err := g.restorePersonalSnapshot(snapshot); err != nil {
		return nil, err
	}
	g.InitializeOutdoorWindow()

	return g, nil
}

func (g *GameState) restoreContinuation(saved save_snapshot.ContinuationSnapshot) {
	if !g.SetActivePlayer(saved.ActivePlayerIndex) {
		panic("validated snapshot active player became unavailable during hydration")
	}
	g.isCriminal = saved.IsCriminal
	g.hmsCapeAutomaticWaitActorSkip = saved.HMSCapeAutomaticWaitActorSkip
	g.beggarKarmaCounter = saved.BeggarKarmaCounter
	g.openingCampaignEncounterReduction = saved.OpeningCampaignEncounterReduction
	g.combatActionTimeCounter = saved.CombatActionTimeCounter
}

func (g *GameState) restorePlayerKnowledge(saved []save_snapshot.PlayerKnowledgeSnapshot) {
	g.playerKnowledge = newPlayerKnowledge()
	for _, entry := range saved {
		restored := KnowledgeEntry{
			Key: KnowledgeKey{
				Category: KnowledgeCategory(entry.Category),
				ID:       entry.Key,
			},
			Source: KnowledgeSource{
				Kind:        KnowledgeSourceKind(entry.Source.Kind),
				DisplayName: entry.Source.DisplayName,
			},
			Unread: entry.Unread,
		}
		if entry.DiscoveredOn != nil {
			restored.DiscoveredOn = &datetime.BritannianDate{
				Year: uint16(entry.DiscoveredOn.Year), Month: byte(entry.DiscoveredOn.Month), Day: byte(entry.DiscoveredOn.Day),
			}
		}
		if entry.SpellFormula != nil {
			restored.SpellFormula = &SpellFormulaKnowledge{
				State:          SpellFormulaState(entry.SpellFormula.State),
				ConfirmedMask:  entry.SpellFormula.ConfirmedMask,
				CandidateMasks: append(make([]uint16, 0, len(entry.SpellFormula.CandidateMasks)), entry.SpellFormula.CandidateMasks...),
			}
		}
		g.playerKnowledge.entries[restored.Key] = restored
	}
}

func (g *GameState) restoreQuestJournal(saved []save_snapshot.QuestJournalEventSnapshot) {
	g.questJournal = newQuestJournal()
	for _, event := range saved {
		restored := QuestJournalEvent{
			Sequence: event.Sequence, Kind: QuestJournalEventKind(event.Kind), Subject: QuestJournalSubject(event.Subject),
			Source: QuestJournalEventSource(event.Source), SourceName: event.SourceName, Acknowledged: event.Acknowledged,
		}
		if event.OccurredOn != nil {
			restored.OccurredOn = &datetime.BritannianDate{
				Year: uint16(event.OccurredOn.Year), Month: byte(event.OccurredOn.Month), Day: byte(event.OccurredOn.Day),
			}
		}
		if !g.questJournal.append(restored) {
			panic("validated quest journal contains a duplicate event")
		}
	}
}

func (g *GameState) restoreDungeonDiscoveries(saved []save_snapshot.DungeonDiscoverySnapshot) {
	g.dungeonDiscoveries = make(map[dungeonDiscoveryKey]dungeonDiscoveryKind, len(saved))
	for _, discovery := range saved {
		key := dungeonDiscoveryKey{
			location: references.Location(discovery.Location),
			floor:    references.FloorNumber(discovery.Floor),
			position: references.Position{
				X: references.Coordinate(discovery.X),
				Y: references.Coordinate(discovery.Y),
			},
		}
		g.dungeonDiscoveries[key] = dungeonDiscoveryKind(discovery.Kind)
	}
}

func (g *GameState) restoreParty(snapshot save_snapshot.SaveSnapshot) error {
	var roster [party_state.NRosterSlots]*party_state.PlayerCharacter
	for index, saved := range snapshot.Party.Roster {
		if !saved.Occupied {
			continue
		}
		if len(saved.Name) > party_state.NMaxPlayerNameSize {
			return fmt.Errorf("hydrating roster slot %d: name is too long", index)
		}
		character := &party_state.PlayerCharacter{Gender: party_state.CharacterGender(saved.Gender), Class: party_state.CharacterClass(saved.Class), Strength: byte(saved.Strength), Dexterity: byte(saved.Dexterity), Intelligence: byte(saved.Intelligence), Exp: uint16(saved.Experience), Level: byte(saved.Level), MonthsAtInn: byte(saved.MonthsAtInn), ArmorClass: byte(saved.ArmorClass), Helmet: byte(saved.Helmet), Armor: byte(saved.Armor), Weapon: byte(saved.Weapon), Shield: byte(saved.Shield), Ring: byte(saved.Ring), Amulet: byte(saved.Amulet), PartyMap: party_state.PartyMap(saved.PartyMap)}
		copy(character.Name[:], saved.Name)
		character.SetMaxHp(uint16(saved.MaximumHP))
		character.SetHp(uint16(saved.CurrentHP))
		character.SetMp(byte(saved.CurrentMP))
		character.SetStatus(party_state.CharacterStatus(saved.Status))
		roster[index] = character
	}
	g.PartyState = *party_state.NewPartyStateFromRoster(roster)
	g.PartyState.Karma = party_state.Karma{Value: snapshot.Party.Karma}
	g.PartyState.RestoreNPCFlags(expandNPCFlags(snapshot.Party.MetNPCs), expandNPCFlags(snapshot.Party.DeadNPCs))
	inv, saved := &g.PartyState.Inventory, snapshot.Inventory
	inv.Provisions.Food.Set(saved.Food)
	inv.Gold.Set(saved.Gold)
	inv.Provisions.Gems.Set(saved.Gems)
	inv.Provisions.Torches.Set(saved.Torches)
	inv.Provisions.Keys.Set(saved.Keys)
	inv.Provisions.SkullKeys.Set(saved.SkullKeys)
	for index, quantity := range saved.Equipment {
		inv.Equipment.Set(references.Equipment(index), quantity)
	}
	for index, quantity := range saved.Spells {
		inv.Spells.Set(references.Spell(index), quantity)
	}
	for index, quantity := range saved.Scrolls {
		inv.Scrolls.Set(references.Scroll(index), quantity)
	}
	for index, quantity := range saved.Potions {
		inv.Potions.Set(references.Potion(index), quantity)
	}
	for index, quantity := range saved.Reagents {
		inv.Reagent.Set(references.Reagent(index), quantity)
	}
	for index, quantity := range saved.QuestItems {
		inv.QuestItems.Set(references.QuestItem(index), quantity)
	}
	for index, quantity := range saved.Moonstones {
		inv.Moonstones.Set(references.Moonstone(index), quantity)
	}

	return nil
}

func expandNPCFlags(flags []save_snapshot.NPCFlagSnapshot) map[references.Location][]bool {
	result := make(map[references.Location][]bool)
	for _, flag := range flags {
		location := references.Location(flag.Location + int(references.Moonglow))
		if result[location] == nil {
			result[location] = make([]bool, party_state.NPcsPerLocation)
		}
		result[location][flag.NPC] = true
	}

	return result
}

func (g *GameState) restoreWorld(snapshot save_snapshot.SaveSnapshot) {
	g.MapState.PlayerLocation = playerLocation(snapshot.World.Position)
	g.LastLargeMapPosition = references.Position{X: references.Coordinate(snapshot.World.LastLargeMapPosition.X), Y: references.Coordinate(snapshot.World.LastLargeMapPosition.Y)}
	g.LastLargeMapFloor = references.FloorNumber(snapshot.World.LastLargeMapPosition.Floor)
	c := snapshot.World.Calendar
	g.DateTime = datetime.UltimaDate{Year: uint16(c.Year), Month: byte(c.Month), Day: byte(c.Day), Hour: byte(c.Hour), Minute: byte(c.Minute), Turn: c.Turns}
	g.ElapsedMs, g.lastProcessedHour, g.LastCamped, g.LastWandererMonth = snapshot.World.ElapsedMs, snapshot.World.LastProcessedHour, snapshot.World.LastCamped, snapshot.World.LastWandererMonth
	g.combatEncounterSequence, g.isSailing = snapshot.World.CombatSequence, false
}

func (g *GameState) initializeSnapshotMapsAndControllers() {
	g.TheOdds = references.NewDefaultTheOdds()
	g.DebugOptions = references.DebugOptions{FreeMove: false, MonsterGen: true}
	g.MapState.LayeredMaps = *map_state.NewLayeredMaps(g.GameReferences.TileReferences, g.GameReferences.OverworldLargeMapReference, g.GameReferences.UnderworldLargeMapReference, g.MapState.XTilesVisibleOnGameScreen, g.MapState.YTilesVisibleOnGameScreen)
	g.LargeMapNPCAIController = make(map[references.World]*ai.NPCAIControllerLargeMap)
	for _, world := range []references.World{references.OVERWORLD, references.UNDERWORLD} {
		g.LargeMapNPCAIController[world] = ai.NewNPCAIControllerLargeMap(ai.NewNPCAIControllerLargeMapInput{World: world, TileRefs: g.GameReferences.TileReferences, MapState: &g.MapState, DebugOptions: &g.DebugOptions, TheOdds: &g.TheOdds, EnemyReferences: g.GameReferences.EnemyReferences, DateTime: &g.DateTime, WindProvider: g, RandomProvider: g.AsRandomProvider()})
	}
	switch g.MapState.PlayerLocation.Location.GetMapType() {
	case references.LargeMapType:
		g.CurrentNPCAIController = g.GetCurrentLargeMapNPCAIController()
	case references.SmallMapType:
		slr := g.locationReference(g.MapState.PlayerLocation.Location)
		g.resetAndCreateSmallMap(slr, g.GameReferences.TileReferences)
		g.CurrentNPCAIController = ai.NewNPCAIControllerSmallMap(slr, g.GameReferences.TileReferences, &g.MapState, &g.DateTime, g,
			basementChestPopulationChecker{state: g, preserveAbsent: false})
		g.CurrentNPCAIController.PopulateMapFirstLoad()
	case references.DungeonMapType:
		g.CurrentNPCAIController = nil
	case references.CombatMapType, references.CampingMapType:
		panic("transient map passed snapshot compatibility validation")
	}
	g.ItemStacksMap = *references.NewItemStacksMap()
}

func (g *GameState) restoreDungeon(saved *save_snapshot.DungeonSnapshot) error {
	if saved == nil {
		return nil
	}

	var cells references.DungeonCells
	index := 0
	for floor := range references.DungeonFloorCount {
		for y := range int(references.YDungeonMapTiles) {
			for x := range int(references.XDungeonMapTiles) {
				cells[floor][y][x] = references.DungeonCell(saved.Cells[index])
				index++
			}
		}
	}
	dungeon, err := newDungeonState(
		references.Location(saved.Location),
		cells,
		references.DungeonFacing(saved.Facing),
		references.DungeonLastDirection(saved.LastDirection),
	)
	if err != nil {
		return fmt.Errorf("hydrating active dungeon: %w", err)
	}
	dungeon.quicknessSkipNext = saved.QuicknessSkipNext
	var assessments DungeonChestAssessments
	for floor := range assessments {
		for y := range assessments[floor] {
			for x := range assessments[floor][y] {
				assessments[floor][y][x] = references.ChestAssessment(saved.ChestAssessments[floor*64+y*8+x])
			}
		}
	}
	if err := dungeon.RestoreChestAssessments(assessments); err != nil {
		return fmt.Errorf("hydrating dungeon chest assessments: %w", err)
	}
	if saved.Wanderer != nil {
		dungeon.wanderer = &DungeonWanderer{
			Monster: references.MonsterType(saved.Wanderer.Monster),
			Floor:   references.FloorNumber(saved.Wanderer.Floor),
			Position: references.Position{
				X: references.Coordinate(saved.Wanderer.X),
				Y: references.Coordinate(saved.Wanderer.Y),
			},
			PreviousPosition: references.Position{
				X: references.Coordinate(saved.Wanderer.PreviousX),
				Y: references.Coordinate(saved.Wanderer.PreviousY),
			},
			OnCeiling: saved.Wanderer.OnCeiling,
		}
	}
	g.installDungeonState(dungeon)

	return nil
}

func (g *GameState) restoreProgression(snapshot save_snapshot.SaveSnapshot) {
	p := snapshot.Progression
	g.SearchManager.restoreState(p.SearchFound, p.SkullKeysFoundDay, p.ReagentCachesFoundDay)
	g.restoreSurfaceCorpsesForCurrentMap()
	for index, stone := range p.Moonstones {
		g.moonstoneSystem.Moonstones[index] = MoonstoneLocation{X: stone.X, Y: stone.Y, Map: stone.Map, Level: stone.Level, State: MoonstoneState(stone.State), IsFound: stone.IsFound}
	}
	g.DungeonStates, g.deferredDungeonRoomProgress, g.hmsCapeActivated = p.Dungeons, p.DungeonRooms, p.HMSCapeActivated
	for index, shrineState := range p.Shrines {
		g.shrineQuestsActive[index], g.shrineQuestsCompleted[index], g.closedShrines[index] = shrineState.Active, shrineState.Completed, shrineState.Ruined
	}
	townAssignments := [references.ShadowlordCount]references.Location{}
	for index, assignment := range p.Shadowlords.TownAssignments {
		townAssignments[index] = references.Location(assignment)
	}
	g.ShadowlordState = &references.ShadowlordState{Alive: p.Shadowlords.Alive, TownAssignments: townAssignments, LastCalled: references.Shadowlord(p.Shadowlords.LastCalled), Summoned: p.Shadowlords.Summoned, AutomaticPresence: references.Shadowlord(p.Shadowlords.AutomaticPresence)}
	g.MagicState = &MagicState{KnownRecipes: make(map[references.Spell]uint16)}
	for spell, recipe := range p.Recipes {
		if recipe.Known {
			restoredSpell := references.Spell(spell)
			mask := recipe.Mask
			if restoredSpell == references.InQuasXen {
				mask = references.GetSpellReagents(restoredSpell)
			}
			g.MagicState.KnownRecipes[restoredSpell] = mask
		}
	}
}

func (g *GameState) restoreEffects(snapshot save_snapshot.SaveSnapshot) {
	e := snapshot.Effects
	g.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectType(e.DurationType), TurnsRemaining: e.DurationTurns, IsToggleItemOnly: e.DurationToggleOnly}
	g.MagicLightMinutesRemaining = e.MagicLightMinutesRemaining
	g.MapState.Lighting.RestoreTorchMinutes(e.CarriedTorchMinutesRemaining)
	g.quicknessWorldActorSkip = e.QuicknessWorldActorSkip
	g.townMountedActorSkip = e.TownMountedActorSkip
	g.townQuicknessActorSkip = e.TownQuicknessActorSkip
	g.outdoorMountedActorSkip = e.OutdoorMountedActorSkip
	g.drunk, g.smashed = e.Drunk, e.Smashed
	g.MapState.Lighting.SetMagicLight(g.MagicLightMinutesRemaining > 0)
	g.GetCurrentLayeredMap().InvalidateLightSourceCache()
	g.AmuletState = ToggleItemState{IsActive: e.Amulet.Active, TurnsRemaining: e.Amulet.Turns}
	g.CrownState = ToggleItemState{IsActive: e.Crown.Active, TurnsRemaining: e.Crown.Turns}
	g.BadgeState = ToggleItemState{IsActive: e.Badge.Active, TurnsRemaining: e.Badge.Turns}
}

func (g *GameState) restoreMapUnits(saved []save_snapshot.MapUnitSnapshot, preserveGeneratedSmallMapUnits bool) error {
	byController := make(map[string][]save_snapshot.MapUnitSnapshot)
	for _, unit := range saved {
		key := fmt.Sprintf("%d/%d", unit.Position.Location, unit.World)
		byController[key] = append(byController[key], unit)
	}
	for _, world := range []references.World{references.UNDERWORLD, references.OVERWORLD} {
		controller := g.LargeMapNPCAIController[world]
		key := fmt.Sprintf("%d/%d", references.BritanniaUnderworld, world)
		units, err := g.hydrateUnits(byController[key], nil)
		if err != nil {
			return err
		}
		controller.GetNpcs().ReplaceWith(units)
		delete(byController, key)
		if g.MapState.PlayerLocation.Location.GetMapType() == references.LargeMapType && g.GetCurrentLargeMapNPCAIController() == controller {
			controller.FreshenExistingNPCsOnMap()
		}
	}
	if small, ok := g.CurrentNPCAIController.(*ai.NPCAIControllerSmallMap); ok {
		key := fmt.Sprintf("%d/0", g.MapState.PlayerLocation.Location)
		savedUnits := byController[key]
		var units []map_units.MapUnit
		var err error
		if preserveGeneratedSmallMapUnits {
			units, err = g.hydrateLegacySmallMapUnits(savedUnits, small.GetNpcs())
		} else {
			units, err = g.hydrateUnits(savedUnits, small.GetNpcs())
		}
		if err != nil {
			return err
		}
		small.GetNpcs().ReplaceWith(units)
		small.FreshenExistingNPCsOnMap()
		delete(byController, key)
	}
	if len(byController) != 0 {
		return errors.New("hydrating map units: unconsumed controller groups")
	}

	return nil
}

func (g *GameState) hydrateLegacySmallMapUnits(
	saved []save_snapshot.MapUnitSnapshot,
	generated *map_units.MapUnits,
) ([]map_units.MapUnit, error) {
	result := make([]map_units.MapUnit, 0, len(*generated)+len(saved))
	candidates := make([]*map_units.NPCFriendly, 0)
	for _, unit := range *generated {
		if unit.Floor() == g.MapState.PlayerLocation.Floor && isLegacySemanticSurfaceObject(unit) {
			if friendly, ok := unit.(*map_units.NPCFriendly); ok &&
				(friendly.NPCReference.GetNPCType() == references.Vehicle || friendly.IsSurfaceChest()) {
				candidates = append(candidates, friendly)
			}

			continue
		}
		result = append(result, unit)
	}
	saved = append([]save_snapshot.MapUnitSnapshot(nil), saved...)
	sort.Slice(saved, func(i, j int) bool { return saved[i].ControllerOrder < saved[j].ControllerOrder })
	matches := matchLegacyGeneratedObjects(saved, candidates)
	usedNPCNumbers := make(map[int]struct{}, len(result)+len(saved))
	for _, unit := range result {
		usedNPCNumbers[unit.MapUnitDetails().NPCNum] = struct{}{}
	}
	for _, match := range matches {
		if match != nil {
			usedNPCNumbers[match.MapUnitDetails().NPCNum] = struct{}{}
		}
	}
	for index := range saved {
		unit := saved[index]
		match := matches[index]
		if match != nil {
			unit.NPCNum = match.MapUnitDetails().NPCNum
			if unit.Kind == save_snapshot.MapUnitKindChest {
				unit.SurfaceChestAuthored = match.IsAuthoredSurfaceChest()
			}
			if unit.Kind == save_snapshot.MapUnitKindVehicle {
				vehicle := *unit.Vehicle
				unit.Vehicle = &vehicle
				unit.Vehicle.OwnedByNPC = match.GetVehicleDetails().IsOwnedByNPC
			}
		} else {
			unit.NPCNum = availableLegacyNPCNumber(unit.NPCNum, usedNPCNumbers)
			if unit.NPCNum < 0 {
				return nil, errors.New("hydrating legacy small-map objects: no free NPC identity")
			}
			usedNPCNumbers[unit.NPCNum] = struct{}{}
		}
		overlay, err := g.hydrateUnits([]save_snapshot.MapUnitSnapshot{unit}, nil)
		if err != nil {
			return nil, err
		}
		if match != nil {
			friendly, ok := overlay[0].(*map_units.NPCFriendly)
			if !ok {
				return nil, fmt.Errorf("hydrating legacy authored object: runtime unit has unexpected type %T", overlay[0])
			}
			friendly.NPCReference = match.NPCReference
		}
		result = append(result, overlay[0])
	}

	return result, nil
}

func matchLegacyGeneratedObjects(
	saved []save_snapshot.MapUnitSnapshot,
	candidates []*map_units.NPCFriendly,
) []*map_units.NPCFriendly {
	matches := make([]*map_units.NPCFriendly, len(saved))
	used := make(map[*map_units.NPCFriendly]struct{}, len(candidates))
	for index, unit := range saved {
		matches[index] = exactLegacyGeneratedObject(unit, candidates, used)
		if matches[index] != nil {
			used[matches[index]] = struct{}{}
		}
	}
	for index, unit := range saved {
		if matches[index] != nil || unit.Kind != save_snapshot.MapUnitKindVehicle {
			continue
		}
		unmatchedSaved := 0
		for otherIndex, other := range saved {
			if matches[otherIndex] == nil && other.Kind == save_snapshot.MapUnitKindVehicle &&
				other.Vehicle.Type == unit.Vehicle.Type {
				unmatchedSaved++
			}
		}
		if unmatchedSaved != 1 {
			continue
		}
		var only *map_units.NPCFriendly
		for _, candidate := range candidates {
			if _, exists := used[candidate]; exists ||
				candidate.GetVehicleDetails().VehicleType != references.VehicleType(unit.Vehicle.Type) {
				continue
			}
			if only != nil {
				only = nil

				break
			}
			only = candidate
		}
		if only != nil {
			matches[index] = only
			used[only] = struct{}{}
		}
	}

	return matches
}

func exactLegacyGeneratedObject(
	unit save_snapshot.MapUnitSnapshot,
	candidates []*map_units.NPCFriendly,
	used map[*map_units.NPCFriendly]struct{},
) *map_units.NPCFriendly {
	if unit.SurfaceChestTransient {
		return nil
	}
	position := references.Position{X: references.Coordinate(unit.Position.X), Y: references.Coordinate(unit.Position.Y)}
	for _, candidate := range candidates {
		if _, exists := used[candidate]; exists || candidate.Pos() != position ||
			candidate.Floor() != references.FloorNumber(unit.Position.Floor) {
			continue
		}
		if unit.Kind == save_snapshot.MapUnitKindChest && candidate.IsSurfaceChest() {
			return candidate
		}
		if unit.Kind == save_snapshot.MapUnitKindVehicle &&
			candidate.GetVehicleDetails().VehicleType == references.VehicleType(unit.Vehicle.Type) {
			return candidate
		}
	}

	return nil
}

func availableLegacyNPCNumber(preferred int, used map[int]struct{}) int {
	if _, exists := used[preferred]; !exists && preferred >= 0 && preferred < map_units.MaximumNpcsPerMap {
		return preferred
	}
	for candidate := range map_units.MaximumNpcsPerMap {
		if _, exists := used[candidate]; !exists {
			return candidate
		}
	}

	return -1
}

func isLegacySemanticSurfaceObject(unit map_units.MapUnit) bool {
	friendly, ok := unit.(*map_units.NPCFriendly)
	if !ok {
		return false
	}
	if friendly.NPCReference.GetNPCType() == references.Vehicle || friendly.IsSurfaceChest() {
		return true
	}

	switch friendly.NPCReference.GetSpriteIndex() { //nolint:exhaustive // Only converted item-object sprites are replaced.
	case indexes.ItemMoney,
		indexes.ItemPotion,
		indexes.ItemScroll,
		indexes.ItemWeapon,
		indexes.ItemShield,
		indexes.ItemKey,
		indexes.ItemGem,
		indexes.ItemHelm,
		indexes.ItemRing,
		indexes.ItemArmour,
		indexes.ItemAnkh,
		indexes.ItemTorch,
		indexes.ItemFood,
		indexes.Moon,
		indexes.Shard,
		indexes.Crown,
		indexes.Sceptre,
		indexes.Amulet:
		return true
	default:
		return false
	}
}

func (g *GameState) hydrateUnits(saved []save_snapshot.MapUnitSnapshot, generated *map_units.MapUnits) ([]map_units.MapUnit, error) {
	sort.Slice(saved, func(i, j int) bool { return saved[i].ControllerOrder < saved[j].ControllerOrder })
	generatedByID := make(map[int]*map_units.NPCFriendly)
	if generated != nil {
		for _, unit := range *generated {
			if friendly, ok := unit.(*map_units.NPCFriendly); ok {
				generatedByID[unit.MapUnitDetails().NPCNum] = friendly
			}
		}
	}
	result := make([]map_units.MapUnit, 0, len(saved))
	for _, unit := range saved {
		var runtime map_units.MapUnit
		switch unit.Kind {
		case "npc":
			friendly, ok := generatedByID[unit.NPCNum]
			if !ok || friendly == nil {
				return nil, fmt.Errorf("hydrating NPC %d: reference unavailable", unit.NPCNum)
			}
			if unit.SurfaceChestNumber != nil {
				friendly.RestoreSurfaceChestNumber(byte(*unit.SurfaceChestNumber))
				friendly.RestoreSurfaceChestAssessment(references.ChestAssessment(unit.SurfaceChestAssessment))
			}
			runtime = friendly
		case "vehicle":
			friendly := map_units.NewNPCFriendlyVehiceNewRef(references.VehicleType(unit.Vehicle.Type), references.Position{X: references.Coordinate(unit.Position.X), Y: references.Coordinate(unit.Position.Y)}, references.FloorNumber(unit.Position.Floor))
			runtime = friendly
		case "chest":
			friendly := g.hydrateSurfaceChest(unit)
			friendly.RestoreSurfaceChestNumber(byte(*unit.SurfaceChestNumber))
			friendly.RestoreSurfaceChestAssessment(references.ChestAssessment(unit.SurfaceChestAssessment))
			runtime = friendly
		case "enemy":
			enemyRef := g.enemyReference(references.MonsterType(unit.ReferenceID))
			if enemyRef == nil {
				return nil, fmt.Errorf("hydrating enemy monster type %d: reference unavailable", unit.ReferenceID)
			}
			reference := *enemyRef
			if unit.EnemyVariant == string(map_units.EnemyVariantPirate) {
				tile := g.GameReferences.TileReferences.GetTile(pirateSpriteForSnapshotFacing(references.Direction(unit.EnemyFacing)))
				reference = references.NewPirateShipEnemyReference(tile)
			}
			enemy := map_units.NewEnemyNPC(reference, unit.NPCNum)
			enemy.SetCurrentHP(unit.EnemyCurrentHP)
			if unit.EnemyVariant == string(map_units.EnemyVariantShadowlord) {
				enemy.SetShadowlordIdentity(references.Shadowlord(*unit.Shadowlord))
			}
			runtime = &enemy
		default:
			return nil, fmt.Errorf("hydrating map unit: unsupported kind %q", unit.Kind)
		}
		runtime.SetPos(references.Position{X: references.Coordinate(unit.Position.X), Y: references.Coordinate(unit.Position.Y)})
		runtime.SetFloor(references.FloorNumber(unit.Position.Floor))
		runtime.SetVisible(unit.Visible)
		if unit.Kind == "vehicle" {
			friendly, ok := runtime.(*map_units.NPCFriendly)
			if !ok {
				return nil, fmt.Errorf("hydrating vehicle: runtime unit has unexpected type %T", runtime)
			}
			restoreVehicle(friendly.GetVehicleDetails(), *unit.Vehicle)
			if friendly.GetVehicleDetails().VehicleType == references.CarpetVehicle &&
				!friendly.GetVehicleDetails().IsOwnedByNPC {
				generatedCarpet := generatedByID[unit.NPCNum]
				if generatedCarpet != nil &&
					generatedCarpet.GetVehicleDetails().VehicleType == references.CarpetVehicle &&
					generatedCarpet.GetVehicleDetails().IsOwnedByNPC {
					friendly.GetVehicleDetails().IsOwnedByNPC = true
				}
			}
		}
		details := runtime.MapUnitDetails()
		details.NPCNum, details.SkipNextTurn = unit.NPCNum, unit.SkipNextTurn
		details.SetAIOverride(
			references.AiType(unit.OverriddenAI),
			references.AIOverrideProvenance(unit.AIOverrideProvenance),
		)
		arrivedScheduleSlot := references.NPCScheduleSlot(0)
		if unit.ArrivedScheduleSlot != nil {
			arrivedScheduleSlot = references.NPCScheduleSlot(*unit.ArrivedScheduleSlot)
		} else if friendly, ok := runtime.(*map_units.NPCFriendly); ok {
			arrivedScheduleSlot = friendly.InferArrivedScheduleSlot()
		}
		details.RestoreArrivedScheduleSlot(arrivedScheduleSlot)
		if unit.ConsumedTalk3ScheduleSlot != nil {
			friendly, ok := runtime.(*map_units.NPCFriendly)
			if !ok || unit.Kind != "npc" || !friendly.RestoreTalk3OneShot(references.NPCScheduleSlot(*unit.ConsumedTalk3ScheduleSlot)) {
				return nil, fmt.Errorf("hydrating NPC %d: persisted TALK3 mutation is incompatible with schedule slot %d", unit.NPCNum, *unit.ConsumedTalk3ScheduleSlot)
			}
		}
		details.PendingInteraction = pendingInteractionFromLegacySnapshot(unit.LegacyWantsToAutoTalk, runtime, g.DateTime)
		path := make([]references.Position, len(unit.CurrentPath))
		for index, position := range unit.CurrentPath {
			path[index] = references.Position{X: references.Coordinate(position.X), Y: references.Coordinate(position.Y)}
		}
		details.SetCurrentPath(path)
		result = append(result, runtime)
	}

	return result, nil
}

func pendingInteractionFromLegacySnapshot(pending bool, unit map_units.MapUnit, date datetime.UltimaDate) map_units.PendingNPCInteraction {
	if !pending {
		return map_units.PendingNPCInteractionNone
	}
	friendly, ok := unit.(*map_units.NPCFriendly)
	if !ok {
		return map_units.PendingNPCInteractionTalk
	}
	action := friendly.GetIndividualBehaviourByUltimaData(date).Ai
	if action == references.ExtortOrAttackOrFollow || action == references.DrudgeWorthThing {
		monster, combatCapable := friendly.NPCReference.SmallMapCombatMonsterType()
		// The persisted boolean predates typed interactions; recover attack contact
		// from restored intent, with Astaroth's dialog marker taking precedence.
		switch {
		case friendly.MapUnitDetails().AIOverrideProvenance() == references.AIOverrideProvenanceAutomaticAstaroth:
			return map_units.PendingNPCInteractionTalk
		case monster == references.MonsterGuard:
			return map_units.PendingNPCInteractionGuardArrest
		case combatCapable:
			return map_units.PendingNPCInteractionHostileCombat
		}
	}

	return map_units.PendingNPCInteractionTalk
}

func pirateSpriteForSnapshotFacing(direction references.Direction) indexes.SpriteIndex {
	switch direction {
	case references.Up:
		return indexes.PirateShipUp
	case references.Down:
		return indexes.PirateShipDown
	case references.Left:
		return indexes.PirateShipLeft
	case references.Right:
		return indexes.PirateShipRight
	case references.NoneDirection:
		panic("pirate snapshot requires a cardinal facing")
	default:
		panic("pirate snapshot has invalid facing")
	}
}

func (g *GameState) enemyReference(monsterType references.MonsterType) *references.EnemyReference {
	return enemyReference(g.GameReferences.EnemyReferences, monsterType)
}

func enemyReference(enemyReferences *references.EnemyReferences, monsterType references.MonsterType) *references.EnemyReference {
	for index := range *enemyReferences {
		if (*enemyReferences)[index].MonsterType == monsterType {
			return &(*enemyReferences)[index]
		}
	}

	return nil
}

func restoreVehicle(vehicle *map_units.VehicleDetails, saved save_snapshot.VehicleSnapshot) {
	vehicle.Restore(map_units.VehicleDetailsSnapshot{VesselID: saved.VesselID, VehicleType: references.VehicleType(saved.Type), CurrentDirection: references.Direction(saved.CurrentDirection), PreviousDirection: references.Direction(saved.PreviousDirection), IsOwnedByNPC: saved.OwnedByNPC, HullHP: saved.Hull, MaxHullHP: map_units.MaxHullHP, SailState: map_units.SailState(saved.SailState), SkiffQuantity: saved.Skiffs, PendingDelivery: saved.PendingDelivery})
	vehicle.HasHMSCape = saved.HasHMSCape
}

func (g *GameState) restoreItemStacks(stacks []save_snapshot.ItemStackSnapshot) error {
	for _, saved := range stacks {
		if saved.Position.Location != int(g.MapState.PlayerLocation.Location) || saved.Position.Floor != int(g.MapState.PlayerLocation.Floor) {
			return errors.New("hydrating item stack: non-current map stack is not representable by runtime ItemStacksMap")
		}
		position := references.Position{X: references.Coordinate(saved.Position.X), Y: references.Coordinate(saved.Position.Y)}
		stack := &references.ItemStack{Items: make(references.ItemStacks, 0, len(saved.Items))}
		for _, item := range saved.Items {
			runtimeItem, ok := references.ItemFromIdentity(references.ItemType(item.Type), item.ID)
			if !ok {
				return fmt.Errorf("hydrating item stack: unsupported item type %d id %d", item.Type, item.ID)
			}
			stack.Items = append(stack.Items, references.ItemAndQuantity{Item: runtimeItem, Quantity: item.Quantity})
		}
		if err := g.ItemStacksMap.Push(&position, stack); err != nil {
			return fmt.Errorf("hydrating item stack: %w", err)
		}
	}

	return nil
}

func (g *GameState) restoreMapMutations(mutations []save_snapshot.MapMutationSnapshot) error {
	for _, mutation := range mutations {
		if mutation.Kind == save_snapshot.MapMutationSetObject && mutation.Value <= 0 {
			continue
		}
		mapType := references.Location(mutation.Position.Location).GetMapType()
		layeredMap := g.MapState.LayeredMaps.GetLayeredMap(mapType, references.FloorNumber(mutation.Position.Floor))
		if layeredMap == nil {
			return fmt.Errorf("hydrating map mutation: location %d floor %d is not loaded", mutation.Position.Location, mutation.Position.Floor)
		}
		position := references.Position{X: references.Coordinate(mutation.Position.X), Y: references.Coordinate(mutation.Position.Y)}
		layer := map_state.MapLayer
		if mutation.Kind == save_snapshot.MapMutationSetObject {
			layer = map_state.MapOverrideLayer
		}
		layeredMap.SetTileByLayer(layer, &position, indexes.SpriteIndex(mutation.Value))
	}

	return nil
}

func playerLocation(position save_snapshot.PositionSnapshot) references.PlayerLocation {
	return references.PlayerLocation{Location: references.Location(position.Location), Position: references.Position{X: references.Coordinate(position.X), Y: references.Coordinate(position.Y)}, Floor: references.FloorNumber(position.Floor)}
}
