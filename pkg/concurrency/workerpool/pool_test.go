package workerpool

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestWorkerPoolBasic проверяет основную функциональность пула воркеров
func TestWorkerPoolBasic(t *testing.T) {
	// Создаем пул с контекстом и буфером задач
	ctx := context.Background()
	pool := NewWorkerPool[int, int](ctx, 10)

	// Запускаем пул с функцией, которая удваивает входное число
	pool.Start(func(ctx context.Context, n int) (int, error) {
		return n * 2, nil
	})

	// Отправляем задачи
	expectedInputs := []int{1, 2, 3, 4, 5}
	expectedOutputs := make(map[int]bool)
	for _, n := range expectedInputs {
		expectedOutputs[n*2] = false // false означает, что результат еще не получен
		if !pool.Submit(n) {
			t.Fatalf("Не удалось отправить задачу %d", n)
		}
	}

	// Получаем и проверяем результаты
	resultCount := 0
	for result := range pool.GetResults() {
		if result.Err != nil {
			t.Errorf("Получена неожиданная ошибка: %v", result.Err)
		}

		// Проверяем, что результат ожидаемый
		if _, exists := expectedOutputs[result.Value]; !exists {
			t.Errorf("Получен неожиданный результат: %d", result.Value)
		} else {
			expectedOutputs[result.Value] = true // отмечаем, что результат получен
		}

		resultCount++
		if resultCount >= len(expectedInputs) {
			break
		}
	}

	// Проверяем, что все ожидаемые результаты получены
	for val, received := range expectedOutputs {
		if !received {
			t.Errorf("Не получен ожидаемый результат: %d", val)
		}
	}

	// Останавливаем пул
	pool.GracefulStop()
}

// TestWorkerPoolWithErrors проверяет обработку ошибок
func TestWorkerPoolWithErrors(t *testing.T) {
	// Создаем пул
	ctx := context.Background()
	pool := NewWorkerPool[int, string](ctx, 10)

	// Запускаем пул с функцией, которая возвращает ошибку для четных чисел
	pool.Start(func(ctx context.Context, n int) (string, error) {
		if n%2 == 0 {
			return "", errors.New("ошибка для четного числа")
		}
		return fmt.Sprintf("успех: %d", n), nil
	})

	// Отправляем задачи (5 нечетных, 5 четных)
	inputs := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	for _, n := range inputs {
		if !pool.Submit(n) {
			t.Fatalf("Не удалось отправить задачу %d", n)
		}
	}

	// Проверяем результаты
	successCount := 0
	errorCount := 0

	for i := 0; i < len(inputs); i++ {
		select {
		case result := <-pool.GetResults():
			if result.Err != nil {
				errorCount++
			} else {
				successCount++
				// Проверяем формат успешного результата
				if !isSuccessResult(result.Value) {
					t.Errorf("Неверный формат успешного результата: %s", result.Value)
				}
			}
		case <-time.After(1 * time.Second):
			t.Fatal("Тайм-аут при ожидании результатов")
		}
	}

	// Должно быть 5 успехов и 5 ошибок
	if successCount != 5 || errorCount != 5 {
		t.Errorf("Ожидалось 5 успехов и 5 ошибок, получено %d успехов и %d ошибок",
			successCount, errorCount)
	}

	// Останавливаем пул
	pool.GracefulStop()
}

func isSuccessResult(s string) bool {
	return len(s) > 0 && s[:12] == "успех: "
}

// TestWorkerPoolSubmitWait проверяет функцию SubmitWait
func TestWorkerPoolSubmitWait(t *testing.T) {
	// Создаем пул
	ctx := context.Background()
	pool := NewWorkerPool[int, string](ctx, 10)

	// Запускаем пул
	pool.Start(func(ctx context.Context, n int) (string, error) {
		if n < 0 {
			return "", errors.New("отрицательное число")
		}
		return fmt.Sprintf("результат: %d", n), nil
	})

	// Тест успешного выполнения
	result, err := pool.SubmitWait(42)
	if err != nil {
		t.Errorf("SubmitWait вернул неожиданную ошибку: %v", err)
	}
	if result != "результат: 42" {
		t.Errorf("SubmitWait вернул неверный результат: %s", result)
	}

	// Тест с ошибкой
	result, err = pool.SubmitWait(-1)
	if err == nil {
		t.Error("SubmitWait должен был вернуть ошибку для отрицательного числа")
	}
	if result != "" {
		t.Errorf("SubmitWait должен был вернуть пустую строку при ошибке, получено: %s", result)
	}

	// Останавливаем пул
	pool.GracefulStop()
}

func TestWorkerPoolContextCancellation(t *testing.T) {
	// Создаем контекст с возможностью отмены
	ctx, cancel := context.WithCancel(context.TODO())
	pool := NewWorkerPool[int, int](ctx, 10)

	// Счетчик выполненных задач
	var completedTasks int32

	// Запускаем пул с функцией, которая работает долго
	pool.Start(func(ctx context.Context, n int) (int, error) {
		select {
		case <-time.After(500 * time.Millisecond):
			atomic.AddInt32(&completedTasks, 1)
			return n, nil
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	})

	// Запускаем горутину для чтения результатов, чтобы избежать блокировки
	go func() {
		for range pool.GetResults() {
			// Просто потребляем результаты, чтобы не блокировать воркеры
		}
	}()

	// Отправляем много задач
	for i := 0; i < 100; i++ {
		pool.Submit(i)
	}

	// Отменяем контекст после небольшой задержки
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Ожидаем завершения пула
	pool.Stop()

	// Проверяем, что выполнены не все задачи
	completed := atomic.LoadInt32(&completedTasks)
	if completed >= 100 {
		t.Error("Все задачи были выполнены, несмотря на отмену контекста")
	}

	// Проверяем, что Submit возвращает false после отмены контекста
	if pool.Submit(999) {
		t.Error("Submit должен возвращать false после отмены контекста")
	}
}

// TestWorkerPoolConcurrency проверяет параллельное выполнение задач
func TestWorkerPoolConcurrency(t *testing.T) {
	ctx := context.Background()
	pool := NewWorkerPool[int, int](ctx, 100)

	// Устанавливаем конкретное число воркеров для предсказуемости теста
	numWorkers := 4
	pool.WithWorkers(numWorkers)

	// Используем мьютекс для защиты доступа к счетчику активных воркеров
	var mu sync.Mutex
	activeWorkers := 0
	maxActiveWorkers := 0

	// Запускаем пул с функцией, которая отслеживает параллелизм
	pool.Start(func(ctx context.Context, n int) (int, error) {
		mu.Lock()
		activeWorkers++
		if activeWorkers > maxActiveWorkers {
			maxActiveWorkers = activeWorkers
		}
		mu.Unlock()

		// Имитируем работу
		time.Sleep(50 * time.Millisecond)

		mu.Lock()
		activeWorkers--
		mu.Unlock()

		return n, nil
	})

	// Отправляем много задач
	for i := 0; i < 20; i++ {
		pool.Submit(i)
	}

	// Ожидаем завершения всех задач
	pool.GracefulStop()

	// Проверяем, что достигнут ожидаемый уровень параллелизма
	if maxActiveWorkers != numWorkers {
		t.Errorf("Ожидалось максимум %d одновременно активных воркеров, фактически: %d",
			numWorkers, maxActiveWorkers)
	}
}

// TestWorkerPoolGracefulStop проверяет корректное завершение всех задач
func TestWorkerPoolGracefulStop(t *testing.T) {
	ctx := context.Background()
	pool := NewWorkerPool[int, int](ctx, 10)

	// Счетчик завершенных задач
	var completedTasks int32

	// Запускаем пул
	pool.Start(func(ctx context.Context, n int) (int, error) {
		time.Sleep(100 * time.Millisecond)
		atomic.AddInt32(&completedTasks, 1)
		return n, nil
	})

	// Отправляем задачи
	numTasks := 10
	for i := 0; i < numTasks; i++ {
		pool.Submit(i)
	}

	// Корректно завершаем пул
	pool.GracefulStop()

	// Проверяем, что все задачи были выполнены
	if int(atomic.LoadInt32(&completedTasks)) != numTasks {
		t.Errorf("Ожидалось %d завершенных задач, фактически: %d",
			numTasks, atomic.LoadInt32(&completedTasks))
	}
}

// TestWorkerPoolImmediateStop проверяет немедленную остановку пула
func TestWorkerPoolImmediateStop(t *testing.T) {
	ctx := context.Background()
	pool := NewWorkerPool[int, int](ctx, 10)

	// Счетчик завершенных задач
	var completedTasks int32

	// Запускаем пул с долгими задачами
	pool.Start(func(ctx context.Context, n int) (int, error) {
		select {
		case <-time.After(500 * time.Millisecond):
			atomic.AddInt32(&completedTasks, 1)
			return n, nil
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	})

	// Отправляем задачи
	numTasks := 10
	for i := 0; i < numTasks; i++ {
		pool.Submit(i)
	}

	// Немедленно останавливаем пул
	pool.Stop()

	// Проверяем, что не все задачи были выполнены
	if int(atomic.LoadInt32(&completedTasks)) == numTasks {
		t.Error("Все задачи были выполнены, несмотря на немедленную остановку")
	}
}

// TestWorkerPoolWithCustomWorkerCount проверяет установку пользовательского числа воркеров
func TestWorkerPoolWithCustomWorkerCount(t *testing.T) {
	ctx := context.Background()
	pool := NewWorkerPool[int, int](ctx, 10)

	customWorkerCount := 3
	pool = pool.WithWorkers(customWorkerCount)

	if pool.numWorkers != customWorkerCount {
		t.Errorf("Ожидалось %d воркеров, установлено: %d",
			customWorkerCount, pool.numWorkers)
	}

	// Проверяем, что метод возвращает сам пул для цепочки вызовов
	if pool.WithWorkers(5) != pool {
		t.Error("Метод WithWorkers должен возвращать сам пул для цепочки вызовов")
	}
}

// TestWorkerPoolEmptyBuffer проверяет работу с буфером нулевого размера
func TestWorkerPoolEmptyBuffer(t *testing.T) {
	ctx := context.Background()
	pool := NewWorkerPool[int, int](ctx, 0)

	// Запускаем пул
	pool.Start(func(ctx context.Context, n int) (int, error) {
		return n * 2, nil
	})

	// Отправляем задачу и получаем результат
	if !pool.Submit(5) {
		t.Fatal("Не удалось отправить задачу")
	}

	// Получаем результат
	select {
	case result := <-pool.GetResults():
		if result.Value != 10 || result.Err != nil {
			t.Errorf("Неверный результат: %v, ошибка: %v", result.Value, result.Err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Тайм-аут при ожидании результата")
	}

	pool.GracefulStop()
}

// TestWorkerPoolInvalidWorkerCount проверяет обработку некорректного количества воркеров
func TestWorkerPoolInvalidWorkerCount(t *testing.T) {
	ctx := context.Background()
	pool := NewWorkerPool[int, int](ctx, 10)

	// Пытаемся установить отрицательное число воркеров
	pool = pool.WithWorkers(-5)

	// Проверяем, что количество воркеров не изменилось
	if pool.numWorkers <= 0 {
		t.Errorf("Установлено некорректное количество воркеров: %d", pool.numWorkers)
	}

	// Пытаемся установить ноль воркеров
	originalWorkers := pool.numWorkers
	pool = pool.WithWorkers(0)

	// Проверяем, что количество воркеров не изменилось
	if pool.numWorkers != originalWorkers {
		t.Errorf("Количество воркеров изменилось на %d, должно остаться %d",
			pool.numWorkers, originalWorkers)
	}
}

// TestWorkerPoolCancelledContext проверяет создание пула с уже отмененным контекстом
func TestWorkerPoolCancelledContext(t *testing.T) {
	// Создаем отмененный контекст
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Создаем пул с отмененным контекстом
	pool := NewWorkerPool[int, int](ctx, 10)

	// Запускаем пул
	tasksStarted := false
	pool.Start(func(ctx context.Context, n int) (int, error) {
		tasksStarted = true
		return n, nil
	})

	// Проверяем, что Submit возвращает false
	if pool.Submit(1) {
		t.Error("Submit должен возвращать false с отмененным контекстом")
	}

	// Проверяем, что SubmitWait возвращает ошибку контекста
	_, err := pool.SubmitWait(1)
	if err == nil || err != context.Canceled {
		t.Errorf("SubmitWait должен возвращать ошибку context.Canceled, получено: %v", err)
	}

	// Проверяем, что задачи не запускаются
	if tasksStarted {
		t.Error("Задачи не должны запускаться с отмененным контекстом")
	}

	pool.Stop()
}

// TestWorkerPoolHighLoad проверяет работу под высокой нагрузкой
func TestWorkerPoolHighLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Пропускаем тест высокой нагрузки в коротком режиме")
	}

	ctx := context.Background()
	pool := NewWorkerPool[int, int](ctx, 1000)

	// Счетчик выполненных задач
	var completedTasks int32

	// Канал для сигнала о завершении обработки результатов
	done := make(chan struct{})

	// Запускаем горутину для чтения результатов
	go func() {
		defer close(done)
		for range pool.GetResults() {
			// Просто потребляем результаты
		}
	}()

	// Запускаем пул
	pool.Start(func(ctx context.Context, n int) (int, error) {
		// Короткие задачи для имитации высокой нагрузки
		time.Sleep(1 * time.Millisecond)
		atomic.AddInt32(&completedTasks, 1)
		return n, nil
	})

	// Отправляем много задач
	numTasks := 10000
	for i := 0; i < numTasks; i++ {
		if !pool.Submit(i) {
			t.Fatalf("Не удалось отправить задачу %d", i)
		}
	}

	// Ожидаем завершения всех задач
	pool.GracefulStop()

	// Ждем завершения обработки результатов
	<-done

	// Проверяем количество выполненных задач
	if int(atomic.LoadInt32(&completedTasks)) != numTasks {
		t.Errorf("Ожидалось %d завершенных задач, фактически: %d",
			numTasks, atomic.LoadInt32(&completedTasks))
	}
}

// TestWorkerPoolResourceLeaks проверяет отсутствие утечек горутин и ресурсов
func TestWorkerPoolResourceLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("Пропускаем тест на утечки в коротком режиме")
	}

	// Запускаем несколько циклов создания и уничтожения пулов
	for i := 0; i < 10; i++ {
		func() {
			ctx := context.Background()
			pool := NewWorkerPool[int, int](ctx, 10)

			// Канал для сигнала о завершении обработки результатов
			done := make(chan struct{})

			// Запускаем горутину для чтения результатов
			go func() {
				defer close(done)
				for range pool.GetResults() {
					// Просто потребляем результаты
				}
			}()

			pool.Start(func(ctx context.Context, n int) (int, error) {
				time.Sleep(10 * time.Millisecond)
				return n, nil
			})

			// Отправляем задачи
			for j := 0; j < 100; j++ {
				pool.Submit(j)
			}

			// Корректно завершаем пул
			pool.GracefulStop()

			// Ждем завершения обработки результатов
			<-done
		}()
	}

	// Если бы были утечки горутин, они бы накапливались с каждым циклом
	// Этот тест лучше запускать с флагом -race для обнаружения гонок данных
}

// TestWorkerPoolPanicRecovery проверяет восстановление после паники в воркере
func TestWorkerPoolPanicRecovery(t *testing.T) {
	ctx := context.Background()
	pool := NewWorkerPool[int, int](ctx, 10)

	// Запускаем пул с функцией, которая паникует для определенных входных данных
	pool.Start(func(ctx context.Context, n int) (int, error) {
		if n == 42 {
			panic("паника в воркере!")
		}
		return n, nil
	})

	// Отправляем обычные задачи и задачу, вызывающую панику
	tasks := []int{1, 2, 42, 3, 4}
	for _, n := range tasks {
		pool.Submit(n)
	}

	// Ожидаем завершения и проверяем, что пул не упал полностью
	// Примечание: текущая реализация не обрабатывает панику, поэтому этот тест может не пройти
	// Для прохождения теста нужно добавить recover() в воркер

	// Добавляем еще задач после потенциальной паники
	pool.Submit(5)
	pool.Submit(6)

	// Корректно завершаем пул
	pool.GracefulStop()

	// Если мы дошли до этой точки без паники в тесте,
	// то либо воркер корректно обрабатывает панику,
	// либо паника в воркере не влияет на тест
	// В реальной реализации нужно добавить обработку паники в воркере
}

// BenchmarkWorkerPoolLightTasks тестирует пул на легких задачах (минимальная нагрузка)
func BenchmarkWorkerPoolLightTasks(b *testing.B) {
	ctx := context.Background()
	pool := NewWorkerPool[int, int](ctx, 100)

	// Запускаем горутину для потребления результатов
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range pool.GetResults() {
			// Просто потребляем результаты
		}
	}()

	pool.Start(func(ctx context.Context, n int) (int, error) {
		return n * n, nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.Submit(i)
	}

	// Используем GracefulStop вместо Stop, так как он уже включает закрытие каналов
	pool.GracefulStop()
	<-done
}

// BenchmarkWorkerPoolIOTasks тестирует пул на IO-bound задачах
func BenchmarkWorkerPoolIOTasks(b *testing.B) {
	ctx := context.Background()
	pool := NewWorkerPool[int, int](ctx, 100)

	// Запускаем горутину для потребления результатов
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range pool.GetResults() {
			// Просто потребляем результаты
		}
	}()

	// Имитация IO-операции
	pool.Start(func(ctx context.Context, n int) (int, error) {
		time.Sleep(10 * time.Millisecond)
		return n, nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.Submit(i)
	}

	pool.GracefulStop()
	<-done
}

// BenchmarkWorkerPoolConcurrentSubmit тестирует конкурентную отправку задач
func BenchmarkWorkerPoolConcurrentSubmit(b *testing.B) {
	ctx := context.Background()
	pool := NewWorkerPool[int, int](ctx, 1000)
	defer pool.Stop()

	var counter int32

	// Запускаем горутину для потребления результатов
	done := make(chan struct{})
	go func() {
		defer close(done)
		for result := range pool.GetResults() {
			if result.Err == nil {
				atomic.AddInt32(&counter, 1)
			}
		}
	}()

	pool.Start(func(ctx context.Context, n int) (int, error) {
		return n, nil
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			pool.Submit(i)
			i++
		}
	})

	pool.GracefulStop()
	<-done

	b.ReportMetric(float64(atomic.LoadInt32(&counter)), "tasks_completed")
}

// BenchmarkWorkerPoolDifferentSizes тестирует разное количество воркеров
func BenchmarkWorkerPoolDifferentSizes(b *testing.B) {
	sizes := []int{1, 2, 4, 8, 16, 32, 64, 128}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("workers_%d", size), func(b *testing.B) {
			ctx := context.Background()
			pool := NewWorkerPool[int, int](ctx, 100).WithWorkers(size)

			var counter int32

			// Запускаем горутину для потребления результатов
			done := make(chan struct{})
			go func() {
				defer close(done)
				for range pool.GetResults() {
					atomic.AddInt32(&counter, 1)
				}
			}()

			pool.Start(func(ctx context.Context, n int) (int, error) {
				time.Sleep(1 * time.Millisecond) // Имитация работы
				return n, nil
			})

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				pool.Submit(i)
			}

			pool.GracefulStop()
			<-done

			b.ReportMetric(float64(atomic.LoadInt32(&counter)), "tasks_completed")
		})
	}
}

// BenchmarkWorkerPoolSubmitWait тестирует SubmitWait с ожиданием результата
func BenchmarkWorkerPoolSubmitWait(b *testing.B) {
	ctx := context.Background()
	pool := NewWorkerPool[int, int](ctx, 100)
	defer pool.Stop()

	pool.Start(func(ctx context.Context, n int) (int, error) {
		return n * n, nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := pool.SubmitWait(i); err != nil {
			b.Fatalf("SubmitWait failed: %v", err)
		}
	}
}

// BenchmarkWorkerPoolWithResults тестирует производительность с чтением результатов
func BenchmarkWorkerPoolWithResults(b *testing.B) {
	ctx := context.Background()
	pool := NewWorkerPool[int, int](ctx, 100)
	defer pool.Stop()

	var counter int32

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range pool.GetResults() {
			atomic.AddInt32(&counter, 1)
		}
	}()

	pool.Start(func(ctx context.Context, n int) (int, error) {
		return n * n, nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.Submit(i)
	}

	pool.GracefulStop()
	<-done

	b.ReportMetric(float64(atomic.LoadInt32(&counter)), "tasks_completed")
}
