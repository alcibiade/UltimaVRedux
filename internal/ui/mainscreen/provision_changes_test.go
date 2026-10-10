package mainscreen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestProvisionChangesCoverageCappedGrantsLossesAndNoops(t *testing.T) {
	inventory := party_state.NewInventory()
	quantities := []party_state.ItemQuantity{
		&inventory.Provisions.Food, &inventory.Gold, &inventory.Provisions.Keys,
		&inventory.Provisions.Gems, &inventory.Provisions.Torches, &inventory.Provisions.SkullKeys,
	}
	identities := []references.Provision{
		references.Food, references.Gold, references.Key, references.Gem, references.Torches, references.SkullKeys,
	}
	names := []string{"Food", "Gold", "Keys", "Gems", "Torches", "Skull keys"}
	var changes ProvisionChanges
	for _, quantity := range quantities {
		quantity.Set(65535)
		require.True(t, quantity.DecrementByOne())
	}
	changes.Observe(SnapshotProvisions(inventory), 0)
	assertNoProvisionChanges(t, changes)
	before := SnapshotProvisions(inventory)
	for index, quantity := range quantities {
		require.NoError(t, inventory.PutItemInInventory(&references.ItemAndQuantity{Item: identities[index], Quantity: 20}))
		require.True(t, quantity.DecrementByOne())
		quantity.IncrementBy(0)
		require.False(t, quantity.DecrementBy(65535))
		quantity.Set(quantity.Get())
	}
	changes.Observe(SnapshotProvisions(inventory), 10)
	for index, row := range changes.Rows() {
		assert.Equal(t, names[index], row.Name)
		assert.Equal(t, uint64(1), row.Gained, "capped grant %s", row.Name)
		assert.Equal(t, uint64(1), row.Lost, "same-update loss %s", row.Name)
		assert.Equal(t, before[index].Total, row.Total, "equal final balance must not cancel feedback")
	}
	rows := changes.Rows()
	changes.Observe(SnapshotProvisions(inventory), 11)
	assert.Equal(t, rows, changes.Rows(), "unchanged observation neither repeats nor refreshes")
}

func TestProvisionChangesRapidFoodConsumptionAndOppositeChanges(t *testing.T) {
	inventory := party_state.NewInventory()
	inventory.Provisions.Food.Set(500)
	inventory.Gold.Set(100)
	var changes ProvisionChanges
	changes.Reset(SnapshotProvisions(inventory), 0)
	for tick := int64(1); tick <= 100; tick++ {
		inventory.Provisions.Food.IncrementBy(6)
		starving, _ := inventory.ConsumeMealTimeFood(6, 6)
		require.False(t, starving)
		changes.Observe(SnapshotProvisions(inventory), tick)
	}
	row := changes.Rows()[0]
	assert.Equal(t, uint64(600), row.Gained)
	assert.Equal(t, uint64(600), row.Lost)
	assert.Equal(t, uint16(500), row.Total)
	assert.Equal(t, int64(100)+ProvisionChangeHoldMs, row.ExpiresMs)
	inventory.Gold.DecrementBy(20)
	changes.Observe(SnapshotProvisions(inventory), 500)
	changes.Observe(SnapshotProvisions(inventory), 100+ProvisionChangeHoldMs)
	assert.Empty(t, changes.Rows()[0].Name, "unrelated gold spending must not prolong food")
	assert.Equal(t, uint64(20), changes.Rows()[1].Lost)
	changes.Observe(SnapshotProvisions(inventory), 500+ProvisionChangeHoldMs)
	assertNoProvisionChanges(t, changes)
	inventory.Provisions.Food.IncrementByOne()
	changes.Observe(SnapshotProvisions(inventory), 7000)
	assert.Equal(t, uint64(1), changes.Rows()[0].Gained, "new window starts fresh")
	assert.Zero(t, changes.Rows()[0].Lost)
}

func TestProvisionChangesBaselineAndClockResetAreSilent(t *testing.T) {
	inventory := party_state.NewInventory()
	inventory.Gold.Set(5000)
	var changes ProvisionChanges
	changes.Observe(SnapshotProvisions(inventory), 100)
	assertNoProvisionChanges(t, changes)
	inventory.Gold.DecrementBy(10)
	changes.Observe(SnapshotProvisions(inventory), 110)
	require.Equal(t, uint64(10), changes.Rows()[1].Lost)
	changes.Observe(SnapshotProvisions(inventory), 0)
	assertNoProvisionChanges(t, changes)
	inventory.Gold.IncrementByOne()
	changes.Observe(SnapshotProvisions(inventory), 1)
	require.Equal(t, uint64(1), changes.Rows()[1].Gained)
	loaded := party_state.NewInventory()
	loaded.Gold.Set(123)
	changes.Observe(SnapshotProvisions(loaded), 2)
	assertNoProvisionChanges(t, changes)
	loaded.Gold.Set(9999)
	changes.Reset(SnapshotProvisions(loaded), 3)
	changes.Observe(SnapshotProvisions(loaded), 4)
	assertNoProvisionChanges(t, changes)
}

func assertNoProvisionChanges(t *testing.T, changes ProvisionChanges) {
	t.Helper()
	for _, row := range changes.Rows() {
		assert.Empty(t, row.Name)
		assert.Zero(t, row.Gained)
		assert.Zero(t, row.Lost)
	}
}
