// Package orderedmap provides a thread-safe, generic ordered map that maintains insertion order
// while providing O(1) lookups. Unlike Go's built-in map, OrderedMap iterates in the order items
// were added, preventing UI flickering and enabling predictable, deterministic behavior.
package orderedmap

import "sync"

// node represents an element in the doubly-linked list
type node[K comparable, V any] struct {
	key   K
	value V
	prev  *node[K, V]
	next  *node[K, V]
}

// OrderedMap maintains insertion order while providing O(1) lookups, deletes, and moves
// This prevents UI flickering from Go's random map iteration order
//
// Thread-safe: All operations use internal locking
// Zero value is usable: var om OrderedMap[K,V] works without initialization
// All operations are O(1) except Range which is O(n)
// Range/RangeBreak: Iterate over snapshot at call time, not live data (safe for re-entrant calls)
type OrderedMap[K comparable, V any] struct {
	mu    sync.RWMutex      // Protects all fields
	index map[K]*node[K, V] // Key -> node pointer (for O(1) lookup)
	head  *node[K, V]       // First node in list
	tail  *node[K, V]       // Last node in list
	len   int               // Number of entries
}

// NewOrderedMap creates a new ordered map
// Note: The zero value is also usable without calling this constructor
func NewOrderedMap[K comparable, V any]() *OrderedMap[K, V] {
	return &OrderedMap[K, V]{
		index: make(map[K]*node[K, V]),
	}
}

// ensureInit initializes the index map if needed (makes zero value usable)
func (om *OrderedMap[K, V]) ensureInit() {
	if om.index == nil {
		om.index = make(map[K]*node[K, V])
	}
}

// appendToEnd adds a node to the end of the list
// Caller must hold om.mu.Lock()
func (om *OrderedMap[K, V]) appendToEnd(n *node[K, V]) {
	if om.tail == nil {
		om.head, om.tail = n, n
		return
	}
	n.prev = om.tail
	n.next = nil
	om.tail.next = n
	om.tail = n
}

// Set adds or updates a key-value pair (O(1))
// If key exists, updates the value in-place (preserves order)
// If key is new, appends to the end
func (om *OrderedMap[K, V]) Set(key K, value V) {
	om.mu.Lock()
	defer om.mu.Unlock()
	om.ensureInit()

	if n, exists := om.index[key]; exists {
		// Update existing value in-place
		n.value = value
		return
	}

	// Create new node and append
	n := &node[K, V]{
		key:   key,
		value: value,
	}
	om.appendToEnd(n)
	om.index[key] = n
	om.len++
}

// Get retrieves a value by key (O(1) lookup)
func (om *OrderedMap[K, V]) Get(key K) (V, bool) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	if n, exists := om.index[key]; exists {
		return n.value, true
	}
	var zero V
	return zero, false
}

// Has checks if a key exists (O(1) lookup)
func (om *OrderedMap[K, V]) Has(key K) bool {
	om.mu.RLock()
	defer om.mu.RUnlock()

	_, exists := om.index[key]
	return exists
}

// Len returns the number of entries (O(1))
func (om *OrderedMap[K, V]) Len() int {
	om.mu.RLock()
	defer om.mu.RUnlock()

	return om.len
}

// Delete removes a key, preserving insertion order (O(1) complexity)
// Returns true if the key existed and was deleted
func (om *OrderedMap[K, V]) Delete(key K) bool {
	om.mu.Lock()
	defer om.mu.Unlock()

	n, exists := om.index[key]
	if !exists {
		return false
	}

	om.removeNode(n)
	delete(om.index, key)
	om.len--

	return true
}

// DeleteWithValue removes a key and returns its value (O(1) complexity)
// Returns the value and true if the key existed, or zero value and false otherwise
func (om *OrderedMap[K, V]) DeleteWithValue(key K) (V, bool) {
	om.mu.Lock()
	defer om.mu.Unlock()

	n, exists := om.index[key]
	if !exists {
		var zero V
		return zero, false
	}

	om.removeNode(n)
	delete(om.index, key)
	om.len--

	return n.value, true
}

// removeNode removes a node from the doubly-linked list
// Caller must hold om.mu.Lock()
func (om *OrderedMap[K, V]) removeNode(n *node[K, V]) {
	if p := n.prev; p != nil {
		p.next = n.next
	} else {
		om.head = n.next
	}
	if q := n.next; q != nil {
		q.prev = n.prev
	} else {
		om.tail = n.prev
	}
	n.prev, n.next = nil, nil // help GC
}

// Clear removes all entries while preserving allocated capacity
func (om *OrderedMap[K, V]) Clear() {
	om.mu.Lock()
	defer om.mu.Unlock()

	// Clear map
	for k := range om.index {
		delete(om.index, k)
	}
	om.head = nil
	om.tail = nil
	om.len = 0
}

// Reset removes all entries and frees allocated memory
func (om *OrderedMap[K, V]) Reset() {
	om.mu.Lock()
	defer om.mu.Unlock()

	om.index = nil
	om.head = nil
	om.tail = nil
	om.len = 0
}

// Range iterates over all entries IN INSERTION ORDER at the time of call
// Takes a snapshot before iterating, so fn can safely call other methods without deadlock
//
// Note: Snapshots both keys and values. Mutations to the value V after the snapshot
// are not reflected in the iteration. For pointer or reference types, changes to the
// underlying data will be visible.
func (om *OrderedMap[K, V]) Range(fn func(key K, value V)) {
	om.mu.RLock()
	// Snapshot keys and values to avoid holding lock during callbacks
	keys := make([]K, 0, om.len)
	values := make([]V, 0, om.len)
	for n := om.head; n != nil; n = n.next {
		keys = append(keys, n.key)
		values = append(values, n.value)
	}
	om.mu.RUnlock()

	for i := range keys {
		fn(keys[i], values[i])
	}
}

// RangeBreak iterates in insertion order and stops when fn returns false
// Takes a snapshot before iterating, so fn can safely call other methods without deadlock
//
// Note: Snapshots both keys and values. Mutations to the value V after the snapshot
// are not reflected in the iteration. For pointer or reference types, changes to the
// underlying data will be visible.
func (om *OrderedMap[K, V]) RangeBreak(fn func(key K, value V) bool) {
	om.mu.RLock()
	// Snapshot keys and values to avoid holding lock during callbacks
	keys := make([]K, 0, om.len)
	values := make([]V, 0, om.len)
	for n := om.head; n != nil; n = n.next {
		keys = append(keys, n.key)
		values = append(values, n.value)
	}
	om.mu.RUnlock()

	for i := range keys {
		if !fn(keys[i], values[i]) {
			return
		}
	}
}

// RangeLocked iterates while holding the RLock (no snapshot allocation)
// WARNING: fn MUST NOT call any OrderedMap methods or deadlock will occur
// Use Range() if you need to modify the map during iteration
func (om *OrderedMap[K, V]) RangeLocked(fn func(key K, value V)) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	for n := om.head; n != nil; n = n.next {
		fn(n.key, n.value)
	}
}

// RangeBreakLocked iterates while holding the RLock and stops when fn returns false
// WARNING: fn MUST NOT call any OrderedMap methods or deadlock will occur
// Use RangeBreak() if you need to modify the map during iteration
func (om *OrderedMap[K, V]) RangeBreakLocked(fn func(key K, value V) bool) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	for n := om.head; n != nil; n = n.next {
		if !fn(n.key, n.value) {
			return
		}
	}
}

// Keys returns a copy of all keys in insertion order
func (om *OrderedMap[K, V]) Keys() []K {
	om.mu.RLock()
	defer om.mu.RUnlock()

	keys := make([]K, 0, om.len)
	for n := om.head; n != nil; n = n.next {
		keys = append(keys, n.key)
	}
	return keys
}

// Values returns a copy of all values in insertion order
func (om *OrderedMap[K, V]) Values() []V {
	om.mu.RLock()
	defer om.mu.RUnlock()

	values := make([]V, 0, om.len)
	for n := om.head; n != nil; n = n.next {
		values = append(values, n.value)
	}
	return values
}

// Front returns the first key-value pair, or zero values if empty
func (om *OrderedMap[K, V]) Front() (K, V, bool) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	if om.head == nil {
		var zeroK K
		var zeroV V
		return zeroK, zeroV, false
	}
	return om.head.key, om.head.value, true
}

// Back returns the last key-value pair, or zero values if empty
func (om *OrderedMap[K, V]) Back() (K, V, bool) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	if om.tail == nil {
		var zeroK K
		var zeroV V
		return zeroK, zeroV, false
	}
	return om.tail.key, om.tail.value, true
}

// At returns the key-value pair at the given index, or zero values if out of bounds (O(n))
func (om *OrderedMap[K, V]) At(i int) (K, V, bool) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	if i < 0 || i >= om.len {
		var zeroK K
		var zeroV V
		return zeroK, zeroV, false
	}

	n := om.head
	for j := 0; j < i; j++ {
		n = n.next
	}
	return n.key, n.value, true
}

// PopFront removes and returns the first key-value pair (O(1) complexity)
// Returns zero values and false if the map is empty
func (om *OrderedMap[K, V]) PopFront() (K, V, bool) {
	om.mu.Lock()
	defer om.mu.Unlock()

	if om.head == nil {
		var zeroK K
		var zeroV V
		return zeroK, zeroV, false
	}

	n := om.head
	om.removeNode(n)
	delete(om.index, n.key)
	om.len--

	return n.key, n.value, true
}

// PopBack removes and returns the last key-value pair (O(1) complexity)
// Returns zero values and false if the map is empty
func (om *OrderedMap[K, V]) PopBack() (K, V, bool) {
	om.mu.Lock()
	defer om.mu.Unlock()

	if om.tail == nil {
		var zeroK K
		var zeroV V
		return zeroK, zeroV, false
	}

	n := om.tail
	om.removeNode(n)
	delete(om.index, n.key)
	om.len--

	return n.key, n.value, true
}

// MoveToEnd moves the given key to the end of the insertion order (O(1) complexity)
// Returns true if the key existed and was moved, false otherwise
// Useful for implementing LRU-like behavior
func (om *OrderedMap[K, V]) MoveToEnd(key K) bool {
	om.mu.Lock()
	defer om.mu.Unlock()

	n, exists := om.index[key]
	if !exists {
		return false
	}

	// Already at end
	if n == om.tail {
		return true
	}

	om.removeNode(n)
	om.appendToEnd(n)

	return true
}

// GetOrSet returns the existing value if key exists, otherwise calls mk to create
// a new value, stores it, and returns it. The second return value indicates whether
// the value already existed (true) or was newly created (false).
//
// WARNING: mk is called while holding the write lock. Keep it fast and simple.
// This avoids double lookups when implementing caching patterns.
func (om *OrderedMap[K, V]) GetOrSet(key K, mk func() V) (V, bool) {
	om.mu.Lock()
	defer om.mu.Unlock()
	om.ensureInit()

	if n, exists := om.index[key]; exists {
		return n.value, true
	}

	// Create new node and append
	v := mk()
	n := &node[K, V]{
		key:   key,
		value: v,
	}
	om.appendToEnd(n)
	om.index[key] = n
	om.len++
	return v, false
}
