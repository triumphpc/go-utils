package workerpool

import (
	"context"
	"fmt"
	"runtime"
	"sync"
)

// Result содержит результат выполнения задачи и возможную ошибку
type Result[R any] struct {
	Value R
	Err   error
}

// WorkerPool - пул воркеров с дженериками
type WorkerPool[T any, R any] struct {
	taskChan   chan T             // Канал для задач
	resultChan chan Result[R]     // Канал для результатов с ошибками
	wg         sync.WaitGroup     // Группа ожидания
	ctx        context.Context    // Контекст для управления жизненным циклом
	cancel     context.CancelFunc // Функция отмены контекста
	numWorkers int                // Количество воркеров
}

// NewWorkerPool создает новый пул воркеров с оптимальным количеством воркеров
func NewWorkerPool[T any, R any](cxt context.Context, taskBuffer int) *WorkerPool[T, R] {
	// Определяем количество воркеров по количеству виртуальных процессоров
	numWorkers := runtime.NumCPU()
	if numWorkers < 1 {
		numWorkers = 1
	}

	ctx, cancel := context.WithCancel(cxt)

	return &WorkerPool[T, R]{
		taskChan:   make(chan T, taskBuffer),
		resultChan: make(chan Result[R], taskBuffer),
		ctx:        ctx,
		cancel:     cancel,
		numWorkers: numWorkers,
	}
}

// WithWorkers устанавливает конкретное количество воркеров
func (wp *WorkerPool[T, R]) WithWorkers(n int) *WorkerPool[T, R] {
	if n > 0 {
		wp.numWorkers = n
	}
	return wp
}

// Start запускает воркеры с обработкой ошибок
func (wp *WorkerPool[T, R]) Start(workerFunc func(context.Context, T) (R, error)) {
	wp.wg.Add(wp.numWorkers)

	for i := 0; i < wp.numWorkers; i++ {
		go func() {
			defer wp.wg.Done()

			for {
				select {
				case <-wp.ctx.Done():
					// Контекст отменен, завершаем работу
					return

				case task, ok := <-wp.taskChan:
					if !ok {
						// Канал закрыт, завершаем работу
						return
					}

					// Выполнение задачи с обработкой паники
					func() {
						defer func() {
							if r := recover(); r != nil {
								// Отправляем информацию о панике как ошибку
								err := fmt.Errorf("panic in worker: %v", r)
								select {
								case <-wp.ctx.Done():
									return
								case wp.resultChan <- Result[R]{Err: err}:
								}
							}
						}()

						// Выполняем задачу
						result, err := workerFunc(wp.ctx, task)

						// Отправка результата, только если контекст не отменен
						select {
						case <-wp.ctx.Done():
							return
						case wp.resultChan <- Result[R]{Value: result, Err: err}:
							// Результат успешно отправлен
						}
					}()
				}
			}
		}()
	}
}

// Submit добавляет задачу в пул
// Возвращает true, если задача была добавлена, и false, если пул закрыт или контекст отменен
func (wp *WorkerPool[T, R]) Submit(task T) bool {
	select {
	case <-wp.ctx.Done():
		// Контекст отменен
		return false
	case wp.taskChan <- task:
		// Задача успешно добавлена
		return true
	}
}

// SubmitWait добавляет задачу в пул и ожидает результат
// Возвращает результат и ошибку, если задача выполнена, или ошибку контекста, если контекст отменен
func (wp *WorkerPool[T, R]) SubmitWait(task T) (R, error) {
	var empty R

	// Проверяем, не отменен ли контекст
	if wp.ctx.Err() != nil {
		return empty, wp.ctx.Err()
	}

	// Создаем канал для получения одного результата
	resultChan := make(chan Result[R], 1)

	// Отправляем задачу
	select {
	case <-wp.ctx.Done():
		return empty, wp.ctx.Err()
	case wp.taskChan <- task:
		// Задача отправлена, ожидаем результат
	}

	// Ждем первый результат из общего канала результатов
	go func() {
		select {
		case result, ok := <-wp.resultChan:
			if ok {
				resultChan <- result
			}
		case <-wp.ctx.Done():
			resultChan <- Result[R]{Value: empty, Err: wp.ctx.Err()}
		}
	}()

	// Получаем результат
	result := <-resultChan
	return result.Value, result.Err
}

// GetResults возвращает канал результатов с ошибками
func (wp *WorkerPool[T, R]) GetResults() <-chan Result[R] {
	return wp.resultChan
}

// Stop останавливает все воркеры, не дожидаясь завершения задач
func (wp *WorkerPool[T, R]) Stop() {
	wp.cancel()
	wp.wg.Wait()

	// Проверяем, не закрыт ли уже канал
	select {
	case _, ok := <-wp.resultChan:
		if ok {
			// Канал не закрыт, можно закрывать
			close(wp.resultChan)
		}
		// Если ok == false, канал уже закрыт
	default:
		// Канал не закрыт и не пуст, закрываем
		close(wp.resultChan)
	}
}

// GracefulStop закрывает канал задач, ожидает завершения всех задач и закрывает пул
func (wp *WorkerPool[T, R]) GracefulStop() {
	// Проверяем, не закрыт ли уже канал
	select {
	case _, ok := <-wp.taskChan:
		if ok {
			// Канал не закрыт, можно закрывать
			close(wp.taskChan)
		}
		// Если ok == false, канал уже закрыт
	default:
		// Канал не закрыт и не пуст, закрываем
		close(wp.taskChan)
	}

	wp.wg.Wait() // Ожидаем завершения всех задач
	wp.cancel()  // Отменяем контекст

	// Проверяем, не закрыт ли уже канал
	select {
	case _, ok := <-wp.resultChan:
		if ok {
			// Канал не закрыт, можно закрывать
			close(wp.resultChan)
		}
		// Если ok == false, канал уже закрыт
	default:
		// Канал не закрыт и не пуст, закрываем
		close(wp.resultChan)
	}
}
