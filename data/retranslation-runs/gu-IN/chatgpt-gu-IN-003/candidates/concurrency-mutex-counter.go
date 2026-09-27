//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter નો સમવર્તી રીતે ઉપયોગ કરવો સુરક્ષિત છે.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc આપેલી કી માટે કાઉન્ટર વધારે છે.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// લૉક કરો જેથી એક સમયે માત્ર એક goroutine જ c.v મેપ ઍક્સેસ કરી શકે.
	c.v[key]++
	c.mu.Unlock()
}

// Value આપેલી કી માટે કાઉન્ટરનું વર્તમાન મૂલ્ય પરત કરે છે.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// લૉક કરો જેથી એક સમયે માત્ર એક goroutine જ c.v મેપ ઍક્સેસ કરી શકે.
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
