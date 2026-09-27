//go:build OMIT

package main

import "fmt"

// Index, s માં x નો ઇન્ડેક્સ પરત કરે છે; x ન મળે તો -1 પરત કરે છે.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v અને x બંનેની ટાઇપ T છે, જેના પર comparable
		// કન્સ્ટ્રેઇન્ટ લાગુ પડે છે; તેથી અહીં == વાપરી શકાય છે.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index પૂર્ણાંકની સ્લાઇસ પર પણ કામ કરે છે.
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index સ્ટ્રિંગની સ્લાઇસ પર પણ કામ કરે છે.
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
