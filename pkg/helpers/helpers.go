package helpers

import (
	"reflect"

	"golang.org/x/exp/constraints"
	"golang.org/x/exp/rand"
)

// Max Generic Max function
func Max[T constraints.Ordered](a, b T) T {
	if a > b {
		return a
	}

	return b
}

// Min Generic Min function
func Min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	}

	return b
}

func AbsInt(a int) int {
	if a < 0 {
		return a * -1
	}

	return a
}

func IsOfTypeInterface(inst, iface interface{}) bool {
	instType := reflect.TypeOf(inst)
	ifaceType := reflect.TypeOf(&iface).Elem() // .Elem() to get the interface type

	return instType.Implements(ifaceType)
}

func OneInXOdds(odds int) bool {
	return rand.Intn(odds) == 0
}

func RandomIntInRange(min, max int) int {
	if min > max {
		// Programming bug: caller must validate min <= max constraint
		panic("min cannot be greater than max")
	}

	return rand.Intn(max-min+1) + min
}

func RandomUInt16InRange(min, max uint16) uint16 {
	if min > max {
		// Programming bug: caller must validate min <= max constraint
		panic("min cannot be greater than max")
	}
	// Compute the inclusive width in int: max-min+1 would wrap to zero in
	// uint16 for the full 0..65535 range and cause Intn(0) to panic.
	width := int(max) - int(min) + 1

	return uint16(int(min) + rand.Intn(width)) //nolint:gosec // The result is in [min,max], both uint16.
}

func PickOneOf[T any](a, b T) T {
	if OneInXOdds(2) {
		return a
	}

	return b
}

func FilterFromSlice[T any](slice []T, keep func(T) bool) []T {
	nIndex := 0 // next write position

	for _, v := range slice { // read position
		if keep(v) {
			slice[nIndex] = v // overwrite; OK even when nIndex==read index
			nIndex++
		}
	}

	return slice[:nIndex] // truncate tail
}

func RoundUp(f float32) int {
	return int(f + 0.5)
}

func Delete[T any](s []T, i int) []T {
	return append(s[:i], s[i+1:]...)
}
