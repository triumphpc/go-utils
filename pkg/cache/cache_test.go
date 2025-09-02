package cache

import (
	"testing"
)

func TestCache(t *testing.T) {
	t.Run("string-int cache", func(t *testing.T) {
		cache := NewCache[string, int]()

		// Test Set and Get with existing key
		cache.Set("age", 25)
		if value, exists := cache.Get("age"); !exists || value != 25 {
			t.Errorf("Expected age=25, exists=true, got value=%d, exists=%v", value, exists)
		}

		// Test Get with non-existing key
		if value, exists := cache.Get("name"); exists {
			t.Errorf("Expected exists=false, got value=%d, exists=%v", value, exists)
		}

		// Test value update
		cache.Set("age", 30)
		if value, exists := cache.Get("age"); !exists || value != 30 {
			t.Errorf("Expected age=30 after update, got value=%d, exists=%v", value, exists)
		}
	})

	t.Run("int-string cache", func(t *testing.T) {
		cache := NewCache[int, string]()

		cache.Set(1, "one")
		cache.Set(2, "two")

		if value, exists := cache.Get(1); !exists || value != "one" {
			t.Errorf("Expected key=1 -> 'one', got '%s', exists=%v", value, exists)
		}

		if value, exists := cache.Get(2); !exists || value != "two" {
			t.Errorf("Expected key=2 -> 'two', got '%s', exists=%v", value, exists)
		}

		if _, exists := cache.Get(3); exists {
			t.Error("Expected key=3 to not exist")
		}
	})

	t.Run("struct key cache", func(t *testing.T) {
		type Point struct {
			X, Y int
		}

		cache := NewCache[Point, string]()
		point := Point{X: 1, Y: 2}

		cache.Set(point, "origin")
		if value, exists := cache.Get(point); !exists || value != "origin" {
			t.Errorf("Expected point -> 'origin', got '%s', exists=%v", value, exists)
		}

		// Test with different point
		otherPoint := Point{X: 3, Y: 4}
		if _, exists := cache.Get(otherPoint); exists {
			t.Error("Expected other point to not exist")
		}
	})

	t.Run("bool value cache", func(t *testing.T) {
		cache := NewCache[string, bool]()

		cache.Set("isActive", true)
		cache.Set("isDeleted", false)

		if value, exists := cache.Get("isActive"); !exists || value != true {
			t.Errorf("Expected isActive=true, got %v, exists=%v", value, exists)
		}

		if value, exists := cache.Get("isDeleted"); !exists || value != false {
			t.Errorf("Expected isDeleted=false, got %v, exists=%v", value, exists)
		}
	})

	t.Run("multiple operations", func(t *testing.T) {
		cache := NewCache[string, float64]()

		// Add multiple values
		cache.Set("pi", 3.14)
		cache.Set("e", 2.71)
		cache.Set("sqrt2", 1.41)

		// Check all values
		testCases := map[string]float64{
			"pi":    3.14,
			"e":     2.71,
			"sqrt2": 1.41,
		}

		for key, expected := range testCases {
			if value, exists := cache.Get(key); !exists || value != expected {
				t.Errorf("Key %s: expected %f, got %f, exists=%v", key, expected, value, exists)
			}
		}

		// Test non-existing key
		if _, exists := cache.Get("unknown"); exists {
			t.Error("Expected unknown key to not exist")
		}
	})

	t.Run("zero value handling", func(t *testing.T) {
		cache := NewCache[string, int]()

		// Get zero value for non-existing key
		if value, exists := cache.Get("nonexistent"); exists || value != 0 {
			t.Errorf("Expected zero value 0, got %d, exists=%v", value, exists)
		}

		// Store and retrieve zero value
		cache.Set("zero", 0)
		if value, exists := cache.Get("zero"); !exists || value != 0 {
			t.Errorf("Expected stored zero value 0, got %d, exists=%v", value, exists)
		}
	})
}
