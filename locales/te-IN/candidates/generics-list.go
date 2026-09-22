//go:build OMIT

package main

// List అనేది ఏ టైప్ విలువలనైనా ఉంచే
// సింగ్లీ-లింక్డ్ లిస్ట్‌ను సూచిస్తుంది.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
