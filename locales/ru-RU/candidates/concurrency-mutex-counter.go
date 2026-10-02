//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter безопасен для конкурентного использования.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc увеличивает счётчик для заданного ключа.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Блокируем, чтобы к отображению c.v одновременно могла обращаться только одна goroutine.
	c.v[key]++
	c.mu.Unlock()
}

// Value возвращает текущее значение счётчика для заданного ключа.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Блокируем, чтобы к отображению c.v одновременно могла обращаться только одна goroutine.
	defer c.mu.Unlock()
	return c.v[key]
}

func main() {
	c := SafeCounter{v: make(map[string]int)}
	for i := 0; i < 1000; i++ {
		go c.Inc("somekey")
	}

	time.Sleep(time.Second)
	fmt.Println(c.Value("somekey"))
}
