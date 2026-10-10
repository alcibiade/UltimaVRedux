package widgets

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func TestMixAll48KeyboardRequests(t *testing.T) {
	// Independent identity table: neither display labels nor the widget lookup
	// generate the expected requests. In particular Mani != Vas Mani.
	cases := []struct {
		keys  string
		spell references.Spell
	}{
		{"IL", references.InLor}, {"GP", references.GravPor}, {"AZ", references.AnZu},
		{"AN", references.AnNox}, {"M", references.Mani}, {"AY", references.AnYlem},
		{"AS", references.AnSanct}, {"AXC", references.AnXenCorp}, {"RH", references.RelHur},
		{"IW", references.InWis}, {"KX", references.KalXen}, {"IXM", references.InXenMani},
		{"VL", references.VasLor}, {"VF", references.VasFlam}, {"IFG", references.InFlamGrav},
		{"ING", references.InNoxGrav}, {"IZG", references.InZuGrav}, {"IP", references.InPor},
		{"AG", references.AnGrav}, {"IS", references.InSanct}, {"ISG", references.InSanctGrav},
		{"UP", references.UusPor}, {"DP", references.DesPor}, {"WQ", references.WisQuas},
		{"IBX", references.InBetXen}, {"AEP", references.AnExPor}, {"IEP", references.InExPor},
		{"VM", references.VasMani}, {"IZ", references.InZu}, {"RT", references.RelTym},
		{"IVPY", references.InVasPorYlem}, {"QAW", references.QuasAnWis}, {"IA", references.InAn},
		{"WAY", references.WisAnYlem}, {"AXE", references.AnXenEx}, {"RXB", references.RelXenBet},
		{"SL", references.SanctLor}, {"XC", references.XenCorp}, {"IQX", references.InQuasXen},
		{"IQW", references.InQuasWis}, {"INH", references.InNoxHur}, {"IQC", references.InQuasCorp},
		{"IMC", references.InManiCorp}, {"KXC", references.KalXenCorp}, {"IVGC", references.InVasGravCorp},
		{"IFH", references.InFlamHur}, {"VRP", references.VasRelPor}, {"AT", references.AnTym},
	}
	require.Len(t, cases, 48)
	catalog := testMixCatalog()
	for _, test := range cases {
		t.Run(test.keys, func(t *testing.T) {
			held := make(map[ebiten.Key]bool)
			keyboard := input.NewKeyboardWithKeyStateSource(-1, func(key ebiten.Key) bool { return held[key] })
			reagents := make([]ReagentInfo, 8)
			for index := range reagents {
				reagents[index] = ReagentInfo{Reagent: references.Reagent(index), Quantity: 99}
			}
			dialog := NewReagentMixingDialog(keyboard, catalog, reagents)
			press := func(key ebiten.Key) {
				held[key] = true
				dialog.Update()
				delete(held, key)
				keyboard.ClearFrameConsumption()
				dialog.Update()
			}
			for _, letter := range test.keys {
				press(ebiten.KeyA + ebiten.Key(letter-'A'))
			}
			require.False(t, dialog.HasPendingMixRequest(), "typing cannot commit")
			press(ebiten.KeyEnter)
			require.False(t, dialog.HasPendingMixRequest(), "choosing a spell cannot commit")
			press(ebiten.KeyEnter)
			require.True(t, dialog.HasPendingMixRequest())
			request := dialog.ConsumePendingMixRequest()
			require.Equal(t, test.spell, request.Spell)
			require.Equal(t, 1, request.Quantity)
			require.Equal(t, references.GetSpellReagents(test.spell), request.ReagentMask)
			for _, reagent := range dialog.reagentInventory {
				require.Equal(t, 99, reagent.Quantity)
			}
		})
	}
}

func TestMixKeyboardQuantityClearAndSpellEdit(t *testing.T) {
	held := make(map[ebiten.Key]bool)
	keyboard := input.NewKeyboardWithKeyStateSource(-1, func(key ebiten.Key) bool { return held[key] })
	dialog := NewReagentMixingDialog(keyboard, testMixCatalog(), []ReagentInfo{{Reagent: references.SulfurAsh, Quantity: 99}})
	press := func(key ebiten.Key) {
		held[key] = true
		dialog.Update()
		delete(held, key)
		keyboard.ClearFrameConsumption()
		dialog.Update()
	}
	press(ebiten.KeyI)
	press(ebiten.KeyL)
	press(ebiten.KeyEnter)
	press(ebiten.Key2)
	press(ebiten.Key3)
	require.Equal(t, "23", dialog.quantityBuffer)
	press(ebiten.KeyDelete)
	require.Empty(t, dialog.quantityBuffer)
	require.Equal(t, references.InLor.ID(), dialog.spell.ID)
	press(ebiten.Key5)
	press(ebiten.KeyBackspace)
	require.Equal(t, 1, dialog.getEffectiveQuantity())
	require.Equal(t, references.InLor.ID(), dialog.spell.ID)
	require.False(t, dialog.HasPendingMixRequest())
}

func TestMixMaximumThatFitsOnlyEditsConfirmedRecipeQuantity(t *testing.T) {
	for _, test := range []struct {
		prepared          uint16
		ingredients, want int
	}{
		{92, 18, 7}, {92, 3, 3}, {0, 99, 99}, {99, 18, 1}, {0, 0, 0},
	} {
		catalog := testMixCatalog()
		catalog[references.InLor].Prepared = test.prepared
		held := map[ebiten.Key]bool{}
		keyboard := input.NewKeyboardWithKeyStateSource(-1, func(key ebiten.Key) bool { return held[key] })
		dialog := NewReagentMixingDialog(keyboard, catalog, []ReagentInfo{{Reagent: references.SulfurAsh, Quantity: test.ingredients}})
		dialog.ChooseSpell(references.InLor, true)
		held[ebiten.KeyControl], held[ebiten.KeyF] = true, true
		dialog.Update()
		require.Equal(t, test.want, dialog.getEffectiveQuantity())
		require.False(t, dialog.HasPendingMixRequest())
		require.Equal(t, test.ingredients, dialog.reagentInventory[0].Quantity)
		if test.prepared == 99 {
			require.Contains(t, dialog.notice, "No room")
		} else if test.ingredients == 0 {
			require.Contains(t, dialog.notice, "Missing ingredients")
		} else {
			clear(held)
			keyboard.ClearFrameConsumption()
			keyboard.SimulateKeyPress(ebiten.KeyEnter)
			dialog.Update()
			require.Equal(t, test.want, dialog.ConsumePendingMixRequest().Quantity)
		}
	}
	dialog := NewReagentMixingDialog(newWidgetTestKeyboard(-1), testMixCatalog(), []ReagentInfo{{Reagent: references.SulfurAsh, Quantity: 18}})
	dialog.ChooseSpell(references.InLor, true)
	dialog.selection = 0
	dialog.fillUsefulQuantity()
	require.Contains(t, dialog.notice, "confirmed recipe")
	require.False(t, dialog.HasPendingMixRequest())
}

func testMixCatalog() []game_state.ZStatsSpell {
	catalog := make([]game_state.ZStatsSpell, 0, references.TotalSpellCount)
	for spell := range references.Spell(references.TotalSpellCount) {
		info := references.GetSpellInfo(spell)
		catalog = append(catalog, game_state.ZStatsSpell{ID: spell.ID(), Name: info.Name, Syllables: info.Name,
			Description: info.Description, EffectLabel: references.SpellEffectLabel(spell), RecipeMask: references.GetSpellReagents(spell),
			FormulaState: string(game_state.SpellFormulaConfirmed)})
	}
	return catalog
}

func TestMixSingleDigitReplacementAndQuantityBoundaries(t *testing.T) {
	for _, maximum := range []int{0, 6, 12, 99} {
		dialog := NewReagentMixingDialog(newWidgetTestKeyboard(0), testMixCatalog(), []ReagentInfo{{Reagent: references.SulfurAsh, Quantity: maximum}})
		dialog.ChooseSpell(references.InLor, true)
		dialog.handleQuantityInput(6)
		dialog.handleQuantityInput(5)
		switch maximum {
		case 0:
			require.Equal(t, 0, dialog.getEffectiveQuantity())
		case 6:
			require.Equal(t, 5, dialog.getEffectiveQuantity())
		case 12:
			require.Equal(t, 6, dialog.getEffectiveQuantity(), "65 is rejected, not silently clamped")
		case 99:
			require.Equal(t, 65, dialog.getEffectiveQuantity())
		}
	}
}

func TestMixRapidRepeatedAndKeypadDigits(t *testing.T) {
	held := map[ebiten.Key]bool{}
	keyboard := input.NewKeyboardWithKeyStateSource(165, func(key ebiten.Key) bool { return held[key] })
	dialog := NewReagentMixingDialog(keyboard, testMixCatalog(), []ReagentInfo{{Reagent: references.SulfurAsh, Quantity: 99}})
	dialog.ChooseSpell(references.InLor, true)
	press := func(key ebiten.Key) {
		held[key] = true
		dialog.Update()
		dialog.Update()
		clear(held)
		keyboard.ClearFrameConsumption()
	}
	press(ebiten.Key1)
	press(ebiten.Key1)
	require.Equal(t, 11, dialog.getEffectiveQuantity(), "separate presses register even inside the movement delay")
	press(ebiten.KeyNumpad9)
	press(ebiten.KeyNumpad9)
	require.Equal(t, 99, dialog.getEffectiveQuantity())
	press(ebiten.KeyBackspace)
	require.Equal(t, 1, dialog.getEffectiveQuantity())
	press(ebiten.Key0)
	require.Equal(t, 1, dialog.getEffectiveQuantity())
	press(ebiten.KeyNumpad1)
	press(ebiten.KeyNumpad0)
	press(ebiten.KeyNumpadEnter)
	require.Equal(t, 10, dialog.ConsumePendingMixRequest().Quantity)
}

func TestMixFinderCancelRestoresPreparation(t *testing.T) {
	held := map[ebiten.Key]bool{}
	keyboard := input.NewKeyboardWithKeyStateSource(-1, func(key ebiten.Key) bool { return held[key] })
	dialog := NewReagentMixingDialog(keyboard, testMixCatalog(), []ReagentInfo{{Reagent: references.SulfurAsh, Quantity: 6}})
	dialog.ChooseSpell(references.InLor, true)
	dialog.handleQuantityInput(5)
	held[ebiten.KeyControl], held[ebiten.KeyB] = true, true
	dialog.Update()
	clear(held)
	keyboard.ClearFrameConsumption()
	require.False(t, dialog.preparing)
	keyboard.SimulateKeyPress(ebiten.KeyEscape)
	dialog.Update()
	require.True(t, dialog.preparing)
	require.Equal(t, 5, dialog.getEffectiveQuantity())
	require.Equal(t, references.InLor.ID(), dialog.spell.ID)
	require.False(t, dialog.IsDismissed())
}
