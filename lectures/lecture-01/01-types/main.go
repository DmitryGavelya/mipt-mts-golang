package main

import (
	"fmt"
)

type Weekday int

const (
	Monday Weekday = iota + 1
	Tuesday
	Wednesday
	Thursday
	Friday
	Saturday
	Sunday
)

func main() {
	var i int
	var f float64
	var b bool
	var s string
	var r rune
	var by byte
	fmt.Printf("int=%d float64=%v bool=%v string=%q rune=%q byte=%d\n", i, f, b, s, r, by)

	var count int = 10
	total := 20
	fmt.Println(count, total)

	var a int32 = 5
	var c int64 = 10
	// sum := a + c // не компилируется: mismatched types int32 and int64
	sum := int64(a) + c
	fmt.Println(sum)

	fmt.Println(Wednesday)

	z := 3 + 4i // complex128 по умолчанию
	fmt.Println(z, real(z), imag(z))

	// варианты записи целочисленных литералов
	hex, oct, bin, million := 0x1F, 0o17, 0b101, 1_000_000
	fmt.Println(hex, oct, bin, million)

	word := "привет"
	fmt.Println(len(word))

	for pos, r := range word {
		fmt.Printf("%c at byte %d\n", r, pos)
	}
}
