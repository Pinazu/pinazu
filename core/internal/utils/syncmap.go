package utils

import "sync"

// SyncMap provides a type-safe wrapper around sync.Map
type SyncMap[K comparable, V any] struct {
	m sync.Map
}

// NewSyncMap creates a new GenericSyncMap instance
func NewSyncMap[K comparable, V any]() *SyncMap[K, V] {
	return &SyncMap[K, V]{}
}

// Store sets the value for a key
func (g *SyncMap[K, V]) Store(key K, value V) {
	g.m.Store(key, value)
}

// Load returns the value stored in the map for a key, or the zero value if no
// value is present. The ok result indicates whether value was found in the map.
func (g *SyncMap[K, V]) Load(key K) (value V, ok bool) {
	v, ok := g.m.Load(key)
	if !ok {
		var zero V
		return zero, false
	}
	return v.(V), true
}

// LoadAndDelete deletes the value for a key, returning the previous value if any.
// The loaded result reports whether the key was present.
func (g *SyncMap[K, V]) LoadAndDelete(key K) (value V, loaded bool) {
	v, loaded := g.m.LoadAndDelete(key)
	if !loaded {
		var zero V
		return zero, false
	}
	return v.(V), true
}

// LoadOrStore returns the existing value for the key if present.
// Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false if stored.
func (g *SyncMap[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	v, loaded := g.m.LoadOrStore(key, value)
	return v.(V), loaded
}

// Delete deletes the value for a key
func (g *SyncMap[K, V]) Delete(key K) {
	g.m.Delete(key)
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
func (g *SyncMap[K, V]) Range(f func(key K, value V) bool) {
	g.m.Range(func(k, v any) bool {
		return f(k.(K), v.(V))
	})
}

// Swap swaps the value for a key and returns the previous value if any.
// The loaded result reports whether the key was present.
func (g *SyncMap[K, V]) Swap(key K, value V) (previous V, loaded bool) {
	v, loaded := g.m.Swap(key, value)
	if !loaded {
		var zero V
		return zero, false
	}
	return v.(V), true
}

// CompareAndSwap swaps the old and new values for key
// if the value stored in the map is equal to old.
// The old value must be of a comparable type.
func (g *SyncMap[K, V]) CompareAndSwap(key K, old, new V) bool {
	return g.m.CompareAndSwap(key, old, new)
}

// CompareAndDelete deletes the entry for key if its value is equal to old.
// The old value must be of a comparable type.
func (g *SyncMap[K, V]) CompareAndDelete(key K, old V) (deleted bool) {
	return g.m.CompareAndDelete(key, old)
}
