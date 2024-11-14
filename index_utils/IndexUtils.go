// Package index_utils contains type definitions necessary for database indices, and any data structure relying on key-value pairs
// in the application
package index_utils

import "cmp"

// Pair contains a comparable key K, and a value V of any type; we use this as a way to simulate pythonic tuples or the Pair in C++
type Pair[K cmp.Ordered, V any] struct {
	Key   K
	Value V
}

// UpdateCheck is a function signature utilized by the skiplist, determining whether to update or insert a value based how UpdateCheck is implemented
type UpdateCheck[K cmp.Ordered, V any] func(curKey K, curVal V, exists bool) (newVal V, err error)
