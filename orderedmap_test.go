package orderedmap

import (
	"sync"
	"testing"
)

func TestNewOrderedMap(t *testing.T) {
	om := NewOrderedMap[string, int]()

	if om == nil {
		t.Fatal("NewOrderedMap returned nil")
	}
	if om.Len() != 0 {
		t.Errorf("Expected length 0, got %d", om.Len())
	}
}

func TestOrderedMap_ZeroValue(t *testing.T) {
	var om OrderedMap[string, int]

	// Should be usable without calling NewOrderedMap
	om.Set("key", 42)

	val, ok := om.Get("key")
	if !ok || val != 42 {
		t.Errorf("Expected zero value to be usable, got val=%d, ok=%v", val, ok)
	}

	if om.Len() != 1 {
		t.Errorf("Expected length 1, got %d", om.Len())
	}
}

func TestOrderedMap_Set_Insert(t *testing.T) {
	om := NewOrderedMap[string, int]()

	om.Set("first", 1)
	om.Set("second", 2)
	om.Set("third", 3)

	if om.Len() != 3 {
		t.Errorf("Expected length 3, got %d", om.Len())
	}

	val, ok := om.Get("first")
	if !ok || val != 1 {
		t.Errorf("Expected first=1, got %d, ok=%v", val, ok)
	}

	val, ok = om.Get("second")
	if !ok || val != 2 {
		t.Errorf("Expected second=2, got %d, ok=%v", val, ok)
	}

	val, ok = om.Get("third")
	if !ok || val != 3 {
		t.Errorf("Expected third=3, got %d, ok=%v", val, ok)
	}
}

func TestOrderedMap_Set_Update(t *testing.T) {
	om := NewOrderedMap[string, int]()

	om.Set("key", 1)
	om.Set("key", 2)
	om.Set("key", 3)

	if om.Len() != 1 {
		t.Errorf("Expected length 1, got %d", om.Len())
	}

	val, ok := om.Get("key")
	if !ok || val != 3 {
		t.Errorf("Expected key=3, got %d, ok=%v", val, ok)
	}
}

func TestOrderedMap_Get_NotFound(t *testing.T) {
	om := NewOrderedMap[string, int]()

	val, ok := om.Get("nonexistent")
	if ok {
		t.Error("Expected ok=false for nonexistent key")
	}
	if val != 0 {
		t.Errorf("Expected zero value, got %d", val)
	}
}

func TestOrderedMap_Has(t *testing.T) {
	om := NewOrderedMap[string, int]()

	om.Set("exists", 42)

	if !om.Has("exists") {
		t.Error("Expected Has to return true for existing key")
	}

	if om.Has("nonexistent") {
		t.Error("Expected Has to return false for nonexistent key")
	}
}

func TestOrderedMap_Delete(t *testing.T) {
	om := NewOrderedMap[string, int]()

	om.Set("first", 1)
	om.Set("second", 2)
	om.Set("third", 3)

	// Delete middle element
	deleted := om.Delete("second")
	if !deleted {
		t.Error("Expected Delete to return true")
	}

	if om.Len() != 2 {
		t.Errorf("Expected length 2 after delete, got %d", om.Len())
	}

	if om.Has("second") {
		t.Error("Expected second to be deleted")
	}

	// Verify order preserved
	keys := om.Keys()
	if len(keys) != 2 || keys[0] != "first" || keys[1] != "third" {
		t.Errorf("Expected [first, third], got %v", keys)
	}

	// Delete first element
	om.Delete("first")
	keys = om.Keys()
	if len(keys) != 1 || keys[0] != "third" {
		t.Errorf("Expected [third], got %v", keys)
	}

	// Delete last element
	om.Delete("third")
	if om.Len() != 0 {
		t.Errorf("Expected length 0, got %d", om.Len())
	}
}

func TestOrderedMap_Delete_NotFound(t *testing.T) {
	om := NewOrderedMap[string, int]()

	deleted := om.Delete("nonexistent")
	if deleted {
		t.Error("Expected Delete to return false for nonexistent key")
	}
}

func TestOrderedMap_DeleteWithValue(t *testing.T) {
	om := NewOrderedMap[string, int]()
	om.Set("a", 1)
	om.Set("b", 2)
	om.Set("c", 3)

	// Delete existing key
	val, ok := om.DeleteWithValue("b")
	if !ok {
		t.Error("Expected DeleteWithValue to return true for existing key")
	}
	if val != 2 {
		t.Errorf("Expected value 2, got %d", val)
	}

	// Verify removed
	if om.Has("b") {
		t.Error("Expected b to be removed")
	}
	if om.Len() != 2 {
		t.Errorf("Expected length 2, got %d", om.Len())
	}

	// Delete nonexistent key
	val, ok = om.DeleteWithValue("nonexistent")
	if ok {
		t.Error("Expected DeleteWithValue to return false for nonexistent key")
	}
	if val != 0 {
		t.Errorf("Expected zero value, got %d", val)
	}

	if om.Len() != 2 {
		t.Errorf("Expected length still 2, got %d", om.Len())
	}
}

func TestOrderedMap_Clear(t *testing.T) {
	om := NewOrderedMap[string, int]()

	om.Set("first", 1)
	om.Set("second", 2)
	om.Set("third", 3)

	om.Clear()

	if om.Len() != 0 {
		t.Errorf("Expected length 0 after clear, got %d", om.Len())
	}

	if om.Has("first") || om.Has("second") || om.Has("third") {
		t.Error("Expected all keys to be cleared")
	}

	// Verify capacity preserved (can add without reallocation)
	om.Set("new", 4)
	if om.Len() != 1 {
		t.Errorf("Expected length 1 after adding to cleared map, got %d", om.Len())
	}
}

func TestOrderedMap_Reset(t *testing.T) {
	om := NewOrderedMap[string, int]()

	om.Set("first", 1)
	om.Set("second", 2)
	om.Set("third", 3)

	om.Reset()

	if om.Len() != 0 {
		t.Errorf("Expected length 0 after reset, got %d", om.Len())
	}

	if om.Has("first") || om.Has("second") || om.Has("third") {
		t.Error("Expected all keys to be reset")
	}

	// Should be usable after reset (zero value usability)
	om.Set("new", 4)
	val, ok := om.Get("new")
	if !ok || val != 4 {
		t.Errorf("Expected map to be usable after reset, got val=%d, ok=%v", val, ok)
	}
}

func TestOrderedMap_Range(t *testing.T) {
	om := NewOrderedMap[string, int]()

	om.Set("first", 1)
	om.Set("second", 2)
	om.Set("third", 3)

	var keys []string
	var values []int

	om.Range(func(k string, v int) {
		keys = append(keys, k)
		values = append(values, v)
	})

	expectedKeys := []string{"first", "second", "third"}
	expectedValues := []int{1, 2, 3}

	if len(keys) != len(expectedKeys) {
		t.Errorf("Expected %d keys, got %d", len(expectedKeys), len(keys))
	}

	for i := range keys {
		if keys[i] != expectedKeys[i] {
			t.Errorf("Key mismatch at index %d: expected %s, got %s", i, expectedKeys[i], keys[i])
		}
		if values[i] != expectedValues[i] {
			t.Errorf("Value mismatch at index %d: expected %d, got %d", i, expectedValues[i], values[i])
		}
	}
}

func TestOrderedMap_Range_Empty(t *testing.T) {
	om := NewOrderedMap[string, int]()

	called := false
	om.Range(func(_ string, _ int) {
		called = true
	})

	if called {
		t.Error("Expected Range to not call function on empty map")
	}
}

func TestOrderedMap_Range_Reentrant(t *testing.T) {
	om := NewOrderedMap[string, int]()

	om.Set("first", 1)
	om.Set("second", 2)

	// Range callback should be able to modify the map without deadlock
	om.Range(func(k string, v int) {
		om.Set("new_"+k, v*10)
	})

	// Verify new entries were added
	val, ok := om.Get("new_first")
	if !ok || val != 10 {
		t.Errorf("Expected new_first=10, got %d, ok=%v", val, ok)
	}

	val, ok = om.Get("new_second")
	if !ok || val != 20 {
		t.Errorf("Expected new_second=20, got %d, ok=%v", val, ok)
	}
}

func TestOrderedMap_RangeBreak(t *testing.T) {
	om := NewOrderedMap[string, int]()

	om.Set("first", 1)
	om.Set("second", 2)
	om.Set("third", 3)

	var keys []string

	om.RangeBreak(func(k string, _ int) bool {
		keys = append(keys, k)
		return k != "second" // Stop after second
	})

	expectedKeys := []string{"first", "second"}

	if len(keys) != len(expectedKeys) {
		t.Errorf("Expected %d keys, got %d", len(expectedKeys), len(keys))
	}

	for i := range keys {
		if keys[i] != expectedKeys[i] {
			t.Errorf("Key mismatch at index %d: expected %s, got %s", i, expectedKeys[i], keys[i])
		}
	}
}

func TestOrderedMap_RangeBreak_NoBreak(t *testing.T) {
	om := NewOrderedMap[string, int]()

	om.Set("first", 1)
	om.Set("second", 2)

	var keys []string

	om.RangeBreak(func(k string, _ int) bool {
		keys = append(keys, k)
		return true // Never break
	})

	if len(keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(keys))
	}
}

func TestOrderedMap_Keys(t *testing.T) {
	om := NewOrderedMap[string, int]()

	om.Set("first", 1)
	om.Set("second", 2)
	om.Set("third", 3)

	keys := om.Keys()

	expectedKeys := []string{"first", "second", "third"}

	if len(keys) != len(expectedKeys) {
		t.Errorf("Expected %d keys, got %d", len(expectedKeys), len(keys))
	}

	for i := range keys {
		if keys[i] != expectedKeys[i] {
			t.Errorf("Key mismatch at index %d: expected %s, got %s", i, expectedKeys[i], keys[i])
		}
	}

	// Verify returned slice is a copy (mutation doesn't affect original)
	keys[0] = "modified"
	originalKeys := om.Keys()
	if originalKeys[0] != "first" {
		t.Error("Keys() should return a copy, not expose internal slice")
	}
}

func TestOrderedMap_Values(t *testing.T) {
	om := NewOrderedMap[string, int]()

	om.Set("first", 1)
	om.Set("second", 2)
	om.Set("third", 3)

	values := om.Values()

	expectedValues := []int{1, 2, 3}

	if len(values) != len(expectedValues) {
		t.Errorf("Expected %d values, got %d", len(expectedValues), len(values))
	}

	for i := range values {
		if values[i] != expectedValues[i] {
			t.Errorf("Value mismatch at index %d: expected %d, got %d", i, expectedValues[i], values[i])
		}
	}

	// Verify returned slice is a copy (mutation doesn't affect original)
	values[0] = 999
	originalValues := om.Values()
	if originalValues[0] != 1 {
		t.Error("Values() should return a copy, not expose internal slice")
	}
}

func TestOrderedMap_Front(t *testing.T) {
	om := NewOrderedMap[string, int]()

	// Empty map
	k, v, ok := om.Front()
	if ok {
		t.Error("Expected Front to return false for empty map")
	}
	if k != "" || v != 0 {
		t.Error("Expected zero values for empty map")
	}

	// With elements
	om.Set("first", 1)
	om.Set("second", 2)
	om.Set("third", 3)

	k, v, ok = om.Front()
	if !ok {
		t.Error("Expected Front to return true")
	}
	if k != "first" || v != 1 {
		t.Errorf("Expected (first, 1), got (%s, %d)", k, v)
	}
}

func TestOrderedMap_Back(t *testing.T) {
	om := NewOrderedMap[string, int]()

	// Empty map
	k, v, ok := om.Back()
	if ok {
		t.Error("Expected Back to return false for empty map")
	}
	if k != "" || v != 0 {
		t.Error("Expected zero values for empty map")
	}

	// With elements
	om.Set("first", 1)
	om.Set("second", 2)
	om.Set("third", 3)

	k, v, ok = om.Back()
	if !ok {
		t.Error("Expected Back to return true")
	}
	if k != "third" || v != 3 {
		t.Errorf("Expected (third, 3), got (%s, %d)", k, v)
	}
}

func TestOrderedMap_InsertionOrder(t *testing.T) {
	om := NewOrderedMap[string, int]()

	// Insert in specific order
	om.Set("zebra", 26)
	om.Set("alpha", 1)
	om.Set("beta", 2)

	// Verify order is preserved (not alphabetical)
	keys := om.Keys()
	expectedKeys := []string{"zebra", "alpha", "beta"}

	if len(keys) != len(expectedKeys) {
		t.Errorf("Expected %d keys, got %d", len(expectedKeys), len(keys))
	}

	for i := range keys {
		if keys[i] != expectedKeys[i] {
			t.Errorf("Expected insertion order preserved, got %v", keys)
			break
		}
	}
}

func TestOrderedMap_Concurrent_SetAndGet(t *testing.T) {
	om := NewOrderedMap[int, int]()
	var wg sync.WaitGroup

	// Number of goroutines and operations
	numGoroutines := 10
	numOpsPerGoroutine := 100

	// Concurrent Sets
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < numOpsPerGoroutine; i++ {
				key := goroutineID*numOpsPerGoroutine + i
				om.Set(key, key*2)
			}
		}(g)
	}

	wg.Wait()

	// Verify all values were set
	expectedLen := numGoroutines * numOpsPerGoroutine
	if om.Len() != expectedLen {
		t.Errorf("Expected length %d, got %d", expectedLen, om.Len())
	}

	// Concurrent Gets
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < numOpsPerGoroutine; i++ {
				key := goroutineID*numOpsPerGoroutine + i
				val, ok := om.Get(key)
				if !ok {
					t.Errorf("Expected key %d to exist", key)
				}
				if val != key*2 {
					t.Errorf("Expected value %d for key %d, got %d", key*2, key, val)
				}
			}
		}(g)
	}

	wg.Wait()
}

func TestOrderedMap_Concurrent_SetUpdateDelete(t *testing.T) {
	om := NewOrderedMap[int, int]()
	var wg sync.WaitGroup

	numGoroutines := 5
	numOps := 100

	// Pre-populate
	for i := 0; i < numOps; i++ {
		om.Set(i, i)
	}

	// Concurrent updates
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < numOps; i++ {
				om.Set(i, i*goroutineID)
			}
		}(g)
	}

	// Concurrent deletes
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numOps/2; i++ {
			om.Delete(i * 2) // Delete even keys
		}
	}()

	wg.Wait()

	// Verify no panics and map is in valid state
	finalLen := om.Len()
	if finalLen < 0 || finalLen > numOps {
		t.Errorf("Invalid final length: %d", finalLen)
	}
}

func TestOrderedMap_Concurrent_RangeWhileModifying(t *testing.T) {
	om := NewOrderedMap[int, int]()
	var wg sync.WaitGroup

	// Pre-populate
	for i := 0; i < 100; i++ {
		om.Set(i, i)
	}

	// Concurrent Range operations
	for g := 0; g < 5; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				sum := 0
				om.Range(func(_, v int) {
					sum += v
				})
			}
		}()
	}

	// Concurrent modifications
	for g := 0; g < 5; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				key := goroutineID*50 + i
				om.Set(key, key*2)
			}
		}(g)
	}

	wg.Wait()

	// Verify no panics and map is accessible
	finalLen := om.Len()
	if finalLen <= 0 {
		t.Errorf("Expected positive length, got %d", finalLen)
	}
}

func TestOrderedMap_Concurrent_ClearWhileReading(t *testing.T) {
	om := NewOrderedMap[int, int]()
	var wg sync.WaitGroup

	// Pre-populate
	for i := 0; i < 100; i++ {
		om.Set(i, i)
	}

	// Concurrent readers
	for g := 0; g < 5; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				om.Keys()
				om.Values()
				om.Len()
				om.Front()
				om.Back()
			}
		}()
	}

	// Clear in the middle
	wg.Add(1)
	go func() {
		defer wg.Done()
		om.Clear()
	}()

	wg.Wait()

	// Verify final state is consistent
	finalLen := om.Len()
	if finalLen < 0 {
		t.Errorf("Invalid length after concurrent clear: %d", finalLen)
	}
}

func TestOrderedMap_Concurrent_ResetWhileReading(t *testing.T) {
	om := NewOrderedMap[int, int]()
	var wg sync.WaitGroup

	// Pre-populate
	for i := 0; i < 100; i++ {
		om.Set(i, i)
	}

	// Concurrent readers
	for g := 0; g < 5; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				om.Keys()
				om.Values()
				om.Len()
				om.Front()
				om.Back()
			}
		}()
	}

	// Reset in the middle
	wg.Add(1)
	go func() {
		defer wg.Done()
		om.Reset()
	}()

	wg.Wait()

	// Verify final state is consistent
	finalLen := om.Len()
	if finalLen < 0 {
		t.Errorf("Invalid length after concurrent reset: %d", finalLen)
	}

	// Should be usable after reset
	om.Set(999, 1)
	if om.Len() != 1 {
		t.Errorf("Expected map to be usable after reset, got length %d", om.Len())
	}
}

func TestOrderedMap_Concurrent_HasOperations(t *testing.T) {
	om := NewOrderedMap[int, int]()
	var wg sync.WaitGroup

	// Pre-populate
	for i := 0; i < 50; i++ {
		om.Set(i, i)
	}

	// Concurrent Has checks
	for g := 0; g < 10; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				om.Has(i % 50)
			}
		}()
	}

	// Concurrent modifications
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 50; i < 100; i++ {
			om.Set(i, i)
		}
	}()

	wg.Wait()

	// Verify final state
	if om.Len() != 100 {
		t.Errorf("Expected length 100, got %d", om.Len())
	}
}

func TestOrderedMap_At(t *testing.T) {
	om := NewOrderedMap[string, int]()

	om.Set("first", 1)
	om.Set("second", 2)
	om.Set("third", 3)

	// Valid indices
	k, v, ok := om.At(0)
	if !ok || k != "first" || v != 1 {
		t.Errorf("Expected (first, 1, true), got (%s, %d, %v)", k, v, ok)
	}

	k, v, ok = om.At(1)
	if !ok || k != "second" || v != 2 {
		t.Errorf("Expected (second, 2, true), got (%s, %d, %v)", k, v, ok)
	}

	k, v, ok = om.At(2)
	if !ok || k != "third" || v != 3 {
		t.Errorf("Expected (third, 3, true), got (%s, %d, %v)", k, v, ok)
	}

	// Out of bounds
	_, _, ok = om.At(-1)
	if ok {
		t.Error("Expected At(-1) to return false")
	}

	_, _, ok = om.At(3)
	if ok {
		t.Error("Expected At(3) to return false")
	}

	_, _, ok = om.At(100)
	if ok {
		t.Error("Expected At(100) to return false")
	}
}

func TestOrderedMap_PopFront(t *testing.T) {
	om := NewOrderedMap[string, int]()

	om.Set("first", 1)
	om.Set("second", 2)
	om.Set("third", 3)

	// Pop first element
	k, v, ok := om.PopFront()
	if !ok || k != "first" || v != 1 {
		t.Errorf("Expected (first, 1, true), got (%s, %d, %v)", k, v, ok)
	}

	if om.Len() != 2 {
		t.Errorf("Expected length 2, got %d", om.Len())
	}

	if om.Has("first") {
		t.Error("Expected first to be removed")
	}

	// Verify order maintained
	keys := om.Keys()
	if len(keys) != 2 || keys[0] != "second" || keys[1] != "third" {
		t.Errorf("Expected [second, third], got %v", keys)
	}

	// Pop until empty
	om.PopFront()
	om.PopFront()

	if om.Len() != 0 {
		t.Errorf("Expected length 0, got %d", om.Len())
	}

	// Pop from empty
	_, _, ok = om.PopFront()
	if ok {
		t.Error("Expected PopFront on empty map to return false")
	}
}

func TestOrderedMap_PopBack(t *testing.T) {
	om := NewOrderedMap[string, int]()

	om.Set("first", 1)
	om.Set("second", 2)
	om.Set("third", 3)

	// Pop last element
	k, v, ok := om.PopBack()
	if !ok || k != "third" || v != 3 {
		t.Errorf("Expected (third, 3, true), got (%s, %d, %v)", k, v, ok)
	}

	if om.Len() != 2 {
		t.Errorf("Expected length 2, got %d", om.Len())
	}

	if om.Has("third") {
		t.Error("Expected third to be removed")
	}

	// Verify order maintained
	keys := om.Keys()
	if len(keys) != 2 || keys[0] != "first" || keys[1] != "second" {
		t.Errorf("Expected [first, second], got %v", keys)
	}

	// Pop until empty
	om.PopBack()
	om.PopBack()

	if om.Len() != 0 {
		t.Errorf("Expected length 0, got %d", om.Len())
	}

	// Pop from empty
	_, _, ok = om.PopBack()
	if ok {
		t.Error("Expected PopBack on empty map to return false")
	}
}

func TestOrderedMap_MoveToEnd(t *testing.T) {
	om := NewOrderedMap[string, int]()

	om.Set("first", 1)
	om.Set("second", 2)
	om.Set("third", 3)

	// Move middle to end
	moved := om.MoveToEnd("second")
	if !moved {
		t.Error("Expected MoveToEnd to return true")
	}

	keys := om.Keys()
	expected := []string{"first", "third", "second"}
	if len(keys) != len(expected) {
		t.Errorf("Expected %v, got %v", expected, keys)
	}
	for i := range keys {
		if keys[i] != expected[i] {
			t.Errorf("Expected %v, got %v", expected, keys)
			break
		}
	}

	// Move first to end
	om.MoveToEnd("first")
	keys = om.Keys()
	expected = []string{"third", "second", "first"}
	for i := range keys {
		if keys[i] != expected[i] {
			t.Errorf("Expected %v, got %v", expected, keys)
			break
		}
	}

	// Move already-at-end (should be no-op)
	om.MoveToEnd("first")
	keys = om.Keys()
	for i := range keys {
		if keys[i] != expected[i] {
			t.Errorf("Expected order unchanged, got %v", keys)
			break
		}
	}

	// Move nonexistent key
	moved = om.MoveToEnd("nonexistent")
	if moved {
		t.Error("Expected MoveToEnd to return false for nonexistent key")
	}
}

func TestOrderedMap_GetOrSet(t *testing.T) {
	om := NewOrderedMap[string, int]()

	callCount := 0
	mk := func() int {
		callCount++
		return 42
	}

	// First call should create
	val, existed := om.GetOrSet("key", mk)
	if existed {
		t.Error("Expected existed=false for new key")
	}
	if val != 42 {
		t.Errorf("Expected value 42, got %d", val)
	}
	if callCount != 1 {
		t.Errorf("Expected mk called once, got %d", callCount)
	}

	// Second call should return existing
	val, existed = om.GetOrSet("key", mk)
	if !existed {
		t.Error("Expected existed=true for existing key")
	}
	if val != 42 {
		t.Errorf("Expected value 42, got %d", val)
	}
	if callCount != 1 {
		t.Errorf("Expected mk not called again, got %d calls", callCount)
	}

	// Verify value is in map
	if om.Len() != 1 {
		t.Errorf("Expected length 1, got %d", om.Len())
	}

	retrievedVal, ok := om.Get("key")
	if !ok || retrievedVal != 42 {
		t.Errorf("Expected to retrieve 42, got %d, ok=%v", retrievedVal, ok)
	}
}

func TestOrderedMap_RangeLocked(t *testing.T) {
	om := NewOrderedMap[string, int]()

	om.Set("first", 1)
	om.Set("second", 2)
	om.Set("third", 3)

	var keys []string
	var values []int

	om.RangeLocked(func(k string, v int) {
		keys = append(keys, k)
		values = append(values, v)
	})

	expectedKeys := []string{"first", "second", "third"}
	expectedValues := []int{1, 2, 3}

	if len(keys) != len(expectedKeys) {
		t.Errorf("Expected %d keys, got %d", len(expectedKeys), len(keys))
	}

	for i := range keys {
		if keys[i] != expectedKeys[i] {
			t.Errorf("Key mismatch at index %d: expected %s, got %s", i, expectedKeys[i], keys[i])
		}
		if values[i] != expectedValues[i] {
			t.Errorf("Value mismatch at index %d: expected %d, got %d", i, expectedValues[i], values[i])
		}
	}
}

func TestOrderedMap_RangeLocked_Empty(t *testing.T) {
	om := NewOrderedMap[string, int]()

	called := false
	om.RangeLocked(func(_ string, _ int) {
		called = true
	})

	if called {
		t.Error("Expected RangeLocked to not call function on empty map")
	}
}

func TestOrderedMap_RangeBreakLocked(t *testing.T) {
	om := NewOrderedMap[string, int]()

	om.Set("first", 1)
	om.Set("second", 2)
	om.Set("third", 3)

	var keys []string

	om.RangeBreakLocked(func(k string, _ int) bool {
		keys = append(keys, k)
		return k != "second" // Stop after second
	})

	expectedKeys := []string{"first", "second"}

	if len(keys) != len(expectedKeys) {
		t.Errorf("Expected %d keys, got %d", len(expectedKeys), len(keys))
	}

	for i := range keys {
		if keys[i] != expectedKeys[i] {
			t.Errorf("Key mismatch at index %d: expected %s, got %s", i, expectedKeys[i], keys[i])
		}
	}
}

func TestOrderedMap_RangeBreakLocked_NoBreak(t *testing.T) {
	om := NewOrderedMap[string, int]()

	om.Set("first", 1)
	om.Set("second", 2)

	var keys []string

	om.RangeBreakLocked(func(k string, _ int) bool {
		keys = append(keys, k)
		return true // Never break
	})

	if len(keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(keys))
	}
}
