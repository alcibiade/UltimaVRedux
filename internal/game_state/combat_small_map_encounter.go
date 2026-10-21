package game_state

import (
	"errors"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/ai"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

var (
	// ErrInvalidSmallMapEncounter indicates that a live town NPC cannot form a source-backed encounter.
	ErrInvalidSmallMapEncounter = errors.New("invalid small-map encounter")
	// ErrSmallMapEncounterPending indicates that another deferred town encounter already owns the boundary.
	ErrSmallMapEncounterPending = errors.New("small-map encounter already pending")
)

// SmallMapEncounterCause identifies why a source-backed town encounter began.
type SmallMapEncounterCause int

const (
	// SmallMapEncounterAttack is an explicit adjacent Attack command.
	SmallMapEncounterAttack SmallMapEncounterCause = iota
	// SmallMapEncounterArrestRefusal follows a refusal to surrender to a Guard.
	SmallMapEncounterArrestRefusal
	// SmallMapEncounterShadowlordAdjacency begins at the pre-movement small-map NPC phase.
	SmallMapEncounterShadowlordAdjacency
	// SmallMapEncounterHostileNPCAdjacency follows contact initiated by an authored town monster.
	SmallMapEncounterHostileNPCAdjacency
)

// SmallMapEncounterSourceKind preserves the source-map provenance of the combatant.
type SmallMapEncounterSourceKind int

const (
	// SmallMapEncounterOrdinaryNPC is an authored slot-backed NPC, including monsters.
	SmallMapEncounterOrdinaryNPC SmallMapEncounterSourceKind = iota
	// SmallMapEncounterAutomaticShadowlord is a town-assignment Shadowlord.
	SmallMapEncounterAutomaticShadowlord
	// SmallMapEncounterYellShadowlord is a Shadowlord summoned by name at a Keep.
	SmallMapEncounterYellShadowlord
)

// SmallMapEncounterConsequences records both required and applied source-map effects.
type SmallMapEncounterConsequences struct {
	KarmaPenalty         int
	ActivateGuards       bool
	PersistentSourceDead bool
	CompleteSourceTurn   bool
	GuardActivation      GuardActivationResult
	GuardActivationCount int
	SceptreReclaimed     bool
	SourceRemoved        bool
}

// SmallMapEncounterOrigin retains the exact source NPC and town state needed on return.
type SmallMapEncounterOrigin struct {
	SourceNPC         *map_units.NPCFriendly
	SourceEnemy       *map_units.NPCEnemy
	SourceKind        SmallMapEncounterSourceKind
	Shadowlord        references.Shadowlord
	Cause             SmallMapEncounterCause
	Location          references.Location
	Floor             references.FloorNumber
	Position          references.Position
	NPCNumber         int
	MonsterType       references.MonsterType
	CombatMap         references.BritanniaCombatMap
	ActivePlayerIndex int
	Consequences      SmallMapEncounterConsequences
}

// SelectSmallMapCombatMap reproduces the town branch of encounter selection.
// See ref:CRIME-0004.
func SelectSmallMapCombatMap(terrain *references.Tile) references.BritanniaCombatMap {
	if isOverworldWaterTerrain(terrain) {
		return references.LandSeaCombatMap
	}
	if terrain == nil {
		return references.BrickCombatMap
	}

	//nolint:exhaustive // Unlisted town terrain uses outwar's indoor default.
	switch terrain.Index {
	case indexes.Swamp:
		return references.SwampCombatMap
	case indexes.Grass:
		return references.GladeCombatMap
	case indexes.Forest1, indexes.Forest2:
		return references.TreedCombatMap
	case indexes.Desert1, indexes.LeftDesert2, indexes.RightDesert2:
		return references.DesertCombatMap
	case indexes.Forest3, indexes.Forest4:
		return references.CleanTreeCombatMap
	case indexes.Hills1, indexes.SmallMountains, indexes.TallMountains,
		indexes.LeftHills2, indexes.RightHills2:
		return references.MountainsCombatMap
	case indexes.LeftWoodFloor2, indexes.RightWoodFloor2:
		return references.BigBridgeCombatMap
	case indexes.BrickFloor:
		return references.BrickCombatMap
	}
	if terrain.IsBridge() {
		return references.BigBridgeCombatMap
	}

	return references.BrickCombatMap
}

// StartSmallMapCombatEncounter starts one validated source-backed town encounter.
func (g *GameState) StartSmallMapCombatEncounter(
	source map_units.MapUnit,
	cause SmallMapEncounterCause,
) (*CombatEncounterResult, error) {
	origin, err := g.buildSmallMapEncounterOrigin(source, cause)
	if err != nil {
		return nil, err
	}

	return g.startSmallMapCombatEncounter(origin)
}

// QueueSmallMapCombatEncounter defers entry until an owning dialog workflow has completed.
func (g *GameState) QueueSmallMapCombatEncounter(
	source map_units.MapUnit,
	cause SmallMapEncounterCause,
) error {
	if g.HasPendingOverworldContact() || g.CombatScene != nil {
		return ErrSmallMapEncounterPending
	}
	origin, err := g.buildSmallMapEncounterOrigin(source, cause)
	if err != nil {
		return err
	}
	g.pendingSmallMapEncounter = origin

	return nil
}

func (g *GameState) startSmallMapCombatEncounter(
	origin *SmallMapEncounterOrigin,
) (*CombatEncounterResult, error) {
	if err := g.validateSmallMapEncounterSource(origin); err != nil {
		return nil, err
	}

	count := 1
	if origin.SourceKind == SmallMapEncounterOrdinaryNPC && origin.MonsterType == references.MonsterGuard {
		count = references.GuardGroupCount
	}
	request := CombatEncounterRequest{
		Map: CombatEncounterMap{
			Territory:    references.BritanniaCombatTerritory,
			BritanniaMap: origin.CombatMap,
		},
		EntryDirection: CombatEntryNorth,
		MonsterType:    origin.MonsterType,
		CountMode:      CombatEncounterCountExact,
		ExactCount:     count,
		FriendsEnabled: origin.MonsterType == references.MonsterGuard,
		Seed:           nil,
	}
	if origin.SourceKind == SmallMapEncounterOrdinaryNPC && smallMapNPCIsGroup(origin.Location, origin.NPCNumber) {
		request.CountMode = CombatEncounterCountAutomatic
		request.FriendsEnabled = true
	}
	result, err := g.startCombatEncounter(request, func(combat *CombatState) error {
		if err := g.applySmallMapEncounterConsequences(origin); err != nil {
			return err
		}
		combat.SmallMapOrigin = origin

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("starting small-map combat: %w", err)
	}

	enemyRef, err := g.enemyReferenceForCombat(origin.MonsterType)
	if err != nil {
		panic("prevalidated small-map combat reference became invalid")
	}
	g.SystemCallbacks.Message.AddRowStr(enemyRef.NamePlural)
	g.SystemCallbacks.Message.AddRowStr("Attacked!")

	return result, nil
}

func (g *GameState) buildSmallMapEncounterOrigin(
	source map_units.MapUnit,
	cause SmallMapEncounterCause,
) (*SmallMapEncounterOrigin, error) {
	if cause != SmallMapEncounterAttack && cause != SmallMapEncounterArrestRefusal &&
		cause != SmallMapEncounterShadowlordAdjacency && cause != SmallMapEncounterHostileNPCAdjacency {
		return nil, fmt.Errorf("%w: unsupported cause %d", ErrInvalidSmallMapEncounter, cause)
	}
	if source == nil && cause == SmallMapEncounterArrestRefusal {
		source = g.findAdjacentArrestGuard()
	}

	location := g.MapState.PlayerLocation.Location
	floor := g.MapState.PlayerLocation.Floor
	origin := &SmallMapEncounterOrigin{
		SourceKind:        SmallMapEncounterOrdinaryNPC,
		Shadowlord:        references.NoShadowlord,
		Cause:             cause,
		Location:          location,
		Floor:             floor,
		ActivePlayerIndex: g.GetActivePlayerIndex(),
		Consequences: SmallMapEncounterConsequences{
			KarmaPenalty:         0,
			ActivateGuards:       false,
			PersistentSourceDead: false,
			CompleteSourceTurn:   false,
		},
	}
	switch typedSource := source.(type) {
	case *map_units.NPCFriendly:
		if typedSource == nil || cause == SmallMapEncounterShadowlordAdjacency {
			return nil, fmt.Errorf("%w: invalid ordinary NPC source", ErrInvalidSmallMapEncounter)
		}
		monsterType, err := smallMapNPCMonsterType(typedSource)
		if err != nil {
			return nil, err
		}
		if cause == SmallMapEncounterArrestRefusal && monsterType != references.MonsterGuard {
			return nil, fmt.Errorf("%w: arrest source is not a Guard", ErrInvalidSmallMapEncounter)
		}
		origin.SourceNPC = typedSource
		origin.Position = typedSource.Pos()
		origin.NPCNumber = typedSource.MapUnitDetails().NPCNum
		origin.MonsterType = monsterType
		origin.CombatMap = SelectSmallMapCombatMap(
			g.GetLayeredMapByCurrentLocation().GetTileTopMapOnlyTile(typedSource.PosPtr()),
		)
		if monsterType.IsWaterMonster() {
			origin.CombatMap = references.LandSeaCombatMap
		}
		if monsterType == references.MonsterShadowlord {
			origin.CombatMap = references.ShadowlordCombatMap
			// Original item-use rules ties Stonegate slots 1..3 to the shard death bits.
			// These authored sources are not automatic or Yell manifestations.
			// See ref:CRIME-0004.
			if location == references.Stonegate && origin.NPCNumber >= 1 && origin.NPCNumber <= 3 {
				origin.Shadowlord = references.Shadowlord(origin.NPCNumber - 1)
			}
		}
		ordinary := monsterType < references.MonsterSeaHorse
		// Original town rules call killnpc before combat, regardless of initiator or outcome.
		// See ref:CRIME-0004.
		origin.Consequences.PersistentSourceDead = ordinary && monsterType != references.MonsterGuard
		switch cause {
		case SmallMapEncounterAttack:
			if ordinary {
				origin.Consequences.KarmaPenalty = 5
			}
			origin.Consequences.ActivateGuards = ordinary || monsterType == references.MonsterDevil
			origin.Consequences.CompleteSourceTurn = true
		case SmallMapEncounterArrestRefusal:
			origin.Consequences.ActivateGuards = true
		case SmallMapEncounterShadowlordAdjacency, SmallMapEncounterHostileNPCAdjacency:
			// Adjacency encounters add no Attack or arrest-refusal consequences.
		}
	case *map_units.NPCEnemy:
		if typedSource == nil || cause == SmallMapEncounterArrestRefusal {
			return nil, fmt.Errorf("%w: invalid enemy source", ErrInvalidSmallMapEncounter)
		}
		lord, sourceKind, err := g.classifySmallMapShadowlordSource(typedSource)
		if err != nil {
			return nil, err
		}
		origin.SourceEnemy = typedSource
		origin.SourceKind = sourceKind
		origin.Shadowlord = lord
		origin.Position = typedSource.Pos()
		origin.NPCNumber = typedSource.MapUnitDetails().NPCNum
		origin.MonsterType = references.MonsterShadowlord
		origin.CombatMap = references.ShadowlordCombatMap
		origin.Consequences.ActivateGuards = false
		origin.Consequences.CompleteSourceTurn = cause == SmallMapEncounterAttack
	default:
		return nil, fmt.Errorf("%w: unsupported source type", ErrInvalidSmallMapEncounter)
	}
	if err := g.validateSmallMapEncounterSource(origin); err != nil {
		return nil, err
	}

	return origin, nil
}

func (g *GameState) validateSmallMapEncounterSource(origin *SmallMapEncounterOrigin) error {
	if origin == nil {
		return fmt.Errorf("%w: source is required", ErrInvalidSmallMapEncounter)
	}
	if g.MapState.PlayerLocation.Location.GetMapType() != references.SmallMapType ||
		g.MapState.PlayerLocation.Location != origin.Location ||
		g.MapState.PlayerLocation.Floor != origin.Floor {
		return fmt.Errorf("%w: source small map is not active", ErrInvalidSmallMapEncounter)
	}
	var source map_units.MapUnit = origin.SourceNPC
	if origin.SourceNPC == nil {
		source = origin.SourceEnemy
	}
	if source == nil {
		return fmt.Errorf("%w: source is required", ErrInvalidSmallMapEncounter)
	}
	if g.CurrentNPCAIController == nil || g.CurrentNPCAIController.GetNpcs() == nil || !source.IsVisible() ||
		source.Pos() != origin.Position || source.Floor() != origin.Floor {
		return fmt.Errorf("%w: source NPC is not active", ErrInvalidSmallMapEncounter)
	}
	player := g.MapState.PlayerLocation.Position
	distance := absInt(int(origin.Position.X-player.X)) + absInt(int(origin.Position.Y-player.Y))
	if distance != 1 {
		return fmt.Errorf("%w: source NPC is not adjacent", ErrInvalidSmallMapEncounter)
	}
	if origin.SourceKind == SmallMapEncounterOrdinaryNPC {
		if origin.SourceNPC == nil || origin.SourceEnemy != nil ||
			origin.NPCNumber < 0 || origin.NPCNumber >= party_state.NPcsPerLocation ||
			origin.SourceNPC.NPCReference.Location != origin.Location {
			return fmt.Errorf("%w: source has no persistent town identity", ErrInvalidSmallMapEncounter)
		}
		monsterType, ok := origin.SourceNPC.NPCReference.SmallMapCombatMonsterType()
		if !ok || monsterType != origin.MonsterType {
			return fmt.Errorf("%w: source combat shape changed", ErrInvalidSmallMapEncounter)
		}
		if origin.Cause == SmallMapEncounterHostileNPCAdjacency {
			action := origin.SourceNPC.GetIndividualBehaviourByUltimaData(g.DateTime).Ai
			if monsterType == references.MonsterGuard ||
				(action != references.ExtortOrAttackOrFollow && action != references.DrudgeWorthThing) ||
				origin.SourceNPC.MapUnitDetails().AIOverrideProvenance() == references.AIOverrideProvenanceAutomaticAstaroth {
				return fmt.Errorf("%w: hostile source is not an active authored attacker", ErrInvalidSmallMapEncounter)
			}
		}
		if origin.Shadowlord != references.NoShadowlord &&
			(g.ShadowlordState == nil || !g.ShadowlordState.IsAlive(origin.Shadowlord) ||
				g.PartyState.IsNPCDead(origin.Location, origin.NPCNumber)) {
			return fmt.Errorf("%w: authored Shadowlord is no longer alive", ErrInvalidSmallMapEncounter)
		}
	} else if err := g.validateSmallMapShadowlordOrigin(origin); err != nil {
		return err
	}
	live := g.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionAndFloorOrNil(
		origin.Position,
		origin.Floor,
	)
	if live == nil || *live != source {
		return fmt.Errorf("%w: source entity identity changed", ErrInvalidSmallMapEncounter)
	}

	return nil
}

func (g *GameState) classifySmallMapShadowlordSource(
	source *map_units.NPCEnemy,
) (references.Shadowlord, SmallMapEncounterSourceKind, error) {
	if source == nil || source.Variant() != map_units.EnemyVariantShadowlord ||
		source.EnemyReference.MonsterType != references.MonsterShadowlord {
		return references.NoShadowlord, SmallMapEncounterOrdinaryNPC,
			fmt.Errorf("%w: enemy is not a Shadowlord", ErrInvalidSmallMapEncounter)
	}
	lord, hasIdentity := source.ShadowlordIdentity()
	if !hasIdentity || g.ShadowlordState == nil || !g.ShadowlordState.IsAlive(lord) {
		return references.NoShadowlord, SmallMapEncounterOrdinaryNPC,
			fmt.Errorf("%w: Shadowlord identity is not active", ErrInvalidSmallMapEncounter)
	}
	location := g.MapState.PlayerLocation
	if g.ShadowlordState.Summoned && g.ShadowlordState.LastCalled == lord &&
		references.IsKeepLocation(location.Location) {
		return lord, SmallMapEncounterYellShadowlord, nil
	}
	if g.ShadowlordState.AutomaticPresence == lord {
		return lord, SmallMapEncounterAutomaticShadowlord, nil
	}

	return references.NoShadowlord, SmallMapEncounterOrdinaryNPC,
		fmt.Errorf("%w: Shadowlord has no source provenance", ErrInvalidSmallMapEncounter)
}

func (g *GameState) validateSmallMapShadowlordOrigin(origin *SmallMapEncounterOrigin) error {
	if origin.SourceNPC != nil || origin.SourceEnemy == nil ||
		origin.MonsterType != references.MonsterShadowlord ||
		origin.CombatMap != references.ShadowlordCombatMap ||
		g.ShadowlordState == nil || !g.ShadowlordState.IsAlive(origin.Shadowlord) {
		return fmt.Errorf("%w: invalid Shadowlord origin", ErrInvalidSmallMapEncounter)
	}
	lord, hasIdentity := origin.SourceEnemy.ShadowlordIdentity()
	if !hasIdentity || lord != origin.Shadowlord {
		return fmt.Errorf("%w: Shadowlord identity changed", ErrInvalidSmallMapEncounter)
	}
	switch origin.SourceKind {
	case SmallMapEncounterAutomaticShadowlord:
		if g.ShadowlordState.AutomaticPresence != origin.Shadowlord {
			return fmt.Errorf("%w: automatic Shadowlord presence changed", ErrInvalidSmallMapEncounter)
		}
	case SmallMapEncounterYellShadowlord:
		if !g.ShadowlordState.Summoned || g.ShadowlordState.LastCalled != origin.Shadowlord ||
			!references.IsKeepLocation(origin.Location) {
			return fmt.Errorf("%w: summoned Shadowlord presence changed", ErrInvalidSmallMapEncounter)
		}
	case SmallMapEncounterOrdinaryNPC:
		return fmt.Errorf("%w: Shadowlord has ordinary source provenance", ErrInvalidSmallMapEncounter)
	}

	return nil
}

func smallMapNPCMonsterType(source *map_units.NPCFriendly) (references.MonsterType, error) {
	if source == nil {
		return references.MonsterNone, fmt.Errorf("%w: source NPC is required", ErrInvalidSmallMapEncounter)
	}
	if monsterType, ok := source.NPCReference.SmallMapCombatMonsterType(); ok {
		return monsterType, nil
	}

	return references.MonsterNone, fmt.Errorf("%w: NPC sprite %d is not a safe combat shape",
		ErrInvalidSmallMapEncounter, source.NPCReference.GetSpriteIndex())
}

// Original NPC initialization marks groups by authored slot, not by monster eligibility.
// See ref:CRIME-0004.
func smallMapNPCIsGroup(location references.Location, slot int) bool {
	//nolint:exhaustive // All other authored slots are single encounters.
	switch location {
	case references.Yew:
		return slot == 15 || slot == 17
	case references.Minoc:
		return slot == 1
	case references.Windemere:
		return slot >= 3 && slot <= 9
	case references.Stonegate:
		return slot >= 5 && slot <= 8
	default:
		return false
	}
}

func (g *GameState) findAdjacentArrestGuard() *map_units.NPCFriendly {
	controller, ok := g.CurrentNPCAIController.(*ai.NPCAIControllerSmallMap)
	if !ok {
		return nil
	}
	player := g.MapState.PlayerLocation.Position
	floor := g.MapState.PlayerLocation.Floor
	for _, unit := range guardActivationSlots(controller) {
		guard, isFriendly := unit.(*map_units.NPCFriendly)
		if !isFriendly || !guard.IsVisible() || guard.Floor() != floor ||
			guard.NPCReference.GetNPCType() != references.Guard {
			continue
		}
		distance := absInt(int(guard.Pos().X-player.X)) + absInt(int(guard.Pos().Y-player.Y))
		if distance == 1 {
			return guard
		}
	}

	return nil
}

func (g *GameState) applySmallMapEncounterConsequences(origin *SmallMapEncounterOrigin) error {
	if origin.SourceKind != SmallMapEncounterOrdinaryNPC {
		if origin.Consequences.SourceRemoved {
			panic("small-map Shadowlord precombat consequences applied more than once")
		}
		origin.Consequences.SceptreReclaimed = g.reclaimSceptreForShadowlordCombat()
		if !g.CurrentNPCAIController.GetNpcs().RemoveEnemy(origin.SourceEnemy) {
			panic("prevalidated Shadowlord source could not be removed")
		}
		origin.Consequences.SourceRemoved = true

		return nil
	}
	if origin.Consequences.GuardActivationCount != 0 {
		panic("small-map encounter consequences applied more than once")
	}
	if origin.Consequences.KarmaPenalty > 0 {
		g.AdjustPartyKarma(-origin.Consequences.KarmaPenalty)
	}
	if origin.Consequences.ActivateGuards {
		activation, err := g.ActivateGuards()
		if err != nil {
			return fmt.Errorf("activating guards for small-map combat: %w", err)
		}
		origin.Consequences.GuardActivation = activation
		origin.Consequences.GuardActivationCount++
		g.SetCriminalStatus(true)
	}
	if origin.Consequences.PersistentSourceDead {
		g.PartyState.SetDeadNpc(origin.Location, origin.NPCNumber)
	}
	if origin.MonsterType == references.MonsterShadowlord {
		origin.Consequences.SceptreReclaimed = g.reclaimSceptreForShadowlordCombat()
	}

	return nil
}

func (g *GameState) resolveSmallMapEncounterOrigin(origin *SmallMapEncounterOrigin) {
	if g.MapState.PlayerLocation.Location != origin.Location ||
		g.MapState.PlayerLocation.Floor != origin.Floor {
		panic("small-map combat returned to the wrong source map")
	}
	units := g.CurrentNPCAIController.GetNpcs()
	var source map_units.MapUnit = origin.SourceNPC
	if origin.SourceNPC == nil {
		source = origin.SourceEnemy
	}
	live := units.GetMapUnitAtPositionAndFloorOrNil(origin.Position, origin.Floor)
	if origin.SourceKind == SmallMapEncounterOrdinaryNPC {
		if live == nil || *live != source {
			panic("small-map combat source NPC changed before return")
		}
		if !units.RemoveNPCAtPositionAndFloor(origin.Position, origin.Floor) {
			panic("small-map combat source NPC could not be removed")
		}
	} else if !origin.Consequences.SourceRemoved || live != nil {
		panic("small-map combat Shadowlord source was not removed before combat")
	}
	if origin.SourceKind == SmallMapEncounterYellShadowlord {
		if g.ShadowlordState == nil || !g.ShadowlordState.Summoned ||
			g.ShadowlordState.LastCalled != origin.Shadowlord {
			panic("summoned Shadowlord state changed before combat return")
		}
		g.ShadowlordState.ClearSummoned()
	}
	g.reloadCurrentSmallMapTerrain()
	g.reinitializeTownShadowlordAfterCombat(origin.Location, g.MapState.PlayerLocation.Position)
	g.CurrentNPCAIController.FreshenExistingNPCsOnMap()
	g.restorePreCombatActivePlayer(origin.ActivePlayerIndex)
	if origin.Consequences.CompleteSourceTurn {
		g.FinishTurn()
	}
}

func (g *GameState) queueAdjacentSmallMapShadowlordEncounter() *map_units.NPCEnemy {
	if g.pendingSmallMapEncounter != nil {
		return g.pendingSmallMapEncounter.SourceEnemy
	}
	if g.HasPendingOverworldContact() {
		return nil
	}
	if g.MapState.PlayerLocation.Location.GetMapType() != references.SmallMapType ||
		g.CurrentNPCAIController == nil || g.CurrentNPCAIController.GetNpcs() == nil {
		return nil
	}
	player := g.MapState.PlayerLocation
	source := g.CurrentNPCAIController.GetNpcs().GetFirstEnemyMatching(func(enemy *map_units.NPCEnemy) bool {
		if !enemy.IsVisible() || enemy.Floor() != player.Floor {
			return false
		}
		if _, hasIdentity := enemy.ShadowlordIdentity(); !hasIdentity {
			return false
		}

		position := enemy.Pos()

		return position.IsNextTo(player.Position, enemy.Floor(), player.Floor)
	})
	if source == nil {
		return nil
	}
	if err := g.QueueSmallMapCombatEncounter(source, SmallMapEncounterShadowlordAdjacency); err != nil {
		return nil
	}

	return source
}
