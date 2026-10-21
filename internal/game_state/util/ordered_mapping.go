// Package util holds small generic helpers for game-state data.
package util

// IDToString pairs a comparable ID with its friendly display name.
type IDToString[T comparable] struct {
	ID           T
	FriendlyName string
}

// OrderedMapping is an ordered list of ID-to-name entries that is searched linearly.
// It is inefficient, but handy for small fixed sets that need a string for a generic
// typed value while keeping their declared order. It is basically a stripped down
// `map` that orders stuff.
type OrderedMapping[T comparable] []IDToString[T]

// GetByID returns the entry whose ID equals thing, or nil when there is none.
func (o *OrderedMapping[T]) GetByID(thing T) *IDToString[T] {
	for _, val := range *o {
		if val.ID == thing {
			return &val
		}
	}

	return nil
}
