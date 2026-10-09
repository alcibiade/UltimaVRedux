package mainscreen

import (
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

const (
	provisionChangeCount = 6
	// ProvisionChangeHoldMs is an idle timeout, refreshed only for a changed resource.
	ProvisionChangeHoldMs int64 = 6000
)

// ProvisionBalance is a detached current quantity and gross movement snapshot.
type ProvisionBalance struct {
	Total    uint16
	Activity party_state.QuantityActivity
}

// ProvisionSnapshot has a fixed presentation order: food, gold, keys, gems,
// torches, skull keys. No inventory entries are allocated while observing it.
type ProvisionSnapshot [provisionChangeCount]ProvisionBalance

// SnapshotProvisions reads all six stacks without changing gameplay.
func SnapshotProvisions(inventory *party_state.Inventory) ProvisionSnapshot {
	return ProvisionSnapshot{
		{Total: inventory.Provisions.Food.Get(), Activity: inventory.Provisions.Food.Activity()},
		{Total: inventory.Gold.Get(), Activity: inventory.Gold.Activity()},
		{Total: inventory.Provisions.Keys.Get(), Activity: inventory.Provisions.Keys.Activity()},
		{Total: inventory.Provisions.Gems.Get(), Activity: inventory.Provisions.Gems.Activity()},
		{Total: inventory.Provisions.Torches.Get(), Activity: inventory.Provisions.Torches.Activity()},
		{Total: inventory.Provisions.SkullKeys.Get(), Activity: inventory.Provisions.SkullKeys.Activity()},
	}
}

// ProvisionChange shows gross gains/losses separately, never their cancelling net.
type ProvisionChange struct {
	Name      string
	Gained    uint64
	Lost      uint64
	Total     uint16
	ExpiresMs int64
}

// ProvisionChanges is a bounded presentation model, not a gameplay event queue.
// Repeated changes coalesce by resource; distinct resources share one display.
type ProvisionChanges struct {
	baseline ProvisionSnapshot
	rows     [provisionChangeCount]ProvisionChange
	lastMs   int64
	ready    bool
}

// Reset silently adopts an initialized/loaded state and forgets old feedback.
func (p *ProvisionChanges) Reset(snapshot ProvisionSnapshot, nowMs int64) {
	*p = ProvisionChanges{baseline: snapshot, rows: [provisionChangeCount]ProvisionChange{}, lastMs: nowMs, ready: true}
}

// Observe runs after authoritative updates. Counters preserve multiple mutations
// even when the ending balance equals the starting balance. Expiry uses only the
// supplied scene clock; draws and unchanged observations never refresh it.
func (p *ProvisionChanges) Observe(snapshot ProvisionSnapshot, nowMs int64) {
	if !p.ready || nowMs < p.lastMs || activityReset(p.baseline, snapshot) {
		p.Reset(snapshot, nowMs)

		return
	}
	labels := [provisionChangeCount]string{"Food", "Gold", "Keys", "Gems", "Torches", "Skull keys"}
	for index, current := range snapshot {
		row := &p.rows[index]
		if nowMs >= row.ExpiresMs {
			*row = ProvisionChange{Name: "", Gained: 0, Lost: 0, Total: 0, ExpiresMs: 0}
		}
		previous := p.baseline[index]
		gained := current.Activity.Gained - previous.Activity.Gained
		lost := current.Activity.Lost - previous.Activity.Lost
		if gained != 0 || lost != 0 {
			row.Name = labels[index]
			row.Gained += gained
			row.Lost += lost
			row.ExpiresMs = nowMs + ProvisionChangeHoldMs
		}
		row.Total = current.Total
	}
	p.baseline = snapshot
	p.lastMs = nowMs
}

func activityReset(before, after ProvisionSnapshot) bool {
	for index, current := range after {
		if current.Activity.Gained < before[index].Activity.Gained || current.Activity.Lost < before[index].Activity.Lost {
			return true
		}
	}

	return false
}

// Rows returns a bounded detached view; empty names identify inactive resources.
func (p *ProvisionChanges) Rows() [provisionChangeCount]ProvisionChange {
	return p.rows
}

// AbsorbReportedPickup excludes an already-reported acquisition from the next
// observation. Other gains and all losses in the same update remain observable.
func (p *ProvisionChanges) AbsorbReportedPickup(item references.Provision, acquired uint16) {
	if !p.ready || acquired == 0 {
		return
	}
	identities := [provisionChangeCount]references.Provision{
		references.Food, references.Gold, references.Key, references.Gem, references.Torches, references.SkullKeys,
	}
	for index, identity := range identities {
		if identity == item {
			p.baseline[index].Activity.Gained += uint64(acquired)

			return
		}
	}
}

// PauseExpiry preserves reading time while scene geometry obscures the panel.
func (p *ProvisionChanges) PauseExpiry(delta int64) {
	if delta <= 0 {
		return
	}
	for index := range p.rows {
		if p.rows[index].Name != "" {
			p.rows[index].ExpiresMs += delta
		}
	}
}
