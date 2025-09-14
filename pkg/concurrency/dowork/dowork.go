// Package dowork предоставляет паттерн для создания управляемых рабочих горутин
// с поддержкой graceful shutdown через done-канал.
package dowork

import (
	"context"
)

// DoWork создает управляемую горутину для обработки данных из входного канала.
// Функция обеспечивает безопасное завершение работы при получении сигнала отмены.
//
// Параметры:
//   - strings: канал с входными данными для обработки
//   - done: канал для сигнала прерывания работы (может быть nil)
//   - processor: функция-обработчик для каждого элемента данных
//
// Возвращает: канал завершения, который закрывается когда горутина завершит работу
//
// Пример использования:
//
//	done := make(chan struct{})
//	completed := DoWork(inputChan, done, func(s string) {
//	    fmt.Println("Processing:", s)
//	})
//
//	// Для отмены: close(done)
//	// Для ожидания завершения: <-completed
func DoWork(
	strings <-chan string,
	done <-chan struct{},
	processor func(string),
) <-chan struct{} {
	completed := make(chan struct{})

	go func() {
		defer func() {
			close(completed)
		}()

		for {
			select {
			case <-done:
				// Получен сигнал завершения
				return
			case s, ok := <-strings:
				if !ok {
					// Входной канал закрыт
					return
				}
				// Обрабатываем данные
				processor(s)
			}
		}
	}()

	return completed
}

// DoWorkWithContext создает управляемую горутину с использованием context.Context
// для более современного подхода к управлению жизненным циклом.
//
// Параметры:
//   - ctx: контекст для управления временем жизни горутины
//   - strings: канал с входными данными
//   - processor: функция-обработчик
//
// Возвращает: канал завершения
func DoWorkWithContext(
	ctx context.Context,
	strings <-chan string,
	processor func(string),
) <-chan struct{} {
	completed := make(chan struct{})

	go func() {
		defer close(completed)

		for {
			select {
			case <-ctx.Done():
				// Контекст отменен
				return
			case s, ok := <-strings:
				if !ok {
					// Входной канал закрыт
					return
				}
				processor(s)
			}
		}
	}()

	return completed
}

// DoWorkBuffered создает рабочую горутину с буферизованной обработкой,
// позволяя обрабатывать несколько элементов одновременно с ограничением.
//
// Параметры:
//   - strings: канал с входными данными
//   - done: канал отмены
//   - processor: функция-обработчик
//   - workers: количество параллельных обработчиков
//
// Возвращает: канал завершения
func DoWorkBuffered(
	strings <-chan string,
	done <-chan struct{},
	processor func(string),
	workers int,
) <-chan struct{} {
	completed := make(chan struct{})

	go func() {
		defer close(completed)

		// Семафор для ограничения параллелизма
		sem := make(chan struct{}, workers)
		defer close(sem)

		for {
			select {
			case <-done:
				return
			case s, ok := <-strings:
				if !ok {
					return
				}

				// Захватываем слот в семафоре
				select {
				case sem <- struct{}{}:
				case <-done:
					return
				}

				// Запускаем обработку в отдельной горутине
				go func(data string) {
					defer func() { <-sem }() // Освобождаем слот
					processor(data)
				}(s)
			}
		}
	}()

	return completed
}
