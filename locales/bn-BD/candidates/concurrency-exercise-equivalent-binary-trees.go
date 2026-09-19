//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk ট্রি t-এর ওপর দিয়ে হেঁটে সব মান
// ট্রি থেকে চ্যানেল ch-তে পাঠায়।
func Walk(t *tree.Tree, ch chan int)

// Same নির্ধারণ করে ট্রি দুটো
// t1 এবং t2 একই মান ধারণ করে কি না।
func Same(t1, t2 *tree.Tree) bool

func main() {
}
