package web

import (
	"testing"
)

func TestClampToZero(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{-10, 0},
		{-1, 0},
		{0, 0},
		{1, 1},
		{10, 10},
		{100, 100},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := clampToZero(tt.input)
			if result != tt.expected {
				t.Errorf("clampToZero(%d) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNewBackpressureController_Validation(t *testing.T) {
	tests := []struct {
		name             string
		normalCapacity   int
		resetInterval    int64
		expectedCapacity int64
		expectedInterval int64
	}{
		{"valid inputs", 100, 60, 100, 60},
		{"zero capacity defaults to 1", 0, 60, 1, 60},
		{"negative capacity defaults to 1", -5, 60, 1, 60},
		{"zero interval defaults to 60", 100, 0, 100, 60},
		{"negative interval defaults to 60", 100, -10, 100, 60},
		{"both invalid", 0, 0, 1, 60},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := NewBackpressureController(tt.normalCapacity, tt.resetInterval)
			if bc.normalCapacity != tt.expectedCapacity {
				t.Errorf("normalCapacity = %d, want %d", bc.normalCapacity, tt.expectedCapacity)
			}
			if bc.resetInterval != tt.expectedInterval {
				t.Errorf("resetInterval = %d, want %d", bc.resetInterval, tt.expectedInterval)
			}
		})
	}
}
