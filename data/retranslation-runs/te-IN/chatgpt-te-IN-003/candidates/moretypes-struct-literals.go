//go:build OMIT

package main

import "fmt"

type Vertex struct {
	X, Y int
}

var (
	v1 = Vertex{1, 2}  // Vertex టైప్‌కు చెందినది
	v2 = Vertex{X: 1}  // Y:0 పరోక్షంగా ఉంటుంది
	v3 = Vertex{}      // X:0 మరియు Y:0
	p  = &Vertex{1, 2} // *Vertex టైప్‌కు చెందినది
)

func main() {
	fmt.Println(v1, p, v2, v3)
}
