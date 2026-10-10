package references

import "strings"

// TavernMenuVariant identifies one original tavern menu family; ref:MERCH-0012.
type TavernMenuVariant byte

// Original meal/drink menu variants; ref:MERCH-0012.
const (
	TavernMenuMuttonAleRations TavernMenuVariant = iota
	TavernMenuWineCheese
	TavernMenuRumBoar
	TavernMenuStoutFruitProvisions
)

// TavernMeal is the meal choice and per-diner base price at one tavern.
type TavernMeal struct {
	Name      string
	Key       byte
	BasePrice int
}

// TavernDrink is the drink choice at one tavern. UnitPrice is zero for Wine,
// whose six bottle prices are provided by TavernWines.
type TavernDrink struct {
	Name       string
	Key        byte
	UnitPrice  int
	IsWineList bool
}

// TavernProvision is the source provision-pack row for one tavern. Available
// distinguishes displayed Rations and Provisions choices from backing rows that
// the original menu does not expose.
type TavernProvision struct {
	Name         string
	Key          byte
	Available    bool
	PackQuantity uint16
	BasePrice    int
}

// TavernInformationMenu is the tavern menu choice that opens paid information.
type TavernInformationMenu struct {
	Name string
	Key  byte
}

// TavernMenu contains the fixed menu and source prices for one active tavern.
type TavernMenu struct {
	MerchantID  MerchantID
	Variant     TavernMenuVariant
	Meal        TavernMeal
	Drink       TavernDrink
	Provision   TavernProvision
	Information TavernInformationMenu
}

// TavernWine is one fixed-price bottle from the original wine list.
type TavernWine struct {
	Name  string
	Price int
}

// TavernInformationTopic is one paid-information keyword in original source
// order. Location retains the source's player-facing location text.
type TavernInformationTopic struct {
	Key       string
	Price     int
	Informant string
	Location  string
}

type tavernCatalogRow struct {
	merchantID         MerchantID
	variant            TavernMenuVariant
	mealBasePrice      int
	provisionBasePrice int
}

// Original food, ale, provisions, and chat response choices; ref:MERCH-0012.
//
//nolint:gochecknoglobals // Canonical original menu variants are immutable after initialization.
var tavernMenuVariants = [4]TavernMenu{
	{
		Variant:     TavernMenuMuttonAleRations,
		Meal:        TavernMeal{Name: "Mutton", Key: 'M'},
		Drink:       TavernDrink{Name: "Ale", Key: 'A', UnitPrice: 1},
		Provision:   TavernProvision{Name: "Rations", Key: 'R', Available: true, PackQuantity: 25},
		Information: TavernInformationMenu{Name: "Chat", Key: 'C'},
	},
	{
		Variant:     TavernMenuWineCheese,
		Meal:        TavernMeal{Name: "Cheese", Key: 'C'},
		Drink:       TavernDrink{Name: "Wine", Key: 'W', IsWineList: true},
		Provision:   TavernProvision{PackQuantity: 25},
		Information: TavernInformationMenu{Name: "Talk", Key: 'T'},
	},
	{
		Variant:     TavernMenuRumBoar,
		Meal:        TavernMeal{Name: "Boar", Key: 'B'},
		Drink:       TavernDrink{Name: "Rum", Key: 'R', UnitPrice: 1},
		Provision:   TavernProvision{PackQuantity: 25},
		Information: TavernInformationMenu{Name: "Help", Key: 'H'},
	},
	{
		Variant:     TavernMenuStoutFruitProvisions,
		Meal:        TavernMeal{Name: "Fruit", Key: 'F'},
		Drink:       TavernDrink{Name: "Stout", Key: 'S', UnitPrice: 1},
		Provision:   TavernProvision{Name: "Provisions", Key: 'P', Available: true, PackQuantity: 25},
		Information: TavernInformationMenu{Name: "Advice", Key: 'A'},
	},
}

// Original meal availability, food prices, and provision prices (ref:MERCH-0012 and ref:MERCH-0014) in
// MerchantDefinition.CatalogRow order. Provision prices are retained even for
// the three menu variants that do not expose a provision choice.
//
//nolint:gochecknoglobals // Canonical original tavern rows are immutable after initialization.
var tavernMenusByCatalogRow = [9]tavernCatalogRow{
	{MerchantTavernMoonglow, TavernMenuMuttonAleRations, 3, 10},
	{MerchantTavernBritain, TavernMenuMuttonAleRations, 4, 15},
	{MerchantTavernJhelom, TavernMenuMuttonAleRations, 5, 20},
	{MerchantTavernYew, TavernMenuRumBoar, 3, 25},
	{MerchantTavernNewMagincia, TavernMenuStoutFruitProvisions, 2, 30},
	{MerchantTavernWestBritanny, TavernMenuWineCheese, 5, 25},
	{MerchantTavernPaws, TavernMenuMuttonAleRations, 3, 20},
	{MerchantTavernBuccaneersDen, TavernMenuRumBoar, 4, 25},
	{MerchantTavernLycaeum, TavernMenuMuttonAleRations, 5, 30},
}

// Original wine display and bottle-price order; ref:MERCH-0012.
//
//nolint:gochecknoglobals // Canonical original wine rows are immutable after initialization.
var tavernWines = [...]TavernWine{
	{Name: "Rose", Price: 18},
	{Name: "Claret", Price: 192},
	{Name: "Sauterne", Price: 79},
	{Name: "Muscatel", Price: 30},
	{Name: "Moselle", Price: 275},
	{Name: "Chablis", Price: 98},
}

// Original paid-information keywords, bribes, names, and town order; ref:MERCH-0014.
//
//nolint:gochecknoglobals // Canonical original information rows are immutable after initialization.
var tavernInformationTopics = [...]TavernInformationTopic{
	{Key: "hone", Price: 50, Informant: "Malik", Location: "Moonglow"},
	{Key: "comp", Price: 75, Informant: "Greyson", Location: "Britain"},
	{Key: "valo", Price: 50, Informant: "Trian", Location: "Jhelom"},
	{Key: "just", Price: 50, Informant: "Jeremy", Location: "Yew"},
	{Key: "sacr", Price: 75, Informant: "Rew", Location: "Minoc"},
	{Key: "hono", Price: 75, Informant: "Gruman", Location: "Trinsic"},
	{Key: "spir", Price: 25, Informant: "Saul", Location: "Skara Brae"},
	{Key: "humi", Price: 50, Informant: "Shirita", Location: "New Magincia"},
	{Key: "dece", Price: 100, Informant: "Malifora", Location: "Moonglow"},
	{Key: "desp", Price: 150, Informant: "Annon", Location: "Britain"},
	{Key: "dest", Price: 75, Informant: "Trian", Location: "Jhelom"},
	{Key: "wron", Price: 150, Informant: "Felespar", Location: "Yew"},
	{Key: "cove", Price: 75, Informant: "the mother of Rew", Location: "Minoc"},
	{Key: "sham", Price: 100, Informant: "Sindar", Location: "Trinsic"},
	{Key: "hyth", Price: 100, Informant: "Kaiko", Location: "New Magincia"},
	{Key: "crow", Price: 200, Informant: "Terrance", Location: "Britain"},
	{Key: "scep", Price: 200, Informant: "Greymarch", Location: "Yew"},
	{Key: "amul", Price: 200, Informant: "Simon and Tessa", Location: "a hidden mountain keep"},
	{Key: "fals", Price: 250, Informant: "Shalineth", Location: "the Lycaeum"},
	{Key: "hatr", Price: 250, Informant: "a daemon", Location: "the desert"},
	{Key: "cowa", Price: 250, Informant: "Lord Malone", Location: "Serpent's Hold"},
	{Key: "astr", Price: 100, Informant: "Zachariah", Location: "Moonglow"},
	{Key: "oppr", Price: 50, Informant: "Tactus", Location: "Minoc"},
	{Key: "brit", Price: 50, Informant: "a daemon", Location: "the desert"},
	{Key: "resi", Price: 200, Informant: "Terrance", Location: "Britain"},
	{Key: "unde", Price: 100, Informant: "Jotham", Location: "a lighthouse south of Britain"},
}

// TavernMenu returns one detached canonical tavern menu.
func (r *MerchantReferences) TavernMenu(merchantID MerchantID) (TavernMenu, bool) {
	if r == nil {
		return TavernMenu{}, false
	}

	merchant, ok := r.Definition(merchantID)
	if !ok || merchant.Family != MerchantFamilyTavern ||
		merchant.CatalogRow < 0 || merchant.CatalogRow >= len(tavernMenusByCatalogRow) {
		return TavernMenu{}, false
	}

	row := tavernMenusByCatalogRow[merchant.CatalogRow]
	if row.merchantID != merchant.ID || int(row.variant) >= len(tavernMenuVariants) {
		return TavernMenu{}, false
	}

	menu := tavernMenuVariants[row.variant]
	menu.MerchantID = row.merchantID
	menu.Meal.BasePrice = row.mealBasePrice
	menu.Provision.BasePrice = row.provisionBasePrice

	return menu, true
}

// TavernWines returns the six fixed-price bottles in source display order.
// The returned slice is detached from the private catalog.
func (r *MerchantReferences) TavernWines() []TavernWine {
	if r == nil {
		return nil
	}

	return append([]TavernWine(nil), tavernWines[:]...)
}

// TavernInformationTopics returns all paid-information topics in source order.
// The returned slice is detached from the private catalog.
func (r *MerchantReferences) TavernInformationTopics() []TavernInformationTopic {
	if r == nil {
		return nil
	}

	return append([]TavernInformationTopic(nil), tavernInformationTopics[:]...)
}

// MatchTavernInformationTopic returns the first source-order topic whose
// normalized key starts the input or immediately follows a literal space. The
// original matcher does not require a word boundary after the key.
func (r *MerchantReferences) MatchTavernInformationTopic(input string) (TavernInformationTopic, bool) {
	if r == nil {
		return TavernInformationTopic{}, false
	}

	normalizedInput := strings.ToLower(input)
	for _, topic := range tavernInformationTopics {
		index := strings.Index(normalizedInput, topic.Key)
		if index == 0 || index > 0 && normalizedInput[index-1] == ' ' {
			return topic, true
		}
	}

	return TavernInformationTopic{}, false
}
