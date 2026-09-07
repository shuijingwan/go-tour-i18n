//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter kan veilig gelijktijdig worden gebruikt.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc verhoogt de teller voor de opgegeven sleutel.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Vergrendel zodat slechts één goroutine tegelijk toegang heeft tot de map c.v.
	c.v[key]++
	c.mu.Unlock()
}

// Value retourneert de huidige waarde van de teller voor de opgegeven sleutel.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Vergrendel zodat slechts één goroutine tegelijk toegang heeft tot de map c.v.
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
