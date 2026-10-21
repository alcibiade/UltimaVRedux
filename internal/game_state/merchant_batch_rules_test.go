package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func setMerchantBatchMode(t *testing.T, state *GameState, batched bool) {
	t.Helper()
	before := state.GameConfig.GameplayConfiguration.BatchMerchantTheft
	state.GameConfig.GameplayConfiguration.BatchMerchantTheft = batched
	t.Cleanup(func() { state.GameConfig.GameplayConfiguration.BatchMerchantTheft = before })
}

func TestMerchantBatchEconomicUnits(t *testing.T) {
	for _, family := range []references.MerchantID{
		references.MerchantArmsBritain, references.MerchantReagentMoonglow, references.MerchantGuildPaws,
	} {
		for _, batched := range []bool{true, false} {
			for _, tight := range []bool{true, false} {
				t.Run(fmt.Sprintf("%s/batched=%t/tight=%t", family, batched, tight), func(t *testing.T) {
					state, npc := newOpenMerchantTestState(t, family)
					setMerchantBatchMode(t, state, batched)
					state.SetRandomSeed(208)
					state.ShadowlordState.AutomaticPresence = references.Faulinei
					state.PartyState.Inventory.Equipment.Set(references.Dagger, 0)
					state.PartyState.Inventory.Reagent.Set(references.Ginseng, 0)
					state.PartyState.Inventory.Provisions.Torches.Set(0)
					state.PartyState.Inventory.Gold.Set(9999)
					session, err := state.BeginMerchantSession(npc)
					require.NoError(t, err)
					var price int
					var bundle uint16
					switch family {
					case references.MerchantArmsBritain:
						quote, quoteErr := state.QuoteArmsBuy(session, references.Dagger)
						require.NoError(t, quoteErr)
						price, bundle = quote.UnitPrice, 1
					case references.MerchantReagentMoonglow:
						quote, quoteErr := state.QuoteReagentOffer(session, references.Ginseng)
						require.NoError(t, quoteErr)
						price, bundle = quote.PricePerBundle, quote.BundleQuantity
					case references.MerchantGuildPaws:
						quote, quoteErr := state.QuoteGuildOffer(session, references.Torches)
						require.NoError(t, quoteErr)
						price, bundle = quote.PricePerBundle, quote.BundleQuantity
					}
					gold := uint16(9999)
					if tight {
						gold = uint16(2 * price)
					}
					state.PartyState.Inventory.Gold.Set(gold)
					assert.Zero(t, state.GetRNG().Calls(), "quotes must not draw theft")
					var count, receipt, theft uint16
					var cost int
					switch family {
					case references.MerchantArmsBritain:
						result, buyErr := state.BuyArmsEquipment(session, ArmsBuyRequest{Equipment: references.Dagger, Quantity: 2})
						require.NoError(t, buyErr)
						count, receipt, theft, cost = result.Quantity, result.ActualReceipt, result.Theft, result.TotalCost
					case references.MerchantReagentMoonglow:
						result, buyErr := state.PurchaseReagentOffer(session, ReagentPurchaseRequest{Reagent: references.Ginseng, BundleCount: 2})
						require.NoError(t, buyErr)
						count, receipt, theft, cost = result.BundleCount, result.ActualReceipt, result.Theft, result.TotalPrice
						assert.Equal(t, bundle*2, result.RequestedQuantity)
					case references.MerchantGuildPaws:
						result, buyErr := state.PurchaseGuildOffer(session, GuildPurchaseRequest{Provision: references.Torches, BundleCount: 2})
						require.NoError(t, buyErr)
						count, receipt, theft, cost = result.BundleCount, result.ActualReceipt, result.Theft, result.TotalCost
						assert.Equal(t, receipt, result.OfferedQuantity)
					}
					expectedCount := uint16(2)
					if tight && !batched {
						expectedCount = 1
					}
					assert.Equal(t, expectedCount, count)
					assert.Equal(t, expectedCount*bundle, receipt)
					assert.Equal(t, int(expectedCount)*price, cost)
					rolls := uint64(expectedCount)
					if batched {
						rolls = 1
					}
					expected := beradrng.New(208)
					remaining := int(gold)
					var expectedTheft uint16
					for range rolls {
						payment := price
						if batched {
							payment = cost
						}
						remaining -= payment
						stolen := min(remaining, expected.Intn(64)+1)
						remaining -= stolen
						expectedTheft += uint16(stolen)
					}
					assert.Equal(t, expectedTheft, theft)
					assert.Equal(t, remaining, int(state.PartyState.Inventory.Gold.Get()))
					assert.Equal(t, rolls, state.GetRNG().Calls())
				})
			}
		}
	}
}

func TestSkaraBraeAllPaidThenFreshCharity(t *testing.T) {
	for _, gold := range []uint16{0, 44, 45, 50, 89, 90} {
		for _, batched := range []bool{true, false} {
			for _, faulinei := range []bool{false, true} {
				t.Run(fmt.Sprintf("gold=%d/batched=%t/faulinei=%t", gold, batched, faulinei), func(t *testing.T) {
					state, npc := newOpenMerchantTestState(t, references.MerchantHealerSkaraBrae)
					setMerchantBatchMode(t, state, batched)
					state.SetRandomSeed(208)
					if faulinei {
						state.ShadowlordState.AutomaticPresence = references.Faulinei
					}
					state.PartyState.Inventory.Gold.Set(gold)
					first := configureHealerTestCharacter(state, 0, party_state.Good, 10, 60)
					second := configureHealerTestCharacter(state, 1, party_state.Good, 20, 60)
					session, err := state.BeginMerchantSession(npc)
					require.NoError(t, err)
					quote, err := state.QuoteHealerServiceForAll(session, references.HealerHeal)
					require.NoError(t, err)
					paid := int(gold / 45)
					if paid == 0 {
						assert.Equal(t, HealerChargeCharitable, quote.Service.ChargeKind)
						assert.Equal(t, 2, quote.Service.RecipientCount)
					} else {
						assert.Equal(t, HealerChargePaid, quote.Service.ChargeKind)
						assert.Equal(t, paid, quote.Service.RecipientCount)
						assert.Equal(t, uint16(paid*45), quote.Service.Charge)
					}
					assert.Len(t, quote.Recipients, quote.Service.RecipientCount)
					assert.Zero(t, state.GetRNG().Calls())
					assert.Equal(t, gold, state.PartyState.Inventory.Gold.Get())
					assert.Equal(t, uint16(10), first.GetCurrentHp())
					result, err := state.ProvideHealerServiceToAll(session, references.HealerHeal)
					require.NoError(t, err)
					completed := quote.Service.RecipientCount
					if faulinei && !batched && paid == 2 {
						completed = 1
					}
					assert.Len(t, result.CharacterIndexes, completed)
					assert.Equal(t, uint16(60), first.GetCurrentHp())
					if paid > 0 {
						assert.Equal(t, uint16(completed*45), result.Charge)
					}
					if completed == 1 {
						assert.Equal(t, []int{0}, result.CharacterIndexes)
						assert.Equal(t, uint16(20), second.GetCurrentHp(), "no automatic charity during paid commit")
						fresh, quoteErr := state.QuoteHealerServiceForAll(session, references.HealerHeal)
						require.NoError(t, quoteErr)
						assert.Equal(t, HealerChargeCharitable, fresh.Service.ChargeKind)
						assert.Equal(t, 1, fresh.Service.RecipientCount)
						beforeCalls, beforeGold := state.GetRNG().Calls(), state.PartyState.Inventory.Gold.Get()
						charity, charityErr := state.ProvideHealerServiceToAll(session, references.HealerHeal)
						require.NoError(t, charityErr)
						assert.Equal(t, []int{1}, charity.CharacterIndexes)
						assert.Zero(t, charity.Charge)
						assert.Equal(t, beforeCalls, state.GetRNG().Calls())
						assert.Equal(t, beforeGold, state.PartyState.Inventory.Gold.Get())
					}
					assert.Equal(t, uint16(60), second.GetCurrentHp())
					if !faulinei || paid == 0 {
						assert.Zero(t, state.GetRNG().Calls())
					} else {
						assert.Equal(t, uint64(1), state.GetRNG().Calls())
					}
				})
			}
		}
	}
}

func TestShipOriginalBatchStopsAfterTheft(t *testing.T) {
	for _, batched := range []bool{true, false} {
		t.Run(fmt.Sprintf("batched=%t", batched), func(t *testing.T) {
			state, npc := newOpenShipShopTestState(t, references.MerchantShipJhelom)
			setMerchantBatchMode(t, state, batched)
			state.SetRandomSeed(208)
			state.ShadowlordState.AutomaticPresence = references.Faulinei
			state.PartyState.Inventory.Gold.Set(680)
			position := shipShopDockPosition(t, state, references.Jhelom)
			frigate := addShipShopVehicle(t, state, references.FrigateVehicle, position, false, 2)
			session, err := state.BeginMerchantSession(npc)
			require.NoError(t, err)
			shop, err := state.GetShipShopState(session)
			require.NoError(t, err)
			require.Equal(t, 340, shop.SkiffQuote.Price)
			request := shipPurchaseRequest(shop, shop.SkiffQuote)
			request.Quantity = 2
			request.ReviewedTotalPrice = 680
			assert.Zero(t, state.GetRNG().Calls())
			result, err := state.PurchaseShip(session, request)
			require.NoError(t, err)
			count := 1
			if batched {
				count = 2
			}
			assert.Equal(t, count, result.Quantity)
			assert.Equal(t, 340*count, result.Price)
			assert.Equal(t, 2+count, frigate.GetVehicleDetails().GetSkiffQuantity())
			assert.Equal(t, uint64(1), state.GetRNG().Calls())
		})
	}
}

func TestPaidHealerAndSkiffBatchDrawOncePerEconomicUnit(t *testing.T) {
	for _, batched := range []bool{true, false} {
		t.Run(fmt.Sprintf("batched=%t", batched), func(t *testing.T) {
			state, npc := newOpenMerchantTestState(t, references.MerchantHealerSkaraBrae)
			setMerchantBatchMode(t, state, batched)
			state.SetRandomSeed(208)
			state.ShadowlordState.AutomaticPresence = references.Faulinei
			state.PartyState.Inventory.Gold.Set(9999)
			configureHealerTestCharacter(state, 0, party_state.Good, 10, 60)
			configureHealerTestCharacter(state, 1, party_state.Good, 20, 60)
			session, err := state.BeginMerchantSession(npc)
			require.NoError(t, err)
			result, err := state.ProvideHealerServiceToAll(session, references.HealerHeal)
			require.NoError(t, err)
			require.Equal(t, []int{0, 1}, result.CharacterIndexes)
			require.Equal(t, uint16(90), result.Charge)
			rolls := uint64(2)
			if batched {
				rolls = 1
			}
			rng := beradrng.New(208)
			var theft uint16
			for range rolls {
				theft += uint16(rng.Intn(64) + 1)
			}
			assert.Equal(t, rolls, state.GetRNG().Calls())
			assert.Equal(t, theft, result.Theft)
			assert.Equal(t, uint16(9999-90)-theft, result.GoldAfterTheft)

			shipState, shipNPC := newOpenShipShopTestState(t, references.MerchantShipJhelom)
			setMerchantBatchMode(t, shipState, batched)
			shipState.SetRandomSeed(208)
			shipState.ShadowlordState.AutomaticPresence = references.Faulinei
			shipState.PartyState.Inventory.Gold.Set(9999)
			position := shipShopDockPosition(t, shipState, references.Jhelom)
			addShipShopVehicle(t, shipState, references.FrigateVehicle, position, false, 2)
			shipSession, err := shipState.BeginMerchantSession(shipNPC)
			require.NoError(t, err)
			shop, err := shipState.GetShipShopState(shipSession)
			require.NoError(t, err)
			request := shipPurchaseRequest(shop, shop.SkiffQuote)
			request.Quantity = 2
			request.ReviewedTotalPrice = shop.SkiffQuote.Price * 2
			shipResult, err := shipState.PurchaseShip(shipSession, request)
			require.NoError(t, err)
			assert.Equal(t, 2, shipResult.Quantity)
			assert.Equal(t, 4, shipResult.SkiffCountAfter)
			assert.Equal(t, request.ReviewedTotalPrice, shipResult.Price)
			assert.Equal(t, rolls, shipState.GetRNG().Calls())
			assert.Equal(t, theft, shipResult.Theft)
		})
	}
}

func TestMerchantBatchModeDoesNotChangeFreeCareOrResurrectionCharity(t *testing.T) {
	for _, batched := range []bool{true, false} {
		for _, merchant := range []references.MerchantID{references.MerchantHealerMinoc, references.MerchantHealerSkaraBrae} {
			t.Run(fmt.Sprintf("%s/batched=%t", merchant, batched), func(t *testing.T) {
				state, npc := newOpenMerchantTestState(t, merchant)
				setMerchantBatchMode(t, state, batched)
				state.SetRandomSeed(208)
				state.ShadowlordState.AutomaticPresence = references.Faulinei
				state.PartyState.Inventory.Gold.Set(0)
				configureHealerTestCharacter(state, 0, party_state.Poisoned, 10, 60)
				configureHealerTestCharacter(state, 1, party_state.Poisoned, 20, 60)
				session, err := state.BeginMerchantSession(npc)
				require.NoError(t, err)
				for _, service := range []references.HealerService{references.HealerCure, references.HealerHeal} {
					result, serviceErr := state.ProvideHealerServiceToAll(session, service)
					require.NoError(t, serviceErr)
					assert.Equal(t, []int{0, 1}, result.CharacterIndexes)
					assert.Zero(t, result.Charge)
					assert.Zero(t, state.GetRNG().Calls())
				}
				configureHealerTestCharacter(state, 1, party_state.Dead, 0, 60)
				_, err = state.ProvideHealerServiceToAll(session, references.HealerResurrect)
				require.ErrorIs(t, err, ErrHealerInsufficientFunds)
				assert.True(t, state.PartyState.GetCharacter(1).IsDead())
				assert.Zero(t, state.GetRNG().Calls())
			})
		}
	}
}
