package references

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTavernMenusContainNineCanonicalRowsAndFourVariants(t *testing.T) {
	t.Parallel()

	refs, err := newMerchantReferenceIndex(originalMerchantDefinitions, nil)
	require.NoError(t, err)
	tests := []struct {
		merchantID  MerchantID
		variant     TavernMenuVariant
		meal        TavernMeal
		drink       TavernDrink
		provision   TavernProvision
		information TavernInformationMenu
	}{
		{
			MerchantTavernMoonglow, TavernMenuMuttonAleRations,
			TavernMeal{"Mutton", 'M', 3}, TavernDrink{"Ale", 'A', 1, false},
			TavernProvision{"Rations", 'R', true, 25, 10}, TavernInformationMenu{"Chat", 'C'},
		},
		{
			MerchantTavernBritain, TavernMenuMuttonAleRations,
			TavernMeal{"Mutton", 'M', 4}, TavernDrink{"Ale", 'A', 1, false},
			TavernProvision{"Rations", 'R', true, 25, 15}, TavernInformationMenu{"Chat", 'C'},
		},
		{
			MerchantTavernJhelom, TavernMenuMuttonAleRations,
			TavernMeal{"Mutton", 'M', 5}, TavernDrink{"Ale", 'A', 1, false},
			TavernProvision{"Rations", 'R', true, 25, 20}, TavernInformationMenu{"Chat", 'C'},
		},
		{
			MerchantTavernYew, TavernMenuRumBoar,
			TavernMeal{"Boar", 'B', 3}, TavernDrink{"Rum", 'R', 1, false},
			TavernProvision{"", 0, false, 25, 25}, TavernInformationMenu{"Help", 'H'},
		},
		{
			MerchantTavernNewMagincia, TavernMenuStoutFruitProvisions,
			TavernMeal{"Fruit", 'F', 2}, TavernDrink{"Stout", 'S', 1, false},
			TavernProvision{"Provisions", 'P', true, 25, 30}, TavernInformationMenu{"Advice", 'A'},
		},
		{
			MerchantTavernWestBritanny, TavernMenuWineCheese,
			TavernMeal{"Cheese", 'C', 5}, TavernDrink{"Wine", 'W', 0, true},
			TavernProvision{"", 0, false, 25, 25}, TavernInformationMenu{"Talk", 'T'},
		},
		{
			MerchantTavernPaws, TavernMenuMuttonAleRations,
			TavernMeal{"Mutton", 'M', 3}, TavernDrink{"Ale", 'A', 1, false},
			TavernProvision{"Rations", 'R', true, 25, 20}, TavernInformationMenu{"Chat", 'C'},
		},
		{
			MerchantTavernBuccaneersDen, TavernMenuRumBoar,
			TavernMeal{"Boar", 'B', 4}, TavernDrink{"Rum", 'R', 1, false},
			TavernProvision{"", 0, false, 25, 25}, TavernInformationMenu{"Help", 'H'},
		},
		{
			MerchantTavernLycaeum, TavernMenuMuttonAleRations,
			TavernMeal{"Mutton", 'M', 5}, TavernDrink{"Ale", 'A', 1, false},
			TavernProvision{"Rations", 'R', true, 25, 30}, TavernInformationMenu{"Chat", 'C'},
		},
	}
	require.Len(t, tavernMenusByCatalogRow, len(tests))
	require.Len(t, tavernMenuVariants, 4)
	seenVariants := make(map[TavernMenuVariant]bool, len(tavernMenuVariants))

	for row, test := range tests {
		merchant, ok := refs.DefinitionForCatalog(MerchantFamilyTavern, row)
		require.True(t, ok, "row %d", row)
		assert.Equal(t, test.merchantID, merchant.ID)

		menu, ok := refs.TavernMenu(test.merchantID)
		require.True(t, ok, test.merchantID)
		assert.Equal(t, TavernMenu{
			MerchantID: test.merchantID, Variant: test.variant,
			Meal: test.meal, Drink: test.drink, Provision: test.provision, Information: test.information,
		}, menu)
		seenVariants[menu.Variant] = true
	}

	assert.Len(t, seenVariants, len(tavernMenuVariants))
}

func TestTavernMenuRejectsNonTavernUnknownAndNilReferences(t *testing.T) {
	t.Parallel()

	refs, err := newMerchantReferenceIndex(originalMerchantDefinitions, nil)
	require.NoError(t, err)
	_, ok := refs.TavernMenu(MerchantInnBritain)
	assert.False(t, ok)
	_, ok = refs.TavernMenu(MerchantID("tavern.unknown"))
	assert.False(t, ok)
	_, ok = (*MerchantReferences)(nil).TavernMenu(MerchantTavernBritain)
	assert.False(t, ok)
}

func TestTavernWinesContainExactSourceListAndReturnDefensiveCopies(t *testing.T) {
	t.Parallel()

	refs := &MerchantReferences{}
	want := []TavernWine{
		{Name: "Rose", Price: 18},
		{Name: "Claret", Price: 192},
		{Name: "Sauterne", Price: 79},
		{Name: "Muscatel", Price: 30},
		{Name: "Moselle", Price: 275},
		{Name: "Chablis", Price: 98},
	}
	wines := refs.TavernWines()
	assert.Equal(t, want, wines)
	wines[0].Name = "Changed"
	wines[0].Price = 1
	assert.Equal(t, want, refs.TavernWines())
	assert.Nil(t, (*MerchantReferences)(nil).TavernWines())
}

func TestTavernInformationTopicsContainExactSourceRowsAndReturnDefensiveCopies(t *testing.T) {
	t.Parallel()

	refs := &MerchantReferences{}
	want := []TavernInformationTopic{
		{"hone", 50, "Malik", "Moonglow"},
		{"comp", 75, "Greyson", "Britain"},
		{"valo", 50, "Trian", "Jhelom"},
		{"just", 50, "Jeremy", "Yew"},
		{"sacr", 75, "Rew", "Minoc"},
		{"hono", 75, "Gruman", "Trinsic"},
		{"spir", 25, "Saul", "Skara Brae"},
		{"humi", 50, "Shirita", "New Magincia"},
		{"dece", 100, "Malifora", "Moonglow"},
		{"desp", 150, "Annon", "Britain"},
		{"dest", 75, "Trian", "Jhelom"},
		{"wron", 150, "Felespar", "Yew"},
		{"cove", 75, "the mother of Rew", "Minoc"},
		{"sham", 100, "Sindar", "Trinsic"},
		{"hyth", 100, "Kaiko", "New Magincia"},
		{"crow", 200, "Terrance", "Britain"},
		{"scep", 200, "Greymarch", "Yew"},
		{"amul", 200, "Simon and Tessa", "a hidden mountain keep"},
		{"fals", 250, "Shalineth", "the Lycaeum"},
		{"hatr", 250, "a daemon", "the desert"},
		{"cowa", 250, "Lord Malone", "Serpent's Hold"},
		{"astr", 100, "Zachariah", "Moonglow"},
		{"oppr", 50, "Tactus", "Minoc"},
		{"brit", 50, "a daemon", "the desert"},
		{"resi", 200, "Terrance", "Britain"},
		{"unde", 100, "Jotham", "a lighthouse south of Britain"},
	}
	topics := refs.TavernInformationTopics()
	assert.Equal(t, want, topics)
	topics[0].Key = "changed"
	topics[0].Price = 1
	assert.Equal(t, want, refs.TavernInformationTopics())
	assert.Nil(t, (*MerchantReferences)(nil).TavernInformationTopics())
}

func TestMatchTavernInformationTopicPreservesOriginalMatchingRules(t *testing.T) {
	t.Parallel()

	refs := &MerchantReferences{}
	tests := []struct {
		name    string
		input   string
		wantKey string
		wantOK  bool
	}{
		{name: "input start", input: "honesty", wantKey: "hone", wantOK: true},
		{name: "case normalized", input: "COMPASSION", wantKey: "comp", wantOK: true},
		{name: "after literal space", input: "tell me about valor", wantKey: "valo", wantOK: true},
		{name: "no trailing boundary required", input: "honey", wantKey: "hone", wantOK: true},
		{name: "first source row wins", input: "compassion then honesty", wantKey: "hone", wantOK: true},
		{name: "invalid first occurrence hides valid later occurrence", input: "dishonesty honesty", wantOK: false},
		{name: "embedded key rejected", input: "dishonesty", wantOK: false},
		{name: "tab is not space", input: "about\thonesty", wantOK: false},
		{name: "punctuation is not space", input: "about,honesty", wantOK: false},
		{name: "empty", input: "", wantOK: false},
		{name: "unknown", input: "gargoyles", wantOK: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			topic, ok := refs.MatchTavernInformationTopic(test.input)
			assert.Equal(t, test.wantOK, ok)
			assert.Equal(t, test.wantKey, topic.Key)
		})
	}

	_, ok := (*MerchantReferences)(nil).MatchTavernInformationTopic("honesty")
	assert.False(t, ok)
}
