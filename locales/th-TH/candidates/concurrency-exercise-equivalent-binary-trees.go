//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk เดินผ่านต้นไม้ t และส่งค่าทั้งหมด
// จากต้นไม้ไปยังแชนเนล ch
func Walk(t *tree.Tree, ch chan int)

// Same ใช้ตรวจว่าต้นไม้
// t1 และ t2 มีค่าเดียวกันหรือไม่
func Same(t1, t2 *tree.Tree) bool

func main() {
}
