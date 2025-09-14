package bridge

import (
	"fmt"
)

func ExampleBridge() {
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

	// Используем Bridge для объединения
	result := Bridge(nil, chanStream)

	for val := range result {
		fmt.Printf("%v ", val)
	}
	// Output: 0 1 2
}

func ExampleBridge_withCancellation() {
	done := make(chan interface{})
	defer close(done)

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

	// Читаем только первые 2 значения
	fmt.Printf("%v ", <-result)
	fmt.Printf("%v ", <-result)

	// Output: 0 1
}
