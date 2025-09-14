package ordone

import (
	"context"
	"testing"
	"time"
)

func TestOrDone(t *testing.T) {
	t.Run("normal operation", func(t *testing.T) {
		source := make(chan interface{}, 3)
		source <- 1
		source <- 2
		source <- 3
		close(source)

		result := OrDone(nil, source)

		expected := []interface{}{1, 2, 3}
		var actual []interface{}

		for val := range result {
			actual = append(actual, val)
		}

		if len(actual) != len(expected) {
			t.Errorf("Expected %d values, got %d", len(expected), len(actual))
		}

		for i, v := range actual {
			if v != expected[i] {
				t.Errorf("At index %d: expected %v, got %v", i, expected[i], v)
			}
		}
	})

	t.Run("with done channel", func(t *testing.T) {
		done := make(chan interface{})
		source := make(chan interface{}, 5)

		// Заполняем источник
		for i := 0; i < 5; i++ {
			source <- i
		}

		result := OrDone(done, source)

		// Читаем 2 значения, затем отменяем
		val1 := <-result
		val2 := <-result
		close(done)

		// Даем время для обработки отмены
		time.Sleep(10 * time.Millisecond)

		// Проверяем, что канал закрыт после отмены
		select {
		case _, ok := <-result:
			if ok {
				t.Error("Expected channel to be closed after done signal")
			}
		default:
			t.Error("Channel should be closed after done signal")
		}

		if val1 != 0 || val2 != 1 {
			t.Errorf("Expected values 0 and 1, got %v and %v", val1, val2)
		}
	})

	t.Run("nil done channel", func(t *testing.T) {
		source := make(chan interface{}, 2)
		source <- "test1"
		source <- "test2"
		close(source)

		result := OrDone(nil, source)

		expected := []interface{}{"test1", "test2"}
		var actual []interface{}

		for val := range result {
			actual = append(actual, val)
		}

		if len(actual) != len(expected) {
			t.Errorf("Expected %d values, got %d", len(expected), len(actual))
		}
	})

	t.Run("immediate cancellation", func(t *testing.T) {
		done := make(chan interface{})
		close(done) // Немедленная отмена

		source := make(chan interface{}, 2)
		source <- "should not be read"
		source <- "should not be read"

		result := OrDone(done, source)

		// Канал должен быть немедленно закрыт
		select {
		case _, ok := <-result:
			if ok {
				t.Error("Expected channel to be closed immediately after done")
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("Timeout waiting for channel to close")
		}
	})

	t.Run("with context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		source := make(chan interface{}, 3)
		source <- 1
		source <- 2
		source <- 3

		// Конвертируем context в done channel
		done := make(chan interface{})
		go func() {
			<-ctx.Done()
			close(done)
		}()

		result := OrDone(done, source)

		// Читаем одно значение и отменяем
		val := <-result
		cancel()

		// Даем время для обработки отмены
		time.Sleep(10 * time.Millisecond)

		if val != 1 {
			t.Errorf("Expected value 1, got %v", val)
		}

		// Проверяем, что канал закрыт
		select {
		case _, ok := <-result:
			if ok {
				t.Error("Expected channel to be closed after context cancellation")
			}
		default:
			t.Error("Channel should be closed after context cancellation")
		}
	})
}

func BenchmarkOrDone(b *testing.B) {
	for i := 0; i < b.N; i++ {
		source := make(chan interface{}, 1000)
		for j := 0; j < 1000; j++ {
			source <- j
		}
		close(source)

		result := OrDone(nil, source)
		for range result {
			// Потребляем все значения
		}
	}
}
