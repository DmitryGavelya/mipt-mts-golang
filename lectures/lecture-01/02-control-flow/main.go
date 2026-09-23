package main

import "fmt"

func classify(n int) string {
	if half := n / 2; half > 100 {
		return "big"
	} else if n < 0 {
		return "negative"
	}

	// switch n {
	// case 0:
	// 	return "zero"
	// case 2:
	// 	return "even"
	// default:
	// 	return "odd"
	// }

	switch {
	case n == 0:
		return "zero"
	case n%2 == 0:
		return "even"
	default:
		return "odd"
	}
}

func main() {
	fmt.Println(classify(4), classify(-1), classify(250))

	for i := 0; i < 3; i++ {
		fmt.Print(i, " ")
	}
	fmt.Println()

	i := 0
	for i < 3 {
		fmt.Print(i, " ")
		i++
	}
	fmt.Println()

	count := 0
	for {
		if count == 3 {
			break
		}
		fmt.Print(count, " ")
		count++
	}
	fmt.Println()

	for i := range 5 {
		fmt.Print(i, " ")
	}
	fmt.Println()

	switch grade := 'B'; grade {
	case 'A':
		fmt.Println("excellent")
		fallthrough
	case 'B':
		fmt.Println("good")
		fallthrough
	case 'C':
		fmt.Println("passed")
	default:
		fmt.Println("failed")
	}

	matrix := [][]int{
		{1, 2, 3},
		{4, 5, 6},
		{7, 8, 9},
	}
	target := 5

search:
	for row, cols := range matrix {
		for col, v := range cols {
			if v == target {
				fmt.Printf("found %d at [%d][%d]\n", target, row, col)
				break search
			}
		}
	}
}
