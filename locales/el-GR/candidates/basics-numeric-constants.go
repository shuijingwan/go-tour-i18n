//go:build OMIT

package main

import "fmt"

const (
	// Δημιουργήστε έναν τεράστιο αριθμό μετατοπίζοντας το bit 1 κατά 100 θέσεις προς τα αριστερά.
	// Με άλλα λόγια, τον δυαδικό αριθμό που αποτελείται από το 1 και ακολουθείται από 100 μηδενικά.
	Big = 1 << 100
	// Μετατοπίστε τον ξανά κατά 99 θέσεις προς τα δεξιά, ώστε να καταλήξουμε στο 1<<1, δηλαδή στο 2.
	Small = Big >> 99
)

func needInt(x int) int { return x*10 + 1 }
func needFloat(x float64) float64 {
	return x * 0.1
}

func main() {
	fmt.Println(needInt(Small))
	fmt.Println(needFloat(Small))
	fmt.Println(needFloat(Big))
}
