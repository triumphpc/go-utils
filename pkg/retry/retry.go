package retry

import (
	"context"
	"math/rand/v2"
	"time"
)

// Retry выполняет операцию с повторными попытками согласно конфигурации.
// Параметры:
//   - ctx: контекст для контроля выполнения и отмены
//   - config: конфигурация повторных попыток (макс. попытки, задержки и т.д.)
//   - operation: функция, которую нужно выполнить с повторными попытками
//
// Возвращает:
//   - результат успешного выполнения операции
//   - ошибку (последнюю ошибку операции или ошибку контекста)
func Retry[T any](ctx context.Context, config Config, operation func() (T, error)) (T, error) {
	var result T
	var err error
	currentDelay := config.InitialDelay // Текущая задержка между попытками

	// Основной цикл попыток выполнения
	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		// Проверяем, не отменен ли контекст
		if ctx.Err() != nil {
			return result, ctx.Err()
		}

		// Выполняем операцию
		result, err = operation()
		if err == nil {
			// Успешное выполнение - возвращаем результат
			return result, nil
		}

		// Если это была последняя попытка - возвращаем ошибку
		if attempt == config.MaxAttempts {
			return result, err
		}

		// Добавляем случайный джиттер к задержке, чтобы избежать эффекта "толпы"
		jitter := time.Duration(rand.Float64() * float64(currentDelay))
		currentDelay += jitter
		// Ограничиваем максимальную задержку
		if currentDelay > config.MaxDelay {
			currentDelay = config.MaxDelay
		}

		// Ожидаем перед следующей попыткой с возможностью прерывания
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(currentDelay):
			// Удваиваем задержку для следующей попытки (экспоненциальный рост)
			currentDelay *= 2
		}
	}

	return result, err
}
