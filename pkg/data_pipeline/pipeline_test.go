package data_pipeline

import (
	"strings"
	"testing"
)

func TestMap(t *testing.T) {
	t.Run("int to int transformation", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5}
		double := func(x int) int { return x * 2 }

		result := Map(input, double)

		expected := []int{2, 4, 6, 8, 10}
		if len(result) != len(expected) {
			t.Errorf("Expected length %d, got %d", len(expected), len(result))
		}

		for i := range expected {
			if result[i] != expected[i] {
				t.Errorf("At index %d: expected %d, got %d", i, expected[i], result[i])
			}
		}
	})

	t.Run("string to int transformation", func(t *testing.T) {
		input := []string{"hello", "world", "test"}
		length := func(s string) int { return len(s) }

		result := Map(input, length)

		expected := []int{5, 5, 4}
		for i := range expected {
			if result[i] != expected[i] {
				t.Errorf("At index %d: expected %d, got %d", i, expected[i], result[i])
			}
		}
	})

	t.Run("int to string transformation", func(t *testing.T) {
		input := []int{1, 2, 3}
		toString := func(x int) string { return string(rune('0' + x)) }

		result := Map(input, toString)

		expected := []string{"1", "2", "3"}
		for i := range expected {
			if result[i] != expected[i] {
				t.Errorf("At index %d: expected %s, got %s", i, expected[i], result[i])
			}
		}
	})

	t.Run("string to string transformation", func(t *testing.T) {
		input := []string{"apple", "banana", "cherry"}
		toUpper := func(s string) string { return strings.ToUpper(s) }

		result := Map(input, toUpper)

		expected := []string{"APPLE", "BANANA", "CHERRY"}
		for i := range expected {
			if result[i] != expected[i] {
				t.Errorf("At index %d: expected %s, got %s", i, expected[i], result[i])
			}
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		input := []int{}
		double := func(x int) int { return x * 2 }

		result := Map(input, double)

		if len(result) != 0 {
			t.Errorf("Expected empty slice, got length %d", len(result))
		}
	})

	t.Run("struct transformation", func(t *testing.T) {
		type Person struct {
			Name string
			Age  int
		}

		type PersonInfo struct {
			Info string
		}

		input := []Person{
			{"Alice", 25},
			{"Bob", 30},
		}

		toInfo := func(p Person) PersonInfo {
			return PersonInfo{Info: p.Name + " is " + string(rune('0'+p.Age/10)) + string(rune('0'+p.Age%10)) + " years old"}
		}

		result := Map(input, toInfo)

		if len(result) != 2 {
			t.Errorf("Expected 2 results, got %d", len(result))
		}
	})
}
