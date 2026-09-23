//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// Ligtas gamitin ang SafeCounter nang magkakasabay.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Dinadagdagan ng Inc ang counter para sa ibinigay na key.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// I-lock upang isang goroutine lang sa bawat pagkakataon ang maka-access sa map na c.v.
	c.v[key]++
	c.mu.Unlock()
}

// Ibinabalik ng Value ang kasalukuyang value ng counter para sa ibinigay na key.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// I-lock upang isang goroutine lang sa bawat pagkakataon ang maka-access sa map na c.v.
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
