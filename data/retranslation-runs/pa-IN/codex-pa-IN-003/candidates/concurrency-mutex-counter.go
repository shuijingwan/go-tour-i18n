//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter ਨੂੰ ਇਕੱਠੇ ਵਰਤਣਾ ਸੁਰੱਖਿਅਤ ਹੈ।
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc ਦਿੱਤੀ key ਲਈ counter ਵਧਾਉਂਦਾ ਹੈ।
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Lock ਕਰੋ ਤਾਂ ਜੋ ਇੱਕ ਸਮੇਂ ਸਿਰਫ਼ ਇੱਕ goroutine ਹੀ ਮੈਪ c.v ਤੱਕ ਪਹੁੰਚ ਕਰ ਸਕੇ।
	c.v[key]++
	c.mu.Unlock()
}

// Value ਦਿੱਤੀ key ਲਈ counter ਦਾ ਮੌਜੂਦਾ ਮੁੱਲ ਵਾਪਸ ਕਰਦਾ ਹੈ।
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Lock ਕਰੋ ਤਾਂ ਜੋ ਇੱਕ ਸਮੇਂ ਸਿਰਫ਼ ਇੱਕ goroutine ਹੀ ਮੈਪ c.v ਤੱਕ ਪਹੁੰਚ ਕਰ ਸਕੇ।
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
