package audio

import (
	"encoding/json"
	"slices"
)

func sortedKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	return keys
}

// AcousticFamilies returns Redux's controlled vocabulary, not generic SDK policy.
func AcousticFamilies() []string { return slices.Clone(acousticFamilies) }

func encodeAudioValue(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}

	return data
}

//nolint:ireturn // Preserve concrete validated metadata types while detaching pointers/maps.
func cloneAudioValue[T any](value T) T {
	var clone T
	if err := json.Unmarshal(encodeAudioValue(value), &clone); err != nil {
		panic(err)
	}

	return clone
}
