package main

import (
	"errors"
	"fmt"
	"strconv"
)

// Решение пишите в этом файле.

// EvalPostfix вычисляет выражение в постфиксной записи. Токены — целые
// числа и операторы + - * /. Деление на ноль возвращает ошибку, а не
// вызывает panic.
func EvalPostfix(tokens []string) (int, error) {
	stack := make([]int, 0, len(tokens))

	for _, token := range tokens {
		switch token {
		case "+", "-", "*", "/":
			if len(stack) < 2 {
				return 0, errors.New("invalid expression: not enough operands")
			}
			right := stack[len(stack)-1]
			left := stack[len(stack)-2]
			stack = stack[:len(stack)-2]

			switch token {
			case "+":
				stack = append(stack, left+right)
			case "-":
				stack = append(stack, left-right)
			case "*":
				stack = append(stack, left*right)
			case "/":
				if right == 0 {
					return 0, errors.New("division by zero")
				}
				stack = append(stack, left/right)
			}
		default:
			value, err := strconv.Atoi(token)
			if err != nil {
				return 0, fmt.Errorf("invalid token %q: %w", token, err)
			}
			stack = append(stack, value)
		}
	}

	if len(stack) != 1 {
		return 0, errors.New("invalid expression: stack size is not 1")
	}

	return stack[0], nil
}

// main по заданию не нужен. Можно добавить сюда построчный ввод
// выражений, чтобы работал `go run main.go`.
func main() {}
