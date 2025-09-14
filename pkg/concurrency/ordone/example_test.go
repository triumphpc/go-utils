package ordone

import (
	"fmt"
	"time"
)

func ExampleOrDone() {
	// Создаем канал с данными
	dataChan := make(chan interface{}, 3)
	dataChan <- "hello"
	dataChan <- "world"
	dataChan <- "!"
	close(dataChan)

	// Используем OrDone для безопасного чтения
	for val := range OrDone(nil, dataChan) {
		fmt.Println(val)
	}
	// Output:
	// hello
	// world
	// !
}

func ExampleOrDone_withCancellation() {
	done := make(chan interface{})
	dataChan := make(chan interface{}, 5)

	// Заполняем канал данными
	for i := 0; i < 5; i++ {
		dataChan <- i
	}

	// Горутина для имитации отмены через некоторое время
	go func() {
		time.Sleep(10 * time.Millisecond)
		close(done)
	}()

	// Чтение с гарантией завершения
	count := 0
	for range OrDone(done, dataChan) {
		count++
		if count >= 2 {
			break
		}
	}

	fmt.Printf("Read %d values before cancellation", count)
	// Output: Read 2 values before cancellation
}
