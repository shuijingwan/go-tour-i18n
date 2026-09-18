//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter an toàn khi sử dụng đồng thời.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc tăng bộ đếm cho khóa đã cho.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Khóa để mỗi lần chỉ một goroutine có thể truy cập map c.v.
	c.v[key]++
	c.mu.Unlock()
}

// Value trả về giá trị hiện tại của bộ đếm cho khóa đã cho.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Khóa để mỗi lần chỉ một goroutine có thể truy cập map c.v.
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
