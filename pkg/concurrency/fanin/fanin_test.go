package fanin

import (
	"sync"
	"testing"
	"time"
)

func TestFanIn_EmptyChannels(t *testing.T) {
	// Тест с пустым списком каналов
	result := FanIn()

	select {
	case _, ok := <-result:
		if ok {
			t.Error("Expected channel to be closed immediately")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected channel to be closed")
	}
}

func TestFanIn_SingleChannel(t *testing.T) {
	// Тест с одним каналом
	input := make(chan any, 3)
	input <- "test1"
	input <- "test2"
	input <- "test3"
	close(input)

	output := FanIn(input)

	// Проверяем, что все данные получены
	expected := []any{"test1", "test2", "test3"}
	var received []any

	for data := range output {
		received = append(received, data)
	}

	if len(received) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(received))
	}
}

func TestFanIn_MultipleChannels(t *testing.T) {
	// Тест с несколькими каналами
	ch1 := make(chan any, 2)
	ch2 := make(chan any, 2)
	ch3 := make(chan any, 2)

	ch1 <- "ch1-1"
	ch1 <- "ch1-2"
	ch2 <- "ch2-1"
	ch2 <- "ch2-2"
	ch3 <- "ch3-1"
	ch3 <- "ch3-2"

	close(ch1)
	close(ch2)
	close(ch3)

	output := FanIn(ch1, ch2, ch3)

	// Собираем все данные
	var received []any
	for data := range output {
		received = append(received, data)
	}

	// Должны получить 6 элементов
	if len(received) != 6 {
		t.Errorf("Expected 6 items, got %d", len(received))
	}
}

func TestFanIn_ConcurrentData(t *testing.T) {
	// Тест с конкурентной отправкой данных
	ch1 := make(chan any)
	ch2 := make(chan any)
	ch3 := make(chan any)

	output := FanIn(ch1, ch2, ch3)

	// Запускаем горутины для отправки данных
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		for i := 0; i < 3; i++ {
			ch1 <- i
		}
		close(ch1)
	}()

	go func() {
		defer wg.Done()
		for i := 10; i < 13; i++ {
			ch2 <- i
		}
		close(ch2)
	}()

	go func() {
		defer wg.Done()
		for i := 20; i < 23; i++ {
			ch3 <- i
		}
		close(ch3)
	}()

	// Собираем результаты
	var received []any
	for data := range output {
		received = append(received, data)
	}

	wg.Wait()

	// Должны получить 9 элементов
	if len(received) != 9 {
		t.Errorf("Expected 9 items, got %d", len(received))
	}
}

func TestFanIn_ChannelClosedImmediately(t *testing.T) {
	// Тест с сразу закрытыми каналами
	ch1 := make(chan any)
	close(ch1)

	ch2 := make(chan any)
	close(ch2)

	output := FanIn(ch1, ch2)

	// Канал должен быть сразу закрыт
	select {
	case _, ok := <-output:
		if ok {
			t.Error("Expected channel to be closed")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected channel to be closed immediately")
	}
}

func TestFanIn_MixedOpenClosedChannels(t *testing.T) {
	// Тест со смесью открытых и закрытых каналов
	ch1 := make(chan any) // Будет закрыт сразу
	close(ch1)

	ch2 := make(chan any, 2) // Будет содержать данные
	ch2 <- "data1"
	ch2 <- "data2"
	close(ch2)

	ch3 := make(chan any) // Будет закрыт сразу
	close(ch3)

	output := FanIn(ch1, ch2, ch3)

	// Должны получить только данные из ch2
	var received []any
	for data := range output {
		received = append(received, data)
	}

	if len(received) != 2 {
		t.Errorf("Expected 2 items, got %d", len(received))
	}

	expected := []any{"data1", "data2"}
	for i, item := range received {
		if item != expected[i] {
			t.Errorf("Expected %v, got %v", expected[i], item)
		}
	}
}

func TestFanIn_WithDifferentDataTypes(t *testing.T) {
	// Тест с разными типами данных
	ch1 := make(chan any, 2)
	ch2 := make(chan any, 2)

	ch1 <- "string"
	ch1 <- 42
	ch2 <- 3.14
	ch2 <- struct{ name string }{name: "test"}

	close(ch1)
	close(ch2)

	output := FanIn(ch1, ch2)

	var received []any
	for data := range output {
		received = append(received, data)
	}

	if len(received) != 4 {
		t.Errorf("Expected 4 items, got %d", len(received))
	}
}

func TestFanIn_ConcurrencySafety(t *testing.T) {
	// Тест на безопасность конкурентности
	channels := make([]<-chan any, 10)
	for i := 0; i < 10; i++ {
		ch := make(chan any, 5)
		for j := 0; j < 5; j++ {
			ch <- i*10 + j
		}
		close(ch)
		channels[i] = ch
	}

	output := FanIn(channels...)

	count := 0
	for range output {
		count++
	}

	if count != 50 {
		t.Errorf("Expected 50 items, got %d", count)
	}
}

func TestFanIn_OrderNotGuaranteed(t *testing.T) {
	// Тест, что порядок не гарантируется
	ch1 := make(chan any, 100)
	ch2 := make(chan any, 100)

	// Заполняем каналы последовательными числами
	for i := 0; i < 100; i++ {
		if i%2 == 0 {
			ch1 <- i
		} else {
			ch2 <- i
		}
	}
	close(ch1)
	close(ch2)

	output := FanIn(ch1, ch2)

	// Проверяем, что порядок нарушен (хотя это не строгий тест)
	unordered := false
	prev := -1
	for data := range output {
		current := data.(int)
		if current < prev {
			unordered = true
			break
		}
		prev = current
	}

	if !unordered {
		t.Log("Note: data happened to be in order, but order is not guaranteed")
	}
}
