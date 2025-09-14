package bridge

// Bridge преобразует канал каналов в простой канал, объединяя все значения
// из вложенных каналов в единый поток. Это полезно для обработки данных
// из нескольких источников как из одного последовательного потока.
//
// Параметры:
//   - done: канал для сигнала прерывания работы (может быть nil)
//   - chanStream: входной канал, содержащий каналы с данными
//
// Возвращает: канал с объединенными значениями из всех вложенных каналов
func Bridge(
	done <-chan interface{},
	chanStream <-chan <-chan interface{},
) <-chan interface{} {
	valStream := make(chan interface{})

	go func() {
		defer close(valStream)

		for {
			// Извлекаем следующий канал из потока каналов
			var stream <-chan interface{}
			select {
			case maybeStream, ok := <-chanStream:
				if !ok {
					return // Поток каналов исчерпан
				}
				stream = maybeStream
			case <-done:
				return // Получен сигнал прерывания
			}

			// Читаем все значения из текущего канала и отправляем в выходной поток
			for val := range orDone(done, stream) {
				select {
				case valStream <- val:
				case <-done:
					return // Получен сигнал прерывания
				}
			}
		}
	}()

	return valStream
}

// orDone создает обертку вокруг канала, которая завершает работу
// при получении сигнала из канала done или закрытии исходного канала
func orDone(done, c <-chan interface{}) <-chan interface{} {
	valStream := make(chan interface{})

	go func() {
		defer close(valStream)

		for {
			select {
			case <-done:
				return
			case v, ok := <-c:
				if !ok {
					return
				}
				select {
				case valStream <- v:
				case <-done:
				}
			}
		}
	}()

	return valStream
}
