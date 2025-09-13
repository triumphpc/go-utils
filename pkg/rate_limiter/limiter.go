package limiter

import (
	"sync"
	"time"
)

// LeakyBucket реализует алгоритм "протекающего ведра" для ограничения скорости
// Позволяет контролировать частоту выполнения операций
type LeakyBucket struct {
	rate     int64         // Скорость протекания (запросов в секунду)
	capacity int64         // Емкость ведра (размер очереди)
	queue    chan struct{} // Ограниченная очередь для хранения запросов
	mu       sync.Mutex    // Мьютекс для обеспечения потокобезопасности
	stopCh   chan struct{} // Канал для сигнала остановки
}

// NewLeakyBucket создает новый экземпляр LeakyBucket
// rate - количество разрешенных запросов в секунду
// capacity - максимальный размер очереди (емкость ведра)
func NewLeakyBucket(rate, capacity int64) *LeakyBucket {
	if rate <= 0 {
		panic("rate must be greater than 0")
	}
	if capacity <= 0 {
		panic("capacity must be greater than 0")
	}

	lb := &LeakyBucket{
		rate:     rate,
		capacity: capacity,
		queue:    make(chan struct{}, capacity),
		stopCh:   make(chan struct{}),
	}
	go lb.leak() // Запускаем горутину для "протекания" ведра
	return lb
}

// Allow проверяет, разрешено ли выполнение операции
// Возвращает true если есть место в очереди (запрос разрешен)
// Возвращает false если очередь полная (запрос отклонен)
func (lb *LeakyBucket) Allow() bool {
	select {
	case lb.queue <- struct{}{}: // Есть место в очереди - запрос принимается
		return true
	default: // Очередь полная - запрос отклоняется
		return false
	}
}

// leak реализует процесс "протекания" ведра
// Удаляет запросы из очереди с заданной скоростью
func (lb *LeakyBucket) leak() {
	// Создаем тикер с интервалом, соответствующим скорости протекания
	ticker := time.NewTicker(time.Second / time.Duration(lb.rate))
	defer ticker.Stop() // Гарантируем остановку тикера при выходе

	for {
		select {
		case <-lb.stopCh: // Получен сигнал остановки
			return // Завершаем работу горутины
		case <-ticker.C: // Сработал тикер - время "протечь"
			select {
			case <-lb.queue: // Удаляем один запрос из очереди (если есть)
				// Здесь можно добавить логику обработки запроса
			default: // Очередь пустая - ничего не делаем
			}
		}
	}
}

// Stop корректно останавливает работу LeakyBucket
// Закрывает канал stopCh, что приводит к завершению горутины leak()
func (lb *LeakyBucket) Stop() {
	close(lb.stopCh) // Отправляем сигнал остановки
}
