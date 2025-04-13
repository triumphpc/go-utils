package ptr

import "testing"

// Ptr takes a value of any type T and returns a pointer to that value.
// This is a convenience function for creating pointers without declaring
// intermediate variables.
//
// Benchmark approach using Ptr function
func BenchmarkPtrFunction(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Ptr(i)
	}
}

// Benchmark traditional approach
func BenchmarkTraditional(b *testing.B) {
	for i := 0; i < b.N; i++ {
		v := i
		_ = &v
	}
}

// Benchmark direct approach (when possible)
func BenchmarkDirect(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = &i // Warning: This takes address of loop variable!
	}
}
