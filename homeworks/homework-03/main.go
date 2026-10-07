package main

import (
	"sort"
	"strconv"
	"strings"
	"sync"
)

func ExecutePipeline(stages ...Stage) {
	var wg sync.WaitGroup
	in := make(chan any)

	for _, stage := range stages {
		out := make(chan any)
		wg.Add(1)
		go func(stage Stage, in, out chan any) {
			defer wg.Done()
			defer close(out)
			stage(in, out)
		}(stage, in, out)
		in = out
	}

	wg.Wait()
}

type primeRequest struct {
	candidate string
	result    chan string
}

func NewPrimePool(workers int) func(string) string {
	requests := make(chan primeRequest)

	for i := 0; i < workers; i++ {
		go func() {
			for request := range requests {
				request.result <- NextPrime(request.candidate)
			}
		}()
	}

	return func(candidate string) string {
		result := make(chan string)
		requests <- primeRequest{candidate: candidate, result: result}
		return <-result
	}
}

func NewCandidateCheck(findPrime func(string) string) Stage {
	return func(in, out chan any) {
		var wg sync.WaitGroup
		for rawID := range in {
			wg.Add(1)
			go func(rawID any) {
				defer wg.Done()
				id := rawID.(int)
				directPrimeCh := make(chan string)
				reservedPrimeCh := make(chan string)

				go func() {
					directCandidate := CandidateBase + id*CandidateStep
					directPrimeCh <- findPrime(strconv.Itoa(directCandidate))
				}()
				go func() {
					reservedCandidate := ReserveCandidate(strconv.Itoa(id))
					reservedPrimeCh <- findPrime(reservedCandidate)
				}()

				directPrime := <-directPrimeCh
				reservedPrime := <-reservedPrimeCh
				direct, _ := strconv.Atoi(directPrime)
				reserved, _ := strconv.Atoi(reservedPrime)
				out <- direct + reserved
			}(rawID)
		}
		wg.Wait()
	}
}

func NewLineCheck(findPrime func(string) string) Stage {
	return func(in, out chan any) {
		var wg sync.WaitGroup
		for rawSum := range in {
			wg.Add(1)
			go func(rawSum any) {
				defer wg.Done()
				sum := rawSum.(int)
				results := make([]string, 6)
				var lineWG sync.WaitGroup
				lineWG.Add(len(results))

				for th := range results {
					go func(th int) {
						defer lineWG.Done()
						candidate := sum + th*FanStep
						results[th] = findPrime(strconv.Itoa(candidate))
					}(th)
				}
				lineWG.Wait()
				out <- strings.Join(results, "")
			}(rawSum)
		}
		wg.Wait()
	}
}

func CombineResults(in, out chan any) {
	var results []string
	for rawResult := range in {
		results = append(results, rawResult.(string))
	}
	sort.Strings(results)
	out <- strings.Join(results, "_")
}

func main() {
	// код запуска не нужен для тестов
}
