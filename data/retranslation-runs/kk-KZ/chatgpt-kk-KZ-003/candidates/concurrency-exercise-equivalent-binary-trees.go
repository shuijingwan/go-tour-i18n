//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk t ағашын аралап, барлық мәндерді
// ағаштан ch арнасына жібереді.
func Walk(t *tree.Tree, ch chan int)

// Same мына ағаштардың
// t1 және t2 бірдей мәндерді қамтитынын анықтайды.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
