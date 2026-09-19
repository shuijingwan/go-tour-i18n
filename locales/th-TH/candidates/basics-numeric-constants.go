//go:build OMIT

package main

import "fmt"

const (
	// สร้างจำนวนขนาดใหญ่มากโดยเลื่อนบิต 1 ไปทางซ้าย 100 ตำแหน่ง
	// กล่าวอีกอย่าง คือเลขฐานสองที่เป็น 1 ตามด้วยศูนย์ 100 ตัว
	Big = 1 << 100
	// เลื่อนกลับไปทางขวา 99 ตำแหน่ง เราจึงได้ 1<<1 หรือ 2
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
