<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, shallowRef, watch } from "vue"
import { provideInstrumentSession } from "./instrument_session"

import {
  envelope,
  type CatalogSnapshot,
  type CommandDefinition,
  type CommandResult,
  type Envelope,
  type SessionReady,
} from "./protocol"
import {
  choicesForParameter,
  hasUnsupportedParameters,
  reconcileDraft,
  resetDraft,
  serializeDraft,
  validateDraft,
  type CommandDraft,
  type FormErrors,
  type OptionSets,
} from "./command_forms"
import PartyCharacterEditor from "./PartyCharacterEditor.vue"
import EquipmentLoadouts from "./EquipmentLoadouts.vue"
import PartyRoster from "./PartyRoster.vue"
import ActiveCombatConsole from "./ActiveCombatConsole.vue"
import LoggingConsole from "./LoggingConsole.vue"
import CombatEncounterComposer from "./CombatEncounterComposer.vue"
import CombatMapCatalog, { type CombatMapCatalogData } from "./CombatMapCatalog.vue"
import CodexCatalog from "./CodexCatalog.vue"
import TileAtlas, { type TileAtlasEntry } from "./TileAtlas.vue"
import SearchLocationAtlas, { type SearchLocationAtlasEntry } from "./SearchLocationAtlas.vue"
import ConsumablesCodex, { type ConsumableCodexEntry } from "./ConsumablesCodex.vue"
import NPCScheduleIndex, { type NPCScheduleEntry } from "./NPCScheduleIndex.vue"
import ConversationScriptExplorer from "./ConversationScriptExplorer.vue"
import MerchantTester from "./MerchantTester.vue"
import MoongateAtlas from "./MoongateAtlas.vue"
import MonsterEcosystemConsole from "./MonsterEcosystemConsole.vue"
import QuestInventoryInspector from "./QuestInventoryInspector.vue"
import SearchItemStacksInspector from "./SearchItemStacksInspector.vue"
import ScenarioLab from "./ScenarioLab.vue"
import SaveLibrary from "./SaveLibrary.vue"
import ShrineProgressInspector from "./ShrineProgressInspector.vue"
import ShrineReference from "./ShrineReference.vue"
import WorkbenchCalendar from "./WorkbenchCalendar.vue"
import WorkbenchWind from "./WorkbenchWind.vue"
import ShipsConsole from "./ShipsConsole.vue"
import DebugOptionsConsole from "./DebugOptionsConsole.vue"
import DisplayConsole from "./DisplayConsole.vue"
import ReduxModificationsConsole from "./ReduxModificationsConsole.vue"
import WorldDiagnostics from "./WorldDiagnostics.vue"
import InventoryLocker from "./InventoryLocker.vue"
import SpellInventory from "./SpellInventory.vue"
import SpellGrimoire, { type SpellGrimoireEntry } from "./SpellGrimoire.vue"
import VFXTestBench from "./VFXTestBench.vue"
import WorldNavigator from "./WorldNavigator.vue"
import DungeonExplorer from "./DungeonExplorer.vue"
import ObservationPane, { type ObservationCommandID } from "./ObservationPane.vue"
import MoonPhaseGlyph from "./MoonPhaseGlyph.vue"
import CommandPalette from "./CommandPalette.vue"
import WorkbenchHome from "./WorkbenchHome.vue"
import { recentNavigationSurfaces, updateNavigationRecents, type NavigationSurface } from "./command_navigation"
import type {
  CalendarStatusData,
  CombatEncounterOptionsData,
  ConversationCatalogData,
  ConversationDetailData,
  MerchantTesterCatalogData,
  MonsterEcosystemData,
  MoongateStatusData,
  QuestItemStatusData,
  SearchStatusData,
  VehicleStatusData,
  VFXStatusData,
  WorldPositionStatusData,
  DungeonExplorerStatusData,
  InventoryLockerStatusData,
  SpellInventoryStatusData,
  ActiveCombatStatusData,
  LoggingStatusData,
	PartyCharacterInspectData,
	DebugOptionsStatusData,
	DisplayStatusData,
	ReduxModificationsStatusData,
	DeathStatusData,
	GuardStatusData,
  ShadowlordStatusData,
  ScenarioStatusData,
  ObservationResultData,
} from "./approved_tool_views"
import type { ShrineStatusData } from "./shrine_views"
import {
  hydrateScenarioComposer,
  cloneScenarioDefinition,
  newScenarioComposerDraft,
  scenarioDraftIsDirty,
  scenarioDraftHasPendingMigrations,
  scenarioIDFromName,
  scenarioLaunchValueIssues,
  scenarioSafeCommands,
	scenarioSourceLabel,
	scenarioStepFromInvocation,
	insertScenarioStep,
	scenarioTerminalStepIndex,
  serializeScenarioDefinition,
  serializeScenarioLaunchValues,
  type ScenarioComposerDraft,
  type ScenarioDefinition,
  type ScenarioDefinitionGetResult,
  type ScenarioDefinitionListResult,
  type ScenarioDefinitionSaveResult,
  type ScenarioDefinitionValidationResult,
  type ScenarioRunStatusResult,
  type ScenarioRunStatus,
  type ScenarioSource,
  type ScenarioStep,
  type ScenarioTask,
  type ScenarioValidationIssue,
} from "./scenarios"
import type { SaveSnapshotGetData, SaveSnapshotListData, SaveSnapshotSummary } from "./saves"
import {
  hydratePartyCharacterDraft,
  partyAlterationPreview,
  partyResourceErrors,
  partyCharacterCommandID,
  partyCharacterFieldIDs,
  partyFieldIsDirty,
  reconcilePartyCharacterDraft,
  serializePartyCharacterDraft,
} from "./party_character_editor"
import { partyEquipmentLoadoutCommandID } from "./party_equipment_loadout"

type ConnectionState = "connecting" | "handshaking" | "ready" | "reconnecting" | "disconnected"
type CatalogWorkspaceID = NavigationSurface["workspace"]
type WorkspaceID = "home" | CatalogWorkspaceID
type HistoryDock = "bottom" | "right"
type ScenarioCaptureIntent = "capture" | "record" | "select"
type ScenarioCaptureDialogMode = "choose" | "new" | "dirty"
type ScenarioDefinitionLoadIntent = "details" | "direct-run" | "capture"
type ScenarioReplacementAction =
  | { type: "load"; reference: string; intent: ScenarioDefinitionLoadIntent; navigateToScenarios?: boolean; historicalRun?: ScenarioRunStatus }
  | { type: "create"; source: ScenarioSource; name?: string }

interface ScenarioDefinitionGetRequest {
  version: number
  draftFingerprint: string
  reference: string
  intent: ScenarioDefinitionLoadIntent
  historicalRun?: ScenarioRunStatus
}

interface ResultEntry extends CommandResult {
  entryID: string
  receivedAt: Date
  ownerCommandID?: string
}

interface PendingRequest {
  commandID: string
  ownerCommandID?: string
  phase: "sending" | "accepted"
  startedAt: number
  lastConfirmedAt: number
  statusRequested: boolean
}

interface CommandCategory {
  name: string
  commands: CommandDefinition[]
}

const resultLimit = 100
const windStatusCommandID = "environment.wind.status"
const loggingStatusCommandID = "logging.status"
const loggingMutationCommandIDs = ["logging.enabled.set", "logging.stdout.set", "logging.level.set", "logging.category.set", "logging.categories.set-all"]
const windSetCommandID = "environment.wind.set"
const shrineStatusCommandID = "shrine.status"
const shrineProgressCommandID = "shrine.progress.update"
const codexQuestStateSetAllCommandID = "codex.quest-state.set-all"
const calendarStatusCommandID = "calendar.status"
const calendarSetTimeCommandID = "calendar.set-time"
const calendarSetDayCommandID = "calendar.set-day"
const searchStatusCommandID = "search.status"
const searchItemStackAddCommandID = "search.item-stack.add"
const searchProgressResetCommandID = "search.progress.reset"
const searchOverlaySetCommandID = "overlay.search.set"
const commandTestSearchCommandID = "command-test.search"
const questItemStatusCommandID = "quest.item.status"
const questItemEnsureCommandID = "quest.item.ensure"
const questItemRemoveCommandID = "quest.item.remove"
const questShardUseCommandID = "quest.shard.use"
const usableInventoryPresetCommandID = "quest.inventory.preset-usable"
const worldPositionStatusCommandID = "world.position.status"
const worldPositionSetCommandID = "world.position.set"
const dungeonExplorerStatusCommandID = "dungeon.explorer.status"
const dungeonExplorerSetupCommandID = "dungeon.explorer.setup"
const movementTeleportCommandID = "movement.teleport"
const movementEnterLocationCommandID = "movement.enter-location"
const movementFloorSetCommandID = "movement.floor.set"
const movementLandmarkTravelCommandID = "movement.landmark.travel"
const legacyWorldSetupCommandIDs = [movementTeleportCommandID, movementEnterLocationCommandID, movementFloorSetCommandID]
const partyKarmaSetCommandID = "party.karma.set"
const partyCharacterInspectCommandID = "party.character.inspect"
const partyRosterMutationCommandIDs = ["party.maximize-levels", "party.status.set", "party.fill", "party.member.add"]
const scenarioStatusCommandID = "scenario.status"
const scenarioStarvationTriggerCommandID = "scenario.starvation.trigger"
const scenarioStarvationHourlyCommandID = "scenario.starvation.force-hourly"
const scenarioGuardsActivateCommandID = "scenario.guards.activate"
const scenarioWaterfallSetupCommandID = "scenario.waterfall.setup"
const scenarioBridgeTrollCommandID = "scenario.bridge-troll.start"
const scenarioWordSetupCommandID = "scenario.word.setup"
const scenarioBlackthornInterrogationCommandID = "scenario.blackthorn-interrogation.start"
const scenarioLordBritishFinaleCommandID = "scenario.lord-british-finale.start"
const scenarioDefinitionListCommandID = "scenario.definition.list"
const scenarioDefinitionGetCommandID = "scenario.definition.get"
const scenarioDefinitionValidateCommandID = "scenario.definition.validate"
const scenarioDefinitionCreateCommandID = "scenario.definition.create"
const scenarioDefinitionUpdateCommandID = "scenario.definition.update"
const scenarioDefinitionDeleteCommandID = "scenario.definition.delete"
const scenarioDefinitionDuplicateCommandID = "scenario.definition.duplicate"
const scenarioRunStartCommandID = "scenario.run.start"
const scenarioRunStatusCommandID = "scenario.run.status"
const scenarioRunCancelCommandID = "scenario.run.cancel"
const saveSnapshotListCommandID = "save.snapshot.list"
const saveSnapshotGetCommandID = "save.snapshot.get"
const saveSnapshotCaptureCommandID = "save.snapshot.capture"
const saveSnapshotLoadCommandID = "save.snapshot.load"
const saveSnapshotDeleteCommandID = "save.snapshot.delete"
const saveSnapshotCommandIDs = [saveSnapshotListCommandID, saveSnapshotGetCommandID, saveSnapshotCaptureCommandID, saveSnapshotLoadCommandID, saveSnapshotDeleteCommandID]
const scenarioCompanionCommandIDs = [
  scenarioDefinitionListCommandID, scenarioDefinitionGetCommandID, scenarioDefinitionValidateCommandID,
  scenarioDefinitionCreateCommandID, scenarioDefinitionUpdateCommandID, scenarioDefinitionDeleteCommandID,
  scenarioDefinitionDuplicateCommandID, scenarioRunStartCommandID, scenarioRunStatusCommandID, scenarioRunCancelCommandID,
]
const commandTestHoleUpCommandID = "command-test.hole-up"
const deathKillPartyCommandID = "death.kill-party"
const deathTestSequenceCommandID = "death.test-sequence"
const observationCommandID = "command-test.telescope"
const observationCommandIDs: ObservationCommandID[] = [observationCommandID, "command-test.spyglass", "command-test.crystal-ball"]
const scenarioMutationCommandIDs = [scenarioStarvationTriggerCommandID, scenarioStarvationHourlyCommandID, scenarioGuardsActivateCommandID, scenarioWaterfallSetupCommandID, scenarioBridgeTrollCommandID, scenarioWordSetupCommandID, scenarioBlackthornInterrogationCommandID, scenarioLordBritishFinaleCommandID, commandTestHoleUpCommandID, deathKillPartyCommandID, deathTestSequenceCommandID]
const moongateStatusCommandID = "moongate.status"
const moongateOverlaySetCommandID = "overlay.moongate.set"
const moongateTravelCommandID = "moongate.travel"
const conversationCatalogCommandID = "conversation.script.catalog"
const conversationInspectCommandID = "conversation.script.inspect"
const combatEncounterOptionsCommandID = "combat.encounter.options"
const combatEncounterStartCommandID = "combat.encounter.start"
const merchantTesterCatalogCommandID = "merchant.tester.catalog"
const merchantTesterPreviewCommandID = "merchant.tester.preview.open"
const merchantTesterLiveCommandID = "merchant.tester.live.open"
const merchantTesterOperationCommandIDs = [merchantTesterPreviewCommandID, merchantTesterLiveCommandID]
const vfxStatusCommandID = "vfx.status"
const vfxKapowCommandID = "vfx.kapow"
const vfxProjectileCommandID = "vfx.projectile"
const environmentEarthquakeCommandID = "environment.earthquake"
const lightingOverlaySetCommandID = "overlay.lighting.set"
const monsterEcosystemStatusCommandID = "monster.ecosystem.status"
const monsterGenerationSetCommandID = "monster.generation.set"
const monsterClearMapEnemiesCommandID = "monster.clear-map-enemies"
const monsterSpawnWhirlpoolCommandID = "monster.spawn-whirlpool"
const monsterSpawnPirateCommandID = "monster.spawn-pirate"
const vehicleStatusCommandID = "vehicle.status"
const vehicleSpawnCommandID = "vehicle.spawn"
const vehicleForceBoardCommandID = "vehicle.force-board"
const shipDamageRawCommandID = "ship.damage-raw"
const shipRepairFullCommandID = "ship.repair-full"
const shipSkiffSetCommandID = "ship.skiff.set"
const shipMutationCommandIDs = [vehicleSpawnCommandID, vehicleForceBoardCommandID, shipDamageRawCommandID, shipRepairFullCommandID, shipSkiffSetCommandID]
const debugOptionsStatusCommandID = "debug.options.status"
const movementFreeMoveSetCommandID = "movement.free-move.set"
const textPlaceholdersSetCommandID = "overlay.text-placeholders.set"
const dialogBoundsSetCommandID = "overlay.dialog-bounds.set"
const unlimitedCastingResourcesSetCommandID = "magic.unlimited-casting-resources.set"
const spellLevelRestrictionsBypassSetCommandID = "magic.spell-level-restrictions-bypass.set"
const debugOptionsMutationCommandIDs = [movementFreeMoveSetCommandID, textPlaceholdersSetCommandID, dialogBoundsSetCommandID, unlimitedCastingResourcesSetCommandID, spellLevelRestrictionsBypassSetCommandID]
const displayStatusCommandID = "display.status"
const displayModeSetCommandID = "display.mode.set"
const displayResolutionSetCommandID = "display.resolution.set"
const displayFullscreenSetCommandID = "display.fullscreen.set"
const displayMutationCommandIDs = [displayModeSetCommandID, displayResolutionSetCommandID, displayFullscreenSetCommandID]
const reduxModificationsStatusCommandID = "redux.modifications.status"
const repeatedMonsterDivisionSetCommandID = "redux.combat.repeated-monster-division.set"
const persistentDungeonDiscoveriesSetCommandID = "redux.dungeons.persist-hidden-discoveries.set"
const subduedUnlitDungeonVisibilitySetCommandID = "redux.dungeons.subdued-unlit-visibility.set"
const smoothSurfaceMovementSetCommandID = "redux.presentation.smooth-surface-movement.set"
const flexibleSextantUseSetCommandID = "redux.navigation.flexible_sextant_use.set"
const reduxModificationCommandIDs = [
  repeatedMonsterDivisionSetCommandID,
  persistentDungeonDiscoveriesSetCommandID,
  subduedUnlitDungeonVisibilitySetCommandID,
  smoothSurfaceMovementSetCommandID,
  flexibleSextantUseSetCommandID,
]
const guardStatusCommandID = "guards.status"
const shadowlordStatusCommandID = "shadowlord.status"
const deathStatusCommandID = "death.status"
const worldDiagnosticCommandIDs = [guardStatusCommandID, shadowlordStatusCommandID, deathStatusCommandID]
const inventoryLockerStatusCommandID = "inventory.locker.status"
const activeCombatStatusCommandID = "combat.status"
const codexCommandIDs = [
  "codex.equipment.catalog", "codex.monster.catalog", "codex.spell.catalog", "codex.tile.catalog",
  "codex.location.catalog", "codex.npc.catalog", "codex.quest.catalog", "codex.save-format.catalog",
  "codex.search-location.catalog", "codex.consumable.catalog",
]
const longCodexCommandIDs = new Set(["codex.equipment.catalog", "codex.monster.catalog", "codex.spell.catalog", "codex.location.catalog"])
const flowCodexCommandIDs = new Set([...longCodexCommandIDs, "codex.tile.catalog", "codex.npc.catalog"])
const activeCombatSleepCommandID = "combat.monster.sleep.set"
const activeCombatExitCommandID = "combat.exit-debug"
const activeCombatSpawnCommandID = "combat.monster.spawn"
const activeCombatRNGSeedSetCommandID = "rng.seed.set"
const activeCombatDamageCommandID = "combat.monster.damage-raw"
const activeCombatKillCommandID = "combat.monster.kill"
const activeCombatKillAllCommandID = "combat.monster.kill-all"
const activeCombatDungeonRoomClearCommandID = "combat.dungeon-room.clear"
const activeCombatClearCommandID = "combat.monster.clear"
const combatPresentationSpeedSetCommandID = "combat.presentation.speed.set"
const activeCombatMutationCommandIDs = [
	activeCombatSleepCommandID, activeCombatExitCommandID, activeCombatSpawnCommandID, activeCombatRNGSeedSetCommandID,
	activeCombatDamageCommandID, activeCombatKillCommandID, activeCombatKillAllCommandID,
	activeCombatDungeonRoomClearCommandID, activeCombatClearCommandID,
	combatPresentationSpeedSetCommandID,
]
const spellInventoryStatusCommandID = "inventory.spell.status"
const spellInventorySetCommandID = "inventory.spell.set"
const spellInventoryPresetMaxCommandID = "inventory.spell.preset-max"
const spellInventoryMutationCommandIDs = [spellInventorySetCommandID, spellInventoryPresetMaxCommandID, "spell.effect.set-remaining", "spell.effect.activate"]
const inventoryLockerMutationCommandIDs = [
	"inventory.provision.set", "inventory.provision.preset", "inventory.reagent.set", "inventory.reagent.preset", "inventory.potion.add",
	"inventory.potion.preset", "inventory.scroll.add", "inventory.scroll.preset", "inventory.equipment.preset",
	"inventory.equipment.grant-preset", "inventory.equipment.set", "inventory.equipment.clear", usableInventoryPresetCommandID,
]
const integratedCommandIDs = new Set([
  loggingStatusCommandID,
  ...loggingMutationCommandIDs,
  windStatusCommandID,
  windSetCommandID,
  "combat.map-catalog.all",
  "shrine.reference.inspect",
  shrineStatusCommandID,
  shrineProgressCommandID,
  codexQuestStateSetAllCommandID,
	partyCharacterInspectCommandID,
  partyCharacterCommandID,
	...partyRosterMutationCommandIDs,
  partyEquipmentLoadoutCommandID,
  calendarStatusCommandID,
  calendarSetTimeCommandID,
  calendarSetDayCommandID,
  searchStatusCommandID,
  searchItemStackAddCommandID,
  searchProgressResetCommandID,
  searchOverlaySetCommandID,
  commandTestSearchCommandID,
  questItemStatusCommandID,
  questItemEnsureCommandID,
  questItemRemoveCommandID,
	questShardUseCommandID,
  worldPositionStatusCommandID,
  worldPositionSetCommandID,
  dungeonExplorerStatusCommandID,
  dungeonExplorerSetupCommandID,
  ...legacyWorldSetupCommandIDs,
  movementLandmarkTravelCommandID,
  partyKarmaSetCommandID,
  scenarioStatusCommandID,
  scenarioStarvationTriggerCommandID,
  scenarioStarvationHourlyCommandID,
  scenarioGuardsActivateCommandID,
  scenarioWaterfallSetupCommandID,
  scenarioBridgeTrollCommandID,
  scenarioWordSetupCommandID,
  scenarioBlackthornInterrogationCommandID,
  scenarioLordBritishFinaleCommandID,
  ...scenarioCompanionCommandIDs,
  ...saveSnapshotCommandIDs,
  commandTestHoleUpCommandID,
  deathKillPartyCommandID,
  deathTestSequenceCommandID,
  moongateStatusCommandID,
  moongateOverlaySetCommandID,
  moongateTravelCommandID,
  conversationCatalogCommandID,
  conversationInspectCommandID,
  combatEncounterOptionsCommandID,
  combatEncounterStartCommandID,
  merchantTesterCatalogCommandID,
  ...merchantTesterOperationCommandIDs,
  vfxStatusCommandID,
  vfxKapowCommandID,
  vfxProjectileCommandID,
  environmentEarthquakeCommandID,
  lightingOverlaySetCommandID,
  monsterEcosystemStatusCommandID,
  monsterGenerationSetCommandID,
  monsterClearMapEnemiesCommandID,
  monsterSpawnWhirlpoolCommandID,
  monsterSpawnPirateCommandID,
  vehicleStatusCommandID,
  vehicleSpawnCommandID,
  vehicleForceBoardCommandID,
  shipDamageRawCommandID,
  shipRepairFullCommandID,
	shipSkiffSetCommandID,
	debugOptionsStatusCommandID,
	...debugOptionsMutationCommandIDs,
	displayStatusCommandID,
  scenarioStatusCommandID,
	...displayMutationCommandIDs,
	reduxModificationsStatusCommandID,
	...reduxModificationCommandIDs,
	...worldDiagnosticCommandIDs,
  inventoryLockerStatusCommandID,
  ...inventoryLockerMutationCommandIDs,
  spellInventoryStatusCommandID,
  ...spellInventoryMutationCommandIDs,
  activeCombatStatusCommandID,
  ...codexCommandIDs,
  ...activeCombatMutationCommandIDs,
  ...observationCommandIDs,
])
const polledIntegratedQueryIDs = new Set([
  loggingStatusCommandID,
  windStatusCommandID,
  shrineStatusCommandID,
  calendarStatusCommandID,
  searchStatusCommandID,
  questItemStatusCommandID,
  worldPositionStatusCommandID,
  dungeonExplorerStatusCommandID,
  moongateStatusCommandID,
  vfxStatusCommandID,
  monsterEcosystemStatusCommandID,
  vehicleStatusCommandID,
  inventoryLockerStatusCommandID,
  spellInventoryStatusCommandID,
  activeCombatStatusCommandID,
	debugOptionsStatusCommandID,
	displayStatusCommandID,
	scenarioStatusCommandID,
])
const connectionTimeoutMs = 5000
const operationStallMs = 10000
const operationReconnectMs = 30000
const sessionStorageKey = "u5r-debug-workbench-session"
const instrumentStorageKey = "u5r-debug-workbench-instrument"
const historyCollapsedStorageKey = "u5r-debug-workbench-history-collapsed"
const historyDockStorageKey = "u5r-debug-workbench-history-dock"
const navigationRecentsStorageKey = "u5r-debug-workbench-navigation-recents"
const deferredScenarioCaptureLimit = 20
const token = document.querySelector<HTMLMetaElement>('meta[name="u5r-workbench-token"]')?.content ?? ""

function restoredHistoryDock(): HistoryDock {
  return localStorage.getItem(historyDockStorageKey) === "right" ? "right" : "bottom"
}

function restoredNavigationRecents(): string[] {
  try {
    const stored = JSON.parse(localStorage.getItem(navigationRecentsStorageKey) ?? "[]")
    return Array.isArray(stored) ? stored.filter((value): value is string => typeof value === "string").slice(0, 8) : []
  } catch {
    return []
  }
}

const connectionState = ref<ConnectionState>("connecting")
const socket = shallowRef<WebSocket | null>(null)
const sessionID = ref(localStorage.getItem(sessionStorageKey) ?? "")
const sceneGeneration = ref(0)
const optionsRevision = ref(0)
const awaitingOptionsRefresh = ref(false)
const commands = ref<CommandDefinition[]>([])
const optionSets = ref<OptionSets>({})
const optionUnavailableReasons = ref<Record<string, string>>({})
const selectedCommandID = ref(localStorage.getItem(instrumentStorageKey) ?? "")
const activeWorkspace = ref<WorkspaceID>("home")
const pageHidden = ref(document.hidden)
const workspaceSelections = new Map<WorkspaceID, string>()
const ownerErrors = ref<Record<string, string>>({})
provideInstrumentSession(computed(() => `${sessionID.value}:${sceneGeneration.value}:${connectionState.value}:${activeWorkspace.value}:${selectedCommandID.value}:${pageHidden.value}`), (message) => {
  ownerErrors.value[selectedCommandID.value] = message
})
const livePollingPaused = ref(false)
const selectedShrineIndex = ref(0)
const conversationSource = ref("towne")
const selectedConversationIndex = ref(-1)
const linkedWorldCoordinates = ref<{ x: number; y: number }>()
const commandSearch = ref("")
const collapsedCategories = ref(new Set<string>())
const drafts = ref<Record<string, CommandDraft>>({})
const dirtyFields = ref<Record<string, string[]>>({})
const formErrors = ref<Record<string, FormErrors>>({})
const formMessages = ref<Record<string, string>>({})
const pending = ref(new Map<string, PendingRequest>())
const results = ref<ResultEntry[]>([])
const latestResults = ref<Record<string, ResultEntry>>({})
const lastSuccessfulWorldPositionResult = ref<ResultEntry>()
const scenarioComposer = ref<ScenarioComposerDraft>(newScenarioComposerDraft())
const scenarioRecording = ref(false)
const scenarioTask = ref<ScenarioTask>("library")
const scenarioTargetActive = ref(false)
const scenarioCaptureDialogOpen = ref(false)
const scenarioCaptureDialogMode = ref<ScenarioCaptureDialogMode>("choose")
const scenarioCaptureIntent = ref<ScenarioCaptureIntent>("select")
const scenarioCaptureResult = ref<ResultEntry>()
const scenarioCaptureNewName = ref("")
const scenarioCaptureSearch = ref("")
const scenarioCaptureSelectedReference = ref("")
const scenarioCaptureExpectedRevision = ref("")
const scenarioCaptureInsertionIndex = ref(0)
const scenarioCaptureError = ref("")
const scenarioCaptureLoadingReference = ref("")
const scenarioReplacementReview = ref<ScenarioReplacementAction>()
const scenarioTargetDialog = ref<HTMLElement | null>(null)
const scenarioReplacementDialog = ref<HTMLElement | null>(null)
const scenarioValidationSnapshot = ref("")
const scenarioValidationContext = ref("")
const scenarioValidationIssues = ref<ScenarioValidationIssue[]>([])
const scenarioValidationIntent = ref<"save" | "review-run" | "direct-run">()
const scenarioReviewedSceneGeneration = ref(0)
const scenarioReviewedBaselineRevision = ref("")
const scenarioReviewedDefinition = ref("")
const scenarioHistoricalRun = ref<ScenarioRunStatus>()
const saveLoadReviewTarget = ref<{ reference: string; revision: string; requestID: string }>()
let rollbackReviewEpoch = 0
const rollbackRequests = new Map<string, { reference: string; revision: string; epoch: number }>()
const scenarioRunStatusChecked = ref(false)
const selectedScenarioRunID = ref("")
const trimmedResults = ref(0)
const reconnectAttempt = ref(0)
const stream = ref<HTMLElement | null>(null)
const historyPanel = ref<HTMLElement | null>(null)
const commandSearchInput = ref<HTMLInputElement | null>(null)
const followResults = ref(true)
const historyCollapsed = ref(localStorage.getItem(historyCollapsedStorageKey) === "true")
const historyDock = ref<HistoryDock>(restoredHistoryDock())
const historyHeight = ref(220)
const historyWidth = ref(420)
const lastCloseReason = ref("")
const retryScheduled = ref(false)
const liveStatus = ref("Workbench initialized.")
const now = ref(Date.now())
const processedRequestIDs = new Set<string>()
const processedRequestOrder: string[] = []
const silentRequestIDs = new Set<string>()
const capturedScenarioTargets = ref(new Map<string, string>())
const deferredScenarioCaptures = ref<ResultEntry[]>([])
const commandPaletteOpen = ref(false)
const commandPaletteLauncher = ref<HTMLButtonElement | null>(null)
const navigationRecents = ref<string[]>(restoredNavigationRecents())
let reconnectTimer: number | undefined
let connectionTimer: number | undefined
let operationTimer: number | undefined
let requestSequence = 0
let connectionGeneration = 0
let localEntrySequence = 0
let stopHistoryResize: (() => void) | undefined
let scenarioTargetPreviousFocus: HTMLElement | null = null
let scenarioReplacementPreviousFocus: HTMLElement | null = null
let commandPalettePreviousFocus: HTMLElement | null = null
let merchantCatalogExplicitRefreshRequired = false
let scenarioReplacementVersion = 0
const scenarioDefinitionGetRequests = new Map<string, ScenarioDefinitionGetRequest>()

watch(selectedCommandID, (commandID) => {
  if (activeWorkspace.value !== "home") workspaceSelections.set(activeWorkspace.value, commandID)
  if (commandID) localStorage.setItem(instrumentStorageKey, commandID)
  else localStorage.removeItem(instrumentStorageKey)
})
watch(historyCollapsed, (collapsed) => localStorage.setItem(historyCollapsedStorageKey, String(collapsed)))
watch(historyDock, (dock) => localStorage.setItem(historyDockStorageKey, dock))
watch([sessionID, sceneGeneration], () => {
  lastSuccessfulWorldPositionResult.value = undefined
}, { flush: "sync" })
watch([sessionID, sceneGeneration, connectionState, activeWorkspace, scenarioTask], () => {
  if (activeWorkspace.value !== "scenarios" || scenarioTask.value !== "run-review" || connectionState.value !== "ready"
    || scenarioReviewedSceneGeneration.value !== sceneGeneration.value) disarmScenarioReview()
}, { flush: "sync" })
watch(sessionID, disarmScenarioReview, { flush: "sync" })
watch([sessionID, sceneGeneration, connectionState, activeWorkspace, selectedCommandID, scenarioTask, selectedScenarioRunID, pageHidden], revokeRollbackReview, { flush: "sync" })

const connectionLabel = computed(() => connectionState.value.replaceAll("_", " ").toUpperCase())
const historyPanelStyle = computed(() => historyDock.value === "right"
  ? { width: historyCollapsed.value ? "38px" : `${historyWidth.value}px` }
  : { height: historyCollapsed.value ? undefined : `${historyHeight.value}px` })
const isReady = computed(() => connectionState.value === "ready")
const supportedCommands = computed(() => commands.value.filter(isSupportedWorkbenchCommand))
const availableCommands = computed(() => supportedCommands.value.filter((command) => commandBelongsToWorkspace(command, activeWorkspace.value)))
const searchableCommands = computed(() => supportedCommands.value.filter((command) => workspaces.some((workspace) => commandBelongsToWorkspace(command, workspace.id))))
const filteredCommands = computed(() => {
  const search = commandSearch.value.trim().toLocaleLowerCase()
  if (!search) return availableCommands.value

  return searchableCommands.value.filter((command) => [
    command.name,
    command.id,
    command.category,
    command.description,
    ...(command.aliases ?? []),
    ...(command.searchTerms ?? []),
  ].some((value) => value.toLocaleLowerCase().includes(search)))
})
const commandCategories = computed<CommandCategory[]>(() => {
  const categories = new Map<string, CommandDefinition[]>()
  for (const command of filteredCommands.value) {
    const categoryName = navigationCategory(command, commandSearch.value.trim() ? primaryWorkspaceForCommand(command) : activeWorkspace.value)
    const category = categories.get(categoryName) ?? []
    category.push(command)
    categories.set(categoryName, category)
  }

  return [...categories]
    .map(([name, categoryCommands]) => ({ name, commands: categoryCommands }))
    .sort((left, right) => navigationCategoryOrder(left.name) - navigationCategoryOrder(right.name))
})
const navigationCommandTotal = computed(() => commandSearch.value.trim() ? searchableCommands.value.length : availableCommands.value.length)
const catalogTabCommandID = computed(() => {
  const visibleCommands = commandCategories.value.flatMap((category) => categoryExpanded(category.name) ? category.commands : [])
  return visibleCommands.some((command) => command.id === selectedCommandID.value) ? selectedCommandID.value : visibleCommands[0]?.id
})
const selectedCommand = computed(() => availableCommands.value.find((command) => command.id === selectedCommandID.value))
const selectedQueryFailure = computed(() => {
  const command = selectedCommand.value
  if (!command) return undefined
  const result = latestResults.value[command.id]
  return result && !result.success ? result : undefined
})
const latestOperationResult = computed(() => {
  const owner = selectedCommand.value
  if (!owner) return undefined
  return [...results.value].reverse().find((result) => {
    const command = commands.value.find((candidate) => candidate.id === result.commandId)
    return command !== undefined && (result.ownerCommandID ?? companionOwnerCommandID(result.commandId)) === owner.id
  })
})
const latestWindResult = computed(() => latestResults.value[windStatusCommandID])
const latestLoggingResult = computed(() => latestResults.value[loggingStatusCommandID])
const latestShrineStatusResult = computed(() => latestResults.value[shrineStatusCommandID])
const latestCombatMapResult = computed(() => latestResults.value["combat.map-catalog.all"])
const latestCalendarResult = computed(() => latestResults.value[calendarStatusCommandID])
const mastheadCalendar = computed(() => latestCalendarResult.value?.data ? calendarStatusData(latestCalendarResult.value.data) : undefined)
const latestSearchResult = computed(() => latestResults.value[searchStatusCommandID])
const latestQuestItemResult = computed(() => latestResults.value[questItemStatusCommandID])
const latestWorldPositionResult = computed(() => latestResults.value[worldPositionStatusCommandID])
const latestDungeonExplorerResult = computed(() => latestResults.value[dungeonExplorerStatusCommandID])
const latestMoongateResult = computed(() => latestResults.value[moongateStatusCommandID])
const latestConversationCatalogResult = computed(() => latestResults.value[conversationCatalogCommandID])
const latestConversationDetailResult = computed(() => latestResults.value[conversationInspectCommandID])
const latestCombatEncounterOptionsResult = computed(() => latestResults.value[combatEncounterOptionsCommandID])
const latestMerchantTesterCatalogResult = computed(() => latestResults.value[merchantTesterCatalogCommandID])
const latestVFXStatusResult = computed(() => latestResults.value[vfxStatusCommandID])
const latestMonsterEcosystemResult = computed(() => latestResults.value[monsterEcosystemStatusCommandID])
const latestVehicleStatusResult = computed(() => latestResults.value[vehicleStatusCommandID])
const latestInventoryLockerResult = computed(() => latestResults.value[inventoryLockerStatusCommandID])
const latestSpellInventoryResult = computed(() => latestResults.value[spellInventoryStatusCommandID])
const latestActiveCombatResult = computed(() => latestResults.value[activeCombatStatusCommandID])
const mastheadActiveCombat = computed(() => latestActiveCombatResult.value?.data
  ? activeCombatStatusData(latestActiveCombatResult.value.data)
  : undefined)
const quickCombatExitDisabled = computed(() => {
	const combat = mastheadActiveCombat.value
	return !isReady.value || commandPending(activeCombatExitCommandID) || !combat?.active || combat.workflowActive
		|| combat.map?.pendingAdvanceTurn || combat.map?.activeMonsterIndex !== -1
})
const latestDebugOptionsResult = computed(() => latestResults.value[debugOptionsStatusCommandID])
const latestDisplayResult = computed(() => latestResults.value[displayStatusCommandID])
const displayModeChoices = computed(() => commands.value
  .find((command) => command.id === displayModeSetCommandID)
  ?.parameters?.find((parameter) => parameter.id === "mode")?.choices ?? [])
const latestReduxModificationsResult = computed(() => latestResults.value[reduxModificationsStatusCommandID])
const freeMoveEnabled = computed(() => latestDebugOptionsResult.value?.data ? debugOptionsStatusData(latestDebugOptionsResult.value.data).freeMove : false)
const latestGuardStatusResult = computed(() => latestResults.value[guardStatusCommandID])
const latestShadowlordStatusResult = computed(() => latestResults.value[shadowlordStatusCommandID])
const latestDeathStatusResult = computed(() => latestResults.value[deathStatusCommandID])
const latestScenarioStatusResult = computed(() => latestResults.value[scenarioStatusCommandID])
const latestScenarioDefinitionListResult = computed(() => latestResults.value[scenarioDefinitionListCommandID])
const latestScenarioRunStatusResult = computed(() => latestResults.value[scenarioRunStatusCommandID] ?? latestResults.value[scenarioRunStartCommandID])
const latestSaveListResult = computed(() => latestResults.value[saveSnapshotListCommandID])
const latestSaveDetailResult = computed(() => latestResults.value[saveSnapshotGetCommandID])
const saveList = computed(() => latestSaveListResult.value?.data ? saveSnapshotListData(latestSaveListResult.value.data) : undefined)
const scenarioDefinitionList = computed(() => latestScenarioDefinitionListResult.value?.data
  ? scenarioDefinitionListData(latestScenarioDefinitionListResult.value.data).definitions
  : [])
const activeScenarioDirty = computed(() => scenarioTargetActive.value && scenarioDraftIsDirty(scenarioComposer.value))
const activeScenarioLabel = computed(() => scenarioComposer.value.savedReference ?? `${scenarioComposer.value.source}:${scenarioComposer.value.definition.id}`)
const activeScenarioStatus = computed(() => scenarioComposer.value.savedReference
  ? activeScenarioDirty.value ? "UNSAVED" : "SAVED"
  : activeScenarioDirty.value ? "UNSAVED" : "DRAFT")
const activeScenarioSemanticState = computed(() => scenarioRecording.value ? "recording" : activeScenarioStatus.value.toLocaleLowerCase())
const scenarioCaptureNewID = computed(() => scenarioIDFromName(scenarioCaptureNewName.value))
const scenarioCaptureNewIDExists = computed(() => scenarioDefinitionList.value.some((definition) =>
  definition.reference === `personal:${scenarioCaptureNewID.value}`,
))
const scenarioCaptureSelectedDefinition = computed(() => scenarioDefinitionList.value.find((definition) =>
  definition.reference === scenarioCaptureSelectedReference.value,
))
const scenarioCaptureInsertionOptions = computed(() => {
  const definition = scenarioCaptureSelectedDefinition.value
  if (!definition) return []
  const capturedCommand = commands.value.find((command) => command.id === scenarioCaptureResult.value?.invocation?.commandId)
  if (capturedCommand?.scenarioTerminal) {
    return [{ index: definition.steps.length, label: "After all setup steps · Final" }]
  }
  const terminalIndex = scenarioTerminalStepIndex(definition.steps, commands.value)
  const maximumIndex = terminalIndex < 0 ? definition.steps.length : terminalIndex
  return Array.from({ length: maximumIndex + 1 }, (_, index) => ({
    index,
    label: scenarioCaptureInsertionLabel(definition.steps, index, terminalIndex),
  }))
})
const scenarioCapturePlacementIssue = computed(() => {
  const definition = scenarioCaptureSelectedDefinition.value
  const capturedCommand = commands.value.find((command) => command.id === scenarioCaptureResult.value?.invocation?.commandId)
  if (!definition || !capturedCommand?.scenarioTerminal) return ""
  return scenarioTerminalStepIndex(definition.steps, commands.value) >= 0
    ? "This scenario already has a final step. Remove or replace it before capturing another terminal command."
    : ""
})
const scenarioDefinitionWritePending = computed(() => [
  scenarioDefinitionValidateCommandID, scenarioDefinitionCreateCommandID, scenarioDefinitionUpdateCommandID,
].some(commandPending))
const latestPartyCharacterInspectResult = computed(() => latestResults.value[partyCharacterInspectCommandID])
const latestPartyEquipmentLoadoutResult = computed(() => latestResults.value[partyEquipmentLoadoutCommandID])
const latestObservationResult = computed(() => [...results.value].reverse().find((result) =>
  observationCommandIDs.includes(result.commandId as ObservationCommandID) && result.success && result.data !== undefined,
))
const observationPending = computed(() => observationCommandIDs.some(commandPending))
const scenarioWordChoices = computed(() => commands.value.find((command) => command.id === scenarioWordSetupCommandID)?.parameters?.find((parameter) => parameter.id === "word")?.choices ?? [])
const worldLandmarkChoices = computed(() => commands.value.find((command) => command.id === movementLandmarkTravelCommandID)?.parameters?.find((parameter) => parameter.id === "landmark")?.choices ?? [])
const dungeonChoices = computed(() => commands.value.find((command) => command.id === dungeonExplorerSetupCommandID)?.parameters?.find((parameter) => parameter.id === "dungeon")?.choices ?? [])
const dungeonFacingChoices = computed(() => commands.value.find((command) => command.id === dungeonExplorerSetupCommandID)?.parameters?.find((parameter) => parameter.id === "facing")?.choices ?? [])
const catalogWorkspaces: { id: CatalogWorkspaceID; label: string; detail: string }[] = [
  { id: "game", label: "Game", detail: "State & setup" },
  { id: "codex", label: "Codex", detail: "Reference" },
  { id: "system", label: "System", detail: "App & display" },
  { id: "scenarios", label: "Scenarios", detail: "Library & runs" },
  { id: "saves", label: "Saves", detail: "Captures & rollback" },
  { id: "tools", label: "Tools", detail: "Tests & workflows" },
]
const workspaces: { id: WorkspaceID; label: string; detail: string }[] = [
  { id: "home", label: "Home", detail: "Start here" },
  ...catalogWorkspaces,
]
const navigationSurfaces = computed<NavigationSurface[]>(() => catalogWorkspaces.flatMap((workspace) => supportedCommands.value
  .filter((command) => commandBelongsToWorkspace(command, workspace.id))
  .map((command) => ({
    id: `${workspace.id}:${command.id}`,
    workspace: workspace.id,
    workspaceLabel: workspace.label,
    label: workspaceCommandName(command, workspace.id),
    description: workspaceCommandDescription(command, workspace.id),
    commandId: command.id,
    commands: navigationSurfaceCommands(command, workspace.id),
  }))))
const homeRecentSurfaces = computed(() => recentNavigationSurfaces(navigationSurfaces.value, navigationRecents.value))
watch(latestScenarioRunStatusResult, (result) => {
  const data = result?.data as ScenarioRunStatusResult | undefined
  const runs = data?.runs?.length ? data.runs : data?.run ? [data.run] : []
  if (!runs.some((run) => run.runId === selectedScenarioRunID.value)) selectedScenarioRunID.value = runs[0]?.runId ?? ""
})
const retryDetail = computed(() => {
  if (retryScheduled.value) return `automatic retry ${reconnectAttempt.value} scheduled`
  if (["connecting", "reconnecting", "handshaking"].includes(connectionState.value)) return "connecting now"
  return "automatic retry stopped"
})

function websocketURL(): string {
  const scheme = location.protocol === "https:" ? "wss:" : "ws:"
  return `${scheme}//${location.host}/ws?token=${encodeURIComponent(token)}`
}

function connect(): void {
  window.clearTimeout(reconnectTimer)
  window.clearTimeout(connectionTimer)
  retryScheduled.value = false
  const generation = ++connectionGeneration
  const previousSocket = socket.value
  socket.value = null
  previousSocket?.close(1000, "connection replaced")
  connectionState.value = reconnectAttempt.value > 0 ? "reconnecting" : "connecting"
  const nextSocket = new WebSocket(websocketURL())
  socket.value = nextSocket
  connectionTimer = window.setTimeout(() => nextSocket.close(1013, "connection timeout"), connectionTimeoutMs)

  nextSocket.addEventListener("open", () => {
    if (generation !== connectionGeneration || socket.value !== nextSocket) {
      nextSocket.close(1000, "stale connection")
      return
    }
    window.clearTimeout(connectionTimer)
    connectionState.value = "handshaking"
    connectionTimer = window.setTimeout(() => nextSocket.close(1013, "session handshake timeout"), connectionTimeoutMs)
    if (!send(envelope("hello", "session.hello", { sessionId: sessionID.value }), nextSocket)) {
      nextSocket.close(1011, "session handshake send failed")
    }
  })
  nextSocket.addEventListener("message", (event) => {
    if (generation !== connectionGeneration || socket.value !== nextSocket) return
    try {
      handleMessage(parseEnvelope(String(event.data)))
    } catch {
      receiveProtocolError({
        version: 1,
        id: "decode",
        type: "protocol.error",
        payload: { code: "invalid_json", message: "The server sent malformed JSON." },
      })
      nextSocket.close(1002, "invalid protocol message")
    }
  })
  nextSocket.addEventListener("close", (event) => {
    if (generation !== connectionGeneration || socket.value !== nextSocket) return
    socket.value = null
    window.clearTimeout(connectionTimer)
    connectionState.value = "disconnected"
    lastCloseReason.value = event.reason || `socket closed (${event.code})`
    if (event.code === 1002 || event.code === 1008) return
    scheduleReconnect()
  })
  nextSocket.addEventListener("error", () => nextSocket.close())
}

function scheduleReconnect(): void {
  reconnectAttempt.value += 1
  retryScheduled.value = true
  const delay = Math.min(500 * 2 ** (reconnectAttempt.value - 1), 5000)
  reconnectTimer = window.setTimeout(connect, delay)
}

function retryNow(): void {
  reconnectAttempt.value = 0
  retryScheduled.value = false
  lastCloseReason.value = ""
  connect()
}

function send(message: Envelope, target = socket.value): boolean {
  if (target?.readyState !== WebSocket.OPEN) return false
  try {
    target.send(JSON.stringify(message))
    return true
  } catch {
    return false
  }
}

function parseEnvelope(raw: string): Envelope {
  const parsed = JSON.parse(raw) as Partial<Envelope>
  if (parsed.version !== 1 || typeof parsed.type !== "string") throw new Error("invalid protocol envelope")
  return parsed as Envelope
}

function handleMessage(message: Envelope): void {
  switch (message.type) {
    case "session.ready": {
      const ready = message.payload as SessionReady
      if (!ready.resumed) {
        lastSuccessfulWorldPositionResult.value = undefined
        reportLostPendingRequests()
        pending.value.clear()
      }
      sessionID.value = ready.sessionId
      sceneGeneration.value = ready.sceneGeneration
      localStorage.setItem(sessionStorageKey, ready.sessionId)
      window.clearTimeout(connectionTimer)
      reconnectAttempt.value = 0
      lastCloseReason.value = ""
      connectionState.value = "ready"
      scenarioRunStatusChecked.value = false
      if (ready.resumed) {
        for (const [requestID, request] of pending.value) {
          pending.value.set(requestID, { ...request, lastConfirmedAt: Date.now(), statusRequested: false })
          send(envelope(requestID, "operation.status", { requestId: requestID }))
        }
      }
      requestCatalog()
      break
    }
    case "catalog.snapshot": {
      const catalog = message.payload as CatalogSnapshot
      commands.value = catalog.commands
      optionSets.value = Object.fromEntries((catalog.optionSets ?? []).map((optionSet) => [optionSet.provider, optionSet.choices]))
      optionUnavailableReasons.value = Object.fromEntries(
        (catalog.optionSets ?? []).filter((optionSet) => optionSet.unavailableReason).map((optionSet) => [optionSet.provider, optionSet.unavailableReason ?? ""]),
      )
      optionsRevision.value = catalog.optionsRevision
      awaitingOptionsRefresh.value = false
      const nextDrafts: Record<string, CommandDraft> = {}
      const nextDirtyFields: Record<string, string[]> = {}
      for (const command of catalog.commands) {
        const existing = drafts.value[command.id] ?? {}
        if (command.id === partyCharacterCommandID || command.id === partyEquipmentLoadoutCommandID) {
          const reconciled = reconcilePartyCharacterDraft(command, existing, dirtyFields.value[command.id] ?? [], optionSets.value)
          nextDrafts[command.id] = reconciled.draft
          if (command.id === partyCharacterCommandID) nextDirtyFields[command.id] = reconciled.dirtyFields
          if (existing.character && existing.character !== reconciled.draft.character) clearCharacterAlterationErrors(command.id)
        } else {
          nextDrafts[command.id] = reconcileDraft(command, existing, optionSets.value)
        }
      }
      drafts.value = nextDrafts
      dirtyFields.value = nextDirtyFields
      if (activeWorkspace.value === "home") {
        if (!catalog.commands.some((command) => isSupportedWorkbenchCommand(command) && command.id === selectedCommandID.value)) {
          selectedCommandID.value = catalog.commands.find(isSupportedWorkbenchCommand)?.id ?? ""
        }
      } else {
        let workspaceCommands = catalog.commands.filter((command) => isSupportedWorkbenchCommand(command) && commandBelongsToWorkspace(command, activeWorkspace.value))
        if (workspaceCommands.length === 0 && catalog.commands.some(isSupportedWorkbenchCommand)) {
		  const firstSupported = catalog.commands.find(isSupportedWorkbenchCommand)
		  activeWorkspace.value = firstSupported ? primaryWorkspaceForCommand(firstSupported) : "tools"
		  workspaceCommands = catalog.commands.filter((command) => isSupportedWorkbenchCommand(command) && commandBelongsToWorkspace(command, activeWorkspace.value))
        }
        if (!workspaceCommands.some((command) => command.id === selectedCommandID.value)) selectedCommandID.value = workspaceCommands[0]?.id ?? ""
      }
      sceneGeneration.value = catalog.sceneGeneration
		void nextTick(() => {
			if (![calendarStatusCommandID, activeCombatStatusCommandID].includes(selectedCommandID.value)) autoExecuteSelectedQuery()
      for (const globalID of [calendarStatusCommandID, activeCombatStatusCommandID]) {
        const latest = latestResults.value[globalID]
        if (!latest || latest.sceneGeneration !== sceneGeneration.value || latest.sessionId !== sessionID.value) runQueryByID(globalID, true)
      }
			runQueryByID(debugOptionsStatusCommandID, true)
			if (!scenarioRunStatusChecked.value) {
				scenarioRunStatusChecked.value = true
				runQueryByID(scenarioRunStatusCommandID, true)
			}
		})
      break
    }
    case "command.accepted": {
      const accepted = message.payload as { requestId: string; commandId: string }
      const existing = pending.value.get(accepted.requestId)
      if (processedRequestIDs.has(accepted.requestId)) {
        send(envelope(`ack-${accepted.requestId}`, "command.ack", { requestId: accepted.requestId }))
      } else if (existing?.phase !== "accepted") {
        pending.value.set(accepted.requestId, {
          commandID: accepted.commandId,
          ownerCommandID: existing?.ownerCommandID,
          phase: "accepted",
          startedAt: existing?.startedAt ?? Date.now(),
          lastConfirmedAt: Date.now(),
          statusRequested: false,
        })
      }
      break
    }
    case "command.completed":
    case "command.failed":
      receiveResult(message.payload as CommandResult)
      break
    case "protocol.error":
      receiveProtocolError(message)
      break
  }
}

function runCommand(command: CommandDefinition, silent = false, onSent?: (requestID: string) => void): boolean {
  if (!isReady.value || !isSupportedWorkbenchCommand(command) || commandPending(command.id) || hasUnsupportedParameters(command, optionSets.value) || commandUnavailableReason(command)) return false
  const errors = validateDraft(command, draftFor(command), optionSets.value)
	if (command.id === partyCharacterCommandID) Object.assign(errors, partyResourceErrors(command, optionSets.value, draftFor(command), dirtyFieldsFor(command)))
  formErrors.value[command.id] = errors
  if (Object.keys(errors).length > 0) {
    formMessages.value[command.id] = "Correct the highlighted fields before running this operation."
    void nextTick(() => firstInvalidControl(command.id)?.focus())
    return false
  }
  if (command.id === partyCharacterCommandID && dirtyFieldsFor(command).length === 0) {
    formMessages.value[command.id] = "Choose at least one character change before applying."
    return false
  }
  if (command.id === partyCharacterCommandID && partyAlterationPreview(command, optionSets.value, draftFor(command), dirtyFieldsFor(command)).length === 0) {
    formMessages.value[command.id] = "The requested values already match the selected character."
    return false
  }
  delete formMessages.value[command.id]
  const requestID = `query-${Date.now()}-${++requestSequence}`
  const sent = send(envelope(requestID, "command.execute", {
    commandId: command.id,
    expectedSceneGeneration: sceneGeneration.value,
    ...(commandUsesDynamicOptions(command) ? { expectedOptionsRevision: optionsRevision.value } : {}),
    arguments: command.id === partyCharacterCommandID
      ? serializePartyCharacterDraft(draftFor(command), dirtyFieldsFor(command))
      : serializeDraft(command, draftFor(command)),
  }))
  if (sent) {
    if (silent) silentRequestIDs.add(requestID)
    pending.value.set(requestID, {
      commandID: command.id,
      phase: "sending",
      startedAt: Date.now(),
      lastConfirmedAt: Date.now(),
      statusRequested: false,
    })
    onSent?.(requestID)
  } else {
    rememberProcessedRequest(requestID)
    appendResult({
      entryID: `transport-${++localEntrySequence}`,
      sessionId: sessionID.value,
      requestId: requestID,
      commandId: command.id,
      sceneGeneration: sceneGeneration.value,
      success: false,
      errorCode: "transport_unavailable",
      errorMessage: "The command could not be sent to the game process.",
      receivedAt: new Date(),
    })
    socket.value?.close(1011, "command send failed")
  }
  return sent
}

function draftFor(command: CommandDefinition): CommandDraft {
  return drafts.value[command.id] ?? {}
}

function dirtyFieldsFor(command: CommandDefinition): string[] {
  return dirtyFields.value[command.id] ?? []
}

function errorFor(commandID: string, parameterID: string): string | undefined {
  return formErrors.value[commandID]?.[parameterID]
}

function describedBy(command: CommandDefinition, parameterID: string): string | undefined {
  const parameter = command.parameters?.find((candidate) => candidate.id === parameterID)
  const identifiers: string[] = []
  if (parameter?.description) identifiers.push(`${command.id}-${parameterID}-help`)
  if (errorFor(command.id, parameterID)) identifiers.push(`${command.id}-${parameterID}-error`)

  return identifiers.length > 0 ? identifiers.join(" ") : undefined
}

function updateDraft(command: CommandDefinition, parameterID: string, value: string): void {
  const current = draftFor(command)
  if (command.id === partyCharacterCommandID && parameterID === "character" && current.character !== value) {
    drafts.value[command.id] = hydratePartyCharacterDraft(command, value, optionSets.value)
    dirtyFields.value[command.id] = []
    clearCharacterAlterationErrors(command.id)
  } else {
    drafts.value[command.id] = { ...current, [parameterID]: value }
    if (command.id === partyCharacterCommandID && parameterID !== "character") {
      const nextDirty = new Set(dirtyFieldsFor(command))
      if (partyFieldIsDirty(command, optionSets.value, current, parameterID, value)) {
        nextDirty.add(parameterID)
      } else {
        nextDirty.delete(parameterID)
      }
      dirtyFields.value[command.id] = [...nextDirty]
    }
  }
  if (formErrors.value[command.id]?.[parameterID]) {
    const errors = { ...formErrors.value[command.id] }
    delete errors[parameterID]
    formErrors.value[command.id] = errors
  }
  delete formMessages.value[command.id]
}

function updateSelectedPartyCharacterDraft(parameterID: string, value: string): void {
  if (!selectedCommand.value) return

  updateDraft(selectedCommand.value, parameterID, value)
}

function updateEquipmentLoadoutCharacter(value: string): void {
	const command = commands.value.find((candidate) => candidate.id === partyEquipmentLoadoutCommandID)
	if (command) updateDraft(command, "character", value)
}

function updateSelectedCombatSection(section: string): void {
  if (!selectedCommand.value || selectedCommand.value.id !== "combat.map-catalog.all") return

  updateDraftAndMaybeRun(selectedCommand.value, "section", section)
}

function resetCommand(command: CommandDefinition): void {
  drafts.value[command.id] = command.id === partyCharacterCommandID
    ? hydratePartyCharacterDraft(command, draftFor(command).character ?? "", optionSets.value)
    : resetDraft(command, optionSets.value)
  if (command.id === partyCharacterCommandID) dirtyFields.value[command.id] = []
  delete formErrors.value[command.id]
  delete formMessages.value[command.id]
}

function selectCommand(command: CommandDefinition): void {
  revokeRollbackReview()
  if (commandSearch.value.trim()) {
    activeWorkspace.value = primaryWorkspaceForCommand(command)
    commandSearch.value = ""
  }
  if (command.id === merchantTesterCatalogCommandID) merchantCatalogExplicitRefreshRequired = false
  selectedCommandID.value = command.id
  if (commandUsesDynamicOptions(command)) requestCatalog()
  void nextTick(autoExecuteSelectedQuery)
}

function selectWorkspace(workspace: WorkspaceID): void {
  revokeRollbackReview()
  if (commandPending(scenarioDefinitionGetCommandID)) scenarioReplacementVersion += 1
  if (activeWorkspace.value !== "home") workspaceSelections.set(activeWorkspace.value, selectedCommandID.value)
  activeWorkspace.value = workspace
  commandSearch.value = ""
  if (workspace === "home") return
  const workspaceCommands = supportedCommands.value.filter((command) => commandBelongsToWorkspace(command, workspace))
  const remembered = workspaceSelections.get(workspace) ?? selectedCommandID.value
  selectedCommandID.value = workspaceCommands.some((command) => command.id === remembered) ? remembered : workspaceCommands[0]?.id ?? ""
  if (selectedCommandID.value === merchantTesterCatalogCommandID) merchantCatalogExplicitRefreshRequired = false
  void nextTick(autoExecuteSelectedQuery)
}

function selectShrine(index: number): void {
  selectedShrineIndex.value = index
}

function openShrineWorkspace(workspace: "codex" | "game"): void {
  selectWorkspace(workspace)
  selectedCommandID.value = shrineStatusCommandID
  void nextTick(autoExecuteSelectedQuery)
}

function openCombatEncounterComposer(): void {
	activeWorkspace.value = "tools"
  commandSearch.value = ""
  selectedCommandID.value = combatEncounterOptionsCommandID
  void nextTick(autoExecuteSelectedQuery)
}

function activateCommand(command: CommandDefinition): void {
  selectCommand(command)
  void nextTick(() => firstInstrumentControl()?.focus())
}

function toggleCategory(category: string): void {
  const next = new Set(collapsedCategories.value)
  if (next.has(category)) {
    next.delete(category)
  } else {
    next.add(category)
  }
  collapsedCategories.value = next
}

function categoryExpanded(category: string): boolean {
  return commandSearch.value.trim() !== "" || !collapsedCategories.value.has(category)
}

function firstInvalidControl(commandID: string): HTMLInputElement | HTMLSelectElement | null {
  return document.querySelector(`[data-command-form="${commandID}"] [aria-invalid="true"]`)
}

function firstInstrumentControl(): HTMLInputElement | HTMLSelectElement | HTMLButtonElement | null {
  return document.querySelector(".instrument form input, .instrument form select, .instrument form button:not(:disabled)")
}

function receiveResult(result: CommandResult): void {
  const firstReceipt = !processedRequestIDs.has(result.requestId)
  send(envelope(`ack-${result.requestId}`, "command.ack", { requestId: result.requestId }))
  if (!firstReceipt) return

  rememberProcessedRequest(result.requestId)
  const ownerCommandID = pending.value.get(result.requestId)?.ownerCommandID
  pending.value.delete(result.requestId)
  sceneGeneration.value = result.sceneGeneration
  const silent = silentRequestIDs.delete(result.requestId)
  const entry = { ...result, ownerCommandID, entryID: `result-${result.requestId}`, receivedAt: new Date() }
  if (result.commandId === worldPositionStatusCommandID && result.success && result.data && result.sessionId === sessionID.value) {
    lastSuccessfulWorldPositionResult.value = entry
  }
  if (silent) {
    latestResults.value = { ...latestResults.value, [result.commandId]: entry }
  } else {
    appendResult(entry)
    if (scenarioRecording.value && scenarioTargetActive.value) appendResultToActiveScenario(entry)
  }
  if (result.success && result.commandId === partyCharacterCommandID) dirtyFields.value[result.commandId] = []
  if (result.errorCode === "stale_options") {
    awaitingOptionsRefresh.value = true
    formMessages.value[result.commandId] = "Available party options changed. Review the refreshed selection before retrying."
  }
  if (result.success && merchantTesterOperationCommandIDs.includes(result.commandId)) merchantCatalogExplicitRefreshRequired = true
  if (!result.success && result.errorCode === "stale_world_position") {
    const owner = result.commandId === dungeonExplorerSetupCommandID
      ? dungeonExplorerStatusCommandID
      : result.commandId === worldPositionSetCommandID ? worldPositionStatusCommandID : ""
    if (owner) void nextTick(() => runQueryByID(owner, true))
  }
  if (!result.success && result.commandId === dungeonExplorerSetupCommandID
    && result.errorCode === "stale_dungeon_state") {
    void nextTick(() => runQueryByID(dungeonExplorerStatusCommandID, true))
  }
  if (result.errorCode === "stale_options" || result.effectiveExecutionMode && result.effectiveExecutionMode !== "query") requestCatalog()
  if (result.success && result.commandId === windSetCommandID) void nextTick(() => runQueryByID(windStatusCommandID, true))
  if (result.success && loggingMutationCommandIDs.includes(result.commandId)) void nextTick(() => runQueryByID(loggingStatusCommandID, true))
  if (result.success && (result.commandId === shrineProgressCommandID || result.commandId === codexQuestStateSetAllCommandID)) void nextTick(() => runQueryByID(shrineStatusCommandID, true))
  if (result.success && (result.commandId === calendarSetTimeCommandID || result.commandId === calendarSetDayCommandID)) void nextTick(() => runQueryByID(calendarStatusCommandID, true))
  if (result.success && (result.commandId === searchItemStackAddCommandID || result.commandId === searchProgressResetCommandID || result.commandId === commandTestSearchCommandID)) void nextTick(() => runQueryByID(searchStatusCommandID, true))
  if (result.success && result.commandId === searchOverlaySetCommandID) void nextTick(() => runQueryByID(searchStatusCommandID, true))
  if (result.success && (result.commandId === questItemEnsureCommandID || result.commandId === questItemRemoveCommandID || result.commandId === questShardUseCommandID)) void nextTick(() => runQueryByID(questItemStatusCommandID, true))
  if (result.success && (result.commandId === worldPositionSetCommandID || result.commandId === movementLandmarkTravelCommandID)) void nextTick(() => runQueryByID(worldPositionStatusCommandID, true))
  if (result.success && result.commandId === dungeonExplorerSetupCommandID) void nextTick(() => runQueryByID(dungeonExplorerStatusCommandID, true))
  if (result.success && scenarioMutationCommandIDs.includes(result.commandId)) void nextTick(() => runQueryByID(scenarioStatusCommandID, true))
  if (result.success && result.commandId === scenarioDefinitionGetCommandID && result.data) {
    const loaded = result.data as ScenarioDefinitionGetResult
    const request = scenarioDefinitionGetRequests.get(result.requestId)
    scenarioDefinitionGetRequests.delete(result.requestId)
    if (!request || request.version !== scenarioReplacementVersion || request.reference !== loaded.reference
      || request.draftFingerprint !== scenarioDraftFingerprint()) {
      if (request?.intent === "capture" && scenarioCaptureLoadingReference.value === request.reference) {
        failScenarioCaptureLoad("The active scenario draft or navigation intent changed while this scenario loaded. Review the target and retry.")
      }
      return
    }
    const captureRevisionChanged = scenarioCaptureLoadingReference.value === loaded.reference
      && Boolean(scenarioCaptureExpectedRevision.value)
      && scenarioCaptureExpectedRevision.value !== loaded.revision
    if (captureRevisionChanged) {
      scenarioCaptureLoadingReference.value = ""
      scenarioCaptureSelectedReference.value = ""
      scenarioCaptureExpectedRevision.value = ""
      scenarioCaptureError.value = "The selected scenario changed after placement review. Review the refreshed definition and choose its insertion position again."
      runQueryByID(scenarioDefinitionListCommandID, true)
    } else {
      scenarioComposer.value = hydrateScenarioComposer(loaded)
      scenarioHistoricalRun.value = request.historicalRun
      if (request.historicalRun?.launch) scenarioComposer.value.launchValues = JSON.parse(JSON.stringify(request.historicalRun.launch.parameterValues)) as Record<string, unknown>
      scenarioValidationIssues.value = []
      if (scenarioCaptureLoadingReference.value === loaded.reference) {
        scenarioCaptureLoadingReference.value = ""
        scenarioTargetActive.value = true
        scenarioRecording.value = false
        finishScenarioTargetSelection()
      } else if (request.intent === "direct-run") {
        scenarioTargetActive.value = false
        scenarioRecording.value = false
        validateScenarioRun("direct-run")
      } else {
        scenarioTargetActive.value = false
        scenarioRecording.value = false
        scenarioTask.value = "details"
      }
    }
  }
  if (!result.success && result.commandId === scenarioDefinitionGetCommandID && scenarioCaptureLoadingReference.value) {
    failScenarioCaptureLoad(result.errorMessage ?? "The scenario could not be loaded.")
  }
  if (result.commandId === scenarioDefinitionGetCommandID) scenarioDefinitionGetRequests.delete(result.requestId)
  if (!result.success && result.commandId === scenarioDefinitionListCommandID && scenarioCaptureDialogOpen.value) {
    scenarioCaptureError.value = result.errorMessage ?? "The scenario library could not be loaded."
  }
  if (result.success && result.commandId === scenarioDefinitionValidateCommandID && result.data) {
    const validation = result.data as ScenarioDefinitionValidationResult
    const intent = scenarioValidationIntent.value
    scenarioValidationIntent.value = undefined
    if (scenarioValidationSnapshot.value !== serializeScenarioDefinition(scenarioComposer.value)
      || ((intent === "review-run" || intent === "direct-run") && (scenarioValidationContext.value !== `${sceneGeneration.value}:${scenarioReviewIdentity()}`
        || scenarioDraftIsDirty(scenarioComposer.value) || scenarioDraftHasPendingMigrations(scenarioComposer.value)))) {
      scenarioValidationIssues.value = [{
        path: "$", code: "definition.changed_during_validation",
        message: "The scenario changed during validation. Review the updated draft and retry.",
      }]
      scenarioValidationSnapshot.value = ""
    } else if (validation.valid && intent === "save") {
      scenarioValidationIssues.value = validation.issues
      dispatchScenarioSave(scenarioValidationSnapshot.value)
    } else if (validation.valid && (intent === "review-run" || intent === "direct-run")) {
      scenarioValidationIssues.value = validation.issues
      scenarioReviewedSceneGeneration.value = sceneGeneration.value
      scenarioReviewedBaselineRevision.value = reviewedScenarioBaselineRevision()
      scenarioTask.value = "run-review"
      scenarioReviewedDefinition.value = scenarioReviewIdentity()
      scenarioValidationSnapshot.value = ""
    } else if (!validation.valid) {
      scenarioValidationIssues.value = validation.issues
      scenarioValidationSnapshot.value = ""
      if (intent === "review-run" || intent === "direct-run") {
        scenarioTargetActive.value = true
        scenarioTask.value = "editor"
      }
    }
  }
  if (result.success && [scenarioDefinitionCreateCommandID, scenarioDefinitionUpdateCommandID].includes(result.commandId) && result.data) {
    const saved = result.data as ScenarioDefinitionSaveResult
    const savedDefinition = scenarioValidationSnapshot.value
      ? JSON.parse(scenarioValidationSnapshot.value) as ScenarioDefinition
      : cloneScenarioDefinition(scenarioComposer.value.definition)
    scenarioComposer.value = {
      ...scenarioComposer.value,
      savedReference: saved.reference,
      loadedRevision: saved.definition.revision,
      originalDefinition: cloneScenarioDefinition(savedDefinition),
    }
    scenarioValidationSnapshot.value = ""
    scenarioValidationIssues.value = []
    scenarioTask.value = "details"
    void nextTick(() => runQueryByID(scenarioDefinitionListCommandID, true))
  }
  if (result.success && result.commandId === scenarioDefinitionDeleteCommandID) {
    scenarioComposer.value = newScenarioComposerDraft()
    scenarioTargetActive.value = false
    scenarioRecording.value = false
    scenarioTask.value = "library"
    void nextTick(() => runQueryByID(scenarioDefinitionListCommandID, true))
  }
  if (!result.success && [scenarioDefinitionValidateCommandID, scenarioDefinitionCreateCommandID, scenarioDefinitionUpdateCommandID].includes(result.commandId)) {
    scenarioValidationIntent.value = undefined
    scenarioValidationSnapshot.value = ""
  }
  if (result.success && result.commandId === scenarioDefinitionDuplicateCommandID && result.data) {
    const duplicated = result.data as ScenarioDefinitionSaveResult
    void nextTick(() => {
      runQueryByID(scenarioDefinitionListCommandID, true)
      getScenarioDefinition(duplicated.reference)
    })
  }
  if (result.success && [scenarioRunStartCommandID, scenarioRunCancelCommandID].includes(result.commandId)) {
	latestResults.value[scenarioRunStatusCommandID] = entry
	const startedRun = (result.data as ScenarioRunStatusResult | undefined)?.run
	if (startedRun) selectedScenarioRunID.value = startedRun.runId
    scenarioTask.value = "latest-run"
    void nextTick(() => runQueryByID(scenarioRunStatusCommandID, true))
  }
  if (result.success && result.commandId === saveSnapshotListCommandID) invalidateStaleScenarioBaselineReview()
  if (result.commandId === saveSnapshotGetCommandID) {
    const rollback = rollbackRequests.get(result.requestId)
    rollbackRequests.delete(result.requestId)
    if (rollback && rollback.epoch === rollbackReviewEpoch && activeWorkspace.value === "saves"
      && selectedCommandID.value === saveSnapshotListCommandID && isReady.value && !pageHidden.value) {
      const snapshot = result.success && result.data ? saveSnapshotDetailData(result.data) : undefined
      if (snapshot?.summary.reference === rollback.reference && snapshot.summary.revision === rollback.revision) {
        saveLoadReviewTarget.value = { ...rollback, requestID: result.requestId }
      } else ownerErrors.value[saveSnapshotListCommandID] = "The recorded rollback is missing, changed, or unavailable. No load was authorized."
    }
  }
  if (result.success && [saveSnapshotCaptureCommandID, saveSnapshotDeleteCommandID].includes(result.commandId)) {
    void nextTick(() => runQueryByID(saveSnapshotListCommandID, true))
  }
  if (result.success && result.commandId === saveSnapshotLoadCommandID) {
    requestCatalog()
    void nextTick(() => {
      runQueryByID(calendarStatusCommandID, true)
      runQueryByID(saveSnapshotListCommandID, true)
      runQueryByID(scenarioStatusCommandID, true)
      runQueryByID(scenarioRunStatusCommandID, true)
    })
  }
  if (result.success && (result.commandId === deathKillPartyCommandID || result.commandId === deathTestSequenceCommandID)) void nextTick(() => runQueryByID(deathStatusCommandID, true))
  if (result.success && (result.commandId === moongateOverlaySetCommandID || result.commandId === moongateTravelCommandID)) void nextTick(() => runQueryByID(moongateStatusCommandID, true))
  if (result.success && [vfxKapowCommandID, vfxProjectileCommandID, environmentEarthquakeCommandID].includes(result.commandId)) void nextTick(() => runQueryByID(vfxStatusCommandID, true))
  if (result.success && result.commandId === lightingOverlaySetCommandID) void nextTick(() => runQueryByID(vfxStatusCommandID, true))
  if (result.success && [monsterGenerationSetCommandID, monsterClearMapEnemiesCommandID, monsterSpawnWhirlpoolCommandID, monsterSpawnPirateCommandID].includes(result.commandId)) {
    void nextTick(() => runQueryByID(monsterEcosystemStatusCommandID, true))
  }
	if (result.success && shipMutationCommandIDs.includes(result.commandId)) void nextTick(() => runQueryByID(vehicleStatusCommandID, true))
	if (result.success && debugOptionsMutationCommandIDs.includes(result.commandId)) void nextTick(() => runQueryByID(debugOptionsStatusCommandID, true))
	if (result.success && reduxModificationCommandIDs.includes(result.commandId)) void nextTick(() => runQueryByID(reduxModificationsStatusCommandID, true))
	if (result.success && displayMutationCommandIDs.includes(result.commandId)) void nextTick(() => runQueryByID(displayStatusCommandID, true))
  if (result.success && inventoryLockerMutationCommandIDs.includes(result.commandId)) void nextTick(() => runQueryByID(inventoryLockerStatusCommandID, true))
  if (result.success && spellInventoryMutationCommandIDs.includes(result.commandId)) void nextTick(() => runQueryByID(spellInventoryStatusCommandID, true))
	if (result.success && activeCombatMutationCommandIDs.includes(result.commandId)) void nextTick(() => runQueryByID(activeCombatStatusCommandID, true))
	if (result.success && (partyRosterMutationCommandIDs.includes(result.commandId) || result.commandId === partyKarmaSetCommandID)) {
		void nextTick(() => runQueryByID(partyCharacterInspectCommandID, true))
	}
	if (result.success && result.commandId === partyEquipmentLoadoutCommandID) {
		void nextTick(() => {
			runQueryByID(partyCharacterInspectCommandID, true)
			runQueryByID(inventoryLockerStatusCommandID, true)
		})
	}
  if ([scenarioDefinitionValidateCommandID, scenarioDefinitionCreateCommandID, scenarioDefinitionUpdateCommandID].includes(result.commandId)) {
    flushDeferredScenarioCaptures()
  }
}

function requestCatalog(): void {
  if (!send(envelope(`catalog-${Date.now()}`, "catalog.request", {}))) socket.value?.close(1011, "catalog request send failed")
}

function isSupportedWorkbenchCommand(command: CommandDefinition): boolean {
  if (integratedCommandIDs.has(command.id)) return true
  return !command.destructive && command.executionModes.some((mode) => mode === "query" || mode === "live_safe")
}

function commandBelongsToWorkspace(command: CommandDefinition, workspace: WorkspaceID): boolean {
  switch (workspace) {
    case "home":
      return false
    case "game":
      return command.id === windStatusCommandID || command.id === calendarStatusCommandID
        || command.id === worldPositionStatusCommandID || command.id === dungeonExplorerStatusCommandID || command.id === moongateStatusCommandID
        || command.id === monsterEcosystemStatusCommandID || command.id === activeCombatStatusCommandID
        || command.id === shrineStatusCommandID || command.id === partyCharacterCommandID
        || command.id === partyEquipmentLoadoutCommandID || command.id === partyCharacterInspectCommandID
        || command.id === searchStatusCommandID || command.id === questItemStatusCommandID
        || command.id === vehicleStatusCommandID || command.id === inventoryLockerStatusCommandID
        || command.id === spellInventoryStatusCommandID || command.id === guardStatusCommandID
    case "codex":
      return command.id === "combat.map-catalog.all" || command.id === shrineStatusCommandID || command.id === conversationCatalogCommandID || codexCommandIDs.includes(command.id)
    case "system":
      return command.id === loggingStatusCommandID || command.id === vfxStatusCommandID
		 || command.id === debugOptionsStatusCommandID || command.id === displayStatusCommandID
		 || command.id === reduxModificationsStatusCommandID
    case "scenarios":
      return command.id === scenarioStatusCommandID
    case "saves":
      return command.id === saveSnapshotListCommandID
    case "tools":
      return command.id === combatEncounterOptionsCommandID
        || command.id === merchantTesterCatalogCommandID
        || command.id === observationCommandID || !integratedCommandIDs.has(command.id)
  }
}

function primaryWorkspaceForCommand(command: CommandDefinition): WorkspaceID {
  if (saveSnapshotCommandIDs.includes(command.id)) return "saves"
  if (command.id === scenarioStatusCommandID) return "scenarios"
  if (codexCommandIDs.includes(command.id) || command.id === "combat.map-catalog.all" || command.id === conversationCatalogCommandID) return "codex"
  if (commandBelongsToWorkspace(command, "system")) return "system"
  if (commandBelongsToWorkspace(command, "game")) return "game"
  return "tools"
}

function workspaceLabel(workspace: WorkspaceID): string {
  return workspaces.find((item) => item.id === workspace)?.label.toLocaleUpperCase() ?? workspace.toLocaleUpperCase()
}

function navigationCategory(command: CommandDefinition, workspace: WorkspaceID): string {
  let category = command.category
  if (workspace === "game") {
    if ([windStatusCommandID, calendarStatusCommandID, worldPositionStatusCommandID, dungeonExplorerStatusCommandID, moongateStatusCommandID, searchStatusCommandID].includes(command.id)) category = "World"
    else if ([partyCharacterCommandID, partyEquipmentLoadoutCommandID, partyCharacterInspectCommandID, inventoryLockerStatusCommandID, spellInventoryStatusCommandID, questItemStatusCommandID].includes(command.id)) category = "Party & Inventory"
    else if ([activeCombatStatusCommandID, monsterEcosystemStatusCommandID].includes(command.id)) category = "Combat & Creatures"
    else category = "Vehicles & Progression"
  } else if (workspace === "system") {
    if (command.id === loggingStatusCommandID) category = "Application"
    else if (command.id === reduxModificationsStatusCommandID) category = "Redux Modifications"
    else if ([displayStatusCommandID, debugOptionsStatusCommandID].includes(command.id)) category = "Game Window"
    else category = "Presentation"
  } else if (workspace === "scenarios") {
    category = "Scenarios"
  } else if (workspace === "tools") {
    if (command.id === observationCommandID) category = "Workflows"
    else if (command.id === combatEncounterOptionsCommandID) category = "Encounter Setup"
    else if (command.id === merchantTesterCatalogCommandID) category = "Merchant Testing"
    else category = "Advanced Commands"
  } else if (workspace === "codex") {
    if ([conversationCatalogCommandID, "codex.npc.catalog"].includes(command.id)) category = "People & Dialogue"
    else if (["combat.map-catalog.all", "codex.tile.catalog", "codex.location.catalog", "codex.search-location.catalog"].includes(command.id)) category = "World & Maps"
    else if (["codex.save-format.catalog"].includes(command.id)) category = "Data Formats"
    else if (command.id === shrineStatusCommandID) category = "Virtues"
    else category = "Rules & Indexes"
  }
  return commandSearch.value.trim() ? `${workspaceLabel(workspace)} / ${category}` : category
}

function navigationCategoryOrder(name: string): number {
  const category = name.includes(" / ") ? name.slice(name.indexOf(" / ") + 3) : name
  const order = ["World", "Party & Inventory", "Combat & Creatures", "Vehicles & Progression", "Rules & Indexes", "World & Maps", "People & Dialogue", "Virtues", "Data Formats", "Application", "Game Window", "Redux Modifications", "Presentation", "Scenarios", "Workflows", "Encounter Setup", "Merchant Testing", "Advanced Commands"]
  const index = order.indexOf(category)
  return index < 0 ? order.length : index
}

function workspaceCommandName(command: CommandDefinition, workspace = activeWorkspace.value): string {
	if (command.id === scenarioStatusCommandID) return "Scenarios"
	if (command.id === saveSnapshotListCommandID) return "Saves"
	if (command.id === observationCommandID) return "Observation"
	if (command.id === guardStatusCommandID) return "World Diagnostics"
	if (command.id === partyCharacterCommandID) return "Character Editor"
	if (command.id === partyEquipmentLoadoutCommandID) return "Equipment Loadouts"
	if (command.id === partyCharacterInspectCommandID) return "Party Roster"
	if (command.id !== shrineStatusCommandID) return command.name
	return workspace === "game" ? "Shrine Progress" : "Shrine Reference"
}

function workspaceCommandDescription(command: CommandDefinition, workspace = activeWorkspace.value): string {
	if (command.id === scenarioStatusCommandID) return "Manage shared and personal scenario definitions, use immutable quick setups, and review retained runs."
	if (command.id === saveSnapshotListCommandID) return "Capture, inspect, load, and revision-safely remove immutable gameplay checkpoints."
	if (command.id === observationCommandID) return "Queue the production Telescope, Spyglass, or Crystal Ball workflow and inspect its expected branch and context."
	if (command.id === guardStatusCommandID) return "Inspect hidden guard hostility, Shadowlord placement, and death/resurrection consequences."
	if (command.id === partyCharacterCommandID) return "Edit one active character's attributes, resources, individual condition, and party position with a pending-change preview."
	if (command.id === partyEquipmentLoadoutCommandID) return "Stage and review one partial six-slot plan for one or all active party members as an atomic operation."
	if (command.id === partyCharacterInspectCommandID) return "Review active Positions 1-6 and inactive companions, then run fingerprint-checked roster and party-wide operations."
  if (command.id !== shrineStatusCommandID) return command.description
  if (workspace === "game") return "Inspect and alter authoritative shrine quest and ruined-state progression."
  return "Browse immutable virtue, mantra, location, and reward reference data."
}

function companionOwnerCommandID(commandID: string): string {
  if (loggingMutationCommandIDs.includes(commandID)) return loggingStatusCommandID
  if (commandID === windSetCommandID) return windStatusCommandID
  if ([shrineProgressCommandID, codexQuestStateSetAllCommandID].includes(commandID)) return shrineStatusCommandID
  if (commandID === "shrine.reference.inspect") return shrineStatusCommandID
  if (partyRosterMutationCommandIDs.includes(commandID) || commandID === partyKarmaSetCommandID) return partyCharacterInspectCommandID
  if (commandID === partyCharacterInspectCommandID) return partyCharacterInspectCommandID
  if (commandID === partyEquipmentLoadoutCommandID) return partyEquipmentLoadoutCommandID
  if (commandID === partyCharacterCommandID) return partyCharacterCommandID
  if ([calendarSetTimeCommandID, calendarSetDayCommandID].includes(commandID)) return calendarStatusCommandID
  if ([searchItemStackAddCommandID, searchProgressResetCommandID, searchOverlaySetCommandID, commandTestSearchCommandID].includes(commandID)) return searchStatusCommandID
  if ([questItemEnsureCommandID, questItemRemoveCommandID, questShardUseCommandID, usableInventoryPresetCommandID].includes(commandID)) return questItemStatusCommandID
  if (commandID === worldPositionSetCommandID || commandID === movementLandmarkTravelCommandID) return worldPositionStatusCommandID
  if (commandID === dungeonExplorerSetupCommandID) return dungeonExplorerStatusCommandID
  if (scenarioCompanionCommandIDs.includes(commandID) || scenarioMutationCommandIDs.includes(commandID)) return scenarioStatusCommandID
  if (saveSnapshotCommandIDs.includes(commandID)) return saveSnapshotListCommandID
  if (observationCommandIDs.includes(commandID as ObservationCommandID)) return observationCommandID
  if ([moongateOverlaySetCommandID, moongateTravelCommandID].includes(commandID)) return moongateStatusCommandID
  if (commandID === conversationInspectCommandID) return conversationCatalogCommandID
  if (commandID === combatEncounterStartCommandID) return combatEncounterOptionsCommandID
  if (merchantTesterOperationCommandIDs.includes(commandID)) return merchantTesterCatalogCommandID
  if ([vfxKapowCommandID, vfxProjectileCommandID, environmentEarthquakeCommandID, lightingOverlaySetCommandID].includes(commandID)) return vfxStatusCommandID
  if ([monsterGenerationSetCommandID, monsterClearMapEnemiesCommandID, monsterSpawnWhirlpoolCommandID, monsterSpawnPirateCommandID].includes(commandID)) return monsterEcosystemStatusCommandID
  if (shipMutationCommandIDs.includes(commandID)) return vehicleStatusCommandID
  if (debugOptionsMutationCommandIDs.includes(commandID)) return debugOptionsStatusCommandID
  if (displayMutationCommandIDs.includes(commandID)) return displayStatusCommandID
  if (reduxModificationCommandIDs.includes(commandID)) return reduxModificationsStatusCommandID
  if (worldDiagnosticCommandIDs.includes(commandID)) return guardStatusCommandID
  if (inventoryLockerMutationCommandIDs.includes(commandID)) return inventoryLockerStatusCommandID
  if (spellInventoryMutationCommandIDs.includes(commandID)) return spellInventoryStatusCommandID
  if (activeCombatMutationCommandIDs.includes(commandID)) return activeCombatStatusCommandID
  return commandID
}

function navigationSurfaceCommands(owner: CommandDefinition, workspace: WorkspaceID): CommandDefinition[] {
  return commands.value.filter((candidate) => {
    if (companionOwnerCommandID(candidate.id) !== owner.id) return false
    if (owner.id !== shrineStatusCommandID) return true
    return workspace === "codex"
      ? candidate.id === shrineStatusCommandID || candidate.id === "shrine.reference.inspect"
      : candidate.id !== "shrine.reference.inspect"
  })
}

function openCommandPalette(): void {
  if (scenarioCaptureDialogOpen.value || scenarioReplacementReview.value || commandPaletteOpen.value) return
  commandPalettePreviousFocus = document.activeElement as HTMLElement | null
  commandPaletteOpen.value = true
  void nextTick(() => setWorkbenchBackgroundInert(true, ".command-palette-modal"))
}

function closeCommandPalette(): void {
  commandPaletteOpen.value = false
  setWorkbenchBackgroundInert(false, ".command-palette-modal")
  void nextTick(() => (commandPalettePreviousFocus ?? commandPaletteLauncher.value)?.focus())
  commandPalettePreviousFocus = null
}

function navigateToSurface(surface: NavigationSurface): void {
  if (commandPending(scenarioDefinitionGetCommandID)) scenarioReplacementVersion += 1
  navigationRecents.value = updateNavigationRecents(navigationRecents.value, surface.id)
  localStorage.setItem(navigationRecentsStorageKey, JSON.stringify(navigationRecents.value))
  activeWorkspace.value = surface.workspace
  commandSearch.value = ""
  if (surface.commandId === merchantTesterCatalogCommandID) merchantCatalogExplicitRefreshRequired = false
  selectedCommandID.value = surface.commandId
  void nextTick(autoExecuteSelectedQuery)
}

function activateNavigationSurface(surface: NavigationSurface): void {
  navigateToSurface(surface)
  closeCommandPalette()
}

function openScenarioFromHome(task: "library" | "latest-run", runID?: string): void {
  scenarioReplacementVersion += 1
  if (runID) selectedScenarioRunID.value = runID
  selectWorkspace("scenarios")
  selectedCommandID.value = scenarioStatusCommandID
  scenarioTask.value = task
  void nextTick(autoExecuteSelectedQuery)
}

function instrumentModeLabel(command: CommandDefinition): string {
  if (command.id === observationCommandID) return "GAME-WINDOW OBSERVATION WORKFLOWS"
  if (command.id === shrineStatusCommandID && activeWorkspace.value === "game") return "LIVE STATE + EXPLICIT ALTERATION"
  if (command.id === shrineStatusCommandID || command.id === "combat.map-catalog.all" || command.id === conversationCatalogCommandID || codexCommandIDs.includes(command.id)) return "IMMUTABLE REFERENCE"
  if (command.id === combatEncounterOptionsCommandID) return "SYNCHRONIZED COMPOSER"
	if (command.id === merchantTesterCatalogCommandID) return "QUERY + PREVIEW + DESTRUCTIVE LIVE"
	if (command.id === worldPositionStatusCommandID) return "POSITION SNAPSHOT + EXPLICIT CONTROL"
  if (command.id === windStatusCommandID || command.id === calendarStatusCommandID || command.id === searchStatusCommandID
    || command.id === questItemStatusCommandID || command.id === dungeonExplorerStatusCommandID) return "LIVE TELEMETRY + EXPLICIT CONTROL"
  if (command.id === moongateStatusCommandID) return "LIVE TELEMETRY + ROUTING REFERENCE"
  if (command.id === vfxStatusCommandID) return "LIVE TELEMETRY + PRESENTATION CONTROL"
	if (command.id === monsterEcosystemStatusCommandID || command.id === vehicleStatusCommandID || command.id === debugOptionsStatusCommandID || command.id === displayStatusCommandID) return "LIVE TELEMETRY + EXPLICIT CONTROL"
  if (command.id === activeCombatStatusCommandID) return "LIVE TACTICAL TELEMETRY + REVIEWED CONTROL"
	if (command.id === partyCharacterCommandID) return "ACTIVE CHARACTER + PREVIEWED ALTERATION"
	if (command.id === partyEquipmentLoadoutCommandID) return "ATOMIC SIX-SLOT LOADOUT REVIEW"
	if (command.id === partyCharacterInspectCommandID) return "LIVE ROSTER + REVIEWED PARTY CONTROL"
	if (command.id === guardStatusCommandID) return "HIDDEN AUTHORITATIVE STATE"
  if (command.id === inventoryLockerStatusCommandID) return "LIVE INVENTORY + REVIEWED ALTERATIONS"
  if (command.id === spellInventoryStatusCommandID) return "LIVE INVENTORY + REVIEWED ALTERATIONS"
  return isMutation(command) ? "LIVE-SAFE MUTATION" : "READ-ONLY QUERY"
}

function instrumentExecutionModes(command: CommandDefinition): string {
	if (command.id !== merchantTesterCatalogCommandID) return command.executionModes.join(" · ")
	return navigationSurfaceCommands(command, activeWorkspace.value)
		.map((candidate) => `${candidate.id === merchantTesterCatalogCommandID ? "catalog" : candidate.id === merchantTesterPreviewCommandID ? "preview" : "live"}: ${candidate.executionModes.join(" / ")}`)
		.join(" · ")
}

function instrumentOwnsMutation(command: CommandDefinition): boolean {
  return isMutation(command) || command.id === windStatusCommandID || command.id === calendarStatusCommandID
    || command.id === searchStatusCommandID || command.id === questItemStatusCommandID || command.id === worldPositionStatusCommandID || command.id === dungeonExplorerStatusCommandID
     || command.id === combatEncounterOptionsCommandID || command.id === vfxStatusCommandID || command.id === monsterEcosystemStatusCommandID
     || command.id === vehicleStatusCommandID || command.id === debugOptionsStatusCommandID || command.id === displayStatusCommandID
     || command.id === inventoryLockerStatusCommandID
     || command.id === spellInventoryStatusCommandID
     || command.id === activeCombatStatusCommandID
	 || command.id === partyCharacterInspectCommandID
	 || command.id === scenarioStatusCommandID
     || command.id === merchantTesterCatalogCommandID
    || command.id === shrineStatusCommandID && activeWorkspace.value === "game"
}

function instrumentWidthClass(command: CommandDefinition): string {
  if (command.id === windStatusCommandID || command.id === calendarStatusCommandID) return "instrument--narrow"
  if (command.id === partyCharacterCommandID || command.id === partyCharacterInspectCommandID || command.id === shrineStatusCommandID && activeWorkspace.value === "game"
    || command.id === searchStatusCommandID || command.id === questItemStatusCommandID || command.id === worldPositionStatusCommandID) return "instrument--editor"
  if (command.id === dungeonExplorerStatusCommandID) return "instrument--reference"
  if (command.id === "combat.map-catalog.all" || command.id === shrineStatusCommandID || command.id === conversationCatalogCommandID || codexCommandIDs.includes(command.id)) return "instrument--reference"
  if (command.id === combatEncounterOptionsCommandID || command.id === vfxStatusCommandID
	 || command.id === monsterEcosystemStatusCommandID || command.id === vehicleStatusCommandID || command.id === debugOptionsStatusCommandID || command.id === displayStatusCommandID) return "instrument--editor"
	if (command.id === guardStatusCommandID) return "instrument--reference"
  if (command.id === inventoryLockerStatusCommandID || command.id === spellInventoryStatusCommandID) return "instrument--reference"
	if (command.id === partyEquipmentLoadoutCommandID) return "instrument--reference"
  if (command.id === activeCombatStatusCommandID) return "instrument--reference"
	if (command.id === merchantTesterCatalogCommandID) return "instrument--reference"
	if (command.id === scenarioStatusCommandID) return "instrument--scenario"
	if (command.id === saveSnapshotListCommandID) return "instrument--scenario"
  if (command.id === observationCommandID) return "instrument--editor"
  return "instrument--form"
}

function runQueryByID(commandID: string, silent: boolean): void {
  const command = supportedCommands.value.find((candidate) => candidate.id === commandID)
  if (command) runCommand(command, silent)
}

function autoExecuteSelectedQuery(): void {
  const command = selectedCommand.value
	if (!command) return
	if (command.id === merchantTesterCatalogCommandID
		&& (merchantCatalogExplicitRefreshRequired || instrumentRequestPending(merchantTesterCatalogCommandID))) return
	if (command.id === scenarioStatusCommandID) {
		runQueryByID(scenarioStatusCommandID, true)
		runQueryByID(scenarioDefinitionListCommandID, true)
		runQueryByID(scenarioRunStatusCommandID, true)
		runQueryByID(saveSnapshotListCommandID, true)
		return
	}
	if (command.id === saveSnapshotListCommandID) {
		runQueryByID(saveSnapshotListCommandID, true)
		return
	}
	if (command.id === partyCharacterCommandID) return
	if (command.id === partyEquipmentLoadoutCommandID) {
		runQueryByID(partyCharacterInspectCommandID, true)
		runQueryByID(inventoryLockerStatusCommandID, true)
		return
	}
	if (isMutation(command)) return
	if (command.id === guardStatusCommandID) {
		for (const commandID of worldDiagnosticCommandIDs) runQueryByID(commandID, true)
		return
	}
  if (activeWorkspace.value === "tools" && ![scenarioStatusCommandID, combatEncounterOptionsCommandID, merchantTesterCatalogCommandID].includes(command.id)) return
  if (["combat.map-catalog.all", combatEncounterOptionsCommandID].includes(command.id) && latestResults.value[command.id]) return
  if (command.id === conversationCatalogCommandID) {
    const loadedSource = (latestResults.value[command.id]?.data as ConversationCatalogData | undefined)?.selectedSource
    if (loadedSource === conversationSource.value) return
    drafts.value[command.id] = { source: conversationSource.value }
  }
  runCommand(command, true)
}

function runSpecializedMutation(commandID: string, argumentsByID: Record<string, string>, onSent?: (requestID: string) => void): boolean {
  const command = supportedCommands.value.find((candidate) => candidate.id === commandID)
  const owner = commandID === usableInventoryPresetCommandID && selectedCommandID.value === inventoryLockerStatusCommandID
    ? inventoryLockerStatusCommandID : companionOwnerCommandID(commandID)
  delete ownerErrors.value[owner]
  const unavailable = !isReady.value ? "Reconnect to the game before running this operation."
    : !command ? "This operation is not available in the current catalog."
    : commandUnavailableReason(command) || (commandPending(commandID) ? "Wait for the pending operation to finish." : "")
  if (unavailable || !command) {
    ownerErrors.value[owner] = unavailable
    return false
  }
  drafts.value[command.id] = argumentsByID
  const sent = runCommand(command, false, (requestID) => {
    const request = pending.value.get(requestID)
    if (request) request.ownerCommandID = owner
    onSent?.(requestID)
  })
  if (!sent) ownerErrors.value[owner] = Object.entries(formErrors.value[commandID] ?? {}).map(([field, error]) => `${field}: ${error}`).join(" ")
    || formMessages.value[commandID] || "The operation could not be sent. Check the connection and catalog constraints."
  return sent
}

function quickExitCombat(): void {
	const combat = mastheadActiveCombat.value
	if (!combat?.active || !combat.encounterId || !combat.snapshotFingerprint) return
	runSpecializedMutation(activeCombatExitCommandID, {
		encounterId: combat.encounterId,
		expectedSnapshot: combat.snapshotFingerprint,
	})
}

function saveScenarioDefinition(): void {
  scenarioValidationIssues.value = []
  scenarioValidationIntent.value = "save"
  scenarioValidationSnapshot.value = serializeScenarioDefinition(scenarioComposer.value)
  runSpecializedMutation(scenarioDefinitionValidateCommandID, {
    definition: scenarioValidationSnapshot.value,
  })
}

function dispatchScenarioSave(definition: string): void {
  if (scenarioComposer.value.savedReference && scenarioComposer.value.loadedRevision) {
    runSpecializedMutation(scenarioDefinitionUpdateCommandID, {
      reference: scenarioComposer.value.savedReference,
      expectedRevision: scenarioComposer.value.loadedRevision,
      definition,
    })
    return
  }
  runSpecializedMutation(scenarioDefinitionCreateCommandID, {
    source: scenarioComposer.value.source,
    definition,
  })
}

function reviewScenarioRun(): void {
  validateScenarioRun("review-run")
}

function validateScenarioRun(intent: "review-run" | "direct-run"): void {
  if (scenarioDraftIsDirty(scenarioComposer.value) || scenarioDraftHasPendingMigrations(scenarioComposer.value)) {
    disarmScenarioReview()
    if (intent === "direct-run") scenarioTask.value = "details"
    ownerErrors.value[scenarioStatusCommandID] = "Save the scenario changes and pending migrations before reviewing a run. Only a saved definition revision can run."
    return
  }
  if (!scenarioComposer.value.savedReference || !scenarioComposer.value.loadedRevision) return
  scenarioValidationIssues.value = []
  scenarioValidationIntent.value = intent
  scenarioValidationContext.value = `${sceneGeneration.value}:${scenarioReviewIdentity()}`
  scenarioValidationSnapshot.value = serializeScenarioDefinition(scenarioComposer.value)
  runSpecializedMutation(scenarioDefinitionValidateCommandID, {
    definition: scenarioValidationSnapshot.value,
  })
}

function deleteScenarioDefinition(): void {
  if (!scenarioComposer.value.savedReference || !scenarioComposer.value.loadedRevision) return
  runSpecializedMutation(scenarioDefinitionDeleteCommandID, {
    reference: scenarioComposer.value.savedReference,
    expectedRevision: scenarioComposer.value.loadedRevision,
  })
}

function duplicateScenarioDefinition(value: { destinationSource: ScenarioSource; destinationId: string; destinationName: string }): void {
  if (!scenarioComposer.value.savedReference || !scenarioComposer.value.loadedRevision) return
  runSpecializedMutation(scenarioDefinitionDuplicateCommandID, {
    sourceReference: scenarioComposer.value.savedReference,
    sourceRevision: scenarioComposer.value.loadedRevision,
    destinationSource: value.destinationSource,
    destinationId: value.destinationId,
    destinationName: value.destinationName,
  })
}

function getScenarioDefinition(reference: string): void {
  requestScenarioReplacement({ type: "load", reference, intent: "details" })
}

function playScenarioDefinition(reference: string): void {
  requestScenarioReplacement({ type: "load", reference, intent: "direct-run" })
}

function reviewHistoricalRun(run: ScenarioRunStatus): void {
  if (!run.launch) return
  requestScenarioReplacement({ type: "load", reference: run.reference, intent: "direct-run", navigateToScenarios: true,
    historicalRun: JSON.parse(JSON.stringify(run)) as ScenarioRunStatus })
}

function cancelRemainingScenarioSteps(run: ScenarioRunStatus): void {
  if (!run.currentProcess || !["queued", "running"].includes(run.state)) return
  runSpecializedMutation(scenarioRunCancelCommandID, { runId: run.runId })
}

function reviewScenarioRollback(run: ScenarioRunStatus): void {
  if (!run.rollback) return
  selectWorkspace("saves")
  selectedCommandID.value = saveSnapshotListCommandID
  const rollback = { ...run.rollback, epoch: rollbackReviewEpoch }
  runSpecializedMutation(saveSnapshotGetCommandID, { reference: rollback.reference }, (requestID) => rollbackRequests.set(requestID, rollback))
}

function revokeRollbackReview(): void {
  rollbackReviewEpoch += 1
  rollbackRequests.clear()
  saveLoadReviewTarget.value = undefined
}

function createScenarioDefinition(source: ScenarioSource): void {
  requestScenarioReplacement({ type: "create", source })
}

function updateScenarioTask(task: ScenarioTask): void {
  if (scenarioDefinitionWritePending.value && task !== "editor") return
  if (commandPending(scenarioDefinitionGetCommandID)) scenarioReplacementVersion += 1
  scenarioTask.value = task
  if (task === "editor") scenarioTargetActive.value = true
}

function scenarioDraftFingerprint(): string {
  return JSON.stringify(scenarioComposer.value)
}

function requestScenarioReplacement(action: ScenarioReplacementAction): void {
  if (scenarioDefinitionWritePending.value) return
  scenarioReplacementVersion += 1
  if (scenarioDraftIsDirty(scenarioComposer.value)) {
    scenarioReplacementReview.value = action
    if (scenarioCaptureDialogOpen.value) scenarioCaptureDialogMode.value = "dirty"
    else {
      scenarioReplacementPreviousFocus = document.activeElement as HTMLElement | null
      void nextTick(() => {
        setWorkbenchBackgroundInert(true, ".scenario-replacement-modal")
        scenarioReplacementDialog.value?.querySelector<HTMLElement>("button:not(:disabled)")?.focus()
      })
    }
    return
  }
  performScenarioReplacement(action, scenarioReplacementVersion)
}

function performScenarioReplacement(action: ScenarioReplacementAction, version: number): void {
  scenarioReplacementReview.value = undefined
  if (!scenarioCaptureDialogOpen.value) closeStandaloneScenarioReplacementReview()
  if (action.type === "create") {
    scenarioHistoricalRun.value = undefined
    const draft = newScenarioComposerDraft()
    draft.source = action.source
    if (action.name) {
      draft.definition.id = scenarioIDFromName(action.name)
      draft.definition.name = action.name
      draft.originalDefinition = cloneScenarioDefinition(draft.definition)
    }
    scenarioComposer.value = draft
    scenarioTargetActive.value = true
    scenarioRecording.value = false
    scenarioTask.value = "editor"
    if (action.name) {
      scenarioCaptureInsertionIndex.value = 0
      finishScenarioTargetSelection()
    }
    return
  }

  if (action.navigateToScenarios) {
    selectWorkspace("scenarios")
    selectedCommandID.value = scenarioStatusCommandID
    scenarioTask.value = "library"
  }

  const draftFingerprint = scenarioDraftFingerprint()
  if (action.intent === "capture") {
    scenarioCaptureDialogMode.value = "choose"
    scenarioCaptureError.value = ""
    scenarioCaptureLoadingReference.value = action.reference
  }
  const sent = runSpecializedMutation(scenarioDefinitionGetCommandID, { reference: action.reference }, (requestID) => {
    scenarioDefinitionGetRequests.set(requestID, {
      version,
      draftFingerprint,
      reference: action.reference,
      intent: action.intent,
      historicalRun: action.historicalRun,
    })
  })
  if (!sent && action.intent === "capture") {
    scenarioCaptureLoadingReference.value = ""
    scenarioCaptureError.value = "The scenario definition could not be requested. Check the Workbench connection and retry."
  }
}

function discardScenarioDraftAndReplace(): void {
  const action = scenarioReplacementReview.value
  if (!action) return
  scenarioComposer.value = {
    ...scenarioComposer.value,
    definition: cloneScenarioDefinition(scenarioComposer.value.originalDefinition),
  }
  performScenarioReplacement(action, scenarioReplacementVersion)
}

function cancelScenarioReplacement(): void {
  scenarioReplacementReview.value = undefined
  if (!scenarioCaptureDialogOpen.value) closeStandaloneScenarioReplacementReview()
  if (scenarioCaptureDialogOpen.value && scenarioCaptureDialogMode.value === "dirty") scenarioCaptureDialogMode.value = "choose"
}

function closeStandaloneScenarioReplacementReview(): void {
  setWorkbenchBackgroundInert(false, ".scenario-replacement-modal")
  const previousFocus = scenarioReplacementPreviousFocus
  scenarioReplacementPreviousFocus = null
  void nextTick(() => previousFocus?.focus())
}

function openActiveScenarioEditor(stepIndex?: number): void {
  if (!scenarioTargetActive.value) return
  selectWorkspace("scenarios")
  selectedCommandID.value = scenarioStatusCommandID
  scenarioTask.value = "editor"
  void nextTick(() => {
    const step = stepIndex === undefined ? undefined : document.querySelector<HTMLElement>(`[data-scenario-screen="editor"] [data-scenario-step="${stepIndex}"]`)
    if (typeof step?.scrollIntoView === "function") step.scrollIntoView({ block: "center" })
    const target = step ?? document.querySelector<HTMLElement>('[data-scenario-screen="editor"] input:not(:disabled), [data-scenario-screen="editor"] select:not(:disabled), [data-scenario-screen="editor"] button:not(:disabled)')
    target?.focus()
  })
}

function updateScenarioComposer(draft: ScenarioComposerDraft): void {
  if (scenarioDefinitionWritePending.value) return
  if (serializeScenarioDefinition(scenarioComposer.value) !== serializeScenarioDefinition(draft)) scenarioValidationIssues.value = []
  scenarioComposer.value = draft
  if (scenarioReviewedDefinition.value && scenarioReviewedDefinition.value !== scenarioReviewIdentity()) disarmScenarioReview()
}

function openScenarioTargetChooser(intent: ScenarioCaptureIntent, result?: ResultEntry): void {
  if (scenarioDefinitionWritePending.value || commandPending(scenarioDefinitionGetCommandID)) {
    liveStatus.value = "Wait for the current scenario definition operation before choosing a capture target."
    return
  }
  scenarioTargetPreviousFocus = document.activeElement as HTMLElement | null
  scenarioCaptureIntent.value = intent
  scenarioCaptureResult.value = result
  scenarioCaptureDialogMode.value = "choose"
  scenarioCaptureDialogOpen.value = true
  scenarioCaptureNewName.value = ""
  scenarioCaptureSearch.value = ""
  scenarioCaptureSelectedReference.value = ""
  scenarioCaptureExpectedRevision.value = ""
  scenarioCaptureInsertionIndex.value = 0
  scenarioCaptureError.value = ""
  scenarioReplacementReview.value = undefined
  runQueryByID(scenarioDefinitionListCommandID, true)
  void nextTick(() => {
    setScenarioBackgroundInert(true)
    focusFirstScenarioTargetControl()
  })
}

function closeScenarioTargetChooser(): void {
  if (scenarioCaptureLoadingReference.value) return
  scenarioCaptureDialogOpen.value = false
  scenarioCaptureResult.value = undefined
  scenarioReplacementReview.value = undefined
  scenarioCaptureSelectedReference.value = ""
  scenarioCaptureExpectedRevision.value = ""
  scenarioCaptureError.value = ""
  setScenarioBackgroundInert(false)
  scenarioTargetPreviousFocus?.focus()
  scenarioTargetPreviousFocus = null
}

function setScenarioBackgroundInert(inert: boolean): void {
  setWorkbenchBackgroundInert(inert, ".scenario-target-modal")
}

function setWorkbenchBackgroundInert(inert: boolean, modalSelector: string): void {
  for (const element of document.querySelectorAll<HTMLElement>(".workbench-shell > *")) {
    if (!element.matches(modalSelector)) element.inert = inert
  }
}

function focusFirstScenarioTargetControl(): void {
  scenarioTargetDialog.value?.querySelector<HTMLElement>("input:not(:disabled), select:not(:disabled), button:not(:disabled)")?.focus()
}

function chooseScenarioTarget(reference: string): void {
  if (scenarioCaptureIntent.value === "capture" && scenarioCaptureResult.value) {
    const definition = scenarioDefinitionList.value.find((candidate) => candidate.reference === reference)
    if (!definition) return
    scenarioCaptureSelectedReference.value = reference
    scenarioCaptureExpectedRevision.value = definition.revision
    const capturedCommand = commands.value.find((command) => command.id === scenarioCaptureResult.value?.invocation?.commandId)
    const terminalIndex = scenarioTerminalStepIndex(definition.steps, commands.value)
    scenarioCaptureInsertionIndex.value = capturedCommand?.scenarioTerminal || terminalIndex < 0 ? definition.steps.length : terminalIndex
    return
  }
  activateScenarioTarget(reference)
}

function confirmScenarioCaptureTarget(): void {
  if (!scenarioCaptureSelectedReference.value) return
  activateScenarioTarget(scenarioCaptureSelectedReference.value)
}

function activateScenarioTarget(reference: string): void {
  if (scenarioTargetActive.value && reference === scenarioComposer.value.savedReference) {
    finishScenarioTargetSelection()
    return
  }
  requestScenarioReplacement({ type: "load", reference, intent: "capture" })
}

function scenarioTargetChoices(source: ScenarioSource): ScenarioDefinitionListResult["definitions"] {
  const search = scenarioCaptureSearch.value.trim().toLocaleLowerCase()
  return scenarioDefinitionList.value.filter((definition) => definition.source === source
    && (!scenarioTargetActive.value || definition.reference !== scenarioComposer.value.savedReference)
    && (!search || [definition.name, definition.id, definition.reference, definition.description]
      .some((value) => value.toLocaleLowerCase().includes(search))))
}

function failScenarioCaptureLoad(message: string): void {
  if (!scenarioCaptureLoadingReference.value) return
  scenarioCaptureLoadingReference.value = ""
  scenarioCaptureError.value = message
}

function requestNewScenarioTarget(): void {
  scenarioCaptureDialogMode.value = "new"
  scenarioCaptureNewName.value = ""
  void nextTick(() => scenarioTargetDialog.value?.querySelector<HTMLInputElement>("[data-scenario-target-name]")?.focus())
}

function returnToActiveScenarioEditor(): void {
  closeScenarioTargetChooser()
  openActiveScenarioEditor()
}

function createScenarioCaptureTarget(): void {
  const name = scenarioCaptureNewName.value.trim()
  const id = scenarioCaptureNewID.value
  if (!name || !id || scenarioCaptureNewIDExists.value) return
  requestScenarioReplacement({ type: "create", source: "personal", name })
}

function finishScenarioTargetSelection(): void {
  const capture = scenarioCaptureResult.value
  const intent = scenarioCaptureIntent.value
  scenarioCaptureDialogOpen.value = false
  scenarioCaptureResult.value = undefined
  scenarioReplacementReview.value = undefined
  scenarioCaptureError.value = ""
  setScenarioBackgroundInert(false)
  scenarioTargetPreviousFocus = null
  const insertedStep = capture ? appendResultToActiveScenario(capture, scenarioCaptureInsertionIndex.value) : undefined
  if (intent === "record") scenarioRecording.value = true
  openActiveScenarioEditor(insertedStep)
}

function toggleScenarioRecording(): void {
  if (scenarioRecording.value) {
    scenarioRecording.value = false
    return
  }
  if (scenarioTargetActive.value) {
    scenarioRecording.value = true
    return
  }
  openScenarioTargetChooser("record")
}

function startScenarioRun(): void {
  if (scenarioDraftIsDirty(scenarioComposer.value) || scenarioDraftHasPendingMigrations(scenarioComposer.value)) {
    disarmScenarioReview()
    ownerErrors.value[scenarioStatusCommandID] = "Save the scenario changes and pending migrations before starting a run. Only a saved definition revision can run."
    return
  }
  if (!scenarioComposer.value.savedReference || !scenarioComposer.value.loadedRevision || scenarioReviewedSceneGeneration.value === 0
    || scenarioTask.value !== "run-review" || scenarioReviewedDefinition.value !== scenarioReviewIdentity()
    || scenarioReviewedSceneGeneration.value !== sceneGeneration.value
    || scenarioReviewedBaselineRevision.value !== reviewedScenarioBaselineRevision()
    || historicalRunIssue.value
    || Object.keys(scenarioLaunchValueIssues(scenarioComposer.value.definition.parameters ?? [], scenarioComposer.value.launchValues)).length) return
  const sent = runSpecializedMutation(scenarioRunStartCommandID, {
    reference: scenarioComposer.value.savedReference,
    expectedRevision: scenarioComposer.value.loadedRevision,
    expectedBaselineRevision: scenarioReviewedBaselineRevision.value,
    reviewedSceneGeneration: String(scenarioReviewedSceneGeneration.value),
    parameterValues: serializeScenarioLaunchValues(scenarioComposer.value.launchValues),
  })
  if (sent) disarmScenarioReview()
}

function scenarioReviewIdentity(): string {
  return JSON.stringify([sessionID.value, scenarioComposer.value.savedReference, scenarioComposer.value.loadedRevision, scenarioComposer.value.definition])
}

const historicalRunIssue = computed(() => {
  const run = scenarioHistoricalRun.value
  if (!run) return ""
  if (run.definitionRevision !== scenarioComposer.value.loadedRevision) return `Historical definition revision ${run.definitionRevision} differs from current revision ${scenarioComposer.value.loadedRevision}. Review Again is unsupported; open the Library explicitly to configure a new run.`
  if (run.baseline.type === "snapshot" && (run.baselineReference !== scenarioComposer.value.definition.baseline.reference || run.baselineRevision !== reviewedScenarioBaselineRevision())) return "The recorded baseline revision is no longer available. Review Again cannot substitute a newer snapshot."
  return ""
})

function disarmScenarioReview(): void {
  if (scenarioValidationIntent.value === "review-run" || scenarioValidationIntent.value === "direct-run") scenarioValidationIntent.value = undefined
  scenarioReviewedSceneGeneration.value = 0
  scenarioReviewedDefinition.value = ""
  if (scenarioTask.value === "run-review") scenarioTask.value = "details"
}

function reviewedScenarioBaselineRevision(): string {
  const baseline = scenarioComposer.value.definition.baseline
  if (baseline.type !== "snapshot" || !baseline.reference) return ""
  return saveList.value?.snapshots.find((snapshot) => snapshot.reference === baseline.reference)?.revision ?? ""
}

function invalidateStaleScenarioBaselineReview(): void {
  if (!scenarioReviewedBaselineRevision.value) return
  const current = reviewedScenarioBaselineRevision()
  if (current === scenarioReviewedBaselineRevision.value) return
  scenarioReviewedSceneGeneration.value = 0
  scenarioReviewedBaselineRevision.value = ""
  if (scenarioTask.value === "run-review") scenarioTask.value = "details"
  liveStatus.value = "Scenario baseline revision changed. Review the run again."
}

function inspectSaveSnapshot(reference: string): void {
  revokeRollbackReview()
  runSpecializedMutation(saveSnapshotGetCommandID, { reference })
}

function useSaveAsScenarioBaseline(snapshot: SaveSnapshotSummary): void {
  if (!scenarioTargetActive.value) return
  scenarioValidationIssues.value = []
  scenarioComposer.value = {
    ...scenarioComposer.value,
    definition: { ...scenarioComposer.value.definition, baseline: { type: "snapshot", reference: snapshot.reference } },
  }
  scenarioReviewedSceneGeneration.value = 0
  scenarioReviewedBaselineRevision.value = ""
  liveStatus.value = `Set ${snapshot.reference} as the active scenario draft baseline.`
}

function saveSnapshotListData(value: unknown): SaveSnapshotListData { return value as SaveSnapshotListData }
function saveSnapshotDetailData(value: unknown) { return (value as SaveSnapshotGetData).snapshot }

function scenarioDefinitionListData(value: unknown): ScenarioDefinitionListResult {
  return value as ScenarioDefinitionListResult
}

function scenarioRunStatusData(value: unknown): ScenarioRunStatusResult {
  return value as ScenarioRunStatusResult
}

function resultEligibleForScenario(result: CommandResult): boolean {
  if (!result.success || !result.invocation || scenarioCompanionCommandIDs.includes(result.invocation.commandId) || saveSnapshotCommandIDs.includes(result.invocation.commandId)) return false
	const command = commands.value.find((candidate) => candidate.id === result.invocation?.commandId)
  return Boolean(command && scenarioSafeCommands([command]).length === 1)
}

function requestAddResultToScenario(result: ResultEntry): void {
  if (!resultEligibleForScenario(result) || capturedScenarioTargets.value.has(result.requestId)) return
  if (!scenarioTargetActive.value) {
    openScenarioTargetChooser("capture", result)
    return
  }
  appendResultToActiveScenario(result)
}

function appendResultToActiveScenario(result: ResultEntry, requestedIndex?: number): number | undefined {
  if (!scenarioTargetActive.value || !resultEligibleForScenario(result) || capturedScenarioTargets.value.has(result.requestId) || !result.invocation) return undefined
  if (scenarioDefinitionWritePending.value) {
    queueDeferredScenarioCapture(result)
    return undefined
	}
	const command = commands.value.find((candidate) => candidate.id === result.invocation?.commandId)
	const terminalIndex = scenarioTerminalStepIndex(scenarioComposer.value.definition.steps, commands.value)
	if (command?.scenarioTerminal && terminalIndex >= 0) {
		liveStatus.value = "Capture not added. This scenario already has a final combat-start step."
		return undefined
	}
	const insertion = insertScenarioStep(
		scenarioComposer.value.definition.steps,
		scenarioStepFromInvocation(result.invocation, command),
		commands.value,
		requestedIndex,
	)
	scenarioValidationIssues.value = []
	scenarioComposer.value = {
    ...scenarioComposer.value,
    definition: {
      ...scenarioComposer.value.definition,
		steps: insertion.steps,
    },
  }
  const captured = new Map(capturedScenarioTargets.value)
  captured.set(result.requestId, scenarioComposer.value.definition.name)
  capturedScenarioTargets.value = captured
  if (scenarioTask.value === "run-review") {
    scenarioReviewedSceneGeneration.value = 0
    scenarioTask.value = "editor"
  }
	liveStatus.value = terminalIndex >= 0
		? `Inserted ${commandName(result.invocation.commandId)} before the final scenario step.`
		: `Added ${commandName(result.invocation.commandId)} to ${scenarioComposer.value.definition.name}.`
	return insertion.index
}

function scenarioCaptureInsertionLabel(steps: ScenarioStep[], index: number, terminalIndex: number): string {
	if (steps.length === 0) return "First step"
	if (index === 0) return "Before Step 01"
	const previous = steps[index - 1]
	const previousName = previous.type === "command" ? commandName(previous.commandId) : "Note"
	if (index === steps.length && terminalIndex < 0) return `After Step ${String(index).padStart(2, "0")} · ${previousName} · End`
	return `After Step ${String(index).padStart(2, "0")} · ${previousName}`
}

function queueDeferredScenarioCapture(result: ResultEntry): void {
  if (deferredScenarioCaptures.value.some((capture) => capture.requestId === result.requestId)) return
  if (deferredScenarioCaptures.value.length >= deferredScenarioCaptureLimit) {
    liveStatus.value = `Scenario capture queue full. ${commandName(result.commandId)} was not captured; review Global History and add it after the save finishes.`
    return
  }
  deferredScenarioCaptures.value = [...deferredScenarioCaptures.value, result]
  liveStatus.value = `Capture queued: ${commandName(result.commandId)} will be added after ${scenarioComposer.value.definition.name} finishes saving.`
}

function flushDeferredScenarioCaptures(): void {
  if (scenarioDefinitionWritePending.value || deferredScenarioCaptures.value.length === 0) return
  const captures = deferredScenarioCaptures.value
  deferredScenarioCaptures.value = []
  for (const capture of captures) appendResultToActiveScenario(capture)
  liveStatus.value = `Added ${captures.length} queued capture${captures.length === 1 ? "" : "s"} to ${scenarioComposer.value.definition.name}.`
}

function scenarioCaptureState(result: ResultEntry): "completed" | "pending" | "ready" | "unavailable" {
  if (capturedScenarioTargets.value.has(result.requestId)) return "completed"
  if (deferredScenarioCaptures.value.some((capture) => capture.requestId === result.requestId)) return "pending"
  return resultEligibleForScenario(result) ? "ready" : "unavailable"
}

function selectConversationSource(source: string): void {
  if (source === conversationSource.value) return
  conversationSource.value = source
  selectedConversationIndex.value = -1
  const command = supportedCommands.value.find((candidate) => candidate.id === conversationCatalogCommandID)
  if (!command) return
  drafts.value[command.id] = { source }
  runCommand(command, true)
}

function inspectConversationScript(index: number): void {
  selectedConversationIndex.value = index
  const command = supportedCommands.value.find((candidate) => candidate.id === conversationInspectCommandID)
  if (!command) return
  drafts.value[command.id] = { source: conversationSource.value, index: String(index) }
  runCommand(command, true)
}

function openNPCConversation(value: { source: string; index: number }): void {
  activeWorkspace.value = "codex"
  selectedCommandID.value = conversationCatalogCommandID
  conversationSource.value = value.source
  selectedConversationIndex.value = value.index
  const catalog = supportedCommands.value.find((candidate) => candidate.id === conversationCatalogCommandID)
  if (catalog) {
    drafts.value[catalog.id] = { source: value.source }
    runCommand(catalog, true)
  }
  inspectConversationScript(value.index)
}

function setMonsterGeneration(value: { enabled: boolean; odds: number }): void {
  runSpecializedMutation(monsterGenerationSetCommandID, { enabled: String(value.enabled), odds: String(value.odds) })
}

function setCalendarTime(value: { hour: number; minute: number }): void {
  runSpecializedMutation(calendarSetTimeCommandID, { hour: String(value.hour), minute: String(value.minute) })
}

function setCalendarDay(value: { day: number }): void {
  runSpecializedMutation(calendarSetDayCommandID, { day: String(value.day) })
}

function addSearchItemStack(value: { x: number; y: number; item: string; quantity: number }): void {
  runSpecializedMutation(searchItemStackAddCommandID, {
    x: String(value.x),
    y: String(value.y),
    item: value.item,
    quantity: String(value.quantity),
  })
}

function resetSearchProgress(): void {
  runSpecializedMutation(searchProgressResetCommandID, { scope: "one_time_locations" })
}

function applyWorldPosition(value: {
  placement: string
  route?: string
  locationId: number
  floor?: number
  x?: number
  y?: number
  expectedLocationId: number
  expectedFloor: number
  expectedX: number
  expectedY: number
  expectedSourceFingerprint: string
}): void {
  const argumentsByID: Record<string, string> = {
    placement: value.placement,
    location: String(value.locationId),
    expectedLocationId: String(value.expectedLocationId),
    expectedFloor: String(value.expectedFloor),
    expectedX: String(value.expectedX),
    expectedY: String(value.expectedY),
    expectedSourceFingerprint: value.expectedSourceFingerprint,
  }
  if (value.route !== undefined) argumentsByID.route = value.route
  if (value.floor !== undefined) argumentsByID.floor = String(value.floor)
  if (value.x !== undefined) argumentsByID.x = String(value.x)
  if (value.y !== undefined) argumentsByID.y = String(value.y)
  runSpecializedMutation(worldPositionSetCommandID, argumentsByID)
}

function applyDungeonSetup(value: {
  dungeon: string
  floor: number
  x: number
  y: number
  facing: string
  lit: boolean
  expectedLocationId: number
  expectedFloor: number
  expectedX: number
  expectedY: number
  expectedDungeonRevision: number
  expectedSourceFingerprint: string
}): void {
  runSpecializedMutation(dungeonExplorerSetupCommandID, {
    dungeon: value.dungeon,
    floor: String(value.floor),
    x: String(value.x),
    y: String(value.y),
    facing: value.facing,
    lit: String(value.lit),
    expectedLocationId: String(value.expectedLocationId),
    expectedFloor: String(value.expectedFloor),
    expectedX: String(value.expectedX),
    expectedY: String(value.expectedY),
    expectedSourceFingerprint: value.expectedSourceFingerprint,
    expectedDungeonRevision: String(value.expectedDungeonRevision),
  })
}

function openWorldNavigator(value: { x: number; y: number }): void {
  activeWorkspace.value = "game"
  commandSearch.value = ""
  linkedWorldCoordinates.value = value
  selectedCommandID.value = worldPositionStatusCommandID
  void nextTick(autoExecuteSelectedQuery)
}

function updateDraftAndMaybeRun(command: CommandDefinition, parameterID: string, value: string): void {
  updateDraft(command, parameterID, value)
  if (activeWorkspace.value !== "tools" && !isMutation(command)) void nextTick(() => runCommand(command, true))
}

function commandUsesDynamicOptions(command: CommandDefinition): boolean {
  return (command.parameters ?? []).some((parameter) => Boolean(parameter.optionsProvider))
}

function isMutation(command: CommandDefinition): boolean {
  return command.executionModes.some((mode) => mode !== "query")
}

function commandUnavailableReason(command: CommandDefinition): string {
  if (awaitingOptionsRefresh.value && commandUsesDynamicOptions(command)) return "Refreshing dynamic options before another operation can run."
  for (const parameter of command.parameters ?? []) {
    if (!parameter.optionsProvider) continue
    if (optionSets.value[parameter.optionsProvider] === undefined) return "Dynamic options are unavailable in the active scene."
    if (optionUnavailableReasons.value[parameter.optionsProvider]) return optionUnavailableReasons.value[parameter.optionsProvider]
  }

  return ""
}

function commandApplyDisabled(command: CommandDefinition): boolean {
  if (!isReady.value || commandPending(command.id) || hasUnsupportedParameters(command, optionSets.value) || commandUnavailableReason(command)) return true
  if (command.id === partyCharacterCommandID) {
    return partyAlterationPreview(command, optionSets.value, draftFor(command), dirtyFieldsFor(command)).length === 0
  }

  return false
}

function clearCharacterAlterationErrors(commandID: string): void {
  const errors = { ...formErrors.value[commandID] }
  for (const fieldID of partyCharacterFieldIDs) delete errors[fieldID]
  formErrors.value[commandID] = errors
  delete formMessages.value[commandID]
}

function receiveProtocolError(message: Envelope): void {
  const payload = message.payload as { code?: string; message?: string }
  const pendingRequest = message.id ? pending.value.get(message.id) : undefined
  if (!pendingRequest && selectedCommandID.value) ownerErrors.value[selectedCommandID.value] = `${payload.code ?? "protocol_error"}: ${payload.message ?? "The connection received an invalid protocol message."}`
  if (message.id) {
    pending.value.delete(message.id)
    silentRequestIDs.delete(message.id)
    scenarioDefinitionGetRequests.delete(message.id)
  }
  if (pendingRequest?.commandID === scenarioDefinitionGetCommandID) {
    failScenarioCaptureLoad(payload.message ?? "The scenario definition request failed.")
  }
  appendResult({
    entryID: `protocol-${++localEntrySequence}`,
    sessionId: sessionID.value,
    requestId: message.id ?? "protocol",
    commandId: pendingRequest?.commandID ?? "protocol",
    ownerCommandID: pendingRequest?.ownerCommandID,
    sceneGeneration: sceneGeneration.value,
    success: false,
    errorCode: payload.code ?? "protocol_error",
    errorMessage: payload.message ?? "Unknown protocol error",
    receivedAt: new Date(),
  })
}

function reportLostPendingRequests(): void {
  for (const [requestID, request] of pending.value) {
    silentRequestIDs.delete(requestID)
    rememberProcessedRequest(requestID)
    scenarioDefinitionGetRequests.delete(requestID)
    appendResult({
      entryID: `lost-${++localEntrySequence}`,
      sessionId: sessionID.value,
      requestId: requestID,
      commandId: request.commandID,
      ownerCommandID: request.ownerCommandID,
      sceneGeneration: sceneGeneration.value,
      success: false,
      errorCode: "session_lost",
      errorMessage: "The previous session was not resumed; this command's outcome is unknown.",
      receivedAt: new Date(),
    })
    if (request.commandID === scenarioDefinitionGetCommandID) {
      failScenarioCaptureLoad("The previous session was not resumed. Review the scenario and retry capture placement.")
    }
  }
}

function appendResult(result: ResultEntry): void {
  results.value.push(result)
  latestResults.value = { ...latestResults.value, [result.commandId]: result }
  const changes = result.changes?.length ? ` ${result.changes.length} changes applied.` : ""
  liveStatus.value = `${commandName(result.commandId)} ${result.success ? "completed" : "failed"}.${changes}`
  if (results.value.length > resultLimit) {
    results.value.shift()
    trimmedResults.value += 1
  }
  if (followResults.value) void nextTick(scrollToLatest)
}

function rememberProcessedRequest(requestID: string): void {
  processedRequestIDs.add(requestID)
  processedRequestOrder.push(requestID)
  if (processedRequestOrder.length > resultLimit * 2) {
    const removed = processedRequestOrder.shift()
    if (removed) processedRequestIDs.delete(removed)
  }
}

function commandPending(commandID: string): boolean {
  return [...pending.value.values()].some((request) => request.commandID === commandID)
}

function instrumentRequestPending(ownerCommandID: string): boolean {
  return [...pending.value.values()].some((request) => companionOwnerCommandID(request.commandID) === ownerCommandID)
}

function commandPendingLabel(commandID: string): string {
  const request = [...pending.value.values()].find((candidate) => candidate.commandID === commandID)
  if (request && now.value - request.lastConfirmedAt >= operationStallMs) return "STALLED · RECONCILING"
  return request?.phase === "accepted" ? "ACCEPTED..." : "SENDING..."
}

function checkPendingOperations(): void {
  now.value = Date.now()
  for (const [requestID, request] of pending.value) {
    const unconfirmedFor = now.value - request.lastConfirmedAt
    if (unconfirmedFor >= operationReconnectMs) {
      socket.value?.close(1011, "operation status timeout")
      return
    }
    if (unconfirmedFor >= operationStallMs && !request.statusRequested) {
      const sent = send(envelope(requestID, "operation.status", { requestId: requestID }))
      if (sent) pending.value.set(requestID, { ...request, statusRequested: true })
    }
  }
	if (!isReady.value || document.hidden) return
  if (selectedCommand.value && commandUsesDynamicOptions(selectedCommand.value)) requestCatalog()
	runQueryByID(calendarStatusCommandID, true)
	runQueryByID(activeCombatStatusCommandID, true)
	if (!selectedCommand.value || [calendarStatusCommandID, activeCombatStatusCommandID].includes(selectedCommand.value.id)) return
	if (instrumentRequestPending(selectedCommand.value.id)) return
	if (selectedCommand.value.id === scenarioStatusCommandID) {
		runQueryByID(scenarioStatusCommandID, true)
		runQueryByID(scenarioRunStatusCommandID, true)
		return
	}
	if (selectedCommand.value.id === partyEquipmentLoadoutCommandID) {
		runQueryByID(partyCharacterInspectCommandID, true)
		runQueryByID(inventoryLockerStatusCommandID, true)
		return
	}
	if (selectedCommand.value.id === partyCharacterInspectCommandID) {
		runQueryByID(partyCharacterInspectCommandID, true)
		return
	}
	if (!polledIntegratedQueryIDs.has(selectedCommand.value.id)) return
  if (selectedCommand.value.id === windStatusCommandID && livePollingPaused.value) return
  runCommand(selectedCommand.value, true)
}

function commandName(commandID: string): string {
  return commands.value.find((command) => command.id === commandID)?.name ?? commandID
}

function formatTime(date: Date): string {
  return date.toLocaleTimeString([], { hour12: false })
}

function toggleHistory(): void {
  historyCollapsed.value = !historyCollapsed.value
}

function toggleHistoryDock(): void {
  stopHistoryResize?.()
  historyDock.value = historyDock.value === "bottom" ? "right" : "bottom"
}

function startHistoryResize(event: PointerEvent): void {
  if (historyCollapsed.value) return
  stopHistoryResize?.()
  const startX = event.clientX
  const startY = event.clientY
  const bounds = historyPanel.value?.getBoundingClientRect()
  const measuredWidth = bounds?.width ?? 0
  const measuredHeight = bounds?.height ?? 0
  const startWidth = measuredWidth > 0 ? measuredWidth : historyWidth.value
  const startHeight = measuredHeight > 0 ? measuredHeight : historyHeight.value
  const move = (moveEvent: PointerEvent) => {
    if (historyDock.value === "right") {
      const maximum = Math.max(360, Math.floor(window.innerWidth * 0.55))
      historyWidth.value = Math.min(maximum, Math.max(300, startWidth + startX - moveEvent.clientX))
    } else {
      const maximum = Math.max(160, Math.floor(window.innerHeight * 0.65))
      historyHeight.value = Math.min(maximum, Math.max(120, startHeight + startY - moveEvent.clientY))
    }
  }
  const stop = () => {
    window.removeEventListener("pointermove", move)
    window.removeEventListener("pointerup", stop)
    stopHistoryResize = undefined
  }
  stopHistoryResize = stop
  window.addEventListener("pointermove", move)
  window.addEventListener("pointerup", stop)
  event.preventDefault()
}

function shrineStatusData(value: unknown): ShrineStatusData {
  return value as ShrineStatusData
}

function combatMapData(value: unknown): CombatMapCatalogData {
  return value as CombatMapCatalogData
}

function calendarStatusData(value: unknown): CalendarStatusData {
  return value as CalendarStatusData
}

function loggingStatusData(value: unknown): LoggingStatusData {
  return value as LoggingStatusData
}

function searchStatusData(value: unknown): SearchStatusData {
  return value as SearchStatusData
}

function questItemStatusData(value: unknown): QuestItemStatusData {
  return value as QuestItemStatusData
}

function worldPositionStatusData(value: unknown): WorldPositionStatusData {
  return value as WorldPositionStatusData
}

function dungeonExplorerStatusData(value: unknown): DungeonExplorerStatusData {
  return value as DungeonExplorerStatusData
}

function moongateStatusData(value: unknown): MoongateStatusData {
  return value as MoongateStatusData
}

function conversationCatalogData(value: unknown): ConversationCatalogData {
  return value as ConversationCatalogData
}

function conversationDetailData(value: unknown): ConversationDetailData {
  return value as ConversationDetailData
}

function combatEncounterOptionsData(value: unknown): CombatEncounterOptionsData {
  return value as CombatEncounterOptionsData
}

function merchantTesterCatalogData(value: unknown): MerchantTesterCatalogData {
  return value as MerchantTesterCatalogData
}

function vfxStatusData(value: unknown): VFXStatusData {
  return value as VFXStatusData
}

function monsterEcosystemData(value: unknown): MonsterEcosystemData {
  return value as MonsterEcosystemData
}

function vehicleStatusData(value: unknown): VehicleStatusData {
  return value as VehicleStatusData
}

function debugOptionsStatusData(value: unknown): DebugOptionsStatusData {
  return value as DebugOptionsStatusData
}

function displayStatusData(value: unknown): DisplayStatusData {
  return value as DisplayStatusData
}

function reduxModificationsStatusData(value: unknown): ReduxModificationsStatusData {
  return value as ReduxModificationsStatusData
}

function guardStatusData(value: unknown): GuardStatusData {
  return value as GuardStatusData
}

function shadowlordStatusData(value: unknown): ShadowlordStatusData {
  return value as ShadowlordStatusData
}

function deathStatusData(value: unknown): DeathStatusData {
  return value as DeathStatusData
}

function inventoryLockerStatusData(value: unknown): InventoryLockerStatusData {
  return value as InventoryLockerStatusData
}

function spellInventoryStatusData(value: unknown): SpellInventoryStatusData {
  return value as SpellInventoryStatusData
}

function activeCombatStatusData(value: unknown): ActiveCombatStatusData {
  return value as ActiveCombatStatusData
}

function observationResultData(value: unknown): ObservationResultData {
  return value as ObservationResultData
}

function windDirection(result: ResultEntry | undefined): string | undefined {
  return (result?.data as { direction?: string } | undefined)?.direction
}

function formatChangeValue(value: unknown): string {
  return typeof value === "string" || typeof value === "number" || typeof value === "boolean" ? String(value) : JSON.stringify(value)
}

function clearResults(): void {
  results.value = []
  trimmedResults.value = 0
}

function onStreamScroll(): void {
  if (!stream.value) return
  const distanceFromBottom = stream.value.scrollHeight - stream.value.scrollTop - stream.value.clientHeight
  followResults.value = distanceFromBottom < 24
}

function toggleFollow(): void {
  followResults.value = !followResults.value
  if (followResults.value) void nextTick(scrollToLatest)
}

function scrollToLatest(): void {
  if (!stream.value) return
  if (typeof stream.value.scrollTo === "function") {
    stream.value.scrollTo({ top: stream.value.scrollHeight })
  } else {
    stream.value.scrollTop = stream.value.scrollHeight
  }
}

function inputValue(event: Event): string {
  return (event.target as HTMLInputElement | HTMLSelectElement).value
}

function onKeyDown(event: KeyboardEvent): void {
  const activeElement = document.activeElement as HTMLElement | null
  if (scenarioReplacementReview.value && !scenarioCaptureDialogOpen.value) {
    if (event.key === "Escape") {
      event.preventDefault()
      cancelScenarioReplacement()
      return
    }
    if (event.key === "Tab") {
      const controls = [...(scenarioReplacementDialog.value?.querySelectorAll<HTMLElement>("button:not(:disabled)") ?? [])]
      if (controls.length > 0) {
        const currentIndex = controls.indexOf(activeElement ?? controls[0])
        const offset = event.shiftKey ? -1 : 1
        controls[(currentIndex + offset + controls.length) % controls.length]?.focus()
        event.preventDefault()
      }
      return
    }
    if (event.key === "F5" || ((event.ctrlKey || event.metaKey) && event.key.toLocaleLowerCase() === "k")) {
      event.preventDefault()
    }
    return
  }
  if ((event.ctrlKey || event.metaKey) && event.key.toLocaleLowerCase() === "k") {
    event.preventDefault()
    openCommandPalette()
    return
  }
  if (commandPaletteOpen.value) {
    if (event.key === "F5") event.preventDefault()
    return
  }
  if (scenarioCaptureDialogOpen.value) {
    if (event.key === "Escape") {
      event.preventDefault()
      closeScenarioTargetChooser()
      return
    }
    if (event.key === "Tab") {
      const controls = [...(scenarioTargetDialog.value?.querySelectorAll<HTMLElement>("button:not(:disabled), input:not(:disabled), select:not(:disabled), [tabindex='0']") ?? [])]
      if (controls.length > 0) {
        const currentIndex = controls.indexOf(activeElement ?? controls[0])
        const offset = event.shiftKey ? -1 : 1
        controls[(currentIndex + offset + controls.length) % controls.length]?.focus()
        event.preventDefault()
      }
      return
    }
    if (event.key === "F5") event.preventDefault()
    return
  }
  if (event.ctrlKey && event.key === "`") {
    event.preventDefault()
    stream.value?.focus()
    return
  }
  if ((event.key === "ArrowDown" || event.key === "ArrowUp") && activeElement?.dataset.commandSelect) {
    const buttons = commandCategories.value
      .flatMap((category) => categoryExpanded(category.name) ? category.commands : [])
      .map((command) => document.querySelector<HTMLButtonElement>(`button[data-command-select="${command.id}"]`))
      .filter((button): button is HTMLButtonElement => button !== null && !button.disabled)
    if (buttons.length === 0) return
    const currentIndex = buttons.indexOf(activeElement as HTMLButtonElement)
    const offset = event.key === "ArrowDown" ? 1 : -1
    const nextButton = buttons[(currentIndex + offset + buttons.length) % buttons.length]
    nextButton?.focus()
    const command = availableCommands.value.find((candidate) => candidate.id === nextButton?.dataset.commandSelect)
    if (command) selectCommand(command)
    event.preventDefault()
    return
  }
  if (event.key !== "F5" || event.ctrlKey || event.altKey || event.metaKey || !selectedCommand.value) return
  event.preventDefault()
  if (selectedCommand.value.id === observationCommandID) return
  if (selectedCommand.value.id === partyEquipmentLoadoutCommandID) { autoExecuteSelectedQuery(); return }
  if (event.shiftKey) {
    formMessages.value[selectedCommand.value.id] = "Staging is not available for this instrument."
    return
  }
  if (selectedCommand.value.id === merchantTesterCatalogCommandID) merchantCatalogExplicitRefreshRequired = false
  if (isMutation(selectedCommand.value)) {
    ownerErrors.value[selectedCommand.value.id] = "F5 only refreshes telemetry. Use the explicit operation controls to stage or apply changes."
    return
  }
  runCommand(selectedCommand.value)
}

function warnBeforeUnload(event: BeforeUnloadEvent): void {
  if (!scenarioDraftIsDirty(scenarioComposer.value)) return
  event.preventDefault()
  event.returnValue = ""
}

function updatePageVisibility(): void {
  pageHidden.value = document.hidden
  if (pageHidden.value) disarmScenarioReview()
}

onMounted(() => {
  operationTimer = window.setInterval(checkPendingOperations, 1000)
  window.addEventListener("keydown", onKeyDown)
  window.addEventListener("beforeunload", warnBeforeUnload)
  document.addEventListener("visibilitychange", updatePageVisibility)
  connect()
})
onBeforeUnmount(() => {
  setScenarioBackgroundInert(false)
  setWorkbenchBackgroundInert(false, ".scenario-replacement-modal")
  setWorkbenchBackgroundInert(false, ".command-palette-modal")
  connectionGeneration += 1
  window.clearTimeout(reconnectTimer)
  window.clearTimeout(connectionTimer)
  window.clearInterval(operationTimer)
  stopHistoryResize?.()
  window.removeEventListener("keydown", onKeyDown)
  window.removeEventListener("beforeunload", warnBeforeUnload)
  document.removeEventListener("visibilitychange", updatePageVisibility)
  socket.value?.close()
})
</script>

<template>
  <main class="workbench-shell" :class="`history-dock-${historyDock}`">
    <header class="masthead">
      <div>
        <p class="eyebrow">ULTIMA V REDUX :: CONTROL PLANE</p>
        <h1>Britannian Switchboard</h1>
        <p class="subtitle">Live telemetry / codex reference / state inspection / operations</p>
      </div>
		<div class="masthead-quick-controls">
		  <button ref="commandPaletteLauncher" type="button" class="command-palette-launcher" data-command-palette-launcher aria-haspopup="dialog" @click="openCommandPalette">GO TO… <kbd>CTRL/⌘K</kbd></button>
		  <section v-if="scenarioTargetActive" class="active-scenario-target" :data-state="activeScenarioSemanticState" :aria-label="`Active scenario ${scenarioComposer.definition.name}, ${activeScenarioSemanticState}`" data-active-scenario>
			<span>{{ scenarioRecording ? 'RECORDING TO SCENARIO' : 'ACTIVE SCENARIO' }}</span>
			<strong>{{ scenarioComposer.definition.name }}</strong>
			<small>{{ scenarioSourceLabel(scenarioComposer.source) }} · {{ scenarioComposer.definition.steps.length }} STEPS · {{ activeScenarioStatus }}</small>
			<div><button type="button" data-active-scenario-edit @click="openActiveScenarioEditor()">EDIT</button><button type="button" data-active-scenario-change :disabled="scenarioDefinitionWritePending" @click="openScenarioTargetChooser('select')">CHANGE</button></div>
		  </section>
		  <button v-else type="button" class="scenario-target-empty" data-active-scenario-select @click="openScenarioTargetChooser('select')">NO ACTIVE SCENARIO<br /><small>SELECT TARGET</small></button>
		  <button
			v-if="mastheadActiveCombat?.active"
			type="button"
			class="combat-exit-quick"
			data-quick-combat-exit
			:disabled="quickCombatExitDisabled"
			title="Discard the active debug encounter and return to its pre-combat location"
			@click="quickExitCombat"
		  >ESCAPE COMBAT</button>
		  <button
			type="button"
			class="free-move-quick"
			role="switch"
			:aria-checked="freeMoveEnabled"
			:disabled="!isReady || commandPending(movementFreeMoveSetCommandID)"
			@click="runSpecializedMutation(movementFreeMoveSetCommandID, { enabled: String(!freeMoveEnabled) })"
		  >FREE MOVE {{ freeMoveEnabled ? 'ON' : 'OFF' }}</button>
		</div>
      <div class="telemetry" aria-live="polite">
        <div v-if="mastheadCalendar" class="masthead-calendar" aria-label="Current Britannian calendar">
          <MoonPhaseGlyph :phase-index="mastheadCalendar.moonPhaseIndex" :label="mastheadCalendar.moonPhase" :size="20" />
          <span><strong>{{ mastheadCalendar.date }}</strong><small>{{ mastheadCalendar.time }} · {{ mastheadCalendar.moonPhase }}</small></span>
        </div>
        <span class="connection" :class="connectionState"><i />{{ connectionLabel }}</span>
        <span>SCENE {{ sceneGeneration || "--" }}</span>
        <small v-if="lastCloseReason">{{ lastCloseReason }} · {{ retryDetail }}</small>
        <button v-if="connectionState === 'disconnected'" type="button" @click="retryNow">RETRY NOW</button>
      </div>
    </header>
    <nav class="workspace-tabs" aria-label="Workbench workspace">
      <button
        v-for="workspace in workspaces"
        :key="workspace.id"
        type="button"
        :data-workspace="workspace.id"
        :class="{ selected: activeWorkspace === workspace.id }"
        :aria-current="activeWorkspace === workspace.id ? 'page' : undefined"
        @click="selectWorkspace(workspace.id)"
      >
        <strong>{{ workspace.label }}</strong><small>{{ workspace.detail }}</small>
      </button>
    </nav>
    <p class="sr-only" aria-live="polite">{{ liveStatus }}</p>

    <section class="switchboard" :class="{ 'scenarios-workspace': activeWorkspace === 'home' || activeWorkspace === 'scenarios' || activeWorkspace === 'saves' }" aria-label="Command switchboard">
      <WorkbenchHome
        v-if="activeWorkspace === 'home'"
        :recents="homeRecentSurfaces"
        :run-status="latestScenarioRunStatusResult?.data ? scenarioRunStatusData(latestScenarioRunStatusResult.data) : undefined"
        :connected="isReady"
        @open-palette="openCommandPalette"
         @open-surface="navigateToSurface"
         @open-scenarios="openScenarioFromHome"
         @review-again="reviewHistoricalRun"
       />
      <aside v-if="activeWorkspace !== 'home' && activeWorkspace !== 'scenarios' && activeWorkspace !== 'saves'" class="command-rail" aria-labelledby="command-title">
        <div class="section-heading">
          <span>01</span>
          <h2 id="command-title">{{ activeWorkspace }}</h2>
          <small>{{ filteredCommands.length }}/{{ navigationCommandTotal }}</small>
        </div>
        <div class="command-search">
          <label class="sr-only" for="command-search">Search commands</label>
          <input id="command-search" ref="commandSearchInput" v-model="commandSearch" type="search" placeholder="SEARCH COMMANDS · CTRL+K" />
        </div>
        <nav class="command-catalog" aria-label="Query command catalog">
          <section v-for="category in commandCategories" :key="category.name">
            <h3>
              <button type="button" class="category-toggle" :aria-expanded="categoryExpanded(category.name)" @click="toggleCategory(category.name)">
                <span>{{ categoryExpanded(category.name) ? '−' : '+' }} {{ category.name }}</span><small>{{ category.commands.length }}</small>
              </button>
            </h3>
            <div v-show="categoryExpanded(category.name)">
              <button
                v-for="command in category.commands"
                :key="command.id"
                type="button"
                class="command-option"
                :data-command-select="command.id"
                :class="{ selected: command.id === selectedCommandID }"
                :aria-current="command.id === selectedCommandID ? 'true' : undefined"
                :tabindex="command.id === catalogTabCommandID ? 0 : -1"
                @click="selectCommand(command)"
                @keydown.enter.prevent="activateCommand(command)"
              >
                <strong>{{ workspaceCommandName(command) }}</strong>
                <small>{{ commandSearch.trim() ? `${workspaceLabel(primaryWorkspaceForCommand(command))} · ${command.id}` : command.id }}</small>
              </button>
            </div>
          </section>
          <p v-if="availableCommands.length > 0 && filteredCommands.length === 0" class="empty-state">NO MATCHING COMMANDS</p>
        </nav>
        <p v-if="availableCommands.length === 0" class="empty-state">
          {{ isReady ? "CATALOG REQUESTED" : "WAITING FOR CONTROL PLANE" }}
        </p>
      </aside>

      <section v-if="activeWorkspace !== 'home'" class="instrument-bank" :class="{ 'is-codex-long': activeWorkspace === 'codex' && selectedCommand ? longCodexCommandIDs.has(selectedCommand.id) : false, 'is-codex-flow': activeWorkspace === 'codex' && selectedCommand ? flowCodexCommandIDs.has(selectedCommand.id) : false }" aria-labelledby="instrument-title">
        <div v-if="activeWorkspace !== 'scenarios' && activeWorkspace !== 'saves'" class="section-heading">
          <span>02</span>
          <h2 id="instrument-title">Active Instrument</h2>
          <small>{{ selectedCommand?.category ?? "STANDBY" }}</small>
        </div>

        <article
          v-if="selectedCommand"
          class="instrument"
          :class="instrumentWidthClass(selectedCommand)"
        >
          <div v-if="activeWorkspace !== 'scenarios' && activeWorkspace !== 'saves'" class="instrument-index">{{ selectedCommand.id }} / {{ instrumentExecutionModes(selectedCommand) }}</div>
          <div v-if="activeWorkspace !== 'scenarios' && activeWorkspace !== 'saves'" class="instrument-title-row">
            <h3>{{ workspaceCommandName(selectedCommand) }}</h3>
            <span class="execution-mode" :class="{ mutation: instrumentOwnsMutation(selectedCommand) }">
              {{ instrumentModeLabel(selectedCommand) }}
            </span>
          </div>
          <p v-if="activeWorkspace !== 'saves'">{{ workspaceCommandDescription(selectedCommand) }}</p>
          <code v-if="activeWorkspace !== 'saves'">{{ selectedCommand.aliases?.join(" · ") }}</code>
          <section v-if="ownerErrors[selectedCommand.id]" class="operation-banner failure" role="alert" data-owner-error>{{ ownerErrors[selectedCommand.id] }}</section>
          <section v-else-if="latestOperationResult" class="operation-banner" :class="latestOperationResult.success ? 'success' : 'failure'" aria-live="polite">
            <header>
              <strong>LAST OPERATION</strong>
              <span>{{ commandName(latestOperationResult.commandId) }} · {{ formatTime(latestOperationResult.receivedAt) }}</span>
            </header>
            <p>{{ latestOperationResult.success ? latestOperationResult.summary : `${latestOperationResult.errorCode} :: ${latestOperationResult.errorMessage}` }}</p>
            <ul v-if="latestOperationResult.changes?.length">
              <li v-for="change in latestOperationResult.changes.slice(0, 3)" :key="change.path">
                <code>{{ change.path }}</code><span>{{ formatChangeValue(change.before) }} → {{ formatChangeValue(change.after) }}</span>
              </li>
            </ul>
            <details v-if="activeWorkspace === 'tools' && latestOperationResult.commandId === selectedCommand.id && latestOperationResult.data">
              <summary>Structured result</summary>
              <pre>{{ JSON.stringify(latestOperationResult.data, null, 2) }}</pre>
            </details>
          </section>
          <section v-if="selectedQueryFailure" class="operation-banner failure" data-query-load-failure role="alert">
            <header><strong>REFERENCE LOAD FAILED</strong><span>{{ commandName(selectedQueryFailure.commandId) }} · {{ selectedQueryFailure.commandId }}</span></header>
            <p>{{ selectedQueryFailure.errorCode }} :: {{ selectedQueryFailure.errorMessage }}</p>
          </section>
          <PartyCharacterEditor
            v-if="selectedCommand.id === partyCharacterCommandID"
            :command="selectedCommand"
            :draft="draftFor(selectedCommand)"
            :dirty-fields="dirtyFieldsFor(selectedCommand)"
            :option-sets="optionSets"
            :errors="formErrors[selectedCommand.id] ?? {}"
            :message="formMessages[selectedCommand.id]"
            :unavailable-reason="commandUnavailableReason(selectedCommand)"
            :pending-label="commandPending(selectedCommand.id) ? commandPendingLabel(selectedCommand.id) : undefined"
            :apply-disabled="commandApplyDisabled(selectedCommand)"
            @update="updateSelectedPartyCharacterDraft"
            @submit="runCommand(selectedCommand)"
            @reset="resetCommand(selectedCommand)"
          />
          <EquipmentLoadouts
            v-else-if="selectedCommand.id === partyEquipmentLoadoutCommandID"
            :command="selectedCommand"
            :draft="draftFor(selectedCommand)"
            :option-sets="optionSets"
            :roster="latestPartyCharacterInspectResult?.data as PartyCharacterInspectData | undefined"
            :locker="latestInventoryLockerResult?.data ? inventoryLockerStatusData(latestInventoryLockerResult.data) : undefined"
            :pending="commandPending(partyEquipmentLoadoutCommandID)"
            :success-token="latestPartyEquipmentLoadoutResult?.success ? latestPartyEquipmentLoadoutResult.requestId : undefined"
            :unavailable-reason="commandUnavailableReason(selectedCommand)"
            @update-character="updateEquipmentLoadoutCharacter"
            @submit="(argumentsByID) => runSpecializedMutation(partyEquipmentLoadoutCommandID, argumentsByID)"
          />
          <PartyRoster
            v-else-if="selectedCommand.id === partyCharacterInspectCommandID"
            :data="latestPartyCharacterInspectResult?.data as PartyCharacterInspectData | undefined"
            :pending-commands="[...partyRosterMutationCommandIDs, partyKarmaSetCommandID].filter(commandPending)"
            :karma-pending="commandPending(partyKarmaSetCommandID)"
            @mutate="(commandID, argumentsByID) => runSpecializedMutation(commandID, argumentsByID)"
          />
          <SaveLibrary
            v-else-if="selectedCommand.id === saveSnapshotListCommandID"
            :data="saveList"
            :detail="latestSaveDetailResult?.data ? saveSnapshotDetailData(latestSaveDetailResult.data) : undefined"
            :pending-list="commandPending(saveSnapshotListCommandID)"
            :pending-detail="commandPending(saveSnapshotGetCommandID)"
            :pending-mutation="commandPending(saveSnapshotCaptureCommandID) || commandPending(saveSnapshotLoadCommandID) || commandPending(saveSnapshotDeleteCommandID)"
            :current-scene="latestScenarioStatusResult?.data ? {
              locationName: (latestScenarioStatusResult.data as ScenarioStatusData).locationName,
              floor: (latestScenarioStatusResult.data as ScenarioStatusData).floor,
              date: (latestScenarioStatusResult.data as ScenarioStatusData).date,
              time: (latestScenarioStatusResult.data as ScenarioStatusData).time,
              partyMembers: (latestScenarioStatusResult.data as ScenarioStatusData).party.length,
            } : undefined"
            :active-scenario="scenarioTargetActive ? scenarioComposer : undefined"
            :active-scenario-dirty="activeScenarioDirty"
            :load-review-target="saveLoadReviewTarget"
            @review-target-consumed="saveLoadReviewTarget = undefined"
            @refresh="runQueryByID(saveSnapshotListCommandID, false)"
            @inspect="inspectSaveSnapshot"
            @capture="(value) => runSpecializedMutation(saveSnapshotCaptureCommandID, value)"
            @load="(value) => runSpecializedMutation(saveSnapshotLoadCommandID, value)"
            @delete="(value) => runSpecializedMutation(saveSnapshotDeleteCommandID, value)"
            @use-baseline="useSaveAsScenarioBaseline"
          />
          <ScenarioLab
            v-else-if="selectedCommand.id === scenarioStatusCommandID"
            :data="latestScenarioStatusResult?.data as ScenarioStatusData | undefined"
            :updated-at="latestScenarioStatusResult?.receivedAt"
            :pending-status="commandPending(scenarioStatusCommandID)"
            :pending-direct="commandPending(scenarioStarvationTriggerCommandID)"
            :pending-hourly="commandPending(scenarioStarvationHourlyCommandID)"
            :pending-guards="commandPending(scenarioGuardsActivateCommandID)"
            :pending-waterfall="commandPending(scenarioWaterfallSetupCommandID)"
            :pending-bridge-troll="commandPending(scenarioBridgeTrollCommandID)"
            :pending-word="commandPending(scenarioWordSetupCommandID)"
            :pending-blackthorn-interrogation="commandPending(scenarioBlackthornInterrogationCommandID)"
            :pending-lord-british-finale="commandPending(scenarioLordBritishFinaleCommandID)"
            :pending-hole-up="commandPending(commandTestHoleUpCommandID)"
            :pending-kill-party="commandPending(deathKillPartyCommandID)"
            :pending-death-sequence="commandPending(deathTestSequenceCommandID)"
            :pending-definition-list="commandPending(scenarioDefinitionListCommandID)"
            :pending-definition-get="commandPending(scenarioDefinitionGetCommandID)"
            :pending-definition-save="commandPending(scenarioDefinitionValidateCommandID) || commandPending(scenarioDefinitionCreateCommandID) || commandPending(scenarioDefinitionUpdateCommandID) || commandPending(scenarioDefinitionDeleteCommandID) || commandPending(scenarioDefinitionDuplicateCommandID)"
            :pending-run-start="commandPending(scenarioRunStartCommandID)"
            :pending-run-cancel="commandPending(scenarioRunCancelCommandID)"
            :historical-run="scenarioHistoricalRun"
            :historical-run-issue="historicalRunIssue"
            :reviewed-baseline-revision="scenarioReviewedBaselineRevision"
            :reviewed-scene-generation="scenarioReviewedSceneGeneration"
            :word-choices="scenarioWordChoices"
            :composer="scenarioComposer"
            :definition-list="latestScenarioDefinitionListResult?.data ? scenarioDefinitionListData(latestScenarioDefinitionListResult.data) : undefined"
            :run-status="latestScenarioRunStatusResult?.data ? scenarioRunStatusData(latestScenarioRunStatusResult.data) : undefined"
            :selected-run-id="selectedScenarioRunID"
            :validation-issues="scenarioValidationIssues"
            :active-task="scenarioTask"
            :commands="commands"
            :option-sets="optionSets"
            :snapshots="saveList?.snapshots ?? []"
            @refresh="runQueryByID(scenarioStatusCommandID, false)"
            @run="(operation) => runSpecializedMutation(operation.commandID, operation.argumentsByID)"
            @update-composer="updateScenarioComposer"
            @create-definition="createScenarioDefinition"
            @list-definitions="runQueryByID(scenarioDefinitionListCommandID, true)"
            @get-definition="getScenarioDefinition"
            @play-definition="playScenarioDefinition"
            @save-definition="saveScenarioDefinition"
            @delete-definition="deleteScenarioDefinition"
            @duplicate-definition="duplicateScenarioDefinition"
            @review-run="reviewScenarioRun"
            @start-run="startScenarioRun"
            @review-again="reviewHistoricalRun"
            @cancel-remaining="cancelRemainingScenarioSteps"
            @review-rollback="reviewScenarioRollback"
            @select-run="selectedScenarioRunID = $event"
            @update-active-task="updateScenarioTask"
          />
          <LoggingConsole
            v-else-if="selectedCommand.id === loggingStatusCommandID"
            :data="latestLoggingResult?.data ? loggingStatusData(latestLoggingResult.data) : undefined"
            :updated-at="latestLoggingResult?.receivedAt"
            :pending="commandPending(loggingStatusCommandID) || loggingMutationCommandIDs.some(commandPending)"
            @mutate="(mutation) => runSpecializedMutation(mutation.commandID, mutation.argumentsByID)"
            @refresh="runQueryByID(loggingStatusCommandID, false)"
          />
          <WorkbenchWind
            v-else-if="selectedCommand.id === windStatusCommandID"
            :direction="windDirection(latestWindResult)"
            :updated-at="latestWindResult?.receivedAt"
            :now="now"
            :paused="livePollingPaused"
            :pending="commandPending(windSetCommandID)"
            @toggle-pause="livePollingPaused = !livePollingPaused"
            @apply="(direction) => runSpecializedMutation(windSetCommandID, { direction })"
          />
          <WorkbenchCalendar
            v-else-if="selectedCommand.id === calendarStatusCommandID"
            :data="latestCalendarResult?.data ? calendarStatusData(latestCalendarResult.data) : undefined"
            :updated-at="latestCalendarResult?.receivedAt"
            :pending-time="commandPending(calendarSetTimeCommandID)"
            :pending-day="commandPending(calendarSetDayCommandID)"
            @set-time="setCalendarTime"
            @set-day="setCalendarDay"
          />
          <SearchItemStacksInspector
            v-else-if="selectedCommand.id === searchStatusCommandID"
            :data="latestSearchResult?.data ? searchStatusData(latestSearchResult.data) : undefined"
            :updated-at="latestSearchResult?.receivedAt"
            :add-pending="commandPending(searchItemStackAddCommandID)"
            :reset-pending="commandPending(searchProgressResetCommandID)"
            :overlay-pending="commandPending(searchOverlaySetCommandID)"
            :search-pending="commandPending(commandTestSearchCommandID)"
            @add="addSearchItemStack"
            @reset="resetSearchProgress"
            @set-overlay="(enabled) => runSpecializedMutation(searchOverlaySetCommandID, { enabled: String(enabled) })"
            @search="(direction) => runSpecializedMutation(commandTestSearchCommandID, { direction })"
          />
          <QuestInventoryInspector
            v-else-if="selectedCommand.id === questItemStatusCommandID"
            :data="latestQuestItemResult?.data ? questItemStatusData(latestQuestItemResult.data) : undefined"
            :updated-at="latestQuestItemResult?.receivedAt"
            :pending="commandPending(questItemEnsureCommandID) || commandPending(questItemRemoveCommandID) || commandPending(questShardUseCommandID)"
            @ensure="(item) => runSpecializedMutation(questItemEnsureCommandID, { item })"
            @remove="(item) => runSpecializedMutation(questItemRemoveCommandID, { item })"
			@use-shard="(argumentsByID) => runSpecializedMutation(questShardUseCommandID, argumentsByID)"
          />
          <WorldNavigator
            v-else-if="selectedCommand.id === worldPositionStatusCommandID"
            :key="`${sessionID}:${sceneGeneration}`"
            :data="lastSuccessfulWorldPositionResult?.data ? worldPositionStatusData(lastSuccessfulWorldPositionResult.data) : undefined"
            :updated-at="lastSuccessfulWorldPositionResult?.receivedAt"
            :pending="commandPending(worldPositionSetCommandID)"
            :refreshing="commandPending(worldPositionStatusCommandID)"
            :stale="Boolean(lastSuccessfulWorldPositionResult && latestWorldPositionResult?.success === false)"
            :landmarks="worldLandmarkChoices"
            :pending-landmark="commandPending(movementLandmarkTravelCommandID)"
            :initial-coordinates="linkedWorldCoordinates"
            @refresh="runQueryByID(worldPositionStatusCommandID, false)"
            @apply="applyWorldPosition"
            @travel-landmark="(landmark) => runSpecializedMutation(movementLandmarkTravelCommandID, { landmark })"
          />
          <DungeonExplorer
            v-else-if="selectedCommand.id === dungeonExplorerStatusCommandID"
            :data="latestDungeonExplorerResult?.data ? dungeonExplorerStatusData(latestDungeonExplorerResult.data) : undefined"
            :updated-at="latestDungeonExplorerResult?.receivedAt"
            :pending="commandPending(dungeonExplorerStatusCommandID) || commandPending(dungeonExplorerSetupCommandID)"
            :dungeons="dungeonChoices"
            :facings="dungeonFacingChoices"
            @apply="applyDungeonSetup"
          />
          <MoongateAtlas
            v-else-if="selectedCommand.id === moongateStatusCommandID"
            :data="latestMoongateResult?.data ? moongateStatusData(latestMoongateResult.data) : undefined"
            :updated-at="latestMoongateResult?.receivedAt"
            :overlay-pending="commandPending(moongateOverlaySetCommandID)"
            :travel-pending="commandPending(moongateTravelCommandID)"
            @navigate="openWorldNavigator"
            @set-overlay="(enabled) => runSpecializedMutation(moongateOverlaySetCommandID, { enabled: String(enabled) })"
            @travel="(value) => runSpecializedMutation(moongateTravelCommandID, value)"
          />
          <template v-else-if="selectedCommand.id === shrineStatusCommandID">
            <ShrineReference
              v-show="activeWorkspace === 'codex'"
              :data="latestShrineStatusResult?.data ? shrineStatusData(latestShrineStatusResult.data) : undefined"
              :selected-index="selectedShrineIndex"
              @select="selectShrine"
              @inspect="openShrineWorkspace('game')"
            />
            <ShrineProgressInspector
              v-show="activeWorkspace === 'game'"
              :data="latestShrineStatusResult?.data ? shrineStatusData(latestShrineStatusResult.data) : undefined"
              :selected-index="selectedShrineIndex"
              :updated-at="latestShrineStatusResult?.receivedAt"
              :pending="commandPending(shrineProgressCommandID)"
              :preset-pending="commandPending(codexQuestStateSetAllCommandID)"
              @select="selectShrine"
              @apply="(argumentsByID) => runSpecializedMutation(shrineProgressCommandID, argumentsByID)"
              @preset="(argumentsByID) => runSpecializedMutation(codexQuestStateSetAllCommandID, argumentsByID)"
              @reference="openShrineWorkspace('codex')"
            />
          </template>
          <CombatMapCatalog
            v-else-if="selectedCommand.id === 'combat.map-catalog.all'"
            :data="latestCombatMapResult?.data ? combatMapData(latestCombatMapResult.data) : undefined"
            :section="draftFor(selectedCommand).section ?? 'all'"
            :pending="commandPending(selectedCommand.id)"
            @section="updateSelectedCombatSection"
          />
          <TileAtlas
            v-else-if="selectedCommand.id === 'codex.tile.catalog'"
            :data="latestResults[selectedCommand.id]?.data as TileAtlasEntry[] | undefined"
          />
          <NPCScheduleIndex
            v-else-if="selectedCommand.id === 'codex.npc.catalog'"
            :data="latestResults[selectedCommand.id]?.data as NPCScheduleEntry[] | undefined"
            @conversation="openNPCConversation"
          />
          <SearchLocationAtlas
            v-else-if="selectedCommand.id === 'codex.search-location.catalog'"
            :data="latestResults[selectedCommand.id]?.data as SearchLocationAtlasEntry[] | undefined"
          />
          <ConsumablesCodex
            v-else-if="selectedCommand.id === 'codex.consumable.catalog'"
            :data="latestResults[selectedCommand.id]?.data as ConsumableCodexEntry[] | undefined"
          />
          <SpellGrimoire
            v-else-if="selectedCommand.id === 'codex.spell.catalog'"
            :data="latestResults[selectedCommand.id]?.data as SpellGrimoireEntry[] | undefined"
          />
          <CodexCatalog
            v-else-if="codexCommandIDs.includes(selectedCommand.id)"
            :title="selectedCommand.name"
            :subtitle="selectedCommand.description"
            :data="latestResults[selectedCommand.id]?.data as Record<string, unknown>[] | undefined"
            :sticky-header="longCodexCommandIDs.has(selectedCommand.id)"
          />
          <ConversationScriptExplorer
            v-else-if="selectedCommand.id === conversationCatalogCommandID"
            :catalog="latestConversationCatalogResult?.data
              && conversationCatalogData(latestConversationCatalogResult.data).selectedSource === conversationSource
              ? conversationCatalogData(latestConversationCatalogResult.data) : undefined"
            :detail="latestConversationDetailResult?.data
              && conversationDetailData(latestConversationDetailResult.data).sourceId === conversationSource
              && conversationDetailData(latestConversationDetailResult.data).index === selectedConversationIndex
              ? conversationDetailData(latestConversationDetailResult.data) : undefined"
            :source="conversationSource"
            :selected-index="selectedConversationIndex"
            :catalog-pending="commandPending(conversationCatalogCommandID)"
            :detail-pending="commandPending(conversationInspectCommandID)"
            @source="selectConversationSource"
            @inspect="inspectConversationScript"
          />
          <CombatEncounterComposer
            v-else-if="selectedCommand.id === combatEncounterOptionsCommandID"
            :options="latestCombatEncounterOptionsResult?.data ? combatEncounterOptionsData(latestCombatEncounterOptionsResult.data) : undefined"
            :pending="commandPending(combatEncounterStartCommandID)"
            @start="(argumentsByID) => runSpecializedMutation(combatEncounterStartCommandID, argumentsByID)"
          />
          <MerchantTester
            v-else-if="selectedCommand.id === merchantTesterCatalogCommandID"
            :catalog="latestMerchantTesterCatalogResult?.data ? merchantTesterCatalogData(latestMerchantTesterCatalogResult.data) : undefined"
            :catalog-scene-generation="latestMerchantTesterCatalogResult?.sceneGeneration"
            :commands="commands"
            :scene-generation="sceneGeneration"
            :pending-preview="commandPending(merchantTesterPreviewCommandID)"
            :pending-live="commandPending(merchantTesterLiveCommandID)"
            @preview="(argumentsByID) => runSpecializedMutation(merchantTesterPreviewCommandID, argumentsByID)"
            @live="(argumentsByID) => runSpecializedMutation(merchantTesterLiveCommandID, argumentsByID)"
          />
          <VFXTestBench
            v-else-if="selectedCommand.id === vfxStatusCommandID"
            :status="latestVFXStatusResult?.data ? vfxStatusData(latestVFXStatusResult.data) : undefined"
            :pending-kapow="commandPending(vfxKapowCommandID)"
            :pending-projectile="commandPending(vfxProjectileCommandID)"
            :pending-earthquake="commandPending(environmentEarthquakeCommandID)"
            :pending-lighting-overlay="commandPending(lightingOverlaySetCommandID)"
            @kapow="(argumentsByID) => runSpecializedMutation(vfxKapowCommandID, argumentsByID)"
            @projectile="(argumentsByID) => runSpecializedMutation(vfxProjectileCommandID, argumentsByID)"
            @earthquake="(argumentsByID) => runSpecializedMutation(environmentEarthquakeCommandID, argumentsByID)"
            @set-lighting-overlay="(enabled) => runSpecializedMutation(lightingOverlaySetCommandID, { enabled: String(enabled) })"
          />
          <MonsterEcosystemConsole
            v-else-if="selectedCommand.id === monsterEcosystemStatusCommandID"
            :data="latestMonsterEcosystemResult?.data ? monsterEcosystemData(latestMonsterEcosystemResult.data) : undefined"
            :pending-generation="commandPending(monsterGenerationSetCommandID)"
            :pending-clear="commandPending(monsterClearMapEnemiesCommandID)"
            :pending-spawn="commandPending(monsterSpawnWhirlpoolCommandID) || commandPending(monsterSpawnPirateCommandID)"
            @set-generation="setMonsterGeneration"
            @clear="(value) => runSpecializedMutation(monsterClearMapEnemiesCommandID, {
              expectedLocationId: String(value.expectedLocationId),
              expectedFloor: String(value.expectedFloor),
              expectedEnemyCount: String(value.expectedEnemyCount),
            })"
            @spawn-whirlpool="runSpecializedMutation(monsterSpawnWhirlpoolCommandID, {})"
            @spawn-pirate="runSpecializedMutation(monsterSpawnPirateCommandID, {})"
          />
		  <DebugOptionsConsole
			v-else-if="selectedCommand.id === debugOptionsStatusCommandID"
			:data="latestDebugOptionsResult?.data ? debugOptionsStatusData(latestDebugOptionsResult.data) : undefined"
			:updated-at="latestDebugOptionsResult?.receivedAt"
			:pending="debugOptionsMutationCommandIDs.some(commandPending)"
			@set-option="(commandID, enabled) => runSpecializedMutation(commandID, { enabled: String(enabled) })"
		  />
		  <DisplayConsole
			v-else-if="selectedCommand.id === displayStatusCommandID"
			:data="latestDisplayResult?.data ? displayStatusData(latestDisplayResult.data) : undefined"
			:modes="displayModeChoices"
			:updated-at="latestDisplayResult?.receivedAt"
			:pending="displayMutationCommandIDs.some(commandPending)"
			@set-mode="(mode) => runSpecializedMutation(displayModeSetCommandID, { mode })"
			@set-resolution="(index) => runSpecializedMutation(displayResolutionSetCommandID, { index: String(index) })"
		  />
		  <ReduxModificationsConsole
			v-else-if="selectedCommand.id === reduxModificationsStatusCommandID"
			:data="latestReduxModificationsResult?.data ? reduxModificationsStatusData(latestReduxModificationsResult.data) : undefined"
			:updated-at="latestReduxModificationsResult?.receivedAt"
			:pending="reduxModificationCommandIDs.some(commandPending)"
			@set-repeated-division="(enabled) => runSpecializedMutation(repeatedMonsterDivisionSetCommandID, { enabled: String(enabled) })"
			@set-persistent-discoveries="(enabled) => runSpecializedMutation(persistentDungeonDiscoveriesSetCommandID, { enabled: String(enabled) })"
			@set-subdued-visibility="(enabled) => runSpecializedMutation(subduedUnlitDungeonVisibilitySetCommandID, { enabled: String(enabled) })"
			@set-smooth-surface-movement="(enabled) => runSpecializedMutation(smoothSurfaceMovementSetCommandID, { enabled: String(enabled) })"
			@set-flexible-sextant-use="(enabled) => runSpecializedMutation(flexibleSextantUseSetCommandID, { enabled: String(enabled) })"
			@refresh="runQueryByID(reduxModificationsStatusCommandID, false)"
		  />
		  <ShipsConsole
            v-else-if="selectedCommand.id === vehicleStatusCommandID"
            :data="latestVehicleStatusResult?.data ? vehicleStatusData(latestVehicleStatusResult.data) : undefined"
            :pending-damage="commandPending(shipDamageRawCommandID)"
            :pending-repair="commandPending(shipRepairFullCommandID)"
			:pending-skiffs="commandPending(shipSkiffSetCommandID)"
			:commands="commands"
			:pending-spawn="commandPending(vehicleSpawnCommandID)"
			:pending-force-board="commandPending(vehicleForceBoardCommandID)"
            @damage="(amount) => runSpecializedMutation(shipDamageRawCommandID, { amount: String(amount) })"
            @repair="runSpecializedMutation(shipRepairFullCommandID, {})"
			@set-skiffs="(quantity) => runSpecializedMutation(shipSkiffSetCommandID, { quantity: String(quantity) })"
			@spawn="(argumentsByID) => runSpecializedMutation(vehicleSpawnCommandID, argumentsByID)"
			@force-board="(type) => runSpecializedMutation(vehicleForceBoardCommandID, { type })"
          />
		  <WorldDiagnostics
			v-else-if="selectedCommand.id === guardStatusCommandID"
			:guards="latestGuardStatusResult?.data ? guardStatusData(latestGuardStatusResult.data) : undefined"
			:shadowlords="latestShadowlordStatusResult?.data ? shadowlordStatusData(latestShadowlordStatusResult.data) : undefined"
			:death="latestDeathStatusResult?.data ? deathStatusData(latestDeathStatusResult.data) : undefined"
			:pending="worldDiagnosticCommandIDs.some(commandPending)"
			@refresh="worldDiagnosticCommandIDs.forEach((commandID) => runQueryByID(commandID, false))"
		  />
          <InventoryLocker
            v-else-if="selectedCommand.id === inventoryLockerStatusCommandID"
            :data="latestInventoryLockerResult?.data ? inventoryLockerStatusData(latestInventoryLockerResult.data) : undefined"
            :updated-at="latestInventoryLockerResult?.receivedAt"
            :commands="commands"
            :pending="inventoryLockerMutationCommandIDs.some(commandPending)"
            @mutate="(mutation) => runSpecializedMutation(mutation.commandID, mutation.argumentsByID)"
          />
          <SpellInventory
            v-else-if="selectedCommand.id === spellInventoryStatusCommandID"
            :data="latestSpellInventoryResult?.data ? spellInventoryStatusData(latestSpellInventoryResult.data) : undefined"
            :updated-at="latestSpellInventoryResult?.receivedAt"
            :pending="spellInventoryMutationCommandIDs.some(commandPending)"
            @mutate="(mutation) => runSpecializedMutation(mutation.commandID, mutation.argumentsByID)"
          />
          <ActiveCombatConsole
            v-else-if="selectedCommand.id === activeCombatStatusCommandID"
            :data="latestActiveCombatResult?.data ? activeCombatStatusData(latestActiveCombatResult.data) : undefined"
            :updated-at="latestActiveCombatResult?.receivedAt"
            :pending="activeCombatMutationCommandIDs.some(commandPending)"
            @mutate="(mutation) => runSpecializedMutation(mutation.commandID, mutation.argumentsByID)"
            @open-encounter="openCombatEncounterComposer"
          />
          <ObservationPane
            v-else-if="selectedCommand.id === observationCommandID"
            :data="latestObservationResult?.data ? observationResultData(latestObservationResult.data) : undefined"
            :command-id="latestObservationResult?.commandId as ObservationCommandID | undefined"
            :updated-at="latestObservationResult?.receivedAt"
            :pending="observationPending"
            @run="(commandID) => runSpecializedMutation(commandID, {})"
          />
          <form v-else :data-command-form="selectedCommand.id" novalidate @submit.prevent="runCommand(selectedCommand)">
            <template v-for="parameter in selectedCommand.parameters ?? []" :key="parameter.id">
              <div class="parameter-field">
                <label :for="`${selectedCommand.id}-${parameter.id}`">
                  {{ parameter.name }}<span v-if="parameter.required" aria-hidden="true"> *</span>
                </label>
                <small v-if="parameter.description" :id="`${selectedCommand.id}-${parameter.id}-help`">{{ parameter.description }}</small>

                <select
                  v-if="parameter.kind === 'enum' || parameter.kind === 'boolean' || parameter.kind === 'character'"
                  :id="`${selectedCommand.id}-${parameter.id}`"
                  :value="draftFor(selectedCommand)[parameter.id] ?? ''"
                  :required="parameter.required"
                  :aria-describedby="describedBy(selectedCommand, parameter.id)"
                  :aria-invalid="Boolean(errorFor(selectedCommand.id, parameter.id))"
                  @change="updateDraftAndMaybeRun(selectedCommand, parameter.id, inputValue($event))"
                >
                  <option value="">{{ parameter.required ? 'SELECT A VALUE' : 'UNSET' }}</option>
                  <template v-if="parameter.kind === 'boolean'">
                    <option value="true">ON</option>
                    <option value="false">OFF</option>
                  </template>
                  <option v-for="choice in choicesForParameter(parameter, optionSets)" :key="choice.value" :value="choice.value">{{ choice.label }}</option>
                </select>

                <input
                  v-else-if="parameter.kind === 'integer'"
                  :id="`${selectedCommand.id}-${parameter.id}`"
                  type="number"
                  step="1"
                  :min="parameter.minimum"
                  :max="parameter.maximum"
                  :value="draftFor(selectedCommand)[parameter.id] ?? ''"
                  :required="parameter.required"
                  :aria-describedby="describedBy(selectedCommand, parameter.id)"
                  :aria-invalid="Boolean(errorFor(selectedCommand.id, parameter.id))"
                  @input="updateDraft(selectedCommand, parameter.id, inputValue($event))"
                />

                <input
                  v-else-if="parameter.kind === 'string'"
                  :id="`${selectedCommand.id}-${parameter.id}`"
                  type="text"
                  :pattern="parameter.pattern"
                  :value="draftFor(selectedCommand)[parameter.id] ?? ''"
                  :required="parameter.required"
                  :aria-describedby="describedBy(selectedCommand, parameter.id)"
                  :aria-invalid="Boolean(errorFor(selectedCommand.id, parameter.id))"
                  @input="updateDraft(selectedCommand, parameter.id, inputValue($event))"
                />

                <p v-else class="schema-error">UNSUPPORTED CONTROL: {{ parameter.kind }}</p>
                <p v-if="errorFor(selectedCommand.id, parameter.id)" :id="`${selectedCommand.id}-${parameter.id}-error`" class="field-error">
                  {{ errorFor(selectedCommand.id, parameter.id) }}
                </p>
              </div>
            </template>

            <p v-if="formMessages[selectedCommand.id]" class="form-error" aria-live="polite">{{ formMessages[selectedCommand.id] }}</p>
            <p v-if="commandUnavailableReason(selectedCommand)" class="disabled-reason" role="status">
              {{ commandUnavailableReason(selectedCommand) }}
            </p>
            <div v-if="activeWorkspace === 'tools' || isMutation(selectedCommand)" class="instrument-actions" :class="{ 'has-reset': Boolean(selectedCommand.parameters?.length) }">
              <button
                type="submit"
                :data-command-id="selectedCommand.id"
                :disabled="commandApplyDisabled(selectedCommand)"
              >
                {{ commandPending(selectedCommand.id) ? commandPendingLabel(selectedCommand.id) : isMutation(selectedCommand) ? "APPLY CHANGES" : "RUN QUERY" }}
              </button>
              <button v-if="selectedCommand.parameters?.length" type="button" class="reset-button" :disabled="commandPending(selectedCommand.id)" @click="resetCommand(selectedCommand)">
                RESET
              </button>
            </div>
          </form>
          <p v-if="!isReady" class="disabled-reason">EXECUTION DISABLED UNTIL CONNECTION IS READY</p>
        </article>
        <p v-else class="empty-state">SELECT AN INSTRUMENT</p>
      </section>
    </section>

    <section
      ref="historyPanel"
      class="history-panel"
      :class="{ collapsed: historyCollapsed }"
      :style="historyPanelStyle"
      :data-state="historyCollapsed ? 'collapsed' : 'expanded'"
      :data-dock="historyDock"
      aria-labelledby="history-title"
    >
      <div v-show="!historyCollapsed" class="history-resize-handle" role="separator" aria-label="Resize global history" :aria-orientation="historyDock === 'right' ? 'vertical' : 'horizontal'" @pointerdown="startHistoryResize" />
      <div class="section-heading result-heading">
        <span>03</span>
        <h2 id="history-title">Global History</h2>
        <div class="stream-actions">
          <button type="button" :aria-expanded="!historyCollapsed" :data-state="historyCollapsed ? 'collapsed' : 'expanded'" data-history-action="collapse" @click="toggleHistory">{{ historyCollapsed ? 'EXPAND HISTORY' : 'COLLAPSE HISTORY' }}</button>
          <button type="button" :data-state="historyDock" data-history-action="dock" @click="toggleHistoryDock">DOCK {{ historyDock === 'bottom' ? 'RIGHT' : 'BOTTOM' }}</button>
          <button type="button" role="switch" :aria-checked="scenarioRecording" :aria-label="scenarioTargetActive ? `Record commands to ${scenarioComposer.definition.name}: ${scenarioRecording ? 'on' : 'off'}` : 'Record commands: unavailable until a scenario is selected'" :data-state="scenarioDefinitionWritePending ? 'pending' : scenarioRecording ? 'recording' : scenarioTargetActive ? 'ready' : 'unavailable'" data-history-action="record" @click="toggleScenarioRecording">{{ scenarioDefinitionWritePending ? `RECORD QUEUED ${deferredScenarioCaptures.length}` : scenarioRecording ? 'RECORDING ON' : scenarioTargetActive ? 'RECORD READY' : 'RECORD UNAVAILABLE' }}</button>
          <button type="button" role="switch" :aria-checked="followResults" :data-state="followResults ? 'active' : 'off'" data-history-action="follow" @click="toggleFollow">FOLLOW {{ followResults ? "ON" : "OFF" }}</button>
          <button type="button" :data-state="results.length ? 'ready' : 'unavailable'" data-history-action="clear" :disabled="results.length === 0" @click="clearResults">CLEAR HISTORY</button>
        </div>
      </div>
      <div v-show="!historyCollapsed" ref="stream" class="result-stream" role="region" aria-label="Global command history" tabindex="0" @scroll="onStreamScroll">
        <p v-if="trimmedResults" class="trimmed">{{ trimmedResults }} OLDER RESULTS TRIMMED</p>
        <article v-for="result in results" :key="result.entryID" class="result-entry" :data-state="result.success ? 'success' : 'failure'" :data-history-request="result.requestId">
          <header>
            <time>{{ formatTime(result.receivedAt) }}</time>
            <strong>{{ commandName(result.commandId) }}</strong>
            <span :class="result.success ? 'success' : 'failure'">{{ result.success ? "SUCCESS" : "FAILED" }}</span>
          </header>
          <p v-if="result.summary" class="summary">{{ result.summary }}</p>
          <p v-if="!result.success" class="error-line">{{ result.errorCode }} :: {{ result.errorMessage }}</p>
          <div class="history-entry-actions">
            <button v-if="resultEligibleForScenario(result)" type="button" :disabled="scenarioCaptureState(result) === 'completed' || scenarioCaptureState(result) === 'pending' || commandPending(scenarioDefinitionGetCommandID)" :data-state="scenarioCaptureState(result)" :data-add-scenario="result.requestId" @click="requestAddResultToScenario(result)">{{ capturedScenarioTargets.has(result.requestId) ? `CAPTURED IN ${capturedScenarioTargets.get(result.requestId)}` : deferredScenarioCaptures.some((capture) => capture.requestId === result.requestId) ? 'CAPTURE QUEUED UNTIL SAVE COMPLETES' : commandPending(scenarioDefinitionGetCommandID) ? 'WAITING FOR SCENARIO LOAD' : scenarioDefinitionWritePending ? 'QUEUE CAPTURE AFTER SAVE' : scenarioTargetActive ? `CAPTURE IN ${scenarioComposer.definition.name}` : 'CHOOSE SCENARIO…' }}</button>
            <span v-else-if="!result.success">ATTACH AS NOTE UNAVAILABLE</span>
          </div>
          <footer>REQ {{ result.requestId }} · SCENE {{ result.sceneGeneration }}</footer>
        </article>
        <div v-if="results.length === 0" class="empty-history">NO COMMAND HISTORY</div>
      </div>
    </section>

    <section v-if="scenarioCaptureDialogOpen" class="scenario-target-modal" role="dialog" aria-modal="true" aria-labelledby="scenario-target-title" data-scenario-target-dialog>
      <div ref="scenarioTargetDialog" class="scenario-target-dialog">
        <header><div><span>GLOBAL HISTORY CAPTURE</span><h2 id="scenario-target-title">{{ scenarioCaptureDialogMode === 'new' ? 'Create Scenario Draft' : scenarioCaptureDialogMode === 'dirty' ? 'Unsaved Scenario Draft' : 'Choose Scenario Target' }}</h2></div><button type="button" :disabled="Boolean(scenarioCaptureLoadingReference)" aria-label="Close scenario target dialog" @click="closeScenarioTargetChooser">CLOSE</button></header>
        <p v-if="scenarioCaptureError" class="error-line" role="alert">{{ scenarioCaptureError }}</p>

        <template v-if="scenarioCaptureDialogMode === 'dirty'">
          <p><strong>{{ scenarioComposer.definition.name }}</strong> has unsaved changes. Replacing it would discard those changes.</p>
          <div class="scenario-target-dialog-actions"><button type="button" @click="cancelScenarioReplacement">CANCEL</button><button type="button" data-scenario-target-return @click="returnToActiveScenarioEditor">RETURN TO EDITOR</button><button type="button" class="danger-button" data-scenario-target-discard @click="discardScenarioDraftAndReplace">DISCARD &amp; CONTINUE</button></div>
        </template>

        <template v-else-if="scenarioCaptureDialogMode === 'new'">
          <label>SCENARIO NAME<input v-model="scenarioCaptureNewName" data-scenario-target-name autofocus placeholder="Waterfall Regression" /></label>
          <dl><div><dt>Destination</dt><dd>Personal / Local</dd></div><div><dt>Reference</dt><dd><code>{{ scenarioCaptureNewID ? `personal:${scenarioCaptureNewID}` : 'Enter a name' }}</code></dd></div></dl>
          <p v-if="scenarioCaptureNewIDExists" class="error-line">A Personal scenario already uses this reference.</p>
          <p>Nothing is saved until you review and save the draft in Scenario Editor.</p>
          <div class="scenario-target-dialog-actions"><button type="button" @click="scenarioCaptureDialogMode = 'choose'">BACK</button><button type="button" data-scenario-target-create :disabled="!scenarioCaptureNewName.trim() || !scenarioCaptureNewID || scenarioCaptureNewIDExists" @click="createScenarioCaptureTarget">CREATE DRAFT &amp; ACTIVATE</button></div>
        </template>

        <template v-else>
          <button v-if="scenarioTargetActive" type="button" class="scenario-target-choice current" data-scenario-target-current @click="finishScenarioTargetSelection"><span>CURRENT ACTIVE</span><strong>{{ scenarioComposer.definition.name }}</strong><small>{{ activeScenarioLabel }} · {{ scenarioComposer.definition.steps.length }} STEPS{{ activeScenarioDirty ? ' · UNSAVED' : '' }}</small></button>
          <label>FIND EXISTING SCENARIO<input v-model="scenarioCaptureSearch" type="search" data-scenario-target-search placeholder="NAME, ID, REFERENCE, OR DESCRIPTION" /></label>
          <section v-for="source in (['personal', 'project'] as ScenarioSource[])" :key="source" class="scenario-target-group">
            <header><strong>{{ scenarioSourceLabel(source) }}</strong><small v-if="source === 'project'">SAVING REQUIRES SHARED-PROJECT REVIEW</small></header>
            <button v-for="definition in scenarioTargetChoices(source)" :key="definition.reference" type="button" class="scenario-target-choice" :class="{ selected: scenarioCaptureSelectedReference === definition.reference }" :aria-pressed="scenarioCaptureSelectedReference === definition.reference" :disabled="Boolean(scenarioCaptureLoadingReference)" :data-scenario-target-reference="definition.reference" @click="chooseScenarioTarget(definition.reference)"><strong>{{ definition.name }}</strong><small>{{ definition.reference }} · {{ definition.steps.length }} STEPS</small></button>
            <p v-if="scenarioTargetChoices(source).length === 0" class="empty-state">NO {{ source.toUpperCase() }} SCENARIOS</p>
          </section>
          <section v-if="scenarioCaptureSelectedDefinition && scenarioCaptureResult" class="scenario-target-placement" data-scenario-target-placement>
            <header><strong>ADD {{ commandName(scenarioCaptureResult.commandId) }}</strong><small>{{ scenarioCaptureSelectedDefinition.name }}</small></header>
            <label>INSERT POSITION<select v-model.number="scenarioCaptureInsertionIndex" data-scenario-target-insertion><option v-for="option in scenarioCaptureInsertionOptions" :key="option.index" :value="option.index">{{ option.label }}</option></select></label>
            <p v-if="scenarioCapturePlacementIssue" class="error-line" role="alert">{{ scenarioCapturePlacementIssue }}</p>
            <p>The server-normalized command and arguments will be added to an unsaved draft, then opened for review.</p>
          </section>
          <p v-if="commandPending(scenarioDefinitionListCommandID)" class="summary">REFRESHING SCENARIO LIBRARY…</p>
          <div class="scenario-target-dialog-actions"><button type="button" @click="closeScenarioTargetChooser">CANCEL</button><button type="button" data-scenario-target-new @click="requestNewScenarioTarget">CREATE NEW PERSONAL…</button><button v-if="scenarioCaptureResult" type="button" data-scenario-target-confirm :disabled="!scenarioCaptureSelectedReference || Boolean(scenarioCaptureLoadingReference) || Boolean(scenarioCapturePlacementIssue)" @click="confirmScenarioCaptureTarget">{{ scenarioCaptureLoadingReference ? 'LOADING…' : 'ADD &amp; OPEN EDITOR' }}</button></div>
        </template>
      </div>
    </section>

    <section v-if="scenarioReplacementReview && !scenarioCaptureDialogOpen" ref="scenarioReplacementDialog" class="scenario-target-modal scenario-replacement-modal" role="dialog" aria-modal="true" aria-labelledby="scenario-replacement-title" data-scenario-replacement-dialog>
      <div class="scenario-target-dialog">
        <header><div><span>SCENARIO DRAFT REPLACEMENT</span><h2 id="scenario-replacement-title">Unsaved Scenario Draft</h2></div><button type="button" @click="cancelScenarioReplacement">CLOSE</button></header>
        <p><strong>{{ scenarioComposer.definition.name }}</strong> has unsaved changes. Replacing it would discard those changes.</p>
        <div class="scenario-target-dialog-actions"><button type="button" data-scenario-replacement-cancel @click="cancelScenarioReplacement">KEEP DRAFT</button><button type="button" class="danger-button" data-scenario-replacement-discard @click="discardScenarioDraftAndReplace">DISCARD &amp; CONTINUE</button></div>
      </div>
    </section>

    <CommandPalette v-if="commandPaletteOpen" :surfaces="navigationSurfaces" :recents="navigationRecents" @close="closeCommandPalette" @activate="activateNavigationSurface" />

    <footer class="keyboard-line">CTRL/⌘K GO TO · TAB NAVIGATE · ENTER ACTIVATE · F5 REFRESH · RESULTS RETAINED {{ results.length }}/{{ resultLimit }}</footer>
  </main>
</template>
