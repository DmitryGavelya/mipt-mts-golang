package main

import "fmt"

// Engine — двигатель мощностью Power лошадиных сил.
type Engine struct {
	Power int
}

// Start запускает двигатель.
func (e Engine) Start() string {
	return fmt.Sprintf("двигатель %d л.с. запущен", e.Power)
}

// Car встраивает Engine: поле Power и метод Start продвигаются в Car.
type Car struct {
	Engine
	Brand string
}

// ElectricCar переопределяет Start, обращаясь к встроенному через имя типа.
type ElectricCar struct {
	Engine
	Brand string
}

// Start запускает электромобиль тихо.
func (c ElectricCar) Start() string {
	return "тихо: " + c.Engine.Start()
}

// Reader умеет читать строку.
type Reader interface{ Read() string }

// Writer умеет записать строку.
type Writer interface{ Write(s string) }

// ReadWriter собран встраиванием из Reader и Writer.
type ReadWriter interface {
	Reader
	Writer
}

func main() {
	c := Car{Engine: Engine{Power: 120}, Brand: "Lada"}
	fmt.Println(c.Power)
	fmt.Println(c.Start())
	e := ElectricCar{Engine: Engine{Power: 200}, Brand: "Tesla"}
	fmt.Println(e.Start()) // переопределённый

	var eng Engine = c.Engine
	fmt.Println(eng.Start())
}
