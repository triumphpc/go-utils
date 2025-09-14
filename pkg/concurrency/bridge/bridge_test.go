package bridge

import (
	"context"
	"testing"
	"time"
)

func TestBridge(t *testing.T) {
	t.Run("basic functionality", func(t *testing.T) {
		// Создаем канал каналов
		chanStream := make(chan (<-chan interface{}))

		go func() {
			defer close(chanStream)
			for i := 0; i < 3; i++ {
				ch := make(chan interface{}, 1)
				ch <- i
				close(ch)
				chanStream <- ch
			}
		}()

		// Используем Bridge
		result := Bridge(nil, chanStream)

		// Проверяем результаты
		expected := []interface{}{0, 1, 2}
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

	t.Run("with cancellation", func(t *testing.T) {
		done := make(chan interface{})
		chanStream := make(chan (<-chan interface{}))

		go func() {
			defer close(chanStream)
			for i := 0; i < 5; i++ {
				ch := make(chan interface{}, 1)
				ch <- i
				close(ch)
				chanStream <- ch
			}
		}()

		result := Bridge(done, chanStream)

		// Читаем первые 2 значения, затем отменяем
		<-result
		<-result
		close(done)

		// Проверяем, что канал закрывается после отмены
		timeout := time.After(100 * time.Millisecond)
		select {
		case _, ok := <-result:
			if ok {
				t.Error("Expected channel to be closed after cancellation")
			}
		case <-timeout:
			t.Error("Timeout waiting for channel to close")
		}
	})

	t.Run("empty channels", func(t *testing.T) {
		chanStream := make(chan (<-chan interface{}))

		go func() {
			defer close(chanStream)
			for i := 0; i < 3; i++ {
				ch := make(chan interface{})
				close(ch) // Пустой канал
				chanStream <- ch
			}
		}()

		result := Bridge(nil, chanStream)

		// Должен вернуть пустой результат
		select {
		case val, ok := <-result:
			if ok {
				t.Errorf("Expected no values, got %v", val)
			}
		case <-time.After(100 * time.Millisecond):
			// Ожидаемое поведение - канал закрыт
		}
	})

	t.Run("multiple values per channel", func(t *testing.T) {
		chanStream := make(chan (<-chan interface{}))

		go func() {
			defer close(chanStream)
			for i := 0; i < 2; i++ {
				ch := make(chan interface{}, 2)
				ch <- i * 10
				ch <- i*10 + 1
				close(ch)
				chanStream <- ch
			}
		}()

		result := Bridge(nil, chanStream)

		expected := []interface{}{0, 1, 10, 11}
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

	t.Run("with context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		chanStream := make(chan (<-chan interface{}))

		go func() {
			defer close(chanStream)
			for i := 0; i < 10; i++ {
				ch := make(chan interface{}, 1)
				ch <- i
				close(ch)
				chanStream <- ch
				time.Sleep(10 * time.Millisecond) // Задержка для теста
			}
		}()

		// Конвертируем context в done channel
		done := make(chan interface{})
		go func() {
			<-ctx.Done()
			close(done)
		}()

		result := Bridge(done, chanStream)

		// Читаем несколько значений и отменяем
		<-result
		<-result
		cancel()

		// Ждем немного для обработки отмены
		time.Sleep(50 * time.Millisecond)

		// Пытаемся прочитать еще - должно быть закрыто
		select {
		case _, ok := <-result:
			if ok {
				t.Error("Expected channel to be closed after context cancellation")
			}
		default:
			t.Error("Channel should be closed or empty after cancellation")
		}
	})
}

func TestOrDone(t *testing.T) {
	t.Run("normal operation", func(t *testing.T) {
		source := make(chan interface{}, 3)
		source <- 1
		source <- 2
		source <- 3
		close(source)

		result := orDone(nil, source)

		expected := []interface{}{1, 2, 3}
		var actual []interface{}

		for val := range result {
			actual = append(actual, val)
		}

		if len(actual) != len(expected) {
			t.Errorf("Expected %d values, got %d", len(expected), len(actual))
		}
	})

	t.Run("with cancellation", func(t *testing.T) {
		source := make(chan interface{})
		done := make(chan interface{})

		result := orDone(done, source)

		// Отменяем до того как источник что-либо отправит
		close(done)

		select {
		case _, ok := <-result:
			if ok {
				t.Error("Expected channel to be closed after cancellation")
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("Timeout waiting for channel to close")
		}
	})
}

// Benchmark для измерения производительности
func BenchmarkBridge(b *testing.B) {
	for i := 0; i < b.N; i++ {
		chanStream := make(chan (<-chan interface{}))

		go func() {
			defer close(chanStream)
			for j := 0; j < 1000; j++ {
				ch := make(chan interface{}, 1)
				ch <- j
				close(ch)
				chanStream <- ch
			}
		}()

		result := Bridge(nil, chanStream)
		for range result {
			// Просто потребляем все значения
		}
	}
}
