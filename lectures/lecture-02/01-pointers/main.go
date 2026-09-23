package main

import "fmt"

func grow(n *int) {
	*n *= 2
}

func growVal(n int) {
	n *= 2
}

// makeCounter возвращает указатель на локальную переменную.
// C это висячий указатель; в Go компилятор переносит c в кучу (escape).
func makeCounter() *int {
	c := 0
	return &c
}

func main() {
	x := 10
	p := &x
	fmt.Println(*p)
	*p = 20
	fmt.Println(x)

	growVal(x)
	fmt.Println(x)
	grow(&x)
	fmt.Println(x)

	var q *int
	fmt.Println(q == nil)
	// fmt.Println(*q)     // паника: invalid memory address or nil pointer dereference

	r := new(int) // *int на свежий int со zero value
	*r = 7
	fmt.Println(*r) // 7

	// Go 1.26: new принимает выражение — сразу указатель на значение.
	// Раньше требовалось: v := 7; r := &v
	seven := new(7)       // *int со значением 7
	name := new("gopher") // *string
	fmt.Println(*seven, *name)

	count := makeCounter()
	*count++
	fmt.Println(*count)
}
