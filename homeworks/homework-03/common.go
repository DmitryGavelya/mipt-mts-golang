package main

import (
	"strconv"
	"sync"
	"time"
)

// Stage - одно звено конвейера. Читает значения из in, пишет результаты в out.
type Stage func(in, out chan any)

const MaxInputDataLen = 100

// Workers - служба поиска простых чисел умеет искать не больше этого числа
// кандидатов одновременно. Ограничение общее на весь конвейер, а не
// отдельное для каждого этапа - оба этапа делят один и тот же пул.
const Workers = 32

// CandidateBase и CandidateStep задают диапазон подбора: кандидаты для
// разных заявок должны быть разнесены заметно дальше типичного промежутка
// между соседними простыми числами, иначе разные заявки могут случайно
// получить одно и то же простое число.
const CandidateBase = 1000003
const CandidateStep = 1009

// FanStep - такой же разнос для 6 параллельных линий одной заявки.
const FanStep = 3001

var (
	entropyMu sync.Mutex
	Salt      = 0 // на проверке будет другое значение
)

// LockEntropy - аппаратный генератор случайности отдаёт следующее значение
// не более чем одному запросу одновременно.
var LockEntropy = func() {
	entropyMu.Lock()
}

var UnlockEntropy = func() {
	entropyMu.Unlock()
}

// ReserveCandidate - берёт у аппаратного генератора случайности следующего
// кандидата под заявку на ключ. Держит не более одного одновременного
// вызова, сериализация уже встроена в саму функцию (см.
// LockEntropy/UnlockEntropy) - используйте её как есть.
var ReserveCandidate = func(data string) string {
	LockEntropy()
	defer UnlockEntropy()
	n, _ := strconv.Atoi(data)
	const registryBase = 2000003
	const registryStep = 2003
	time.Sleep(10 * time.Millisecond)
	return strconv.Itoa(registryBase + n*registryStep + Salt)
}

// NextPrime - находит пробным делением ближайшее простое число, не меньшее
// переданного. Сама функция не проверяет Workers - лимит на число
// одновременных вызовов держит вызывающий код.
var NextPrime = func(data string) string {
	n, _ := strconv.Atoi(data)
	n += Salt
	if n <= 2 {
		n = 2
	} else if n%2 == 0 {
		n++
	}
	for !isPrime(n) {
		n += 2
	}
	time.Sleep(500 * time.Millisecond)
	return strconv.Itoa(n)
}

func isPrime(n int) bool {
	if n < 2 {
		return false
	}
	if n%2 == 0 {
		return n == 2
	}
	for i := 3; i*i <= n; i += 2 {
		if n%i == 0 {
			return false
		}
	}
	return true
}
