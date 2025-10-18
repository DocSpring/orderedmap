# orderedmap

A thread-safe, generic ordered map for Go that maintains insertion order while providing O(1) operations for lookups, deletes, and moves.

[![Test](https://github.com/DocSpring/orderedmap/actions/workflows/test.yml/badge.svg)](https://github.com/DocSpring/orderedmap/actions/workflows/test.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/DocSpring/orderedmap.svg)](https://pkg.go.dev/github.com/DocSpring/orderedmap)
[![Go Report Card](https://goreportcard.com/badge/github.com/DocSpring/orderedmap)](https://goreportcard.com/report/github.com/DocSpring/orderedmap)
[![codecov](https://codecov.io/gh/DocSpring/orderedmap/branch/main/graph/badge.svg)](https://codecov.io/gh/DocSpring/orderedmap)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Features

- **Insertion order preservation** - Iterates in the order items were added (unlike Go's built-in maps)
- **O(1) operations** - Fast lookups, deletes, and moves using map + doubly-linked list
- **Thread-safe** - All operations use internal locking (RWMutex)
- **Zero-value usable** - No constructor required: `var om OrderedMap[K,V]` just works
- **Generic** - Works with any comparable key type and any value type
- **Snapshot-based iteration** - Range/RangeBreak take snapshots, preventing deadlocks even if callbacks modify the map

## Installation

```bash
go get github.com/DocSpring/orderedmap
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/DocSpring/orderedmap"
)

func main() {
    // Create a new ordered map
    om := orderedmap.NewOrderedMap[string, int]()

    // Or use zero value (no constructor needed)
    var om2 orderedmap.OrderedMap[string, int]

    // Add entries
    om.Set("zebra", 26)
    om.Set("alpha", 1)
    om.Set("beta", 2)

    // Get values (O(1) lookup)
    val, ok := om.Get("alpha")
    fmt.Println(val, ok) // 1 true

    // Iterate in insertion order
    om.Range(func(key string, value int) {
        fmt.Printf("%s: %d\n", key, value)
    })
    // Output:
    // zebra: 26
    // alpha: 1
    // beta: 2
}
```

## Why OrderedMap?

Go's built-in `map[K]V` has **random iteration order** by design. This causes problems when you need:

- **Stable UI rendering** - Prevent flickering when displaying map contents in a terminal UI
- **Predictable output** - Generate consistent results across runs
- **FIFO/insertion-order semantics** - Process items in the order they were added
- **Deterministic testing** - Avoid flaky tests from random map iteration

## Implementation

OrderedMap uses a **map + doubly-linked list** hybrid approach:

- **map[K]*node** - O(1) lookups by key
- **Doubly-linked list** - O(1) deletes and moves, preserves insertion order
- **Cached length** - O(1) len() operation

This combination provides optimal performance for all operations except indexed access (At), which is O(n).

## API

### Creating

```go
// Constructor (recommended)
om := orderedmap.NewOrderedMap[string, int]()

// Zero value (also works)
var om orderedmap.OrderedMap[string, int]
```

### Basic Operations

```go
// Set - Insert or update (O(1))
om.Set("key", 42)

// Get - Retrieve value (O(1))
val, ok := om.Get("key")

// Has - Check existence (O(1))
if om.Has("key") { ... }

// Delete - Remove entry (O(1))
deleted := om.Delete("key")

// DeleteWithValue - Remove entry and return its value (O(1))
val, deleted := om.DeleteWithValue("key")

// Len - Get size (O(1))
count := om.Len()
```

### Iteration

```go
// Range - Iterate over all entries (insertion order)
om.Range(func(key string, value int) {
    fmt.Printf("%s: %d\n", key, value)
})

// RangeBreak - Iterate with early exit
om.RangeBreak(func(key string, value int) bool {
    fmt.Printf("%s: %d\n", key, value)
    return key != "stop" // return false to break
})
```

**Important:** Range and RangeBreak take a **snapshot** before iterating, so the callback can safely modify the map without causing deadlocks.

**Note:** The snapshot copies both keys and values. For value types (int, struct, etc.), mutations after the snapshot are not reflected in the iteration. For pointer or reference types (*struct, map, slice), changes to the underlying data will be visible.

### Zero-Allocation Iteration

```go
// RangeLocked - Iterate without allocating a snapshot
om.RangeLocked(func(key string, value int) {
    // Process items
})
```

**⚠️ WARNING:** The callback in `RangeLocked` **MUST NOT** call any OrderedMap methods or deadlock will occur. Use `Range()` if you need to modify the map during iteration.

### Access First/Last

```go
// Front - Get first entry (O(1))
key, val, ok := om.Front()

// Back - Get last entry (O(1))
key, val, ok := om.Back()

// At - Get entry at index (O(n))
key, val, ok := om.At(5)
```

### Queue Operations

```go
// PopFront - Remove and return first entry (O(1))
key, val, ok := om.PopFront()

// PopBack - Remove and return last entry (O(1))
key, val, ok := om.PopBack()

// MoveToEnd - Move key to end of order (O(1))
moved := om.MoveToEnd("key")
```

### Cache Patterns

```go
// GetOrSet - Atomic get-or-create (O(1))
val, existed := om.GetOrSet("key", func() int {
    return expensiveComputation()
})
```

**⚠️ WARNING:** The `mk` function in `GetOrSet` is called **while holding the write lock**. Keep it fast and simple to avoid blocking other operations.

### Bulk Operations

```go
// Keys - Get all keys (returns copy)
keys := om.Keys()

// Values - Get all values (returns copy)
values := om.Values()

// Clear - Remove all entries (preserves capacity)
om.Clear()

// Reset - Remove all entries and free memory
om.Reset()
```

## Performance Characteristics

| Operation | Complexity | Notes |
|-----------|-----------|-------|
| Set | **O(1)** | Amortized due to map growth |
| Get | **O(1)** | Hash map lookup |
| Has | **O(1)** | Hash map lookup |
| Delete | **O(1)** | Doubly-linked list removal |
| DeleteWithValue | **O(1)** | Returns value during removal |
| PopFront | **O(1)** | Remove head of list |
| PopBack | **O(1)** | Remove tail of list |
| MoveToEnd | **O(1)** | Relink nodes |
| GetOrSet | **O(1)** | Atomic get-or-create |
| Len | **O(1)** | Cached length |
| Range | **O(n)** | Snapshot + iteration |
| RangeLocked | **O(n)** | No allocation, holds lock |
| Keys/Values | **O(n)** | Returns defensive copy |
| Front/Back | **O(1)** | Direct pointer access |
| At | **O(n)** | Must traverse list |

## Thread Safety

All operations are thread-safe via internal `sync.RWMutex`:

- **Reads** (Get, Has, Len, Range, Keys, Values, Front, Back) use RLock
- **Writes** (Set, Delete, Clear, Reset, Pop*, MoveToEnd) use Lock

```go
var om orderedmap.OrderedMap[int, string]

// Safe concurrent access from multiple goroutines
go func() { om.Set(1, "a") }()
go func() { om.Set(2, "b") }()
go func() { val, _ := om.Get(1) }()
```

### Re-entrant Callbacks

Range and RangeBreak take snapshots **before** calling your callback, releasing the lock during iteration. This means:

✅ **Safe** - Callbacks can modify the map without deadlock:

```go
om.Range(func(k string, v int) {
    om.Set("new_"+k, v*2) // No deadlock!
    om.Delete("old")      // No deadlock!
})
```

⚠️ **Snapshot semantics** - The callback sees the state at the time Range was called, not live updates

❌ **RangeLocked is NOT safe** for re-entrant calls:

```go
om.RangeLocked(func(k string, v int) {
    om.Set("new", 1) // DEADLOCK!
})
```

## Use Cases

### Terminal UI - Stable Rendering

```go
// Prevent UI flickering in terminal applications
var checks orderedmap.OrderedMap[string, Status]

// Updates maintain order
checks.Set("database", StatusOK)
checks.Set("cache", StatusOK)
checks.Set("api", StatusFail)

// Always renders in same order (no flicker)
checks.Range(func(name string, status Status) {
    fmt.Printf("  %s %s\n", statusIcon(status), name)
})
```

### LRU Cache

```go
type LRUCache struct {
    items   orderedmap.OrderedMap[string, []byte]
    maxSize int
}

func (c *LRUCache) Get(key string) ([]byte, bool) {
    if val, ok := c.items.Get(key); ok {
        c.items.MoveToEnd(key) // O(1) - move to back for LRU
        return val, true
    }
    return nil, false
}

func (c *LRUCache) Set(key string, value []byte) {
    c.items.Set(key, value)
    c.items.MoveToEnd(key) // O(1) - newest at back

    // Evict oldest if over capacity
    for c.items.Len() > c.maxSize {
        c.items.PopFront() // O(1) - remove oldest
    }
}
```

### FIFO Queue with Deduplication

```go
// Process unique items in order received
var queue orderedmap.OrderedMap[string, Task]

queue.Set(task.ID, task) // Dedupe by ID
for {
    id, task, ok := queue.PopFront() // O(1)
    if !ok { break }
    process(task)
}
```

### Configuration Manager

```go
type ConfigManager struct {
    settings orderedmap.OrderedMap[string, string]
}

func (cm *ConfigManager) Load(file string) {
    // Settings maintain file order for display
    cm.settings.Set("timeout", "30s")
    cm.settings.Set("retries", "3")
    cm.settings.Set("endpoint", "https://api.example.com")
}

func (cm *ConfigManager) Display() {
    cm.settings.Range(func(key, val string) {
        fmt.Printf("%s = %s\n", key, val)
    })
}
```

## Comparison with Alternatives

| Feature | OrderedMap | `map[K]V` | `container/list` |
|---------|-----------|-----------|------------------|
| Insertion order | ✅ | ❌ (random) | ✅ |
| O(1) lookup | ✅ | ✅ | ❌ (O(n)) |
| O(1) delete | ✅ | ✅ | ✅ |
| O(1) move | ✅ | ❌ | ✅ |
| Thread-safe | ✅ | ❌ | ❌ |
| Generic | ✅ | ✅ | ✅ (Go 1.18+) |
| Zero value usable | ✅ | ✅ | ❌ |

## Requirements

- Go 1.18+ (uses generics)

## Testing

```bash
# Run tests
go test -v

# Run tests with race detector
go test -v -race

# Check coverage
go test -v -coverprofile=coverage.txt
go tool cover -html=coverage.txt

# Run benchmarks
go test -bench=. -benchmem
```

Current test coverage: **100%**

## Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure 100% test coverage
5. Run `go fmt` and `go vet`
6. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) file for details

## Author

DocSpring, Inc. ([@DocSpring](https://github.com/DocSpring))

## Acknowledgments

Inspired by the need for stable UI rendering in terminal applications and the lack of a simple, thread-safe ordered map with O(1) operations in Go's standard library.
