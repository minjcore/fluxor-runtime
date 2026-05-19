package core

import (
	"sync"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// List is a thread-safe generic list implementation
// It provides common list operations with concurrent access support
type List[T any] struct {
	mu    sync.RWMutex
	items []T
}

// NewList creates a new empty list
func NewList[T any]() *List[T] {
	return &List[T]{
		items: make([]T, 0),
	}
}

// NewListWithCapacity creates a new list with the specified initial capacity
func NewListWithCapacity[T any](capacity int) *List[T] {
	failfast.NonNegative(capacity, "capacity")
	return &List[T]{
		items: make([]T, 0, capacity),
	}
}

// Add appends an item to the end of the list
func (l *List[T]) Add(item T) {
	failfast.NotNil(l, "list")
	l.mu.Lock()
	defer l.mu.Unlock()
	l.items = append(l.items, item)
}

// AddAll appends all items from another slice to the list
func (l *List[T]) AddAll(items []T) {
	failfast.NotNil(l, "list")
	if len(items) == 0 {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.items = append(l.items, items...)
}

// Insert inserts an item at the specified index
func (l *List[T]) Insert(index int, item T) {
	failfast.NotNil(l, "list")
	l.mu.Lock()
	defer l.mu.Unlock()
	failfast.If(index >= 0 && index <= len(l.items), "index out of bounds: %d (size: %d)", index, len(l.items))
	
	// Insert at index
	l.items = append(l.items[:index], append([]T{item}, l.items[index:]...)...)
}

// Remove removes the item at the specified index
func (l *List[T]) Remove(index int) T {
	failfast.NotNil(l, "list")
	l.mu.Lock()
	defer l.mu.Unlock()
	failfast.If(index >= 0 && index < len(l.items), "index out of bounds: %d (size: %d)", index, len(l.items))
	
	item := l.items[index]
	l.items = append(l.items[:index], l.items[index+1:]...)
	return item
}

// RemoveItem removes the first occurrence of the specified item
// Returns true if the item was found and removed, false otherwise
func (l *List[T]) RemoveItem(item T) bool {
	failfast.NotNil(l, "list")
	l.mu.Lock()
	defer l.mu.Unlock()
	
	for i, v := range l.items {
		// Use type assertion or comparison
		// For comparable types, we can use ==
		// For non-comparable types, we need a custom comparison
		if any(v) == any(item) {
			l.items = append(l.items[:i], l.items[i+1:]...)
			return true
		}
	}
	return false
}

// RemoveItemFunc removes the first item that matches the predicate
// Returns true if an item was found and removed, false otherwise
func (l *List[T]) RemoveItemFunc(predicate func(T) bool) bool {
	failfast.NotNil(l, "list")
	failfast.NotNil(predicate, "predicate")
	l.mu.Lock()
	defer l.mu.Unlock()
	
	for i, v := range l.items {
		if predicate(v) {
			l.items = append(l.items[:i], l.items[i+1:]...)
			return true
		}
	}
	return false
}

// Get returns the item at the specified index
func (l *List[T]) Get(index int) T {
	failfast.NotNil(l, "list")
	l.mu.RLock()
	defer l.mu.RUnlock()
	failfast.If(index >= 0 && index < len(l.items), "index out of bounds: %d (size: %d)", index, len(l.items))
	return l.items[index]
}

// Set replaces the item at the specified index
func (l *List[T]) Set(index int, item T) {
	failfast.NotNil(l, "list")
	l.mu.Lock()
	defer l.mu.Unlock()
	failfast.If(index >= 0 && index < len(l.items), "index out of bounds: %d (size: %d)", index, len(l.items))
	l.items[index] = item
}

// Size returns the number of items in the list
func (l *List[T]) Size() int {
	failfast.NotNil(l, "list")
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.items)
}

// IsEmpty returns true if the list is empty
func (l *List[T]) IsEmpty() bool {
	return l.Size() == 0
}

// Clear removes all items from the list
func (l *List[T]) Clear() {
	failfast.NotNil(l, "list")
	l.mu.Lock()
	defer l.mu.Unlock()
	l.items = make([]T, 0)
}

// Contains returns true if the list contains the specified item
func (l *List[T]) Contains(item T) bool {
	failfast.NotNil(l, "list")
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	for _, v := range l.items {
		if any(v) == any(item) {
			return true
		}
	}
	return false
}

// ContainsFunc returns true if the list contains an item that matches the predicate
func (l *List[T]) ContainsFunc(predicate func(T) bool) bool {
	failfast.NotNil(l, "list")
	failfast.NotNil(predicate, "predicate")
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	for _, v := range l.items {
		if predicate(v) {
			return true
		}
	}
	return false
}

// IndexOf returns the index of the first occurrence of the specified item
// Returns -1 if the item is not found
func (l *List[T]) IndexOf(item T) int {
	failfast.NotNil(l, "list")
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	for i, v := range l.items {
		if any(v) == any(item) {
			return i
		}
	}
	return -1
}

// IndexOfFunc returns the index of the first item that matches the predicate
// Returns -1 if no item matches
func (l *List[T]) IndexOfFunc(predicate func(T) bool) int {
	failfast.NotNil(l, "list")
	failfast.NotNil(predicate, "predicate")
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	for i, v := range l.items {
		if predicate(v) {
			return i
		}
	}
	return -1
}

// ToSlice returns a copy of the list as a slice
func (l *List[T]) ToSlice() []T {
	failfast.NotNil(l, "list")
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	result := make([]T, len(l.items))
	copy(result, l.items)
	return result
}

// ForEach applies the function to each item in the list
func (l *List[T]) ForEach(fn func(T)) {
	failfast.NotNil(l, "list")
	failfast.NotNil(fn, "function")
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	for _, item := range l.items {
		fn(item)
	}
}

// Filter returns a new list containing only items that match the predicate
func (l *List[T]) Filter(predicate func(T) bool) *List[T] {
	failfast.NotNil(l, "list")
	failfast.NotNil(predicate, "predicate")
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	result := NewList[T]()
	for _, item := range l.items {
		if predicate(item) {
			result.items = append(result.items, item)
		}
	}
	return result
}

// MapList applies the function to each item and returns a new list with the results
// This is a standalone function because Go doesn't allow methods on generic types to have their own type parameters
func MapList[T any, R any](l *List[T], fn func(T) R) *List[R] {
	failfast.NotNil(l, "list")
	failfast.NotNil(fn, "function")
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	result := NewList[R]()
	for _, item := range l.items {
		result.items = append(result.items, fn(item))
	}
	return result
}

// Find returns the first item that matches the predicate
// Returns the zero value and false if no item matches
func (l *List[T]) Find(predicate func(T) bool) (T, bool) {
	failfast.NotNil(l, "list")
	failfast.NotNil(predicate, "predicate")
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	for _, item := range l.items {
		if predicate(item) {
			return item, true
		}
	}
	var zero T
	return zero, false
}

// First returns the first item in the list
// Returns the zero value and false if the list is empty
func (l *List[T]) First() (T, bool) {
	failfast.NotNil(l, "list")
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	if len(l.items) == 0 {
		var zero T
		return zero, false
	}
	return l.items[0], true
}

// Last returns the last item in the list
// Returns the zero value and false if the list is empty
func (l *List[T]) Last() (T, bool) {
	failfast.NotNil(l, "list")
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	if len(l.items) == 0 {
		var zero T
		return zero, false
	}
	return l.items[len(l.items)-1], true
}
