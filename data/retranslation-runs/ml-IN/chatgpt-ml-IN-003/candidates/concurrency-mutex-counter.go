//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter കൺകറൻറായി ഉപയോഗിക്കാൻ സുരക്ഷിതമാണ്.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc നൽകിയ key-നുള്ള counter വർധിപ്പിക്കുന്നു.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// ഒരേസമയം ഒരു goroutine-ന് മാത്രം c.v മാപ്പിലേക്ക് പ്രവേശിക്കാനാകുന്നവിധം ലോക്ക് ചെയ്യുക.
	c.v[key]++
	c.mu.Unlock()
}

// Value നൽകിയ key-നുള്ള counter-ന്റെ നിലവിലെ മൂല്യം തിരികെ നൽകുന്നു.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// ഒരേസമയം ഒരു goroutine-ന് മാത്രം c.v മാപ്പിലേക്ക് പ്രവേശിക്കാനാകുന്നവിധം ലോക്ക് ചെയ്യുക.
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
