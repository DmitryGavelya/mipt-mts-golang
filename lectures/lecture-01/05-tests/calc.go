package calc

import "fmt"

func Add(a, b int) int {
	return a + b
}

func Divide(a, b int) (int, error) {
	if b == 0 {
		return 0, fmt.Errorf("division by zero")
	}
	return a / b, nil
}

func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
