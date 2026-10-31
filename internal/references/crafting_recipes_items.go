package references

// REDUX DEVIATION: CRAFT-0013: the original sells weapons, armour, shields, torches and keys
// in shops and finds potions and scrolls, but makes none of them. With Gameplay.Crafting.Enabled,
// stations on deed maps make the mundane original items, the original potions and scrolls, and
// refine common reagents into rarer ones; see ref:CRAFT-0013 to ref:CRAFT-0017.

// Original items made by crafting (ref:CRAFT-0013 to ref:CRAFT-0017). Outputs are the original
// item identities, so crafted goods fit equipment, shops and saves unchanged.

// equipmentRecipe makes one original equipment item at a station.
func equipmentRecipe(
	id CraftRecipeID, equipment Equipment, category CraftCategory, made uint16, minutes int, station CraftStation,
	inputs ...CraftGoods,
) CraftRecipe {
	return CraftRecipe{
		ID: id, Name: equipment.String(), Category: category, Minutes: minutes, Station: station,
		Inputs: inputs, Outputs: []CraftGoods{EquipmentGoods(equipment, made)},
	}
}

// CraftableEquipment returns every equipment item a recipe makes, in recipe order. Only mundane
// items are craftable: magic, quest and precious items stay shop and quest rewards
// (ref:CRAFT-0013).
func CraftableEquipment() []Equipment {
	recipes := craftRecipesForEquipment()
	items := make([]Equipment, 0, len(recipes))
	for _, recipe := range recipes {
		for _, output := range recipe.Outputs {
			if output.Kind == CraftGoodsEquipment {
				items = append(items, Equipment(output.ID))
			}
		}
	}

	return items
}

// craftRecipesForEquipment returns the mundane weapons, armour and shields (ref:CRAFT-0013) and
// the torches and keys (ref:CRAFT-0014). Metal is forged from Ingots; wooden, leather and cloth
// goods are made at the Workbench.
func craftRecipesForEquipment() []CraftRecipe {
	forge, bench := CraftStationForge, CraftStationWorkbench
	ingots := func(quantity uint16) CraftGoods { return MaterialGoods(Ingots, quantity) }
	planks := func(quantity uint16) CraftGoods { return MaterialGoods(Planks, quantity) }

	return []CraftRecipe{
		equipmentRecipe(RecipeDagger, Dagger, CraftCategoryBlades, 1, 30, forge, ingots(1)),
		equipmentRecipe(RecipeMainGauche, MainGauche, CraftCategoryBlades, 1, 45, forge, ingots(2)),
		equipmentRecipe(RecipeShortSword, ShortSword, CraftCategoryBlades, 1, 60, forge, ingots(3), planks(1)),
		equipmentRecipe(RecipeLongSword, LongSword, CraftCategoryBlades, 1, 90, forge, ingots(4), planks(1)),
		equipmentRecipe(RecipeTwoHandedSword, TwoHSword, CraftCategoryBlades, 1, 150, forge, ingots(7), planks(1)),

		equipmentRecipe(RecipeThrowingAxe, ThrowingAxe, CraftCategoryHafted, 1, 30, forge, ingots(1), planks(1)),
		equipmentRecipe(RecipeMace, Mace, CraftCategoryHafted, 1, 60, forge, ingots(3), planks(1)),
		equipmentRecipe(RecipeMorningStar, MorningStar, CraftCategoryHafted, 1, 75, forge, ingots(4), planks(1)),
		equipmentRecipe(RecipeTwoHandedHammer, TwoHHammer, CraftCategoryHafted, 1, 90, forge, ingots(5), planks(2)),
		equipmentRecipe(RecipeTwoHandedAxe, TwoHAxe, CraftCategoryHafted, 1, 120, forge, ingots(6), planks(2)),
		equipmentRecipe(RecipeHalberd, Halberd, CraftCategoryHafted, 1, 150, forge, ingots(8), planks(3)),

		equipmentRecipe(RecipeClub, Club, CraftCategoryWoodenWeapons, 1, 15, bench, planks(2)),
		equipmentRecipe(RecipeSling, Sling, CraftCategoryWoodenWeapons, 1, 20, bench, planks(1), MaterialGoods(Forage, 1)),
		equipmentRecipe(RecipeSpear, Spear, CraftCategoryWoodenWeapons, 1, 30, bench, planks(2), ingots(1)),
		equipmentRecipe(RecipeBow, Bow, CraftCategoryWoodenWeapons, 1, 90, bench, planks(6), MaterialGoods(Forage, 1)),
		equipmentRecipe(RecipeArrows, Arrows, CraftCategoryWoodenWeapons, 10, 30, bench, planks(1), ingots(1)),
		equipmentRecipe(RecipeCrossbow, Crossbow, CraftCategoryWoodenWeapons, 1, 120, bench, planks(4), ingots(2)),
		equipmentRecipe(RecipeQuarrels, Quarrels, CraftCategoryWoodenWeapons, 10, 30, bench, planks(1), ingots(1)),

		equipmentRecipe(RecipeClothArmour, ClothArmour, CraftCategoryArmour, 1, 45, bench, MaterialGoods(Forage, 4)),
		equipmentRecipe(RecipeLeatherArmour, LeatherArmour, CraftCategoryArmour, 1, 60, bench, planks(4), MaterialGoods(Forage, 2)),
		equipmentRecipe(RecipeRingMail, RingMail, CraftCategoryArmour, 1, 90, forge, ingots(5)),
		equipmentRecipe(RecipeScaleMail, ScaleMail, CraftCategoryArmour, 1, 120, forge, ingots(6)),
		equipmentRecipe(RecipeChainMail, ChainMail, CraftCategoryArmour, 1, 180, forge, ingots(8)),
		equipmentRecipe(RecipePlateMail, PlateMail, CraftCategoryArmour, 1, 240, forge, ingots(12)),

		equipmentRecipe(RecipeLeatherHelm, LeatherHelm, CraftCategoryHelmsShields, 1, 30, bench, planks(2), MaterialGoods(Forage, 1)),
		equipmentRecipe(RecipeChainCoif, ChainCoif, CraftCategoryHelmsShields, 1, 60, forge, ingots(2)),
		equipmentRecipe(RecipeIronHelm, IronHelm, CraftCategoryHelmsShields, 1, 75, forge, ingots(3)),
		equipmentRecipe(RecipeSpikedHelm, SpikedHelm, CraftCategoryHelmsShields, 1, 90, forge, ingots(4)),
		equipmentRecipe(RecipeSmallShield, SmallShield, CraftCategoryHelmsShields, 1, 45, bench, planks(3), ingots(1)),
		equipmentRecipe(RecipeLargeShield, LargeShield, CraftCategoryHelmsShields, 1, 60, bench, planks(5), ingots(2)),
		equipmentRecipe(RecipeSpikedShield, SpikedShield, CraftCategoryHelmsShields, 1, 90, forge, ingots(4), planks(3)),

		{ID: RecipeTorches, Name: "Torches", Category: CraftCategoryFood, Minutes: 10, Station: bench,
			Inputs:  []CraftGoods{MaterialGoods(Wood, 1), MaterialGoods(Forage, 1)},
			Outputs: []CraftGoods{ProvisionGoods(Torches, 2)}},
		{ID: RecipeKey, Name: "Key", Category: CraftCategoryFood, Minutes: 20, Station: forge,
			Inputs:  []CraftGoods{ingots(1)},
			Outputs: []CraftGoods{ProvisionGoods(Key, 1)}},
	}
}

// alchemyRecipe makes one potion or scroll, or refines a reagent, at the Alchemy table.
func alchemyRecipe(
	id CraftRecipeID, name string, category CraftCategory, minutes int, output CraftGoods, inputs ...CraftGoods,
) CraftRecipe {
	return CraftRecipe{
		ID: id, Name: name, Category: category, Minutes: minutes, Station: CraftStationAlchemyTable,
		Inputs: inputs, Outputs: []CraftGoods{output},
	}
}

// scrollRecipe writes a scroll from one of each reagent of its spell (GetSpellReagents), in
// reagent order, and one Planks as paper (ref:CRAFT-0016).
func scrollRecipe(id CraftRecipeID, scroll Scroll, spell Spell) CraftRecipe {
	mask := GetSpellReagents(spell)
	inputs := make([]CraftGoods, 0, 7)
	for reagent := SulfurAsh; reagent <= MandrakeRoot; reagent++ {
		if mask&(1<<reagent) != 0 {
			inputs = append(inputs, ReagentGoods(reagent, 1))
		}
	}
	inputs = append(inputs, MaterialGoods(Planks, 1))
	name := scroll.Name()

	return alchemyRecipe(id, name[:len(name)-len(" Scroll")], CraftCategoryScrolls, 60, ScrollGoods(scroll, 1), inputs...)
}

// craftRecipesForAlchemy returns the potions (ref:CRAFT-0015), scrolls (ref:CRAFT-0016) and
// reagent refining (ref:CRAFT-0017).
func craftRecipesForAlchemy() []CraftRecipe {
	one := func(reagent Reagent) CraftGoods { return ReagentGoods(reagent, 1) }
	potion := func(id CraftRecipeID, color Potion, inputs ...CraftGoods) CraftRecipe {
		return alchemyRecipe(id, color.Name(), CraftCategoryPotions, 30, PotionGoods(color, 1), inputs...)
	}
	refine := func(id CraftRecipeID, made, from Reagent) CraftRecipe {
		return alchemyRecipe(id, made.Name(), CraftCategoryRefining, 30, ReagentGoods(made, 1), ReagentGoods(from, 3))
	}

	return []CraftRecipe{
		potion(RecipeBluePotion, Blue, one(SulfurAsh), one(Ginseng), one(Garlic)),
		potion(RecipeYellowPotion, Yellow, one(Ginseng), one(SpiderSilk)),
		potion(RecipeRedPotion, Red, one(Ginseng), one(Garlic)),
		potion(RecipeGreenPotion, Green, one(BloodMoss), one(NightShade)),
		potion(RecipeOrangePotion, Orange, one(Ginseng), one(SpiderSilk), one(NightShade)),
		potion(RecipePurplePotion, Purple, one(SpiderSilk), one(BloodMoss), one(MandrakeRoot)),
		potion(RecipeBlackPotion, Black, one(BloodMoss), one(NightShade), one(MandrakeRoot)),
		potion(RecipeWhitePotion, White, one(SulfurAsh), one(MandrakeRoot)),

		scrollRecipe(RecipeScrollVasLor, ScrollVasLor, VasLor),
		scrollRecipe(RecipeScrollRelHur, ScrollRelHur, RelHur),
		scrollRecipe(RecipeScrollInSanct, ScrollInSanct, InSanct),
		scrollRecipe(RecipeScrollInAn, ScrollInAn, InAn),
		scrollRecipe(RecipeScrollInQuasWis, ScrollInQuasWis, InQuasWis),
		scrollRecipe(RecipeScrollKalXenCorp, ScrollKalXenCorp, KalXenCorp),
		scrollRecipe(RecipeScrollInManiCorp, ScrollInManiCorp, InManiCorp),
		scrollRecipe(RecipeScrollAnTym, ScrollAnTym, AnTym),

		refine(RecipeRefineBlackPearl, BlackPearl, SulfurAsh),
		refine(RecipeRefineNightShade, NightShade, BloodMoss),
		refine(RecipeRefineMandrakeRoot, MandrakeRoot, Ginseng),
	}
}
