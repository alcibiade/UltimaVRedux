// Package util holds small generic helpers for game-state data.
package util

// IdToString pairs a comparable ID with its friendly display name.
type IdToString[T comparable] struct {
	Id           T
	FriendlyName string
}

// OrderedMapping is an ordered list of ID-to-name entries that is searched linearly.
// It is inefficient, but handy for small fixed sets that need a string for a generic
// typed value while keeping their declared order. It is basically a stripped down
// `map` that orders stuff.
type OrderedMapping[T comparable] []IdToString[T]

// GetById returns the entry whose Id equals thing, or nil when there is none.
func (o *OrderedMapping[T]) GetById(thing T) *IdToString[T] {
	for _, val := range *o {
		if val.Id == thing {
			return &val
		}
	}

	return nil
}
