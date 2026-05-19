package core

import (
	"testing"
)

func TestNewList(t *testing.T) {
	list := NewList[int]()
	if list == nil {
		t.Fatal("NewList returned nil")
	}
	if list.Size() != 0 {
		t.Errorf("Expected size 0, got %d", list.Size())
	}
	if !list.IsEmpty() {
		t.Error("Expected list to be empty")
	}
}

func TestNewListWithCapacity(t *testing.T) {
	list := NewListWithCapacity[int](10)
	if list == nil {
		t.Fatal("NewListWithCapacity returned nil")
	}
	if list.Size() != 0 {
		t.Errorf("Expected size 0, got %d", list.Size())
	}
}

func TestAdd(t *testing.T) {
	list := NewList[int]()
	list.Add(1)
	list.Add(2)
	list.Add(3)
	
	if list.Size() != 3 {
		t.Errorf("Expected size 3, got %d", list.Size())
	}
	
	if list.Get(0) != 1 {
		t.Errorf("Expected 1 at index 0, got %d", list.Get(0))
	}
	if list.Get(1) != 2 {
		t.Errorf("Expected 2 at index 1, got %d", list.Get(1))
	}
	if list.Get(2) != 3 {
		t.Errorf("Expected 3 at index 2, got %d", list.Get(2))
	}
}

func TestAddAll(t *testing.T) {
	list := NewList[int]()
	list.AddAll([]int{1, 2, 3})
	
	if list.Size() != 3 {
		t.Errorf("Expected size 3, got %d", list.Size())
	}
	
	list.AddAll([]int{4, 5})
	if list.Size() != 5 {
		t.Errorf("Expected size 5, got %d", list.Size())
	}
}

func TestInsert(t *testing.T) {
	list := NewList[int]()
	list.Add(1)
	list.Add(3)
	list.Insert(1, 2)
	
	if list.Size() != 3 {
		t.Errorf("Expected size 3, got %d", list.Size())
	}
	if list.Get(1) != 2 {
		t.Errorf("Expected 2 at index 1, got %d", list.Get(1))
	}
}

func TestRemove(t *testing.T) {
	list := NewList[int]()
	list.Add(1)
	list.Add(2)
	list.Add(3)
	
	removed := list.Remove(1)
	if removed != 2 {
		t.Errorf("Expected removed item 2, got %d", removed)
	}
	if list.Size() != 2 {
		t.Errorf("Expected size 2, got %d", list.Size())
	}
	if list.Get(0) != 1 {
		t.Errorf("Expected 1 at index 0, got %d", list.Get(0))
	}
	if list.Get(1) != 3 {
		t.Errorf("Expected 3 at index 1, got %d", list.Get(1))
	}
}

func TestRemoveItem(t *testing.T) {
	list := NewList[int]()
	list.Add(1)
	list.Add(2)
	list.Add(3)
	
	removed := list.RemoveItem(2)
	if !removed {
		t.Error("Expected RemoveItem to return true")
	}
	if list.Size() != 2 {
		t.Errorf("Expected size 2, got %d", list.Size())
	}
	
	removed = list.RemoveItem(99)
	if removed {
		t.Error("Expected RemoveItem to return false for non-existent item")
	}
}

func TestRemoveItemFunc(t *testing.T) {
	list := NewList[int]()
	list.Add(1)
	list.Add(2)
	list.Add(3)
	
	removed := list.RemoveItemFunc(func(x int) bool { return x == 2 })
	if !removed {
		t.Error("Expected RemoveItemFunc to return true")
	}
	if list.Size() != 2 {
		t.Errorf("Expected size 2, got %d", list.Size())
	}
}

func TestGet(t *testing.T) {
	list := NewList[string]()
	list.Add("a")
	list.Add("b")
	list.Add("c")
	
	if list.Get(0) != "a" {
		t.Errorf("Expected 'a' at index 0, got %s", list.Get(0))
	}
	if list.Get(1) != "b" {
		t.Errorf("Expected 'b' at index 1, got %s", list.Get(1))
	}
}

func TestSet(t *testing.T) {
	list := NewList[int]()
	list.Add(1)
	list.Add(2)
	list.Set(0, 10)
	
	if list.Get(0) != 10 {
		t.Errorf("Expected 10 at index 0, got %d", list.Get(0))
	}
}

func TestClear(t *testing.T) {
	list := NewList[int]()
	list.Add(1)
	list.Add(2)
	list.Clear()
	
	if list.Size() != 0 {
		t.Errorf("Expected size 0 after Clear, got %d", list.Size())
	}
	if !list.IsEmpty() {
		t.Error("Expected list to be empty after Clear")
	}
}

func TestContains(t *testing.T) {
	list := NewList[int]()
	list.Add(1)
	list.Add(2)
	list.Add(3)
	
	if !list.Contains(2) {
		t.Error("Expected list to contain 2")
	}
	if list.Contains(99) {
		t.Error("Expected list not to contain 99")
	}
}

func TestContainsFunc(t *testing.T) {
	list := NewList[int]()
	list.Add(1)
	list.Add(2)
	list.Add(3)
	
	if !list.ContainsFunc(func(x int) bool { return x > 2 }) {
		t.Error("Expected list to contain item > 2")
	}
	if list.ContainsFunc(func(x int) bool { return x > 10 }) {
		t.Error("Expected list not to contain item > 10")
	}
}

func TestIndexOf(t *testing.T) {
	list := NewList[int]()
	list.Add(1)
	list.Add(2)
	list.Add(3)
	
	if list.IndexOf(2) != 1 {
		t.Errorf("Expected index 1 for value 2, got %d", list.IndexOf(2))
	}
	if list.IndexOf(99) != -1 {
		t.Errorf("Expected index -1 for non-existent value, got %d", list.IndexOf(99))
	}
}

func TestIndexOfFunc(t *testing.T) {
	list := NewList[int]()
	list.Add(1)
	list.Add(2)
	list.Add(3)
	
	idx := list.IndexOfFunc(func(x int) bool { return x > 2 })
	if idx != 2 {
		t.Errorf("Expected index 2, got %d", idx)
	}
	
	idx = list.IndexOfFunc(func(x int) bool { return x > 10 })
	if idx != -1 {
		t.Errorf("Expected index -1, got %d", idx)
	}
}

func TestToSlice(t *testing.T) {
	list := NewList[int]()
	list.Add(1)
	list.Add(2)
	list.Add(3)
	
	slice := list.ToSlice()
	if len(slice) != 3 {
		t.Errorf("Expected slice length 3, got %d", len(slice))
	}
	
	// Modify slice should not affect list
	slice[0] = 99
	if list.Get(0) != 1 {
		t.Error("Modifying slice should not affect list")
	}
}

func TestForEach(t *testing.T) {
	list := NewList[int]()
	list.Add(1)
	list.Add(2)
	list.Add(3)
	
	sum := 0
	list.ForEach(func(x int) {
		sum += x
	})
	
	if sum != 6 {
		t.Errorf("Expected sum 6, got %d", sum)
	}
}

func TestFilter(t *testing.T) {
	list := NewList[int]()
	list.Add(1)
	list.Add(2)
	list.Add(3)
	list.Add(4)
	list.Add(5)
	
	filtered := list.Filter(func(x int) bool { return x%2 == 0 })
	if filtered.Size() != 2 {
		t.Errorf("Expected filtered size 2, got %d", filtered.Size())
	}
	if filtered.Get(0) != 2 || filtered.Get(1) != 4 {
		t.Error("Filtered list should contain 2 and 4")
	}
}

func TestMapList(t *testing.T) {
	list := NewList[int]()
	list.Add(1)
	list.Add(2)
	list.Add(3)
	
	mapped := MapList(list, func(x int) string {
		return string(rune('0' + x))
	})
	
	if mapped.Size() != 3 {
		t.Errorf("Expected mapped size 3, got %d", mapped.Size())
	}
	if mapped.Get(0) != "1" {
		t.Errorf("Expected '1' at index 0, got %s", mapped.Get(0))
	}
}

func TestFind(t *testing.T) {
	list := NewList[int]()
	list.Add(1)
	list.Add(2)
	list.Add(3)
	
	item, found := list.Find(func(x int) bool { return x > 2 })
	if !found {
		t.Error("Expected Find to return true")
	}
	if item != 3 {
		t.Errorf("Expected item 3, got %d", item)
	}
	
	_, found = list.Find(func(x int) bool { return x > 10 })
	if found {
		t.Error("Expected Find to return false for non-existent item")
	}
}

func TestFirst(t *testing.T) {
	list := NewList[int]()
	list.Add(1)
	list.Add(2)
	
	first, found := list.First()
	if !found {
		t.Error("Expected First to return true")
	}
	if first != 1 {
		t.Errorf("Expected first item 1, got %d", first)
	}
	
	emptyList := NewList[int]()
	_, found = emptyList.First()
	if found {
		t.Error("Expected First to return false for empty list")
	}
}

func TestLast(t *testing.T) {
	list := NewList[int]()
	list.Add(1)
	list.Add(2)
	
	last, found := list.Last()
	if !found {
		t.Error("Expected Last to return true")
	}
	if last != 2 {
		t.Errorf("Expected last item 2, got %d", last)
	}
	
	emptyList := NewList[int]()
	_, found = emptyList.Last()
	if found {
		t.Error("Expected Last to return false for empty list")
	}
}

func TestConcurrentAccess(t *testing.T) {
	list := NewList[int]()
	
	// Test concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(val int) {
			for j := 0; j < 100; j++ {
				list.Add(val*100 + j)
			}
			done <- true
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Should have 1000 items
	if list.Size() != 1000 {
		t.Errorf("Expected size 1000, got %d", list.Size())
	}
}
