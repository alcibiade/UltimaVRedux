package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	conversationcatalog "github.com/bradhannah/UltimaVRedux/internal/conversation/catalog"
	"github.com/bradhannah/UltimaVRedux/internal/conversation/semantic"
	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

const (
	vfxKapowDurationMs        = 500
	vfxEarthquakePixelsScale  = 12.0
	debugProjectileTilePixels = 16
	debugDirectionEastName    = "east"
	debugDirectionNorthName   = "north"
	debugDirectionSouthName   = "south"
	debugDirectionWestName    = "west"
	debugProjectileArrow      = "arrow"
	debugProjectileBlue       = "blue"
	debugProjectileGreen      = "green"
	debugProjectileRed        = "red"
	debugVehicleNone          = "none"
	debugVehicleHorse         = "horse"
	debugVehicleCarpet        = "carpet"
	debugVehicleSkiff         = "skiff"
	debugVehicleFrigate       = "frigate"
	debugVehicleAvatar        = "avatar"
	debugVehicleDock          = "dock"
)

var errInvalidSecondBatchDebugArgument = errors.New("invalid second-batch debug argument")

type debugPositionResult struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type conversationCatalogResult struct {
	Sources        []conversationSourceResult  `json:"sources"`
	SelectedSource string                      `json:"selectedSource"`
	Scripts        []conversationSummaryResult `json:"scripts"`
}

type conversationSourceResult struct {
	ID          string `json:"id"`
	ScriptCount int    `json:"scriptCount"`
}

type conversationSummaryResult struct {
	Index         int    `json:"index"`
	Dialog        int    `json:"dialog"`
	SemanticID    string `json:"semanticId"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	TopicCount    int    `json:"topicCount"`
	QuestionCount int    `json:"questionCount"`
	Patched       bool   `json:"patched"`
}

type conversationDetailResult struct {
	SourceID    string `json:"sourceId"`
	Index       int    `json:"index"`
	Dialog      int    `json:"dialog"`
	SemanticID  string `json:"semanticId"`
	Schema      string `json:"schema"`
	Patched     bool   `json:"patched"`
	RuntimeYAML string `json:"runtimeYaml"`
}

type combatEncounterOptionsResult struct {
	BritanniaMaps   []combatMapSummary             `json:"britanniaMaps"`
	Dungeons        []combatEncounterDungeonResult `json:"dungeons"`
	EntryDirections []debugOptionResult            `json:"entryDirections"`
	CountModes      []debugOptionResult            `json:"countModes"`
	Monsters        []combatEncounterMonsterResult `json:"monsters"`
}

type debugOptionResult struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type combatEncounterDungeonResult struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Rooms []int  `json:"rooms"`
}

type combatEncounterMonsterResult struct {
	ID           int                          `json:"id"`
	Name         string                       `json:"name"`
	Strength     int                          `json:"strength"`
	Dexterity    int                          `json:"dexterity"`
	Intelligence int                          `json:"intelligence"`
	Armour       int                          `json:"armour"`
	Damage       int                          `json:"damage"`
	HitPoints    int                          `json:"hitPoints"`
	MaxGroup     int                          `json:"maxGroup"`
	Friend       *combatEncounterFriendResult `json:"friend,omitempty"`
}

type combatEncounterFriendResult struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type combatEncounterStartResult struct {
	Territory    string                `json:"territory"`
	MapIndex     int                   `json:"mapIndex"`
	Dungeon      int                   `json:"dungeon"`
	DungeonRoom  int                   `json:"dungeonRoom"`
	MonsterType  int                   `json:"monsterType"`
	MonsterName  string                `json:"monsterName"`
	Seed         *uint64               `json:"seed,omitempty"`
	PrimaryCount int                   `json:"primaryCount"`
	FriendCount  int                   `json:"friendCount"`
	TotalCount   int                   `json:"totalCount"`
	Positions    []debugPositionResult `json:"positions"`
}

type vfxStatusResult struct {
	MapType                string                  `json:"mapType"`
	Player                 debugPositionResult     `json:"player"`
	MaximumX               int                     `json:"maximumX"`
	MaximumY               int                     `json:"maximumY"`
	VisualEffects          []visualEffectResult    `json:"visualEffects"`
	ActiveProjectile       *activeProjectileResult `json:"activeProjectile,omitempty"`
	LightingOverlayEnabled bool                    `json:"lightingOverlayEnabled"`
	EarthquakeActive       bool                    `json:"earthquakeActive"`
}

type visualEffectResult struct {
	WorldPosition *debugPositionResult `json:"worldPosition,omitempty"`
	SpriteIndex   int                  `json:"spriteIndex"`
	ScaleMinimum  float64              `json:"scaleMinimum"`
	ScaleMaximum  float64              `json:"scaleMaximum"`
	ScreenX       int                  `json:"screenX"`
	ScreenY       int                  `json:"screenY"`
	Type          string               `json:"type"`
	Duration      float64              `json:"duration"`
	Elapsed       float64              `json:"elapsed"`
}

type activeProjectileResult struct {
	FromTileX    float64 `json:"fromTileX"`
	FromTileY    float64 `json:"fromTileY"`
	ToTileX      float64 `json:"toTileX"`
	ToTileY      float64 `json:"toTileY"`
	CurrentTileX float64 `json:"currentTileX"`
	CurrentTileY float64 `json:"currentTileY"`
	Type         string  `json:"type"`
	TypeID       int     `json:"typeId"`
	Active       bool    `json:"active"`
}

type monsterEcosystemResult struct {
	LocationID         int                  `json:"locationId"`
	LocationName       string               `json:"locationName"`
	MapType            string               `json:"mapType"`
	Floor              int                  `json:"floor"`
	GenerationEnabled  bool                 `json:"generationEnabled"`
	GenerationBaseOdds int                  `json:"generationBaseOdds"`
	Capacity           int                  `json:"capacity"`
	CurrentUnitCount   int                  `json:"currentUnitCount"`
	EnemyCount         int                  `json:"enemyCount"`
	FriendlyCount      int                  `json:"friendlyCount"`
	VehicleCount       int                  `json:"vehicleCount"`
	Enemies            []monsterEnemyResult `json:"enemies"`
}

type monsterEnemyResult struct {
	RuntimeToken         int                 `json:"runtimeToken"`
	MonsterType          int                 `json:"monsterType"`
	Name                 string              `json:"name"`
	Position             debugPositionResult `json:"position"`
	Floor                int                 `json:"floor"`
	Visible              bool                `json:"visible"`
	ActivelyAttacks      bool                `json:"activelyAttacks"`
	CanFlyOverWater      bool                `json:"canFlyOverWater"`
	CanPassThroughWalls  bool                `json:"canPassThroughWalls"`
	DoNotMove            bool                `json:"doNotMove"`
	HasWhirlpoolMovement bool                `json:"hasWhirlpoolMovement"`
	SkipNextTurn         bool                `json:"skipNextTurn"`
	IsSandEnemy          bool                `json:"isSandEnemy"`
	IsWaterEnemy         bool                `json:"isWaterEnemy"`
}

type monsterClearResult struct {
	Removed           int    `json:"removed"`
	Supported         bool   `json:"supported"`
	UnsupportedReason string `json:"unsupportedReason,omitempty"`
}

type monsterSpawnResult struct {
	Spawned         bool                 `json:"spawned"`
	Position        *debugPositionResult `json:"position,omitempty"`
	RejectionReason string               `json:"rejectionReason,omitempty"`
}

type vehicleStatusResult struct {
	VehicleType             int                 `json:"vehicleType"`
	VehicleName             string              `json:"vehicleName"`
	CurrentDirection        string              `json:"currentDirection"`
	PreviousDirection       string              `json:"previousDirection"`
	OwnedByNPC              bool                `json:"ownedByNpc"`
	HullHP                  int                 `json:"hullHp"`
	MaximumHullHP           int                 `json:"maximumHullHp"`
	SailState               string              `json:"sailState"`
	SkiffQuantity           int                 `json:"skiffQuantity"`
	Position                debugPositionResult `json:"position"`
	Floor                   int                 `json:"floor"`
	HMSCapeActive           bool                `json:"hmsCapeActive"`
	RawHullDamageAvailable  bool                `json:"rawHullDamageAvailable"`
	FullHullRepairAvailable bool                `json:"fullHullRepairAvailable"`
}

type vehicleSpawnResult struct {
	VehicleType  int                 `json:"vehicleType"`
	VehicleName  string              `json:"vehicleName"`
	Placement    string              `json:"placement"`
	Dock         string              `json:"dock,omitempty"`
	LocationID   int                 `json:"locationId"`
	LocationName string              `json:"locationName"`
	Position     debugPositionResult `json:"position"`
	Floor        int                 `json:"floor"`
	UnitCount    int                 `json:"unitCount"`
}

type vehicleForceBoardResult struct {
	VehicleStatus   vehicleStatusResult `json:"vehicleStatus"`
	TerrainBypassed bool                `json:"terrainBypassed"`
	MapUnitCount    int                 `json:"mapUnitCount"`
}

func (g *GameScene) executeSecondBatchSceneDebugCommand(request debugging.Request) (debugging.Result, bool) {
	switch request.CommandID {
	case debugging.CommandConversationScriptCatalog:
		return g.executeConversationScriptCatalog(request), true
	case debugging.CommandConversationScriptInspect:
		return g.executeConversationScriptInspect(request), true
	case debugging.CommandVFXStatus:
		status := g.currentVFXStatus()
		status.LightingOverlayEnabled = g.gameState.DebugOptions.LightOverlay

		return debugSuccessResult(request, "VFX status loaded.", status), true
	case debugging.CommandVFXKapow:
		return g.executeVFXKapow(request), true
	case debugging.CommandVFXProjectile:
		return g.executeVFXProjectile(request), true
	case debugging.CommandEnvironmentEarthquake:
		return g.executeVFXEarthquake(request), true
	default:
		return debugging.Result{
			SessionID: "", RequestID: "", CommandID: "", SceneGeneration: 0,
			Invocation:             nil,
			EffectiveExecutionMode: "", Success: false, Summary: "", ErrorCode: "", ErrorMessage: "",
			Changes: nil, Data: nil,
		}, false
	}
}

func (g *GameScene) executeVFXEarthquake(request debugging.Request) debugging.Result {
	if g.gameState.EventQueueManager.HasActiveWorkflows() {
		return debugging.FailureResult(request, "gameplay_workflow_active", "earthquake presentation is unavailable while a gameplay workflow is active")
	}
	durationTenths := mustDebugIntegerArgument(request, "durationTenths")
	intensityTenths := mustDebugIntegerArgument(request, "intensityTenths")
	durationMs := durationTenths * 100
	intensity := float64(intensityTenths) / 10
	if !g.gameState.QueueDebugEarthquakeFeedback(durationMs, intensity) {
		return debugging.FailureResult(request, "earthquake_already_active", "an earthquake presentation is already active")
	}

	return debugSuccessResultWithMode(request, debugging.ExecutionPresentation, "Earthquake presentation queued.", struct {
		DurationTenths      int     `json:"durationTenths"`
		DurationMs          int     `json:"durationMs"`
		IntensityTenths     int     `json:"intensityTenths"`
		Intensity           float64 `json:"intensity"`
		MaximumOffsetPixels float64 `json:"maximumOffsetPixels"`
		Queued              bool    `json:"queued"`
	}{
		DurationTenths: durationTenths, DurationMs: durationMs,
		IntensityTenths: intensityTenths, Intensity: intensity,
		MaximumOffsetPixels: intensity * vfxEarthquakePixelsScale, Queued: true,
	})
}

func (g *GameScene) executeConversationScriptCatalog(request debugging.Request) debugging.Result {
	semanticCatalog, failure := g.debugConversationCatalog(request)
	if failure != nil {
		return *failure
	}
	source := mustDebugStringArgument(request, "source")
	keys := semanticCatalog.OriginalKeys(semantic.OriginalSource(source))
	result := conversationCatalogResult{
		Sources: make([]conversationSourceResult, 0), SelectedSource: source,
		Scripts: make([]conversationSummaryResult, 0, len(keys)),
	}
	for _, originalSource := range []semantic.OriginalSource{
		semantic.OriginalSourceCastle, semantic.OriginalSourceKeep,
		semantic.OriginalSourceTowne, semantic.OriginalSourceDwelling,
	} {
		result.Sources = append(result.Sources, conversationSourceResult{
			ID: string(originalSource), ScriptCount: len(semanticCatalog.OriginalKeys(originalSource)),
		})
	}
	for _, key := range keys {
		program, exists := semanticCatalog.Original(key.Source, key.Dialog)
		if !exists {
			panic("enumerated semantic conversation is missing from its immutable catalog")
		}
		definition, definitionErr := program.Definition()
		if definitionErr != nil {
			return debugging.FailureResult(request, "conversation_script_query_failed", definitionErr.Error())
		}
		result.Scripts = append(result.Scripts, conversationSummaryResult{
			Index: key.Dialog - 1, Dialog: key.Dialog, SemanticID: program.ID(),
			Name: semanticBodyText(definition.Name), Description: semanticBodyText(definition.Description),
			TopicCount: len(definition.Topics), QuestionCount: len(definition.Questions), Patched: program.Patched(),
		})
	}

	return debugSuccessResult(
		request,
		fmt.Sprintf("Conversation catalog loaded: %s (%d scripts).", source, len(result.Scripts)),
		result,
	)
}

func semanticBodyText(body semantic.Body) string {
	var text strings.Builder
	for _, action := range body {
		var content semantic.RichText
		switch value := action.(type) {
		case *semantic.SayAction:
			content = value.Content
		case *semantic.NarrateAction:
			content = value.Content
		default:
			continue
		}
		for _, part := range content {
			switch value := part.(type) {
			case *semantic.TextPart:
				text.WriteString(value.Text)
			case *semantic.RunicPart:
				text.WriteString(value.Text)
			case *semantic.LineBreakPart:
				text.WriteByte('\n')
			}
		}
	}

	return text.String()
}

func (g *GameScene) executeConversationScriptInspect(request debugging.Request) debugging.Result {
	semanticCatalog, failure := g.debugConversationCatalog(request)
	if failure != nil {
		return *failure
	}
	source := mustDebugStringArgument(request, "source")
	index := mustDebugIntegerArgument(request, "index")
	dialog := index + 1
	program, exists := semanticCatalog.Original(semantic.OriginalSource(source), dialog)
	if !exists {
		return debugging.FailureResult(request, "conversation_script_not_found", fmt.Sprintf("semantic conversation is unavailable: %s/%d", source, dialog))
	}

	return debugSuccessResult(
		request,
		fmt.Sprintf("Conversation script loaded: %s dialog %d.", source, dialog),
		newConversationDetailResult(source, index, program),
	)
}

func (g *GameScene) debugConversationCatalog(request debugging.Request) (*conversationcatalog.Catalog, *debugging.Result) {
	if g.gameState.GameReferences == nil || g.gameState.GameReferences.ConversationCatalog == nil {
		result := debugging.FailureResult(
			request,
			"conversation_references_unavailable",
			"semantic conversation catalog is not loaded",
		)

		return nil, &result
	}

	return g.gameState.GameReferences.ConversationCatalog, nil
}

func (g *GameScene) executeVFXKapow(request debugging.Request) debugging.Result {
	position, err := g.debugPositionFromMode(request, "targetMode", "x", "y")
	if err != nil {
		return debugging.FailureResult(request, "invalid_vfx_position", err.Error())
	}
	g.gameState.EventQueueManager.TriggerFeedback("debug_workbench.vfx.kapow."+request.ID, event_queue.PriorityNormal,
		func(sequence *event_queue.SequentialEventBuilder) {
			sequence.First(func() { g.playKapowAtPosition(position) }).
				Animate(vfxKapowDurationMs, func(float32) {}).
				Finally(func() {})
		})
	result := debugSuccessResultWithMode(request, debugging.ExecutionPresentation, "Kapow presentation queued.", struct {
		Position debugPositionResult `json:"position"`
	}{Position: newDebugPosition(position)})

	return result
}

func (g *GameScene) executeVFXProjectile(request debugging.Request) debugging.Result {
	if g.activeProjectile != nil && g.activeProjectile.active {
		return debugging.FailureResult(request, "projectile_already_active", "a projectile is already active")
	}
	if g.gameState.EventQueueManager.HasActiveWorkflows() {
		return debugging.FailureResult(
			request,
			"gameplay_workflow_active",
			"projectile presentation is unavailable while a gameplay workflow is active",
		)
	}
	from, err := g.debugPositionFromMode(request, "fromMode", "fromX", "fromY")
	if err != nil {
		return debugging.FailureResult(request, "invalid_vfx_position", err.Error())
	}
	target, err := g.validDebugMapPosition(
		mustDebugIntegerArgument(request, "toX"),
		mustDebugIntegerArgument(request, "toY"),
	)
	if err != nil {
		return debugging.FailureResult(request, "invalid_vfx_position", err.Error())
	}
	projectileType, ok := debugProjectileType(mustDebugStringArgument(request, "projectileType"))
	if !ok {
		return debugging.FailureResult(request, "invalid_projectile_type", "unsupported projectile type")
	}
	duration := CalculateProjectileDuration(
		float64(from.X)*debugProjectileTilePixels,
		float64(from.Y)*debugProjectileTilePixels,
		float64(target.X)*debugProjectileTilePixels,
		float64(target.Y)*debugProjectileTilePixels,
		projectileType,
		g.gameState.MapState.PlayerLocation.Location,
	)
	g.gameState.EventQueueManager.TriggerFeedback("debug_workbench.vfx.projectile."+request.ID, event_queue.PriorityNormal,
		func(sequence *event_queue.SequentialEventBuilder) {
			sequence.First(func() { g.startProjectile(from, target, projectileType) }).
				Animate(duration, g.updateProjectileProgress).
				Finally(g.clearProjectile)
		})

	return debugSuccessResultWithMode(request, debugging.ExecutionPresentation, "Projectile presentation queued.", struct {
		From       debugPositionResult `json:"from"`
		To         debugPositionResult `json:"to"`
		Type       string              `json:"type"`
		DurationMs uint64              `json:"durationMs"`
	}{
		From: newDebugPosition(from), To: newDebugPosition(target),
		Type: mustDebugStringArgument(request, "projectileType"), DurationMs: duration,
	})
}

func (g *GameScene) debugPositionFromMode(
	request debugging.Request,
	modeID, xID, yID string,
) (references.Position, error) {
	switch mustDebugStringArgument(request, modeID) {
	case "player":
		position := g.gameState.MapState.PlayerLocation.Position

		return g.validDebugMapPosition(int(position.X), int(position.Y))
	case "coordinates":
		coordinateX, xSet, err := optionalDebugIntegerArgument(request, xID)
		if err != nil {
			return references.Position{}, err
		}
		coordinateY, ySet, err := optionalDebugIntegerArgument(request, yID)
		if err != nil {
			return references.Position{}, err
		}
		if !xSet || !ySet {
			return references.Position{}, fmt.Errorf(
				"%w: coordinate mode requires both x and y",
				errInvalidSecondBatchDebugArgument,
			)
		}

		return g.validDebugMapPosition(coordinateX, coordinateY)
	default:
		return references.Position{}, fmt.Errorf(
			"%w: unsupported position mode",
			errInvalidSecondBatchDebugArgument,
		)
	}
}

func (g *GameScene) validDebugMapPosition(coordinateX, coordinateY int) (references.Position, error) {
	maximumX, maximumY := debugMapMaxima(g.gameState.MapState.PlayerLocation.Location.GetMapType())
	if coordinateX < 0 || coordinateX > maximumX || coordinateY < 0 || coordinateY > maximumY {
		return references.Position{}, fmt.Errorf(
			"%w: position (%d,%d) is outside current map bounds 0..%d,0..%d",
			errInvalidSecondBatchDebugArgument,
			coordinateX,
			coordinateY,
			maximumX,
			maximumY,
		)
	}

	return references.Position{
		X: references.Coordinate(coordinateX), //nolint:gosec // Logical bounds are validated above.
		Y: references.Coordinate(coordinateY), //nolint:gosec // Logical bounds are validated above.
	}, nil
}

func (g *GameScene) currentVFXStatus() vfxStatusResult {
	location := g.gameState.MapState.PlayerLocation
	maximumX, maximumY := debugMapMaxima(location.Location.GetMapType())
	status := vfxStatusResult{
		MapType: debugMapTypeName(location.Location.GetMapType()), Player: newDebugPosition(location.Position),
		MaximumX: maximumX, MaximumY: maximumY,
		VisualEffects: make([]visualEffectResult, 0, len(g.visualEffects)), ActiveProjectile: nil,
		LightingOverlayEnabled: false, EarthquakeActive: g.gameState.IsDebugEarthquakeFeedbackActive(),
	}
	for _, effect := range g.visualEffects {
		item := visualEffectResult{
			WorldPosition: nil, SpriteIndex: int(effect.SpriteIndex), ScaleMinimum: effect.ScaleMin,
			ScaleMaximum: effect.ScaleMax, ScreenX: effect.X, ScreenY: effect.Y, Type: effect.Type,
			Duration: effect.Duration, Elapsed: effect.Elapsed,
		}
		if effect.WorldPos != nil {
			position := newDebugPosition(*effect.WorldPos)
			item.WorldPosition = &position
		}
		status.VisualEffects = append(status.VisualEffects, item)
	}
	if projectile := g.activeProjectile; projectile != nil {
		status.ActiveProjectile = &activeProjectileResult{
			FromTileX: projectile.fromTileX, FromTileY: projectile.fromTileY, ToTileX: projectile.toTileX,
			ToTileY: projectile.toTileY, CurrentTileX: projectile.currentTileX, CurrentTileY: projectile.currentTileY,
			Type:   debugProjectileTypeName(projectile.projectileType),
			TypeID: int(projectile.projectileType), Active: projectile.active,
		}
	}

	return status
}

//nolint:cyclop // Stable command dispatch is clearer as one exhaustive switch.
func executeSecondBatchStateDebugCommand(request debugging.Request, state *game_state.GameState) debugging.Result {
	switch request.CommandID {
	case debugging.CommandCombatEncounterOptions:
		return executeCombatEncounterOptions(request, state)
	case debugging.CommandCombatEncounterStart:
		return executeCombatEncounterStart(request, state)
	case debugging.CommandMonsterEcosystemStatus:
		return debugSuccessResult(
			request,
			"Monster ecosystem status loaded.",
			newMonsterEcosystemResult(state.CurrentMonsterEcosystemSnapshot()),
		)
	case debugging.CommandMonsterGenerationSet:
		return executeMonsterGenerationSet(request, state)
	case debugging.CommandMonsterClearMapEnemies:
		return executeMonsterClearMapEnemies(request, state)
	case debugging.CommandMonsterSpawnWhirlpool:
		return executeMonsterSpawn(request, state, true)
	case debugging.CommandMonsterSpawnPirate:
		return executeMonsterSpawn(request, state, false)
	case debugging.CommandVehicleStatus:
		return debugSuccessResult(
			request,
			"Vehicle status loaded.",
			newVehicleStatusResult(state.CurrentPartyVehicleSnapshot()),
		)
	case debugging.CommandVehicleSpawn:
		return executeVehicleSpawn(request, state)
	case debugging.CommandVehicleForceBoard:
		return executeVehicleForceBoard(request, state)
	case debugging.CommandShipDamageRaw:
		return executeShipDamageRaw(request, state)
	case debugging.CommandShipRepairFull:
		return executeShipRepairFull(request, state)
	case debugging.CommandShipSkiffSet:
		return executeShipSkiffSet(request, state)
	default:
		panic("unhandled second-batch state command")
	}
}

func executeCombatEncounterOptions(request debugging.Request, state *game_state.GameState) debugging.Result {
	if state.GameReferences == nil || state.GameReferences.EnemyReferences == nil {
		return debugging.FailureResult(request, "combat_references_unavailable", "enemy references are not loaded")
	}
	result := combatEncounterOptionsResult{
		BritanniaMaps: make([]combatMapSummary, 0, references.TotalCombatMaps),
		Dungeons:      make([]combatEncounterDungeonResult, 0, references.MaxDungeonCombatIndex+1),
		EntryDirections: []debugOptionResult{
			{ID: int(game_state.CombatEntryWest), Name: debugDirectionWestName},
			{ID: int(game_state.CombatEntryEast), Name: debugDirectionEastName},
			{ID: int(game_state.CombatEntryNorth), Name: debugDirectionNorthName},
			{ID: int(game_state.CombatEntrySouth), Name: debugDirectionSouthName},
		},
		CountModes: []debugOptionResult{
			{ID: int(game_state.CombatEncounterCountAutomatic), Name: "automatic"},
			{ID: int(game_state.CombatEncounterCountExact), Name: "exact"},
			{ID: int(game_state.CombatEncounterCountMaximum), Name: "maximum"},
		},
		Monsters: make([]combatEncounterMonsterResult, 0),
	}
	for index := range references.TotalCombatMaps {
		combatMap := references.NewCombatMapReference(index, references.BritanniaCombatTerritory)
		result.BritanniaMaps = append(result.BritanniaMaps, combatMapSummary{Index: index, Name: combatMap.MapName})
	}
	for dungeon, name := range references.DungeonNames() {
		rooms := make([]int, references.MaxDungeonRooms)
		for room := range rooms {
			rooms[room] = room
		}
		result.Dungeons = append(result.Dungeons, combatEncounterDungeonResult{ID: dungeon, Name: name, Rooms: rooms})
	}
	for _, enemy := range *state.GameReferences.EnemyReferences {
		if enemy.MaxPerMap < 1 {
			continue
		}
		monster := combatEncounterMonsterResult{
			ID: enemy.MonsterType.Index(), Name: enemy.Name, Strength: enemy.Strength, Dexterity: enemy.Dexterity,
			Intelligence: enemy.Intelligence, Armour: enemy.Armour, Damage: enemy.Damage, HitPoints: enemy.HitPoints,
			MaxGroup: enemy.MaxPerMap, Friend: nil,
		}
		if enemy.Friend != nil && enemy.Friend.MonsterType.IsValid() {
			monster.Friend = &combatEncounterFriendResult{ID: enemy.Friend.MonsterType.Index(), Name: enemy.Friend.Name}
		}
		result.Monsters = append(result.Monsters, monster)
	}

	return debugSuccessResult(request, "Combat encounter options loaded.", result)
}

func executeCombatEncounterStart(request debugging.Request, state *game_state.GameState) debugging.Result {
	input, err := combatEncounterRequestFromDebug(request)
	if err != nil {
		return debugging.FailureResult(request, "invalid_combat_encounter", err.Error())
	}
	encounter, err := state.StartCombatEncounter(input)
	if err != nil {
		switch {
		case errors.Is(err, game_state.ErrCombatEncounterAlreadyActive):
			return debugging.FailureResult(request, "combat_encounter_active", err.Error())
		case errors.Is(err, game_state.ErrInvalidCombatEncounter):
			return debugging.FailureResult(request, "invalid_combat_encounter", err.Error())
		default:
			return debugging.FailureResult(request, "combat_encounter_start_failed", err.Error())
		}
	}
	positions := make([]debugPositionResult, 0, len(encounter.Positions))
	for _, position := range encounter.Positions {
		positions = append(positions, newDebugPosition(position))
	}
	result := combatEncounterStartResult{
		Territory: debugCombatTerritoryName(encounter.Territory), MapIndex: encounter.MapIndex,
		Dungeon: int(encounter.Dungeon), DungeonRoom: encounter.DungeonRoom,
		MonsterType: input.MonsterType.Index(), MonsterName: input.MonsterType.Name(), Seed: encounter.Seed,
		PrimaryCount: encounter.PrimaryCount, FriendCount: encounter.FriendCount,
		TotalCount: encounter.PrimaryCount + encounter.FriendCount, Positions: positions,
	}
	debugResult := debugSuccessResultWithMode(
		request,
		debugging.ExecutionSynchronized,
		"Combat encounter started.",
		result,
	)
	appendSemanticChange(&debugResult.Changes, "combat.active", false, true)

	return debugResult
}

//nolint:cyclop,funlen // Conditional parameter validation mirrors the command's typed modes.
func combatEncounterRequestFromDebug(request debugging.Request) (game_state.CombatEncounterRequest, error) {
	input := game_state.CombatEncounterRequest{
		Map: game_state.CombatEncounterMap{
			Territory: 0, BritanniaMap: 0, Dungeon: 0, DungeonRoom: 0,
		},
		EntryDirection: 0, MonsterType: references.MonsterNone,
		CountMode: 0, ExactCount: 0, FriendsEnabled: true, Seed: nil,
	}
	switch mustDebugStringArgument(request, "territory") {
	case "britannia":
		value, supplied, err := optionalDebugIntegerArgument(request, "britanniaMap")
		if err != nil || !supplied {
			return input, invalidSecondBatchArgument("Britannia territory requires britanniaMap")
		}
		input.Map.Territory = references.BritanniaCombatTerritory
		input.Map.BritanniaMap = references.BritanniaCombatMap(value)
	case debugMapDungeon:
		dungeon, dungeonSet, err := optionalDebugIntegerArgument(request, "dungeon")
		if err != nil {
			return input, err
		}
		room, roomSet, err := optionalDebugIntegerArgument(request, "dungeonRoom")
		if err != nil {
			return input, err
		}
		if !dungeonSet || !roomSet {
			return input, invalidSecondBatchArgument("dungeon territory requires dungeon and dungeonRoom")
		}
		input.Map.Territory = references.DungeonCombatTerritory
		input.Map.Dungeon, input.Map.DungeonRoom = references.DungeonCombatIndex(dungeon), room
	default:
		return input, invalidSecondBatchArgument("unsupported combat territory")
	}
	entry, found := debugCombatEntryDirection(mustDebugStringArgument(request, "entryDirection"))
	if !found {
		return input, invalidSecondBatchArgument("unsupported combat entry direction")
	}
	input.EntryDirection = entry
	monsterValue, err := strconv.Atoi(mustDebugStringArgument(request, "monsterType"))
	if err != nil || !references.MonsterType(monsterValue).IsValid() {
		return input, invalidSecondBatchArgument("invalid monster type")
	}
	input.MonsterType = references.MonsterType(monsterValue)
	countMode, ok := debugCombatCountMode(mustDebugStringArgument(request, "countMode"))
	if !ok {
		return input, invalidSecondBatchArgument("unsupported combat count mode")
	}
	input.CountMode = countMode
	exactCount, exactSet, err := optionalDebugIntegerArgument(request, "exactCount")
	if err != nil {
		return input, err
	}
	if countMode == game_state.CombatEncounterCountExact && !exactSet {
		return input, invalidSecondBatchArgument("exact count mode requires exactCount")
	}
	if countMode != game_state.CombatEncounterCountExact && exactSet {
		return input, invalidSecondBatchArgument("exactCount is only valid with exact count mode")
	}
	input.ExactCount = exactCount
	if friends, set, optionalErr := optionalDebugBooleanArgument(request, "friends"); optionalErr != nil {
		return input, optionalErr
	} else if set {
		input.FriendsEnabled = friends
	}
	if seed, set, optionalErr := optionalDebugIntegerArgument(request, "seed"); optionalErr != nil {
		return input, optionalErr
	} else if set {
		if seed < 0 {
			return input, invalidSecondBatchArgument("seed cannot be negative")
		}
		value := uint64(seed)
		input.Seed = &value
	}

	return input, nil
}

func executeMonsterGenerationSet(request debugging.Request, state *game_state.GameState) debugging.Result {
	before := state.CurrentMonsterEcosystemSnapshot()
	enabled, _, err := optionalDebugBooleanArgument(request, "enabled")
	if err != nil {
		return debugging.FailureResult(request, "invalid_monster_generation", err.Error())
	}
	odds := mustDebugIntegerArgument(request, "odds")
	if err = state.SetMonsterGenerationBaseOdds(odds); err != nil {
		return debugging.FailureResult(request, "invalid_monster_generation", err.Error())
	}
	state.SetMonsterGenerationEnabled(enabled)
	after := state.CurrentMonsterEcosystemSnapshot()
	result := debugSuccessResultWithMode(
		request,
		debugging.ExecutionLiveSafe,
		"Monster generation updated.",
		newMonsterEcosystemResult(after),
	)
	appendSemanticChange(&result.Changes, "monster.generation.enabled", before.GenerationEnabled, after.GenerationEnabled)
	appendSemanticChange(
		&result.Changes,
		"monster.generation.baseOdds",
		before.GenerationBaseOdds,
		after.GenerationBaseOdds,
	)

	return result
}

func executeMonsterClearMapEnemies(request debugging.Request, state *game_state.GameState) debugging.Result {
	snapshot := state.CurrentMonsterEcosystemSnapshot()
	expectedLocationID := mustDebugIntegerArgument(request, "expectedLocationId")
	expectedFloor := mustDebugIntegerArgument(request, "expectedFloor")
	expectedEnemyCount := mustDebugIntegerArgument(request, "expectedEnemyCount")
	if int(snapshot.Location) != expectedLocationID || int(snapshot.Floor) != expectedFloor ||
		snapshot.EnemyCount != expectedEnemyCount {
		return debugging.FailureResult(
			request,
			"stale_monster_ecosystem",
			"the current map or enemy roster changed after clear review",
		)
	}
	before := snapshot.EnemyCount
	removed, supported := state.ClearCurrentMapEnemies()
	data := monsterClearResult{Removed: removed, Supported: supported, UnsupportedReason: ""}
	if !supported {
		data.UnsupportedReason = "current map controller does not support authoritative enemy clearing"
	}
	result := debugSuccessResultWithMode(
		request,
		debugging.ExecutionAlteration,
		fmt.Sprintf("Removed %d current-map enemies.", removed),
		data,
	)
	if supported {
		after := state.CurrentMonsterEcosystemSnapshot().EnemyCount
		appendSemanticChange(&result.Changes, "monster.ecosystem.enemyCount", before, after)
	}

	return result
}

func executeMonsterSpawn(request debugging.Request, state *game_state.GameState, whirlpool bool) debugging.Result {
	before := state.CurrentMonsterEcosystemSnapshot().EnemyCount
	var position references.Position
	var spawned bool
	if whirlpool {
		position, spawned = state.ForceSpawnWhirlpoolAtRandomLocationWithPosition()
	} else {
		position, spawned = state.ForceSpawnPirateAtRandomLocationWithPosition()
	}
	data := monsterSpawnResult{Spawned: spawned, Position: nil, RejectionReason: ""}
	name := "Pirate"
	if whirlpool {
		name = "Whirlpool"
	}
	if !spawned {
		data.RejectionReason = monsterSpawnRejection(state, whirlpool)

		return debugSuccessResultWithMode(request, debugging.ExecutionLiveSafe, name+" spawn rejected.", data)
	}
	positionResult := newDebugPosition(position)
	data.Position = &positionResult
	result := debugSuccessResultWithMode(request, debugging.ExecutionLiveSafe, name+" spawned.", data)
	appendSemanticChange(
		&result.Changes,
		"monster.ecosystem.enemyCount",
		before,
		state.CurrentMonsterEcosystemSnapshot().EnemyCount,
	)

	return result
}

func monsterSpawnRejection(state *game_state.GameState, whirlpool bool) string {
	if state.MapState.PlayerLocation.Location.GetMapType() != references.LargeMapType {
		return "current map is not a Britannia large map"
	}
	if whirlpool && state.HasAnyWhirlpool() {
		return "a whirlpool is already present"
	}

	return "no valid off-screen deep-water spawn position was available"
}

func executeShipDamageRaw(request debugging.Request, state *game_state.GameState) debugging.Result {
	operation, err := state.ApplyRawPartyVehicleHullDamage(mustDebugIntegerArgument(request, "amount"))
	if err != nil {
		return debugging.FailureResult(request, "invalid_ship_hull_operation", err.Error())
	}
	result := debugSuccessResultWithMode(
		request,
		debugging.ExecutionSynchronized,
		"Ship hull damaged.",
		newVehicleStatusResult(operation.After),
	)
	appendSemanticChange(&result.Changes, "vehicle.hullHp", operation.Before.HullHP, operation.After.HullHP)

	return result
}

func executeVehicleSpawn(request debugging.Request, state *game_state.GameState) debugging.Result {
	vehicleType, ok := debugVehicleType(mustDebugStringArgument(request, "type"))
	if !ok || (vehicleType != references.FrigateVehicle && vehicleType != references.SkiffVehicle) {
		return vehicleDebugFailure(request, game_state.ErrDebugVehicleUnsupportedType)
	}
	placementValue := mustDebugStringArgument(request, "placement")
	spawnRequest := game_state.DebugVehicleSpawnRequest{VehicleType: vehicleType}
	switch placementValue {
	case debugVehicleAvatar:
		spawnRequest.Placement = game_state.DebugVehicleSpawnAtAvatar
	case debugVehicleDock:
		spawnRequest.Placement = game_state.DebugVehicleSpawnAtDock
		dockValue, exists := request.Arguments["dock"]
		if !exists || len(dockValue) == 0 || string(dockValue) == "null" {
			return debugging.FailureResult(request, "vehicle_dock_required", "dock is required when placement is dock")
		}
		dock, found := debugVehicleDockLocation(mustDebugStringArgument(request, "dock"))
		if !found {
			return vehicleDebugFailure(request, game_state.ErrDebugVehicleInvalidLocation)
		}
		spawnRequest.DockLocation = dock
	default:
		return vehicleDebugFailure(request, game_state.ErrDebugVehicleInvalidLocation)
	}

	operation, err := state.SpawnDebugVehicle(spawnRequest)
	if err != nil {
		return vehicleDebugFailure(request, err)
	}
	data := vehicleSpawnResult{
		VehicleType: int(vehicleType), VehicleName: debugVehicleName(vehicleType), Placement: placementValue,
		LocationID: int(operation.After.Location), LocationName: operation.After.Location.String(),
		Position: newDebugPosition(operation.After.Position), Floor: int(operation.After.Floor),
		UnitCount: operation.After.UnitCount,
	}
	if placementValue == debugVehicleDock {
		data.Dock = debugVehicleDockName(operation.After.DockLocation)
	}
	result := debugSuccessResultWithMode(request, debugging.ExecutionSynchronized, "Vehicle spawned.", data)
	appendSemanticChange(&result.Changes, "vehicle.map.unitCount", operation.Before.UnitCount, operation.After.UnitCount)
	appendSemanticChange(&result.Changes, "vehicle.map.targetVehicleType", nil, debugVehicleName(vehicleType))

	return result
}

func executeVehicleForceBoard(request debugging.Request, state *game_state.GameState) debugging.Result {
	vehicleType, ok := debugVehicleType(mustDebugStringArgument(request, "type"))
	if !ok {
		return vehicleDebugFailure(request, game_state.ErrDebugVehicleUnsupportedType)
	}
	operation, err := state.ForceBoardDebugVehicle(game_state.DebugForceBoardRequest{VehicleType: vehicleType})
	if err != nil {
		return vehicleDebugFailure(request, err)
	}
	result := debugSuccessResultWithMode(request, debugging.ExecutionSynchronized, "Party force-boarded vehicle; terrain restrictions bypassed.", vehicleForceBoardResult{
		VehicleStatus: newVehicleStatusResult(operation.PartyAfter), TerrainBypassed: true, MapUnitCount: operation.MapAfter.UnitCount,
	})
	appendSemanticChange(&result.Changes, "vehicle.type", debugVehicleName(operation.PartyBefore.VehicleType), debugVehicleName(operation.PartyAfter.VehicleType))
	appendSemanticChange(&result.Changes, "vehicle.skiffQuantity", operation.PartyBefore.SkiffQuantity, operation.PartyAfter.SkiffQuantity)
	appendSemanticChange(&result.Changes, "vehicle.map.unitCount", operation.MapBefore.UnitCount, operation.MapAfter.UnitCount)

	return result
}

func vehicleDebugFailure(request debugging.Request, err error) debugging.Result {
	switch {
	case errors.Is(err, game_state.ErrDebugVehicleUnsupportedType):
		return debugging.FailureResult(request, "unsupported_vehicle_type", "vehicle type is not supported by this operation")
	case errors.Is(err, game_state.ErrDebugVehicleUnsupportedContext):
		return debugging.FailureResult(request, "unsupported_vehicle_context", "vehicle operation is not supported in the current gameplay context")
	case errors.Is(err, game_state.ErrDebugVehicleInvalidLocation):
		return debugging.FailureResult(request, "invalid_vehicle_location", "vehicle destination is invalid")
	case errors.Is(err, game_state.ErrDebugVehicleMapUnavailable):
		return debugging.FailureResult(request, "vehicle_map_unavailable", "authoritative vehicle map controller is unavailable")
	case errors.Is(err, game_state.ErrDebugVehiclePositionOccupied):
		return debugging.FailureResult(request, "vehicle_position_occupied", "vehicle destination is occupied")
	case errors.Is(err, game_state.ErrDebugVehicleMapFull):
		return debugging.FailureResult(request, "vehicle_map_full", "vehicle destination map has no free unit slot")
	default:
		panic("unhandled debug vehicle error: " + err.Error())
	}
}

func executeShipRepairFull(request debugging.Request, state *game_state.GameState) debugging.Result {
	operation, err := state.RepairPartyVehicleHullToMaximum()
	if err != nil {
		return debugging.FailureResult(request, "invalid_ship_hull_operation", err.Error())
	}
	result := debugSuccessResultWithMode(
		request,
		debugging.ExecutionSynchronized,
		"Ship hull fully repaired.",
		newVehicleStatusResult(operation.After),
	)
	appendSemanticChange(&result.Changes, "vehicle.hullHp", operation.Before.HullHP, operation.After.HullHP)

	return result
}

func executeShipSkiffSet(request debugging.Request, state *game_state.GameState) debugging.Result {
	operation, err := state.SetPartyFrigateSkiffQuantity(mustDebugIntegerArgument(request, "quantity"))
	if err != nil {
		return debugging.FailureResult(request, "invalid_ship_skiff_operation", err.Error())
	}
	result := debugSuccessResultWithMode(
		request,
		debugging.ExecutionSynchronized,
		"Frigate skiff quantity updated.",
		newVehicleStatusResult(operation.After),
	)
	appendSemanticChange(
		&result.Changes,
		"vehicle.skiffQuantity",
		operation.Before.SkiffQuantity,
		operation.After.SkiffQuantity,
	)

	return result
}

func newMonsterEcosystemResult(snapshot game_state.MonsterEcosystemSnapshot) monsterEcosystemResult {
	result := monsterEcosystemResult{
		LocationID: int(snapshot.Location), LocationName: snapshot.Location.String(),
		MapType: debugMapTypeName(snapshot.MapType),
		Floor:   int(snapshot.Floor), GenerationEnabled: snapshot.GenerationEnabled,
		GenerationBaseOdds: snapshot.GenerationBaseOdds, Capacity: snapshot.Capacity,
		CurrentUnitCount: snapshot.CurrentUnitCount, EnemyCount: snapshot.EnemyCount,
		FriendlyCount: snapshot.FriendlyCount, VehicleCount: snapshot.VehicleCount,
		Enemies: make([]monsterEnemyResult, 0, len(snapshot.Enemies)),
	}
	for _, enemy := range snapshot.Enemies {
		result.Enemies = append(result.Enemies, monsterEnemyResult{
			RuntimeToken: enemy.RuntimeToken, MonsterType: enemy.MonsterType.Index(), Name: enemy.Name,
			Position: newDebugPosition(enemy.Position), Floor: int(enemy.Floor), Visible: enemy.Visible,
			ActivelyAttacks: enemy.ActivelyAttacks, CanFlyOverWater: enemy.CanFlyOverWater,
			CanPassThroughWalls: enemy.CanPassThroughWalls, DoNotMove: enemy.DoNotMove,
			HasWhirlpoolMovement: enemy.HasWhirlpoolMovement, SkipNextTurn: enemy.SkipNextTurn,
			IsSandEnemy: enemy.IsSandEnemy, IsWaterEnemy: enemy.IsWaterEnemy,
		})
	}

	return result
}

func newVehicleStatusResult(snapshot game_state.PartyVehicleSnapshot) vehicleStatusResult {
	return vehicleStatusResult{
		VehicleType: int(snapshot.VehicleType), VehicleName: debugVehicleName(snapshot.VehicleType),
		CurrentDirection:  debugDirectionName(snapshot.CurrentDirection),
		PreviousDirection: debugDirectionName(snapshot.PreviousDirection),
		OwnedByNPC:        snapshot.IsOwnedByNPC, HullHP: snapshot.HullHP, MaximumHullHP: snapshot.MaxHullHP,
		SailState: snapshot.SailState.String(), SkiffQuantity: snapshot.SkiffQuantity,
		Position: newDebugPosition(snapshot.Position), Floor: int(snapshot.Floor), HMSCapeActive: snapshot.HMSCapeActive,
		RawHullDamageAvailable: snapshot.RawHullDamageAvailable, FullHullRepairAvailable: snapshot.FullHullRepairAvailable,
	}
}

func newConversationDetailResult(source string, index int, program *conversationcatalog.Program) conversationDetailResult {
	return conversationDetailResult{
		SourceID: source, Index: index, Dialog: index + 1,
		SemanticID: program.ID(), Schema: semantic.SchemaVersion, Patched: program.Patched(),
		RuntimeYAML: string(program.CanonicalYAML()),
	}
}

func newDebugPosition(position references.Position) debugPositionResult {
	return debugPositionResult{X: int(position.X), Y: int(position.Y)}
}

func debugCombatEntryDirection(value string) (game_state.CombatEntryDirection, bool) {
	switch value {
	case debugDirectionWestName:
		return game_state.CombatEntryWest, true
	case debugDirectionEastName:
		return game_state.CombatEntryEast, true
	case debugDirectionNorthName:
		return game_state.CombatEntryNorth, true
	case debugDirectionSouthName:
		return game_state.CombatEntrySouth, true
	default:
		return 0, false
	}
}

func debugCombatCountMode(value string) (game_state.CombatEncounterCountMode, bool) {
	switch value {
	case "automatic":
		return game_state.CombatEncounterCountAutomatic, true
	case "exact":
		return game_state.CombatEncounterCountExact, true
	case "maximum":
		return game_state.CombatEncounterCountMaximum, true
	default:
		return 0, false
	}
}

func debugCombatTerritoryName(territory references.CombatMapTerritory) string {
	if territory == references.DungeonCombatTerritory {
		return debugMapDungeon
	}

	return "britannia"
}

func debugProjectileType(value string) (references.MissileType, bool) {
	types := map[string]references.MissileType{
		debugProjectileArrow: references.MissileArrow, "axe": references.MissileAxe, "rock": references.MissileRock,
		"cannonball": references.MissileCannonBall, "violet": references.MissileViolet,
		debugProjectileRed: references.MissileRed, debugProjectileBlue: references.MissileBlue,
		debugProjectileGreen: references.MissileGreen,
	}
	projectileType, ok := types[value]

	return projectileType, ok
}

func debugProjectileTypeName(projectileType references.MissileType) string {
	for _, name := range []string{
		debugProjectileArrow, "axe", "rock", "cannonball", "violet", debugProjectileRed,
		debugProjectileBlue, debugProjectileGreen,
	} {
		if candidate, _ := debugProjectileType(name); candidate == projectileType {
			return name
		}
	}

	return debugMapUnknown
}

func debugVehicleName(vehicleType references.VehicleType) string {
	switch vehicleType {
	case references.NoPartyVehicle:
		return debugVehicleNone
	case references.CarpetVehicle:
		return debugQuestCarpet
	case references.HorseVehicle:
		return debugVehicleHorse
	case references.SkiffVehicle:
		return debugVehicleSkiff
	case references.FrigateVehicle:
		return debugVehicleFrigate
	case references.NPC:
		return "npc"
	default:
		return debugMapUnknown
	}
}

func debugVehicleType(value string) (references.VehicleType, bool) {
	switch value {
	case debugVehicleHorse:
		return references.HorseVehicle, true
	case debugVehicleCarpet:
		return references.CarpetVehicle, true
	case debugVehicleSkiff:
		return references.SkiffVehicle, true
	case debugVehicleFrigate:
		return references.FrigateVehicle, true
	default:
		return references.NoPartyVehicle, false
	}
}

func debugVehicleDockLocation(value string) (references.Location, bool) {
	switch value {
	case "jhelom":
		return references.Jhelom, true
	case "minoc":
		return references.Minoc, true
	case "east_britanny":
		return references.EastBritanny, true
	case "buccaneers_den":
		return references.BuccaneersDen, true
	default:
		return 0, false
	}
}

func debugVehicleDockName(location references.Location) string {
	switch location {
	case references.Jhelom:
		return "jhelom"
	case references.Minoc:
		return "minoc"
	case references.EastBritanny:
		return "east_britanny"
	case references.BuccaneersDen:
		return "buccaneers_den"
	default:
		return debugMapUnknown
	}
}

func debugDirectionName(direction references.Direction) string {
	switch direction {
	case references.NoneDirection:
		return debugVehicleNone
	case references.Up:
		return debugDirectionNorthName
	case references.Down:
		return debugDirectionSouthName
	case references.Left:
		return debugDirectionWestName
	case references.Right:
		return debugDirectionEastName
	default:
		return debugMapUnknown
	}
}

func invalidSecondBatchArgument(message string) error {
	return fmt.Errorf("%w: %s", errInvalidSecondBatchDebugArgument, message)
}
