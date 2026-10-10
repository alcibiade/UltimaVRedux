package main

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	conversationcatalog "github.com/bradhannah/UltimaVRedux/internal/conversation/catalog"
	"github.com/bradhannah/UltimaVRedux/internal/conversation/semantic"
	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestApplicationOwnsCodexReferencesBeforeGameplayScene(t *testing.T) {
	scene := &scenarioRunnerTestScene{ready: true}
	game := NewGame(scene)
	loadCalls := 0
	game.debugReferencesLoader = func(_ *config.UltimaVConfiguration) (*references.GameReferences, error) {
		loadCalls++
		return &references.GameReferences{InventoryItemReferences: references.NewInventoryItemsReferences()}, nil //nolint:exhaustruct // Quest catalog needs only inventory references.
	}

	result := game.ExecuteLocalDebugCommand(debugging.Request{
		SessionID: "test", ID: "quest", CommandID: debugging.CommandCodexQuestCatalog,
	})

	require.True(t, result.Success, result.ErrorMessage)
	assert.Equal(t, "Quest artifact ledger loaded.", result.Summary)
	assert.Empty(t, scene.executed, "application-owned references must not dispatch through GameScene")
	result = game.ExecuteLocalDebugCommand(debugging.Request{
		SessionID: "test", ID: "quest-again", CommandID: debugging.CommandCodexQuestCatalog,
	})
	require.True(t, result.Success, result.ErrorMessage)
	assert.Equal(t, 1, loadCalls, "immutable references should be cached across queries")
}

func TestApplicationReportsCodexReferenceLoadFailure(t *testing.T) {
	game := NewGame(&scenarioRunnerTestScene{ready: true})
	game.debugReferencesLoader = func(_ *config.UltimaVConfiguration) (*references.GameReferences, error) {
		return nil, errors.New("missing reference data")
	}

	result := game.ExecuteLocalDebugCommand(debugging.Request{
		SessionID: "test", ID: "quest", CommandID: debugging.CommandCodexQuestCatalog,
	})

	assert.False(t, result.Success)
	assert.Equal(t, "codex_reference_load_failed", result.ErrorCode)
	assert.Contains(t, result.ErrorMessage, "missing reference data")
}

func TestCodexEquipmentCatalogUsesStableOriginalEquipmentIDs(t *testing.T) {
	entries := newCodexEquipmentCatalog()

	require.Len(t, entries, 48)
	assert.Equal(t, 0, entries[0].ID)
	assert.Equal(t, "Leather Helm", entries[0].Name)
	assert.Equal(t, 47, entries[47].ID)
	assert.Equal(t, "Ankh", entries[47].Name)
}

func TestConversationSourceForLocationMatchesRuntimeTalkSelection(t *testing.T) {
	assert.Equal(t, "towne", conversationSourceForLocation(references.Britain))
	assert.Equal(t, "castle", conversationSourceForLocation(references.LordBritishsCastle))
	assert.Equal(t, "dwelling", conversationSourceForLocation(references.IolosHut))
	assert.Equal(t, "keep", conversationSourceForLocation(references.Lycaeum))
	assert.Empty(t, conversationSourceForLocation(references.Doom))
}

func TestCodexNPCLinksUseSemanticProgramsAndActualDialogIdentity(t *testing.T) {
	definition := &semantic.Conversation{
		Schema: semantic.SchemaVersion, Protocol: semantic.ProtocolUltima5, ID: "original.towne.007",
		Origin:      &semantic.OriginalOrigin{Source: semantic.OriginalSourceTowne, Dialog: 7},
		Name:        semantic.Body{&semantic.SayAction{Content: semantic.RichText{&semantic.TextPart{Text: "Runtime bard"}}}},
		Description: semantic.Body{&semantic.NarrateAction{Content: semantic.RichText{&semantic.TextPart{Text: "a musician"}}}},
		Greeting:    semantic.Body{}, Job: semantic.Body{}, Farewell: semantic.Body{}, Topics: []semantic.Topic{}, Questions: map[string]semantic.Question{},
	}
	catalog, err := conversationcatalog.New(definition)
	require.NoError(t, err)
	entries, err := newCodexNPCCatalog(&references.GameReferences{
		ConversationCatalog: catalog,
		NPCReferences: &references.NPCReferences{Npcs: []references.NPCReference{
			{Location: references.Britain, DialogNumber: 7},
			{Location: references.Britain, DialogNumber: 1},
			{Location: references.Lycaeum, DialogNumber: 7},
			{Location: references.Britain, DialogNumber: byte(references.DialogGuard)},
			{Location: references.Britain, DialogNumber: byte(references.NoResponse)},
		}},
	})
	require.NoError(t, err)
	require.Len(t, entries, 5)
	assert.True(t, entries[0].HasConversation)
	assert.Equal(t, "Runtime bard", entries[0].ConversationName)
	assert.Equal(t, "a musician", entries[0].ConversationDescription)
	assert.Equal(t, "towne", entries[0].ConversationSource)
	assert.Equal(t, 6, entries[0].ConversationIndex)
	assert.Equal(t, "Missing semantic conversation", entries[1].ConversationKind)
	assert.False(t, entries[1].HasConversation)
	assert.False(t, entries[2].HasConversation, "the same dialog in another source must not resolve")
	assert.False(t, entries[3].HasConversation)
	assert.Equal(t, -1, entries[3].ConversationIndex)
	assert.Equal(t, "No response", entries[4].ConversationKind)
}

func TestCodexSpellCatalogIncludesEveryCircleAndReagentRecipe(t *testing.T) {
	entries := newCodexSpellCatalog()

	require.Len(t, entries, 48)
	assert.Equal(t, "In Lor", entries[0].Name)
	assert.Equal(t, 1, entries[0].Circle)
	assert.Equal(t, []string{"Sulfurous Ash"}, entries[0].Reagents)
	assert.Equal(t, "An Tym", entries[47].Name)
	assert.Equal(t, 8, entries[47].Circle)
	assert.Contains(t, entries[references.InZuGrav].ReduxTargeting, "range-15")
	assert.Contains(t, entries[references.InZuGrav].Formula, "without a combat save")
	assert.Contains(t, entries[references.InZu].ReduxTargeting, "footprint preview")
	assert.Contains(t, entries[references.InZu].Formula, "target INT")
	assert.Equal(t, "Complete", entries[references.WisAnYlem].Status)
	assert.Contains(t, entries[references.WisAnYlem].Audio, "SoundXRay")
}

func TestCodexSpellCatalogHasCompleteReviewedAuditData(t *testing.T) {
	entries := newCodexSpellCatalog()

	for _, entry := range entries {
		assert.NotEmpty(t, entry.Status, entry.Name+" status")
		assert.NotEmpty(t, entry.OriginalBehavior, entry.Name+" original behavior")
		assert.NotEmpty(t, entry.ReduxBehavior, entry.Name+" Redux behavior")
		assert.NotEmpty(t, entry.OriginalTargeting, entry.Name+" original targeting")
		assert.NotEmpty(t, entry.ReduxTargeting, entry.Name+" Redux targeting")
		assert.NotEmpty(t, entry.ActorVisual, entry.Name+" actor visual")
		assert.NotEmpty(t, entry.ScreenVisual, entry.Name+" screen visual")
		assert.NotEmpty(t, entry.StatusIndicator, entry.Name+" status indicator")
		assert.NotEmpty(t, entry.TurnOrder, entry.Name+" turn order")
		assert.NotEmpty(t, entry.CombatEvents, entry.Name+" combat events")
		assert.NotEmpty(t, entry.AreaEffect, entry.Name+" area effect")
		assert.NotEmpty(t, entry.Failure, entry.Name+" failure behavior")
		assert.NotEmpty(t, entry.Formula, entry.Name+" formula")
		assert.NotEmpty(t, entry.Audio, entry.Name+" audio")
		assert.NotEmpty(t, entry.Opportunities, entry.Name+" opportunities")
		assert.NotEmpty(t, entry.CodeReferences, entry.Name+" code references")
		assert.NotEmpty(t, entry.OriginalReferences, entry.Name+" original references")
	}
}

func TestCodexSaveFormatIncludesStableCoreOffsets(t *testing.T) {
	fields := newCodexSaveFormat()

	require.NotEmpty(t, fields)
	assert.Equal(t, "0x0000", fields[0].Offset)
	assert.Contains(t, fields[4].Description, "stable item-ID order")
}

func TestCodexSpecializedEntriesUseBrowserWireFieldNames(t *testing.T) {
	tileJSON, err := json.Marshal(codexTileEntry{ID: 12, Name: "Grass"})
	require.NoError(t, err)
	assert.JSONEq(t, `{"id":12,"name":"Grass","description":"","lookDescription":"","speed":0,"light":0,"animationFrames":0,"animated":false,"blocksLight":false,"enemy":false,"npc":false,"building":false,"walk":false,"horse":false,"skiff":false,"frigate":false,"carpet":false,"npcPassable":false}`, string(tileJSON))

	npcJSON, err := json.Marshal(codexNPCEntry{LocationID: 2, SpriteIndex: 256, ConversationName: "Iolo"})
	require.NoError(t, err)
	assert.Contains(t, string(npcJSON), `"locationID":2`)
	assert.Contains(t, string(npcJSON), `"spriteIndex":256`)
	assert.Contains(t, string(npcJSON), `"conversationName":"Iolo"`)
	assert.NotContains(t, string(npcJSON), `"LocationID"`)
}

func TestCodexSearchLocationCatalogDecodesOriginalPayloadsWithoutLiveState(t *testing.T) {
	entries := newCodexSearchLocationCatalog()

	require.Len(t, entries, 113)
	assert.Equal(t, 0, entries[0].Index)
	assert.Equal(t, 0x0b, entries[0].RawObjectType)
	assert.Equal(t, references.MysticArmour.ID(), entries[0].ItemID)
	assert.Equal(t, "one_time", entries[0].RepeatPolicy)
	assert.Equal(t, "Mystic Armour", entries[0].ItemName)
	assert.False(t, entries[0].Suspicious)

	skullKeys := entries[14]
	assert.Equal(t, "Skull Keys", skullKeys.ItemName)
	assert.Equal(t, "daily", skullKeys.RepeatPolicy)
	assert.Equal(t, []string{"different_day_from_last_skull_keys"}, skullKeys.Requirements)
	assert.Equal(t, 5, skullKeys.PickupQuantity)

	assert.Equal(t, "conditional", entries[13].RepeatPolicy)
	assert.Equal(t, []string{"no_keys", "no_object_at_location"}, entries[13].Requirements)
	assert.Equal(t, "conditional", entries[15].RepeatPolicy)
	assert.Equal(t, []string{"no_glass_sword", "no_object_at_location"}, entries[15].Requirements)

	encoded, err := json.Marshal(skullKeys)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "found")
}

func TestCodexConsumableCatalogSeparatesOriginalAndReduxBehavior(t *testing.T) {
	entries := newCodexConsumableCatalog()

	require.Len(t, entries, 16)
	assert.Equal(t, "Orange", entries[4].Name)
	assert.Contains(t, entries[4].OriginalBehavior, "sleep")
	assert.Contains(t, entries[4].ReduxBehavior, "sleep")
	assert.Contains(t, entries[4].ReduxConsumptionTiming, "after target confirmation")
	assert.Equal(t, "Any confirmed active party member", entries[14].ReduxTarget)
	assert.Equal(t, "complete", entries[13].Support)
	assert.Contains(t, entries[13].ReduxBehavior, "guaranteed-loyal")
	assert.Contains(t, entries[15].ReduxBehavior, "Stonegate or Doom consumes")
	assert.Contains(t, entries[15].ReduxBehavior, "combat use resolves normally regardless of origin")
	for _, scroll := range entries[8:] {
		assert.NotContains(t, scroll.ReduxConsumptionTiming, "absorption")
	}
	for _, entry := range entries {
		assert.NotEmpty(t, entry.OriginalConsumptionTiming)
		assert.Contains(t, []string{"complete", "partial", "unavailable"}, entry.Support)
	}
}

func TestCodexTileAndQuestCatalogsUseLoadedAuthoritativeMetadata(t *testing.T) {
	tiles := references.NewTileReferences()
	looks := &references.LookReferences{LookData: []byte("LOOK2 description\x00")}
	tileEntries := newCodexTileCatalog(tiles, looks)
	require.NotEmpty(t, tileEntries)
	var tileZero codexTileEntry
	for _, entry := range tileEntries {
		if entry.ID == 0 {
			tileZero = entry
			break
		}
	}
	assert.Equal(t, "LOOK2 description", tileZero.LookDescription)

	items := references.NewInventoryItemsReferences()
	questEntries := newCodexQuestCatalog(items)
	require.NotEmpty(t, questEntries)
	assert.Equal(t, items.GetReferenceByItem(references.Amulet).ItemDescription, questEntries[0].Description)
	assert.NotEmpty(t, questEntries[0].Description)
}

func TestTileAtlasCombinesRuntimeFactsWithInspectorAnnotations(t *testing.T) {
	tiles := references.NewTileReferences()
	tiles.GetTile(1).LightEmission = 7
	entries := newCodexTileCatalog(tiles, &references.LookReferences{})
	byID := make(map[int]codexTileEntry, len(entries))
	for _, entry := range entries {
		byID[entry.ID] = entry
	}
	assert.Equal(t, "Do Not Draw", byID[-1].Description)
	assert.Equal(t, "largest forest", byID[10].Description)
	assert.True(t, byID[1].Animated)
	assert.Equal(t, 2, byID[1].AnimationFrames)
	assert.Equal(t, 7, byID[1].Light, "inspector must read live reference properties, not an annotation copy")
	assert.Equal(t, tiles.GetTile(1).IsWalkingPassable(), byID[1].Walk)
	assert.Equal(t, tiles.GetTile(10).BlocksLight, byID[10].BlocksLight)
}
