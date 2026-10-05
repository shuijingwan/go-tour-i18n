//go:build OMIT

package main

import "fmt"

// Index מחזירה את האינדקס של x בתוך s, או -1 אם x לא נמצא.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v ו־x הם מטיפוס T, שעליו חל אילוץ comparable,
		// ולכן אפשר להשתמש כאן ב־==.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index פועלת על פרוסה של int
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index פועלת גם על פרוסה של string
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
