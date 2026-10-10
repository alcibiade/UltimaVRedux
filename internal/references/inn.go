package references

// InnReference contains the fixed source data for one active original inn.
type InnReference struct {
	MerchantID    MerchantID
	BasePrice     int
	GuestCapacity int
	SleepLocation PlayerLocation
	WakeLocation  PlayerLocation
}

// Original inn prices, rooms, and rest coordinates (ref:MERCH-0009) in
// MerchantDefinition.CatalogRow order. Wake positions are the source's
// unconditional one-tile east movement after morning restoration.
//
//nolint:gochecknoglobals // Canonical original inn rows are immutable after initialization.
var innsByCatalogRow = [6]InnReference{
	{
		MerchantID: MerchantInnBritain, BasePrice: 2, GuestCapacity: 3,
		SleepLocation: PlayerLocation{Location: Britain, Position: Position{X: 21, Y: 10}, Floor: 0},
		WakeLocation:  PlayerLocation{Location: Britain, Position: Position{X: 22, Y: 10}, Floor: 0},
	},
	{
		MerchantID: MerchantInnJhelom, BasePrice: 3, GuestCapacity: 4,
		SleepLocation: PlayerLocation{Location: Jhelom, Position: Position{X: 15, Y: 7}, Floor: 1},
		WakeLocation:  PlayerLocation{Location: Jhelom, Position: Position{X: 16, Y: 7}, Floor: 1},
	},
	{
		MerchantID: MerchantInnSkaraBrae, BasePrice: 2, GuestCapacity: 3,
		SleepLocation: PlayerLocation{Location: SkaraBrae, Position: Position{X: 25, Y: 9}, Floor: 0},
		WakeLocation:  PlayerLocation{Location: SkaraBrae, Position: Position{X: 26, Y: 9}, Floor: 0},
	},
	{
		MerchantID: MerchantInnNorthBritanny, BasePrice: 3, GuestCapacity: 2,
		SleepLocation: PlayerLocation{Location: NorthBritanny, Position: Position{X: 20, Y: 1}, Floor: 0},
		WakeLocation:  PlayerLocation{Location: NorthBritanny, Position: Position{X: 21, Y: 1}, Floor: 0},
	},
	{
		MerchantID: MerchantInnPaws, BasePrice: 2, GuestCapacity: 2,
		SleepLocation: PlayerLocation{Location: Paws, Position: Position{X: 27, Y: 6}, Floor: 0},
		WakeLocation:  PlayerLocation{Location: Paws, Position: Position{X: 28, Y: 6}, Floor: 0},
	},
	{
		MerchantID: MerchantInnBuccaneersDen, BasePrice: 3, GuestCapacity: 2,
		SleepLocation: PlayerLocation{Location: BuccaneersDen, Position: Position{X: 7, Y: 26}, Floor: 0},
		WakeLocation:  PlayerLocation{Location: BuccaneersDen, Position: Position{X: 8, Y: 26}, Floor: 0},
	},
}

// Inn returns one detached canonical inn row.
func (r *MerchantReferences) Inn(merchantID MerchantID) (InnReference, bool) {
	if r == nil {
		return InnReference{}, false
	}

	merchant, ok := r.Definition(merchantID)
	if !ok || merchant.Family != MerchantFamilyInn ||
		merchant.CatalogRow < 0 || merchant.CatalogRow >= len(innsByCatalogRow) {
		return InnReference{}, false
	}
	inn := innsByCatalogRow[merchant.CatalogRow]
	if inn.MerchantID != merchant.ID {
		return InnReference{}, false
	}

	return inn, true
}
