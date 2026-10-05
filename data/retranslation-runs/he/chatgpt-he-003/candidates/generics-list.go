//go:build OMIT

package main

// List מייצגת רשימה מקושרת חד־כיוונית שמחזיקה
// ערכים מכל טיפוס.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
