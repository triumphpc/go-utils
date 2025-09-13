package limiter

import (
	"sync"
	"testing"
	"time"
)

func TestNewLeakyBucket(t *testing.T) {
	tests := []struct {
		name     string
		rate     int64
		capacity int64
		wantErr  bool
	}{
		{"Valid parameters", 10, 100, false},
		{"Zero rate", 0, 100, true},
		{"Zero capacity", 10, 0, true},
		{"Negative rate", -1, 100, true},
		{"Negative capacity", 10, -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); (r != nil) != tt.wantErr {
					t.Errorf("NewLeakyBucket() panic = %v, wantErr %v", r, tt.wantErr)
				}
			}()

			lb := NewLeakyBucket(tt.rate, tt.capacity)
			if lb == nil && !tt.wantErr {
				t.Error("NewLeakyBucket() returned nil")
			}

			if lb != nil {
				defer lb.Stop()
			}
		})
	}
}

func TestLeakyBucket_Allow(t *testing.T) {
	t.Run("Should allow requests within capacity", func(t *testing.T) {
		capacity := int64(5)
		lb := NewLeakyBucket(10, capacity)
		defer lb.Stop()

		// Должны разрешить все запросы в пределах емкости
		for i := 0; i < int(capacity); i++ {
			if !lb.Allow() {
				t.Errorf("Allow() returned false on request %d, expected true", i)
			}
		}
	})

	t.Run("Should reject requests beyond capacity", func(t *testing.T) {
		capacity := int64(3)
		lb := NewLeakyBucket(10, capacity)
		defer lb.Stop()

		// Заполняем очередь
		for i := 0; i < int(capacity); i++ {
			lb.Allow()
		}

		// Следующий запрос должен быть отклонен
		if lb.Allow() {
			t.Error("Allow() returned true when bucket should be full")
		}
	})

	t.Run("Should allow requests after leak", func(t *testing.T) {
		rate := int64(2) // 2 запроса в секунду
		capacity := int64(2)
		lb := NewLeakyBucket(rate, capacity)
		defer lb.Stop()

		// Заполняем очередь
		lb.Allow()
		lb.Allow()

		// Очередь полная - запрос отклонен
		if lb.Allow() {
			t.Error("Allow() should return false when bucket is full")
		}

		// Ждем, пока ведро "протечет"
		time.Sleep(time.Second/time.Duration(rate) + 50*time.Millisecond)

		// Теперь должен быть доступен слот
		if !lb.Allow() {
			t.Error("Allow() should return true after leak")
		}
	})
}

func TestLeakyBucket_ConcurrentAccess(t *testing.T) {
	capacity := int64(100)
	lb := NewLeakyBucket(1000, capacity) // Высокая скорость для теста
	defer lb.Stop()

	var wg sync.WaitGroup
	allowed := make(chan bool, 200) // Буферизованный канал для результатов

	// Запускаем множество горутин
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allowed <- lb.Allow()
		}()
	}

	wg.Wait()
	close(allowed)

	// Подсчитываем разрешенные запросы
	var allowedCount int
	for result := range allowed {
		if result {
			allowedCount++
		}
	}

	// Не должно быть разрешено больше запросов, чем емкость
	if allowedCount > int(capacity) {
		t.Errorf("Allowed %d requests, but capacity is only %d", allowedCount, capacity)
	}

	t.Logf("Allowed %d out of 200 requests (capacity: %d)", allowedCount, capacity)
}

func TestLeakyBucket_LeakRate(t *testing.T) {
	rate := int64(5) // 5 запросов в секунду
	capacity := int64(10)
	lb := NewLeakyBucket(rate, capacity)
	defer lb.Stop()

	// Заполняем очередь полностью
	for i := 0; i < int(capacity); i++ {
		if !lb.Allow() {
			t.Fatalf("Failed to fill bucket on request %d", i)
		}
	}

	// Ждем 1 секунду + небольшой запас
	time.Sleep(1100 * time.Millisecond)

	// Должно освободиться примерно rate слотов
	allowedCount := 0
	for i := 0; i < int(rate)+2; i++ { // +2 для погрешности
		if lb.Allow() {
			allowedCount++
		}
	}

	// Должно быть доступно примерно rate слотов
	expectedMin := int(rate) - 1 // Допускаем погрешность в 1
	expectedMax := int(rate) + 1

	if allowedCount < expectedMin || allowedCount > expectedMax {
		t.Errorf("Expected %d-%d available slots after 1 second, got %d",
			expectedMin, expectedMax, allowedCount)
	}
}

func TestLeakyBucket_ZeroRateEdgeCase(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic with zero rate, but none occurred")
		}
	}()

	NewLeakyBucket(0, 10) // Должен вызвать панику
}

func TestLeakyBucket_HighPrecisionRate(t *testing.T) {
	// Тест для высокой скорости обработки
	highRate := int64(1000)
	lb := NewLeakyBucket(highRate, 5000)
	defer lb.Stop()

	// Быстро заполняем очередь
	for i := 0; i < 1000; i++ {
		lb.Allow()
	}

	// Короткая пауза
	time.Sleep(10 * time.Millisecond)

	// Должны быть доступны некоторые слоты
	allowed := 0
	for i := 0; i < 100; i++ {
		if lb.Allow() {
			allowed++
		}
	}

	if allowed == 0 {
		t.Error("No slots available even with high leak rate")
	}
}

func BenchmarkLeakyBucket_Allow(b *testing.B) {
	lb := NewLeakyBucket(1000000, 1000000) // Высокие значения для бенчмарка
	defer lb.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			lb.Allow()
		}
	})
}

func BenchmarkLeakyBucket_AllowWithLeak(b *testing.B) {
	lb := NewLeakyBucket(100000, 100000)
	defer lb.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			lb.Allow()
		}
	})
}
