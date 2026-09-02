//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // i를 가리킵니다
	fmt.Println(*p) // 포인터를 통해 i를 읽습니다
	*p = 21         // 포인터를 통해 i를 설정합니다
	fmt.Println(i)  // i의 새 값을 확인합니다

	p = &j         // j를 가리킵니다
	*p = *p / 37   // 포인터를 통해 j를 나눕니다
	fmt.Println(j) // j의 새 값을 확인합니다
}
