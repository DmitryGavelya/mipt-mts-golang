package main

import (
	"fmt"
	"math"
)

// Shaper — всё, что умеет вернуть свою площадь.
type Shaper interface {
	Area() float64
}

// Rect — прямоугольник со сторонами W и H.
type Rect struct{ W, H float64 }

// Area возвращает площадь прямоугольника.
func (r Rect) Area() float64 { return r.W * r.H }

// String реализует fmt.Stringer: fmt.Println печатает Rect через него,
// а не через отражение полей.
func (r Rect) String() string { return fmt.Sprintf("%g×%g", r.W, r.H) }

// Circle — круг радиуса R.
type Circle struct{ R float64 }

// Area возвращает площадь круга.
func (c Circle) Area() float64 { return math.Pi * c.R * c.R }

// totalArea суммирует площади фигур. Rect и Circle
func totalArea(shapes []Shaper) float64 {
	var sum float64
	for _, s := range shapes {
		sum += s.Area()
	}
	return sum
}

func describe(s Shaper) {
	switch v := s.(type) {
	case Rect:
		fmt.Printf("прямоугольник %gx%g\n", v.W, v.H)
	case Circle:
		fmt.Printf("круг r=%g\n", v.R)
	default:
		fmt.Printf("неизвестная фигура %T\n", v)
	}
}

func main() {
	shapes := []Shaper{Rect{2, 3}, Circle{1}}
	fmt.Println(totalArea(shapes))

	for _, s := range shapes {
		describe(s)
	}

	// Rect реализует fmt.Stringer — Println берёт String(), а не поля
	fmt.Println(Rect{4, 5}) // 4×5

	var x any = 42
	if n, ok := x.(int); ok {
		fmt.Println("это int:", n)
	}

	var r *Rect
	var s Shaper = r
	fmt.Printf("%v %T\n", s == nil, s)
}
