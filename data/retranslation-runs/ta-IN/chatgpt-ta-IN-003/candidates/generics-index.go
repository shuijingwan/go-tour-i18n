//go:build OMIT

package main

import "fmt"

// Index, s-இல் x இருக்கும் குறியிடத்தைத் திருப்பித் தருகிறது; கிடைக்காவிட்டால் -1.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v மற்றும் x ஆகியவை comparable கட்டுப்பாடு கொண்ட T வகையைச் சேர்ந்தவை,
		// எனவே இங்கே == ஐப் பயன்படுத்தலாம்.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index, முழு எண்களின் துண்டத்தில் செயல்படுகிறது
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index, சரங்களின் துண்டத்திலும் செயல்படுகிறது
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
