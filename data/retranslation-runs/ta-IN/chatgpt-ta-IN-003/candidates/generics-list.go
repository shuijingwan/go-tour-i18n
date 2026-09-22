//go:build OMIT

package main

// List என்பது எந்த வகையிலான
// மதிப்புகளையும் வைத்திருக்கும் ஒற்றை-இணைக்கப்பட்ட பட்டியலைக் குறிக்கிறது.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
