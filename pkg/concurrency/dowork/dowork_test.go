package dowork

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestDoWork_BasicFunctionality(t *testing.T) {
	t.Run("normal processing", func(t *testing.T) {
		input := make(chan string, 3)
		done := make(chan struct{})
		defer close(done)

		var processed []string
		var mu sync.Mutex

		processor := func(s string) {
			mu.Lock()
			defer mu.Unlock()
			processed = append(processed, s)
		}

		completed := DoWork(input, done, processor)

		// Отправляем данные
		input <- "test1"
		input <- "test2"
		input <- "test3"
		close(input)

		// Ждем завершения
		<-completed

		expected := []string{"test1", "test2", "test3"}
		if len(processed) != len(expected) {
			t.Errorf("Expected %d processed items, got %d", len(expected), len(processed))
		}

		for i, v := range processed {
			if v != expected[i] {
				t.Errorf("At index %d: expected %s, got %s", i, expected[i], v)
			}
		}
	})

	t.Run("with cancellation", func(t *testing.T) {
		input := make(chan string)
		done := make(chan struct{})

		processed := make([]string, 0)
		var mu sync.Mutex

		processor := func(s string) {
			mu.Lock()
			defer mu.Unlock()
			processed = append(processed, s)
		}

		completed := DoWork(input, done, processor)

		// Отправляем одно значение
		input <- "test1"

		// Даем время на обработку
		time.Sleep(10 * time.Millisecond)

		// Отменяем
		close(done)

		// Ждем завершения
		<-completed

		if len(processed) != 1 || processed[0] != "test1" {
			t.Errorf("Expected ['test1'], got %v", processed)
		}
	})

	t.Run("immediate cancellation", func(t *testing.T) {
		input := make(chan string)
		done := make(chan struct{})
		close(done) // Немедленная отмена

		processed := false
		processor := func(s string) {
			processed = true
		}

		completed := DoWork(input, done, processor)

		// Должно завершиться немедленно
		select {
		case <-completed:
			// Ожидаемое поведение
		case <-time.After(100 * time.Millisecond):
			t.Error("Timeout waiting for completion")
		}

		if processed {
			t.Error("No processing should occur with immediate cancellation")
		}
	})
}

func TestDoWorkWithContext(t *testing.T) {
	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		input := make(chan string)

		var processed []string
		processor := func(s string) {
			processed = append(processed, s)
		}

		completed := DoWorkWithContext(ctx, input, processor)

		// Отправляем данные
		input <- "test1"
		input <- "test2"

		// Отменяем контекст
		cancel()

		// Ждем завершения
		<-completed

		// Могли обработать 0, 1 или 2 элемента в зависимости от timing
		if len(processed) > 2 {
			t.Errorf("Unexpected number of processed items: %d", len(processed))
		}
	})

	t.Run("context timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		input := make(chan string)
		processed := false

		processor := func(s string) {
			processed = true
		}

		completed := DoWorkWithContext(ctx, input, processor)

		// Ждем таймаута контекста
		<-completed

		if processed {
			t.Error("No processing should occur with context timeout")
		}
	})
}

func TestDoWorkBuffered(t *testing.T) {
	t.Run("buffered processing", func(t *testing.T) {
		input := make(chan string, 5)
		done := make(chan struct{})
		defer close(done)

		var processed []string
		var mu sync.Mutex
		var wg sync.WaitGroup

		processor := func(s string) {
			defer wg.Done()
			mu.Lock()
			defer mu.Unlock()
			processed = append(processed, s)
			time.Sleep(10 * time.Millisecond) // Имитация работы
		}

		completed := DoWorkBuffered(input, done, processor, 2) // Максимум 2 параллельных

		// Отправляем данные
		wg.Add(5)
		for i := 0; i < 5; i++ {
			input <- fmt.Sprintf("test%d", i)
		}
		close(input)

		// Ждем завершения всех обработчиков
		wg.Wait()
		<-completed

		if len(processed) != 5 {
			t.Errorf("Expected 5 processed items, got %d", len(processed))
		}
	})

	t.Run("buffered with cancellation", func(t *testing.T) {
		input := make(chan string)
		done := make(chan struct{})

		processed := 0
		var mu sync.Mutex

		processor := func(s string) {
			mu.Lock()
			defer mu.Unlock()
			processed++
			time.Sleep(100 * time.Millisecond)
		}

		completed := DoWorkBuffered(input, done, processor, 3)

		// Отправляем несколько значений
		go func() {
			for i := 0; i < 10; i++ {
				input <- fmt.Sprintf("test%d", i)
			}
		}()

		// Даем немного времени на обработку
		time.Sleep(50 * time.Millisecond)

		// Отменяем
		close(done)

		// Ждем завершения
		<-completed

		if processed > 3 { // Не должно быть больше чем workers
			t.Errorf("Expected at most 3 processed items, got %d", processed)
		}
	})
}

func BenchmarkDoWork(b *testing.B) {
	b.Run("sequential", func(b *testing.B) {
		input := make(chan string, b.N)
		done := make(chan struct{})
		defer close(done)

		processor := func(s string) {
			// Минимальная обработка
		}

		for i := 0; i < b.N; i++ {
			input <- "test"
		}
		close(input)

		completed := DoWork(input, done, processor)
		<-completed
	})

	b.Run("buffered", func(b *testing.B) {
		input := make(chan string, b.N)
		done := make(chan struct{})
		defer close(done)

		var wg sync.WaitGroup
		processor := func(s string) {
			defer wg.Done()
			// Минимальная обработка
		}

		wg.Add(b.N)
		for i := 0; i < b.N; i++ {
			input <- "test"
		}
		close(input)

		completed := DoWorkBuffered(input, done, processor, 10)
		wg.Wait()
		<-completed
	})
}
