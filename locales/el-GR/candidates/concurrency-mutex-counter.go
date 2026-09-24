//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// Η SafeCounter είναι ασφαλής για ταυτόχρονη χρήση.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Η Inc αυξάνει τον μετρητή για το δοσμένο κλειδί.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Κλείδωμα, ώστε μόνο μία goroutine κάθε φορά να μπορεί να προσπελάσει τον πίνακα αντιστοίχισης c.v.
	c.v[key]++
	c.mu.Unlock()
}

// Η Value επιστρέφει την τρέχουσα τιμή του μετρητή για το δοσμένο κλειδί.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Κλείδωμα, ώστε μόνο μία goroutine κάθε φορά να μπορεί να προσπελάσει τον πίνακα αντιστοίχισης c.v.
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
