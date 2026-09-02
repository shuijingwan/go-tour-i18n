//go:build OMIT

package main

// List는 모든 타입의 값을 담는
// 단일 연결 리스트를 나타냅니다.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
