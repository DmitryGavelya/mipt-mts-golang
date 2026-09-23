package main

import "fmt"

// Point — точка на плоскости с целочисленными координатами.
type Point struct {
	X, Y int
}

// Dist возвращает манхэттенское расстояние между p и o.
func (p Point) Dist(o Point) int {
	return abs(p.X-o.X) + abs(p.Y-o.Y)
}

// Move сдвигает точку на (dx, dy).
func (p *Point) Move(dx, dy int) {
	p.X += dx
	p.Y += dy
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// Counter — счётчик. Метод можно повесить на любой именованный тип,
// не только на структуру.
type Counter int

func (c *Counter) Inc() { *c++ }

func main() {
	a := Point{X: 1, Y: 2} // литерал по именам полей
	b := Point{4, 6}       // позиционный литерал
	var z Point            // zero value
	fmt.Printf("%v %+v %#v\n", z, z, z)

	fmt.Println(a.Dist(b)) // 7

	a.Move(10, 10)
	fmt.Println(a)

	var c Counter
	c.Inc()
	c.Inc()
	fmt.Println(c)

	// var nilPoint *Point
	// nilPoint.Dist(b)
}
