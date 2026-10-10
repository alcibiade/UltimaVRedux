package references

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_WithinN_One(t *testing.T) {
	position1 := Position{X: 0, Y: 0}
	assert.True(t, position1.IsWithinN(&Position{X: 0, Y: 0}, 1))
	assert.True(t, position1.IsWithinN(&Position{X: 1, Y: 1}, 1))

	assert.False(t, position1.IsWithinN(&Position{X: 2, Y: 2}, 1))
	assert.False(t, position1.IsWithinN(&Position{X: 2, Y: 1}, 1))
	assert.False(t, position1.IsWithinN(&Position{X: 1, Y: 2}, 1))
}

func Test_WithinN_OneHundred(t *testing.T) {
	position1 := Position{X: 100, Y: 100}

	assert.True(t, position1.IsWithinN(&Position{X: 100, Y: 100}, 5))
	assert.True(t, position1.IsWithinN(&Position{X: 50, Y: 50}, 51))

	assert.False(t, position1.IsWithinN(&Position{X: 2, Y: 2}, 1))
	assert.False(t, position1.IsWithinN(&Position{X: 2, Y: 1}, 1))
	assert.False(t, position1.IsWithinN(&Position{X: 1, Y: 2}, 1))
}

func TestPositionAnimationHashWidensCoordinatesBeforePacking(t *testing.T) {
	tests := []struct {
		position Position
		want     int32
	}{
		{position: Position{X: 128, Y: 0}, want: 32768},
		{position: Position{X: 146, Y: 241}, want: 37617},
		{position: Position{X: 200, Y: 5}, want: 51205},
		{position: Position{X: 255, Y: 255}, want: 65535},
	}
	for _, test := range tests {
		assert.Equal(t, test.want, test.position.AnimationHash())
	}
}
