//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// Бұл әдіс T түрі I интерфейсін іске асыратынын білдіреді,
// бірақ мұны айқын түрде жариялау қажет емес.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
