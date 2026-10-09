package references

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatBritannianCoordinatesUsesOriginalYThenXPunctuation(t *testing.T) {
	tests := []struct {
		name string
		x    Coordinate
		y    Coordinate
		want string
	}{
		{"origin", 0x00, 0x00, "A'A\", A'A\""},
		{"different nibbles", 0x2f, 0xa1, "K'B\", C'P\""},
		{"maximum", 0xff, 0xff, "P'P\", P'P\""},
		{"wraps to encoded byte", 0x100, -1, "P'P\", A'A\""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, FormatBritannianCoordinates(test.x, test.y))
		})
	}
}
