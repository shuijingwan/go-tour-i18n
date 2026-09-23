//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter selamat digunakan secara serentak.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc menambah pembilang bagi kunci yang diberikan.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Kunci supaya hanya satu goroutine pada satu-satu masa boleh mengakses peta c.v.
	c.v[key]++
	c.mu.Unlock()
}

// Value mengembalikan nilai semasa pembilang bagi kunci yang diberikan.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Kunci supaya hanya satu goroutine pada satu-satu masa boleh mengakses peta c.v.
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
