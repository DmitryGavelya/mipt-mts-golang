package main

import "fmt"

// Map применяет f к каждому элементу s и возвращает новый слайс.
func Map[T, U any](s []T, f func(T) U) []U {
	r := make([]U, len(s))
	for i, v := range s {
		r[i] = f(v)
	}
	return r
}

// Number — ограничение: целые и float. ~ разрешает типы с таким
// underlying-типом — например, Celsius ниже.
type Number interface {
	~int | ~int64 | ~float64
}

// Celsius — именованный тип на основе float64. Под ~float64 подходит,
// под голый float64 — нет.
type Celsius float64

// Sum складывает элементы слайса любого числового типа.
func Sum[T Number](s []T) T {
	var total T
	for _, v := range s {
		total += v
	}
	return total
}

// Set — множество элементов типа K. K обязан быть comparable,
// иначе его нельзя использовать как ключ мапы.
type Set[K comparable] struct {
	m map[K]struct{}
}

func NewSet[K comparable]() *Set[K] {
	return &Set[K]{m: make(map[K]struct{})}
}

func (s *Set[K]) Add(k K) { s.m[k] = struct{}{} }

// Has сообщает, есть ли элемент в множестве.
func (s *Set[K]) Has(k K) bool {
	_, ok := s.m[k]
	return ok
}

func main() {
	nums := []int{1, 2, 3, 4}
	labels := Map(nums, func(n int) string { return fmt.Sprintf("#%d", n) })
	fmt.Println(labels)

	fmt.Println(Sum([]int{1, 2, 3}))       // 6
	fmt.Println(Sum([]float64{1.5, 2.5}))  // 4
	fmt.Println(Sum([]Celsius{18.5, 2.0})) // 20.5 — Celsius прошёл по ~float64

	seen := NewSet[string]()
	seen.Add("go")
	fmt.Println(seen.Has("go"), seen.Has("rust")) // true false
}
