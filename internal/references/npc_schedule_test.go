package references

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/datetime"
)

func TestNPCScheduleDesiredSlotAndMerchantParity(t *testing.T) {
	schedule := NPCSchedule{Time: [4]byte{6, 12, 18, 0}}
	tests := []struct {
		hour byte
		want NPCScheduleSlot
	}{
		{hour: 0, want: 1},
		{hour: 5, want: 1},
		{hour: 6, want: 0},
		{hour: 11, want: 0},
		{hour: 12, want: 1},
		{hour: 17, want: 1},
		{hour: 18, want: 2},
		{hour: 23, want: 2},
	}
	for _, test := range tests {
		got := schedule.DesiredSlot(datetime.UltimaDate{Hour: test.hour})
		if got != test.want {
			t.Errorf("DesiredSlot(hour %d) = %d, want %d", test.hour, got, test.want)
		}
		if got.IsMerchantSlot() != (test.want == 1) {
			t.Errorf("slot %d merchant parity does not match original odd-slot rule", got)
		}
	}
}
