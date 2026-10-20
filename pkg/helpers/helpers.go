// Package helpers provides shared numeric, reflection, and slice utilities.
package helpers

import (
	"cmp"
	"reflect"
)

// Max returns the greater value, choosing b when the comparison is unordered.
//
//nolint:ireturn // The result retains the caller's concrete ordered type.
func Max[T cmp.Ordered](a, b T) T {
	if a > b {
		return a
	}

	return b
}

// Min returns the lesser value, choosing b when the comparison is unordered.
//
//nolint:ireturn // The result retains the caller's concrete ordered type.
func Min[T cmp.Ordered](a, b T) T {
	if a < b {
		return a
	}

	return b
}

// AbsInt returns the absolute value of a, subject to integer overflow.
func AbsInt(a int) int {
	if a < 0 {
		return a * -1
	}

	return a
}

// IsOfTypeInterface checks whether inst's type implements the interface holding iface.
func IsOfTypeInterface(inst, iface interface{}) bool {
	instType := reflect.TypeOf(inst)
	ifaceType := reflect.TypeOf(&iface).Elem() // .Elem() to get the interface type

	return instType.Implements(ifaceType)
}

// FilterFromSlice retains matching elements in order, reusing the input's backing array.
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

// RoundUp adds one half to f and truncates the result toward zero.
func RoundUp(f float32) int {
	const half = 0.5

	return int(f + half)
}

// Delete removes the element at i, reusing the input's backing array.
func Delete[T any](s []T, i int) []T {
	return append(s[:i], s[i+1:]...)
}
