//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter는 동시에 사용해도 안전합니다.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc는 주어진 키의 카운터를 증가시킵니다.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// 한 번에 하나의 goroutine만 맵 c.v에 접근할 수 있도록 잠급니다.
	c.v[key]++
	c.mu.Unlock()
}

// Value는 주어진 키의 현재 카운터 값을 반환합니다.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// 한 번에 하나의 goroutine만 맵 c.v에 접근할 수 있도록 잠급니다.
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
