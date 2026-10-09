package map_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtendTorchMinutesAddsAndSaturates(t *testing.T) {
	lighting := Lighting{}
	lighting.RestoreTorchMinutes(100)

	assert.Equal(t, 220, lighting.ExtendTorchMinutes(120, 255))
	assert.Equal(t, 255, lighting.ExtendTorchMinutes(120, 255))
}

func TestExtendTorchMinutesRejectsInvalidInputs(t *testing.T) {
	lighting := Lighting{}
	assert.PanicsWithValue(t, "torch extension minutes cannot be negative", func() {
		lighting.ExtendTorchMinutes(-1, 255)
	})
	assert.PanicsWithValue(t, "torch maximum cannot be negative", func() {
		lighting.ExtendTorchMinutes(1, -1)
	})
}
