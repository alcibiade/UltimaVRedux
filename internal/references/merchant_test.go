package references

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOriginalMerchantDefinitions(t *testing.T) {
	t.Parallel()

	refs, err := newMerchantReferenceIndex(originalMerchantDefinitions, nil)
	require.NoError(t, err)
	require.Len(t, refs.Definitions(), 46)

	expectedFamilyCounts := map[MerchantFamily]int{
		MerchantFamilyArms:    9,
		MerchantFamilyTavern:  9,
		MerchantFamilyHorse:   3,
		MerchantFamilyShip:    4,
		MerchantFamilyReagent: 5,
		MerchantFamilyGuild:   3,
		MerchantFamilyHealer:  7,
		MerchantFamilyInn:     6,
	}
	actualFamilyCounts := make(map[MerchantFamily]int, len(expectedFamilyCounts))

	for _, definition := range refs.Definitions() {
		actualFamilyCounts[definition.Family]++

		resolved, resolveErr := refs.Resolve(
			DialogNumber(definition.Family),
			definition.Location,
			definition.NPCIndex,
		)
		require.NoError(t, resolveErr)
		assert.Equal(t, definition, resolved)

		catalogDefinition, ok := refs.DefinitionForCatalog(definition.Family, definition.CatalogRow)
		require.True(t, ok)
		assert.Equal(t, definition, catalogDefinition)
	}
	assert.Equal(t, expectedFamilyCounts, actualFamilyCounts)
}

func TestOriginalMerchantDefinitionsCoverEveryIDAndFamilyRow(t *testing.T) {
	t.Parallel()

	refs, err := newMerchantReferenceIndex(originalMerchantDefinitions, nil)
	require.NoError(t, err)

	expectedIDs := []MerchantID{
		MerchantArmsBritain,
		MerchantArmsJhelom,
		MerchantArmsYew,
		MerchantArmsMinoc,
		MerchantArmsTrinsic,
		MerchantArmsCastleBritannia,
		MerchantArmsBuccaneersDen,
		MerchantArmsBordermarch,
		MerchantArmsSerpentsHold,
		MerchantTavernMoonglow,
		MerchantTavernBritain,
		MerchantTavernJhelom,
		MerchantTavernYew,
		MerchantTavernNewMagincia,
		MerchantTavernWestBritanny,
		MerchantTavernPaws,
		MerchantTavernBuccaneersDen,
		MerchantTavernLycaeum,
		MerchantHorseTrinsic,
		MerchantHorseNorthBritanny,
		MerchantHorsePaws,
		MerchantShipJhelom,
		MerchantShipMinoc,
		MerchantShipEastBritanny,
		MerchantShipBuccaneersDen,
		MerchantReagentMoonglow,
		MerchantReagentYew,
		MerchantReagentSkaraBrae,
		MerchantReagentCove,
		MerchantReagentLycaeum,
		MerchantGuildNewMagincia,
		MerchantGuildPaws,
		MerchantGuildBuccaneersDen,
		MerchantHealerMinoc,
		MerchantHealerTrinsic,
		MerchantHealerSkaraBrae,
		MerchantHealerEastBritanny,
		MerchantHealerCove,
		MerchantHealerLycaeum,
		MerchantHealerEmpathAbbey,
		MerchantInnBritain,
		MerchantInnJhelom,
		MerchantInnSkaraBrae,
		MerchantInnNorthBritanny,
		MerchantInnPaws,
		MerchantInnBuccaneersDen,
	}
	require.Len(t, expectedIDs, 46)
	seen := make(map[MerchantID]struct{}, len(expectedIDs))
	for _, id := range expectedIDs {
		_, duplicate := seen[id]
		assert.False(t, duplicate, "duplicate expected merchant ID %q", id)
		seen[id] = struct{}{}

		_, ok := refs.Definition(id)
		assert.True(t, ok, id)
	}

	for family := MerchantFamilyArms; family <= MerchantFamilyInn; family++ {
		for row := range merchantCatalogRowCount(family) {
			_, ok := refs.DefinitionForCatalog(family, row)
			assert.True(t, ok, "family 0x%02x row %d", byte(family), row)
		}
	}
}

func TestOriginalGuildMerchantMappings(t *testing.T) {
	t.Parallel()

	refs, err := newMerchantReferenceIndex(originalMerchantDefinitions, nil)
	require.NoError(t, err)

	expected := []MerchantDefinition{
		{MerchantGuildNewMagincia, MerchantFamilyGuild, NewMagincia, 1, 0, "Braunam", "The Den"},
		{MerchantGuildPaws, MerchantFamilyGuild, Paws, 1, 1, "Danfits", "The Guild"},
		{MerchantGuildBuccaneersDen, MerchantFamilyGuild, BuccaneersDen, 1, 2, "Daem", "The Nemesis"},
	}
	for _, definition := range expected {
		actual, ok := refs.DefinitionForCatalog(MerchantFamilyGuild, definition.CatalogRow)
		require.True(t, ok)
		assert.Equal(t, definition, actual)
	}
}

func TestMerchantDefinitionCanonicalMappings(t *testing.T) {
	t.Parallel()

	refs, err := newMerchantReferenceIndex(originalMerchantDefinitions, nil)
	require.NoError(t, err)

	tests := []MerchantDefinition{
		{MerchantArmsCastleBritannia, MerchantFamilyArms, LordBritishsCastle, 30, 5, "Max", "North Star Armoury"},
		{MerchantTavernLycaeum, MerchantFamilyTavern, Lycaeum, 10, 8, "Rob", "The Folley Tap"},
		{MerchantHorseNorthBritanny, MerchantFamilyHorse, NorthBritanny, 6, 1, "Theoan", "The Stablehouse"},
		{MerchantShipEastBritanny, MerchantFamilyShip, EastBritanny, 2, 2, "Master Hawkins", "The Oaken Oar"},
		{MerchantReagentCove, MerchantFamilyReagent, Cove, 1, 3, "Enlor", "Mysticism"},
		{MerchantGuildBuccaneersDen, MerchantFamilyGuild, BuccaneersDen, 1, 2, "Daem", "The Nemesis"},
		{MerchantHealerEmpathAbbey, MerchantFamilyHealer, EmpathAbbey, 9, 6, "Jessip", "The Empath"},
		{MerchantInnJhelom, MerchantFamilyInn, Jhelom, 1, 1, "Gremnor", "The Warrior's Stead"},
	}

	for _, expected := range tests {
		actual, ok := refs.Definition(expected.ID)
		require.True(t, ok, expected.ID)
		assert.Equal(t, expected, actual)
	}
}

func TestMerchantResolutionRejectsUnsupportedAndMismatchedRows(t *testing.T) {
	t.Parallel()

	refs, err := newMerchantReferenceIndex(originalMerchantDefinitions, nil)
	require.NoError(t, err)

	for _, code := range []DialogNumber{MerchantThreshold, 0x89, DialogGuard} {
		_, resolveErr := refs.Resolve(code, Britain, 1)
		assert.ErrorContains(t, resolveErr, "unsupported merchant dialog code")
	}

	_, err = refs.Resolve(DialogWeaponDealer, Britain, 2)
	assert.ErrorContains(t, err, "NPC index mismatch")

	_, err = refs.Resolve(DialogWeaponDealer, Moonglow, 1)
	assert.ErrorContains(t, err, "no active merchant")

	// These original source rows have no active, valid merchant NPC.
	for _, orphan := range []merchantCatalogKey{
		{family: MerchantFamilyHorse, row: 3},
		{family: MerchantFamilyGuild, row: 3},
		{family: MerchantFamilyHealer, row: 7},
	} {
		_, ok := refs.DefinitionForCatalog(orphan.family, orphan.row)
		assert.False(t, ok)
	}

	_, err = refs.Resolve(DialogHorseDealer, Lycaeum, 0)
	assert.ErrorContains(t, err, "no active merchant")
}

func TestMerchantReferencesRejectInvalidDefinitionRows(t *testing.T) {
	t.Parallel()

	definitions := append([]MerchantDefinition(nil), originalMerchantDefinitions...)
	definitions[18].CatalogRow = 3
	_, err := newMerchantReferenceIndex(definitions, nil)
	assert.ErrorContains(t, err, "invalid catalog row")

	definitions = append([]MerchantDefinition(nil), originalMerchantDefinitions...)
	definitions[1].ID = definitions[0].ID
	_, err = newMerchantReferenceIndex(definitions, nil)
	assert.ErrorContains(t, err, "duplicate merchant ID")
}

func TestMerchantReferencesRetrievalIsDetached(t *testing.T) {
	t.Parallel()

	texts := map[MerchantTextID]string{"TEST": "original"}
	refs, err := newMerchantReferenceIndex(originalMerchantDefinitions, texts)
	require.NoError(t, err)

	definitions := refs.Definitions()
	definitions[0].OwnerName = "changed"
	definition, ok := refs.Definition(MerchantArmsBritain)
	require.True(t, ok)
	assert.Equal(t, "Gwenneth", definition.OwnerName)

	texts["TEST"] = "changed"
	text, ok := refs.Text("TEST")
	require.True(t, ok)
	assert.Equal(t, "original", text)
}
