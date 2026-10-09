package game_state

// CalculateMerchantBuyPrice applies the original Intelligence modifier with
// the exact original shop rules integer operation ordering.
// See ref:MERCH-0002.
func CalculateMerchantBuyPrice(basePrice, intelligence int) int {
	return basePrice + ((100-3*intelligence)*basePrice)/100
}
