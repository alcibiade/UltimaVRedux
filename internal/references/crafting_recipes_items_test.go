package references

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// magicEquipment lists every original equipment item that is magic, a quest item or made of a
// precious material. None of them may be crafted (ref:CRAFT-0013).
var magicEquipment = []Equipment{
	MagicShield, JewelShield, MysticArmour, ChaosSword, MagicBow, SilverSword, MagicAxe, GlassSword,
	JeweledSword, MysticSword, RingInvisibility, RingProtection, RingRegeneration, AmuletOfTurning,
	SpikedCollar, Ankh,
}

// mundaneEquipment is every craftable item. Flaming Oil is mundane but not craftable: deeds
// yield no lamp oil.
var mundaneEquipment = []Equipment{
	Dagger, MainGauche, ShortSword, LongSword, TwoHSword,
	ThrowingAxe, Mace, MorningStar, TwoHHammer, TwoHAxe, Halberd,
	Club, Sling, Spear, Bow, Arrows, Crossbow, Quarrels,
	ClothArmour, LeatherArmour, RingMail, ScaleMail, ChainMail, PlateMail,
	LeatherHelm, ChainCoif, IronHelm, SpikedHelm, SmallShield, LargeShield, SpikedShield,
}

func TestNoMagicItemIsCraftable(t *testing.T) {
	craftable := CraftableEquipment()
	for _, item := range magicEquipment {
		assert.NotContains(t, craftable, item, "%s must stay a shop or quest reward", item)
	}
	for _, recipe := range CraftRecipes() {
		for _, output := range recipe.Outputs {
			if output.Kind == CraftGoodsEquipment {
				assert.Contains(t, mundaneEquipment, Equipment(output.ID), recipe.Name)
			}
		}
	}
	assert.ElementsMatch(t, mundaneEquipment, craftable, "each mundane item has exactly one recipe")
	assert.NotContains(t, craftable, FlamingOil)
}

func TestEquipmentRecipesUseTheirMaterialsStation(t *testing.T) {
	for _, recipe := range craftRecipesForEquipment() {
		usesIngots, usesPlanks := false, false
		for _, input := range recipe.Inputs {
			require.NotEqual(t, CraftGoodsEquipment, input.Kind, "%s: equipment is never an input", recipe.Name)
			usesIngots = usesIngots || (input.Kind == CraftGoodsMaterial && Material(input.ID) == Ingots)
			usesPlanks = usesPlanks || (input.Kind == CraftGoodsMaterial && Material(input.ID) == Planks)
		}
		switch recipe.Station {
		case CraftStationForge:
			assert.True(t, usesIngots, "%s: forged goods are metal", recipe.Name)
		case CraftStationWorkbench:
			assert.False(t, usesIngots && !usesPlanks, "%s: workbench goods are mostly wood", recipe.Name)
		default:
			t.Errorf("%s: unexpected station %s", recipe.Name, recipe.Station.Name())
		}
	}
}

func TestCraftedEquipmentNeverUndercutsTheShopByMuch(t *testing.T) {
	// One Ingots takes about 50 minutes of mining and smelting, and Planks take a few; a forged
	// item's labour (its minutes plus its ingots) grows with its shop price.
	for _, recipe := range craftRecipesForEquipment() {
		output := recipe.Outputs[0]
		if output.Kind != CraftGoodsEquipment || Equipment(output.ID).IsAmmo() {
			continue
		}
		price := Equipment(output.ID).GetBasePrice()
		assert.Positive(t, price, recipe.Name)
		labour := recipe.Minutes
		for _, input := range recipe.Inputs {
			if input.Kind == CraftGoodsMaterial && Material(input.ID) == Ingots {
				labour += 50 * int(input.Quantity)
			}
		}
		assert.GreaterOrEqual(t, labour, 15, recipe.Name)
		assert.GreaterOrEqual(t, labour*2, price, "%s: %d minutes of labour for %d gold", recipe.Name, labour, price)
	}
}

func TestScrollsUseTheirSpellsReagentsAndPaper(t *testing.T) {
	spells := map[Scroll]Spell{
		ScrollVasLor: VasLor, ScrollRelHur: RelHur, ScrollInSanct: InSanct, ScrollInAn: InAn,
		ScrollInQuasWis: InQuasWis, ScrollKalXenCorp: KalXenCorp, ScrollInManiCorp: InManiCorp, ScrollAnTym: AnTym,
	}
	made := map[Scroll]bool{}
	for _, recipe := range CraftRecipes() {
		if recipe.Category != CraftCategoryScrolls {
			continue
		}
		require.Len(t, recipe.Outputs, 1)
		scroll := Scroll(recipe.Outputs[0].ID)
		made[scroll] = true
		assert.Equal(t, CraftStationAlchemyTable, recipe.Station)
		var mask uint16
		for _, input := range recipe.Inputs {
			if input.Kind == CraftGoodsReagent {
				mask |= 1 << input.ID
				assert.Equal(t, uint16(1), input.Quantity)
			}
		}
		assert.Equal(t, GetSpellReagents(spells[scroll]), mask, recipe.Name)
		assert.Contains(t, recipe.Inputs, MaterialGoods(Planks, 1), "%s: Planks are the paper", recipe.Name)
	}
	assert.Len(t, made, NumberOfScrolls)
}

func TestEveryPotionHasOneFixedRecipe(t *testing.T) {
	made := map[Potion]int{}
	inputs := map[string]Potion{}
	for _, recipe := range CraftRecipes() {
		if recipe.Category != CraftCategoryPotions {
			continue
		}
		potion := Potion(recipe.Outputs[0].ID)
		made[potion]++
		assert.Equal(t, CraftStationAlchemyTable, recipe.Station)
		key := DescribeCraftGoods(recipe.Inputs)
		_, duplicate := inputs[key]
		assert.False(t, duplicate, "%s shares its reagents with another potion", recipe.Name)
		inputs[key] = potion
	}
	for _, potion := range PotionCatalog() {
		assert.Equal(t, 1, made[potion], potion.Name())
	}
}

// TestRefiningMakesEveryAlchemyRecipeReachable checks that every reagent a potion or scroll needs
// grows on some deed terrain, or is refined from one that does (ref:CRAFT-0017).
func TestRefiningMakesEveryAlchemyRecipeReachable(t *testing.T) {
	grown := map[Reagent]bool{}
	for _, terrain := range DeedTerrains() {
		for _, quota := range DeedProfile(terrain).Nodes {
			for _, reagent := range quota.Reagents {
				grown[reagent] = true
			}
		}
	}
	reachable := map[Reagent]bool{}
	for reagent := range grown {
		reachable[reagent] = true
	}
	refinings := 0
	for _, recipe := range CraftRecipes() {
		if recipe.Category != CraftCategoryRefining {
			continue
		}
		refinings++
		require.Len(t, recipe.Inputs, 1)
		input, output := recipe.Inputs[0], recipe.Outputs[0]
		assert.Equal(t, CraftGoodsReagent, input.Kind)
		assert.Equal(t, CraftGoodsReagent, output.Kind)
		assert.Equal(t, uint16(3), input.Quantity, "a poor rate: 3 to 1")
		assert.Equal(t, uint16(1), output.Quantity)
		assert.True(t, grown[Reagent(input.ID)], "%s refines a grown reagent", recipe.Name)
		assert.False(t, grown[Reagent(output.ID)], "%s makes a rarer reagent", recipe.Name)
		reachable[Reagent(output.ID)] = true
	}
	assert.Equal(t, 3, refinings)
	for _, reagent := range ReagentCatalog() {
		assert.True(t, reachable[reagent], reagent.Name())
	}
}

func TestDescribeCraftGoodsCompactShortensReagents(t *testing.T) {
	assert.Equal(t, "SA Gi Ga, 1 Planks", DescribeCraftGoodsCompact([]CraftGoods{
		ReagentGoods(SulfurAsh, 1), ReagentGoods(Ginseng, 1), ReagentGoods(Garlic, 1), MaterialGoods(Planks, 1),
	}))
	assert.Equal(t, "3 SA", DescribeCraftGoodsCompact([]CraftGoods{ReagentGoods(SulfurAsh, 3)}))
	assert.Equal(t, "4 Ingots, 1 Planks", DescribeCraftGoodsCompact([]CraftGoods{MaterialGoods(Ingots, 4), MaterialGoods(Planks, 1)}))
	assert.Equal(t, "Blue Potion", PotionGoods(Blue, 1).Name())
	assert.Equal(t, "In Mani Corp Scroll", ScrollGoods(ScrollInManiCorp, 1).Name())
	assert.Equal(t, "Two-Handed Sword", EquipmentGoods(TwoHSword, 1).Name())
}

func TestCraftCategoriesStayShortEnoughForTheMenu(t *testing.T) {
	counts := map[CraftCategory]int{}
	for _, recipe := range CraftRecipes() {
		counts[recipe.Category]++
	}
	for _, category := range CraftCategories() {
		assert.LessOrEqual(t, counts[category], 9, category.Name())
		assert.Positive(t, counts[category], category.Name())
	}
}
