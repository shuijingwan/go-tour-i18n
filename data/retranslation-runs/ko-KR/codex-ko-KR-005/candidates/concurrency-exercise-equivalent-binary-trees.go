//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk는 트리 t를 순회하며 트리의 모든 값을
// 채널 ch로 보냅니다.
func Walk(t *tree.Tree, ch chan int)

// Same은 두 트리
// t1과 t2가 같은 값을 포함하는지 확인합니다.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
