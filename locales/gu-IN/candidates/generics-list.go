//go:build OMIT

package main

// List એવી સિંગલી-લિંક્ડ લિસ્ટ દર્શાવે છે જેમાં
// કોઈપણ ટાઇપનાં મૂલ્યો રાખી શકાય છે.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
