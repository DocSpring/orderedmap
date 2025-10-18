package orderedmap

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestOrderedMap_GetOrSet_RaceWithLongMk tests concurrent GetOrSet with long-running mk function
func TestOrderedMap_GetOrSet_RaceWithLongMk(t *testing.T) {
	om := NewOrderedMap[string, int]()

	var mkCallCount atomic.Int32

	mk := func() int {
		mkCallCount.Add(1)
		time.Sleep(10 * time.Millisecond) // Simulate expensive computation
		return 42
	}

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	results := make([]struct {
		val     int
		existed bool
	}, goroutines)

	// Launch multiple goroutines trying to GetOrSet the same key
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			val, existed := om.GetOrSet("key", mk)
			results[idx].val = val
			results[idx].existed = existed
		}(i)
	}

	wg.Wait()

	// Verify:
	// 1. All goroutines got the same value
	// 2. mk was called exactly once (lock protects it)
	// 3. Exactly one goroutine saw existed=false, rest saw existed=true
	existedFalseCount := 0
	for i := range results {
		if results[i].val != 42 {
			t.Errorf("Goroutine %d got wrong value: %d", i, results[i].val)
		}
		if !results[i].existed {
			existedFalseCount++
		}
	}

	if mkCallCount.Load() != 1 {
		t.Errorf("Expected mk called exactly once, got %d", mkCallCount.Load())
	}

	if existedFalseCount != 1 {
		t.Errorf("Expected exactly one goroutine to see existed=false, got %d", existedFalseCount)
	}

	// Verify readers can access while GetOrSet is running
	var wg2 sync.WaitGroup
	wg2.Add(2)

	go func() {
		defer wg2.Done()
		om.GetOrSet("slow_key", func() int {
			time.Sleep(50 * time.Millisecond)
			return 99
		})
	}()

	go func() {
		defer wg2.Done()
		time.Sleep(10 * time.Millisecond) // Let GetOrSet acquire lock first
		// This read should work concurrently with the slow mk() execution
		val, ok := om.Get("key")
		if !ok || val != 42 {
			t.Errorf("Expected to read key=42 while other GetOrSet runs, got %d, ok=%v", val, ok)
		}
	}()

	wg2.Wait()
}

// Helper functions for invariant checking
func checkLenInvariant[K comparable, V any](t *testing.T, om *OrderedMap[K, V], name string) {
	t.Helper()
	if om.len != len(om.index) {
		t.Errorf("%s: len=%d != len(index)=%d", name, om.len, len(om.index))
	}
}

func checkNodeCountInvariant[K comparable, V any](t *testing.T, om *OrderedMap[K, V], name string) {
	t.Helper()
	count := 0
	for n := om.head; n != nil; n = n.next {
		count++
	}
	if count != om.len {
		t.Errorf("%s: counted %d nodes in list, len=%d", name, count, om.len)
	}
}

func checkIndexPointsToListNodes[K comparable, V any](t *testing.T, om *OrderedMap[K, V], name string) {
	t.Helper()
	for key, node := range om.index {
		found := false
		for n := om.head; n != nil; n = n.next {
			if n == node {
				found = true
				if n.key != key {
					t.Errorf("%s: index[%v] points to node with key %v", name, key, n.key)
				}
				break
			}
		}
		if !found {
			t.Errorf("%s: index[%v] points to node not in list", name, key)
		}
	}
}

func checkListIntegrity[K comparable, V any](t *testing.T, om *OrderedMap[K, V], name string) {
	t.Helper()
	// Check head/tail boundaries
	if om.head != nil && om.head.prev != nil {
		t.Errorf("%s: head.prev should be nil", name)
	}
	if om.tail != nil && om.tail.next != nil {
		t.Errorf("%s: tail.next should be nil", name)
	}
	// Forward links
	for n := om.head; n != nil; n = n.next {
		if n.next != nil && n.next.prev != n {
			t.Errorf("%s: broken forward link: node.next.prev != node", name)
		}
	}
	// Backward links
	for n := om.tail; n != nil; n = n.prev {
		if n.prev != nil && n.prev.next != n {
			t.Errorf("%s: broken backward link: node.prev.next != node", name)
		}
	}
}

// TestOrderedMap_Invariants checks all internal invariants
func TestOrderedMap_Invariants(t *testing.T) {
	om := NewOrderedMap[string, int]()

	checkInvariants := func(name string) {
		t.Helper()
		om.mu.RLock()
		defer om.mu.RUnlock()

		checkLenInvariant(t, om, name)
		checkNodeCountInvariant(t, om, name)
		checkIndexPointsToListNodes(t, om, name)
		checkListIntegrity(t, om, name)
	}

	checkInvariants("initial empty")

	// Test operations and verify invariants after each
	om.Set("a", 1)
	checkInvariants("after Set(a)")

	om.Set("b", 2)
	om.Set("c", 3)
	checkInvariants("after Set(b,c)")

	om.Set("b", 22) // Update
	checkInvariants("after Set(b) update")

	om.Delete("a")
	checkInvariants("after Delete(a) - head")

	om.Set("d", 4)
	om.Set("e", 5)
	om.Delete("d") // Middle delete
	checkInvariants("after Delete(d) - middle")

	om.Delete("e") // Tail delete
	checkInvariants("after Delete(e) - tail")

	om.MoveToEnd("b")
	checkInvariants("after MoveToEnd(b)")

	om.PopFront()
	checkInvariants("after PopFront")

	om.Set("f", 6)
	om.PopBack()
	checkInvariants("after PopBack")

	om.Clear()
	checkInvariants("after Clear")

	om.Set("x", 1)
	om.Reset()
	if om.index != nil {
		// After Reset, check if re-initialized
		checkInvariants("after Reset and re-init")
	}
}

// FuzzOrderedMap tests random operation sequences
func FuzzOrderedMap(f *testing.F) {
	// Seed with some interesting sequences
	f.Add([]byte{0, 1, 2, 3, 4, 5}) // Set operations
	f.Add([]byte{0, 0, 1, 1, 2, 2}) // Duplicate sets
	f.Add([]byte{0, 2, 0, 2, 0, 2}) // Set, delete, set, delete
	f.Add([]byte{0, 0, 0, 3, 3, 3}) // Sets then clears

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 {
			return
		}

		om := NewOrderedMap[int, int]()

		// Interpret bytes as operations
		for i, b := range data {
			key := int(b) % 10 // Use keys 0-9
			op := i % 7        // 7 operation types

			switch op {
			case 0: // Set
				om.Set(key, i)
			case 1: // Delete
				om.Delete(key)
			case 2: // MoveToEnd
				om.MoveToEnd(key)
			case 3: // PopFront
				om.PopFront()
			case 4: // PopBack
				om.PopBack()
			case 5: // Clear
				om.Clear()
			case 6: // GetOrSet
				om.GetOrSet(key, func() int { return i })
			}

			// Check invariants after every operation
			om.mu.RLock()
			actualLen := om.len
			indexLen := len(om.index)

			// Count nodes
			count := 0
			for n := om.head; n != nil; n = n.next {
				count++
				if count > 100 { // Prevent infinite loop
					break
				}
			}
			om.mu.RUnlock()

			if actualLen != indexLen || actualLen != count {
				t.Fatalf("Invariant violated after op %d: len=%d, len(index)=%d, node_count=%d",
					op, actualLen, indexLen, count)
			}
		}
	})
}

// TestOrderedMap_PropertyOrderPreserved tests that insertion order is always preserved
func TestOrderedMap_PropertyOrderPreserved(t *testing.T) {
	om := NewOrderedMap[string, int]()

	// Insert in specific order
	insertOrder := []string{"z", "a", "m", "b", "y"}
	for i, key := range insertOrder {
		om.Set(key, i)
	}

	// Verify order preserved
	keys := om.Keys()
	for i, key := range keys {
		if key != insertOrder[i] {
			t.Errorf("Order not preserved: expected %v, got %v", insertOrder, keys)
			break
		}
	}

	// Update should NOT change order
	om.Set("a", 100)
	om.Set("z", 200)

	keys = om.Keys()
	for i, key := range keys {
		if key != insertOrder[i] {
			t.Errorf("Order changed after update: expected %v, got %v", insertOrder, keys)
			break
		}
	}
}

// TestOrderedMap_PropertyDeleteMaintainsList tests Delete correctly maintains doubly-linked list
func TestOrderedMap_PropertyDeleteMaintainsList(t *testing.T) {
	testCases := []struct {
		name       string
		deleteKey  string
		deleteFunc func(*OrderedMap[string, int], string)
	}{
		{"Delete head", "first", func(om *OrderedMap[string, int], k string) { om.Delete(k) }},
		{"Delete middle", "second", func(om *OrderedMap[string, int], k string) { om.Delete(k) }},
		{"Delete tail", "third", func(om *OrderedMap[string, int], k string) { om.Delete(k) }},
		{"PopFront", "first", func(om *OrderedMap[string, int], _ string) { om.PopFront() }},
		{"PopBack", "third", func(om *OrderedMap[string, int], _ string) { om.PopBack() }},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			om := NewOrderedMap[string, int]()
			om.Set("first", 1)
			om.Set("second", 2)
			om.Set("third", 3)

			tc.deleteFunc(om, tc.deleteKey)

			// Check doubly-linked list integrity
			om.mu.RLock()
			defer om.mu.RUnlock()

			// Forward traversal
			count := 0
			for n := om.head; n != nil; n = n.next {
				count++
				if n.next != nil && n.next.prev != n {
					t.Errorf("After %s: broken forward link at key=%s", tc.name, n.key)
				}
			}

			if count != om.len {
				t.Errorf("After %s: counted %d nodes, expected %d", tc.name, count, om.len)
			}

			// Backward traversal
			count = 0
			for n := om.tail; n != nil; n = n.prev {
				count++
				if n.prev != nil && n.prev.next != n {
					t.Errorf("After %s: broken backward link at key=%s", tc.name, n.key)
				}
			}

			if count != om.len {
				t.Errorf("After %s: counted %d nodes backward, expected %d", tc.name, count, om.len)
			}
		})
	}
}

// TestOrderedMap_Reentrancy_RangeSafe tests that Range allows reentrant modifications
func TestOrderedMap_Reentrancy_RangeSafe(t *testing.T) {
	om := NewOrderedMap[string, int]()
	om.Set("a", 1)
	om.Set("b", 2)
	om.Set("c", 3)

	// Range should allow modifications without deadlock
	callCount := 0
	om.Range(func(k string, v int) {
		callCount++
		// These should NOT deadlock
		om.Set("new_"+k, v*10)
		om.Delete("b")
		_, _ = om.Get("a")
		_ = om.Has("c")
	})

	if callCount != 3 {
		t.Errorf("Expected Range to iterate 3 times, got %d", callCount)
	}

	// Verify modifications occurred
	if !om.Has("new_a") {
		t.Error("Expected new_a to exist")
	}
	if om.Has("b") {
		t.Error("Expected b to be deleted")
	}
}

// TestOrderedMap_BigN_MemoryAndAllocs tests with large dataset
func TestOrderedMap_BigN_MemoryAndAllocs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping big-N test in short mode")
	}

	om := NewOrderedMap[int, int]()
	const N = 10_000 // Reduced from 1M for faster tests

	// Insert entries
	for i := 0; i < N; i++ {
		om.Set(i, i*2)
	}

	if om.Len() != N {
		t.Errorf("Expected length %d, got %d", N, om.Len())
	}

	// Random deletes (10% of entries)
	rng := rand.New(rand.NewSource(42))
	deleteCount := N / 10
	for i := 0; i < deleteCount; i++ {
		key := rng.Intn(N)
		om.Delete(key)
	}

	// Verify invariants still hold
	om.mu.RLock()
	if om.len != len(om.index) {
		t.Errorf("Invariants violated: len=%d, len(index)=%d", om.len, len(om.index))
	}
	om.mu.RUnlock()

	// Test allocation count for Get (should be zero)
	allocs := testing.AllocsPerRun(100, func() {
		_, _ = om.Get(42)
	})
	if allocs > 0 {
		t.Errorf("Get should not allocate, got %.2f allocs/run", allocs)
	}

	// Test allocation count for Set on existing key (should be zero)
	om.Set(12345, 99)
	allocs = testing.AllocsPerRun(100, func() {
		om.Set(12345, 100)
	})
	if allocs > 0 {
		t.Errorf("Set (update) should not allocate, got %.2f allocs/run", allocs)
	}

	// Test RangeLocked doesn't allocate
	allocs = testing.AllocsPerRun(10, func() {
		count := 0
		om.RangeLocked(func(_, _ int) {
			count++
			if count > 100 {
				return // Early exit to keep test fast
			}
		})
	})
	if allocs > 0 {
		t.Errorf("RangeLocked should not allocate, got %.2f allocs/run", allocs)
	}
}

// Benchmark Get operation
func BenchmarkOrderedMap_Get(b *testing.B) {
	sizes := []int{10, 100, 1000, 10000}

	for _, size := range sizes {
		om := NewOrderedMap[int, int]()
		for i := 0; i < size; i++ {
			om.Set(i, i*2)
		}

		b.Run(benchName("size", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = om.Get(i % size)
			}
		})
	}
}

// Benchmark Set operation (new keys)
func BenchmarkOrderedMap_Set_New(b *testing.B) {
	sizes := []int{10, 100, 1000, 10000}

	for _, size := range sizes {
		b.Run(benchName("size", size), func(b *testing.B) {
			om := NewOrderedMap[int, int]()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				om.Set(i%size, i)
			}
		})
	}
}

// Benchmark Set operation (updates)
func BenchmarkOrderedMap_Set_Update(b *testing.B) {
	sizes := []int{10, 100, 1000, 10000}

	for _, size := range sizes {
		om := NewOrderedMap[int, int]()
		for i := 0; i < size; i++ {
			om.Set(i, i*2)
		}

		b.Run(benchName("size", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				om.Set(i%size, i)
			}
		})
	}
}

// Benchmark Range operation
func BenchmarkOrderedMap_Range(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		om := NewOrderedMap[int, int]()
		for i := 0; i < size; i++ {
			om.Set(i, i*2)
		}

		b.Run(benchName("size", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				om.Range(func(_, _ int) {})
			}
		})
	}
}

// Benchmark RangeLocked operation
func BenchmarkOrderedMap_RangeLocked(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		om := NewOrderedMap[int, int]()
		for i := 0; i < size; i++ {
			om.Set(i, i*2)
		}

		b.Run(benchName("size", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				om.RangeLocked(func(_, _ int) {})
			}
		})
	}
}

// Benchmark Delete operation
func BenchmarkOrderedMap_Delete(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		b.Run(benchName("size", size), func(b *testing.B) {
			b.StopTimer()
			for i := 0; i < b.N; i++ {
				om := NewOrderedMap[int, int]()
				for j := 0; j < size; j++ {
					om.Set(j, j*2)
				}
				b.StartTimer()
				om.Delete(size / 2) // Delete middle element
				b.StopTimer()
			}
		})
	}
}

// Benchmark MoveToEnd operation
func BenchmarkOrderedMap_MoveToEnd(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		om := NewOrderedMap[int, int]()
		for i := 0; i < size; i++ {
			om.Set(i, i*2)
		}

		b.Run(benchName("size", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				om.MoveToEnd(i % size)
			}
		})
	}
}

// Benchmark GetOrSet operation
func BenchmarkOrderedMap_GetOrSet(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		om := NewOrderedMap[int, int]()
		mk := func() int { return 42 }

		b.Run(benchName("existing_size", size), func(b *testing.B) {
			// Pre-populate
			for i := 0; i < size; i++ {
				om.Set(i, i*2)
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				om.GetOrSet(i%size, mk)
			}
		})

		b.Run(benchName("new_size", size), func(b *testing.B) {
			om := NewOrderedMap[int, int]()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				om.GetOrSet(i, mk)
			}
		})
	}
}

// Benchmark concurrent operations
func BenchmarkOrderedMap_Concurrent(b *testing.B) {
	om := NewOrderedMap[int, int]()
	for i := 0; i < 1000; i++ {
		om.Set(i, i*2)
	}

	b.Run("ReadHeavy_90_10", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				if i%10 == 0 {
					om.Set(i%1000, i)
				} else {
					_, _ = om.Get(i % 1000)
				}
				i++
			}
		})
	})

	b.Run("WriteHeavy_10_90", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				if i%10 == 0 {
					_, _ = om.Get(i % 1000)
				} else {
					om.Set(i%1000, i)
				}
				i++
			}
		})
	})

	b.Run("Mixed_50_50", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				if i%2 == 0 {
					_, _ = om.Get(i % 1000)
				} else {
					om.Set(i%1000, i)
				}
				i++
			}
		})
	})
}

// Helper for benchmark naming
func benchName(prefix string, value int) string {
	return prefix + "_" + itoa(value)
}

// Simple itoa for benchmark names
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf) - 1
	for n > 0 {
		buf[i] = byte('0' + n%10)
		n /= 10
		i--
	}
	return string(buf[i+1:])
}
