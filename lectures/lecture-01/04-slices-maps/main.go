package main

import "fmt"

func main() {
	var arr [5]int
	fmt.Println(arr)
	fmt.Printf("%#v\n", arr)

	lit := [...]int{10, 20, 30}
	fmt.Println(lit)

	arrCopy := arr
	arrCopy[0] = 1
	fmt.Println(arr, arrCopy)

	fromArr := lit[1:3]
	fromArr[0] = 99
	fmt.Println(fromArr, lit)

	base := []int{1, 2, 3, 4, 5}
	// fmt.Printf("base=%v len=%d cap=%d\n", base, len(base), cap(base))
	window := base[1:3]
	fmt.Printf("window=%v len=%d cap=%d\n", window, len(window), cap(window))
	window[0] = 99
	fmt.Println(base)
	fmt.Println(window)

	a := make([]int, 2)
	fmt.Printf("a: len=%d cap=%d\n", len(a), cap(a))
	b := append(a, 1)
	fmt.Printf("b: len=%d cap=%d (cap вырос -> новый массив)\n", len(b), cap(b))
	b[0] = 100
	fmt.Println(a, b)

	// var _ map[[]int]string // не компилируется
	scores := map[string]int{"alice": 0}
	v, ok := scores["alice"]
	fmt.Println(v, ok)
	v, ok = scores["bob"]
	fmt.Println(v, ok)

	delete(scores, "alice")
	fmt.Println(len(scores), scores)

	arrMap := map[string][2]int{"a": {1, 2}}
	// arrMap["a"][0] = 9 // не компилируется
	pair := arrMap["a"]
	pair[0] = 9
	arrMap["a"] = pair
	arrMap["mkmkmk"] = [2]int{0, 2}
	fmt.Println(arrMap)

	nested := map[string]map[string]int{"a": {"x": 1}}
	nested["a"]["x"] = 9
	fmt.Println(nested)

	var m map[string]int
	fmt.Println(m["missing"])

	value, exists := m["missing"]
	fmt.Println(value, exists)

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("recovered:", r)
		}
	}()
	m["key"] = 1 // паника
}
