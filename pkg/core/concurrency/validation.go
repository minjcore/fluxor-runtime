package concurrency

import "fmt"

// failFast panics with an error (fail-fast principle)
func failFast(err error) {
	if err != nil {
		panic(fmt.Errorf("fail-fast: %w", err))
	}
}

// failFastIf panics if condition is true
func failFastIf(condition bool, message string) {
	if condition {
		panic(fmt.Errorf("fail-fast: %s", message))
	}
}
