package main

import "fmt"

func add(a, b int) int {
	return a + b
}

// counter — замыкание
func counter() func() int {
	count := 0
	return func() int {
		count++
		return count
	}
}

func divide(a, b int) (result int, err error) {
	if b == 0 {
		err = fmt.Errorf("division by zero")
		return
	}
	result = a / b
	return
}

func sum(nums ...int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}

func deferOrder() {
	for i := 0; i < 3; i++ {
		defer fmt.Println("deferred", i)
	}
	fmt.Println("done scheduling")
}

func main() {
	fmt.Println(add(2, 3))

	next := counter()
	fmt.Println(next(), next(), next())

	res, err := divide(10, 0)
	fmt.Println(res, err)

	fmt.Println(sum(1, 2, 3, 4))

	deferOrder()
}
