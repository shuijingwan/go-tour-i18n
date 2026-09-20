//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter کو concurrent طور پر محفوظ طریقے سے استعمال کیا جا سکتا ہے۔
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc دی گئی key کے counter کو ایک بڑھاتا ہے۔
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Lock کریں تاکہ ایک وقت میں صرف ایک goroutine میپ c.v تک رسائی حاصل کر سکے۔
	c.v[key]++
	c.mu.Unlock()
}

// Value دی گئی key کے counter کی موجودہ قدر واپس کرتا ہے۔
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Lock کریں تاکہ ایک وقت میں صرف ایک goroutine میپ c.v تک رسائی حاصل کر سکے۔
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
