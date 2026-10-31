package main

import (
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

/*
это тест на проверку того, что ExecutePipeline - действительно конвейер.
неправильное поведение: накапливать результаты выполнения одного этапа, а
потом слать их в следующий - это не позволяет запускать на конвейере
бесконечные задачи.
правильное поведение: обеспечить беспрепятственный поток.
*/
func TestConveyorIsFreeFlowing(t *testing.T) {
	var ok = true
	var received uint32
	freeFlowStages := []Stage{
		Stage(func(in, out chan any) {
			out <- 1
			time.Sleep(10 * time.Millisecond)
			curr := atomic.LoadUint32(&received)
			// если этап накапливает значения, то пока вся функция не
			// отработает - дальше они не пойдут. тут проверяется, что
			// счётчик увеличился в следующем этапе - значит, значение
			// дошло раньше, чем текущий этап отработал целиком.
			if curr == 0 {
				ok = false
			}
		}),
		Stage(func(in, out chan any) {
			for range in {
				atomic.AddUint32(&received, 1)
			}
		}),
	}
	ExecutePipeline(freeFlowStages...)
	if !ok || received == 0 {
		t.Errorf("no value free flow - не копите значения")
	}
}

func TestPrimeBatch(t *testing.T) {
	testExpected := "300001730030173006013300902330120173015037_300302930060373009043301204130150373018031_300611930090893012071301507130180913021077_300908930120673015071301809130210773024071_301210130151093018091302108930241013027121_301510930180913021077302408330270773030107_301809130210913024101302712130301073033097_302113130241373027121303011330331613036119_302413730271313030143303316130361513039139_302716130301433033161303615130391633042157_303017930331673036193303917330421733045173_303316130361513039163304215730451573048163_303622130392633042203304520930482113051227_303926330422093045209304821130512273054211_304219130451913048197305122730541973057227_304519130481973051227305419730572273060203_304821130512273054217305722730602413063217_305122730542213057227306024130632533066223_305428330572533060257306325330662533069263_305728130602873063287306629330693233072301"
	testResult := "NOT_SET"

	// защита от попыток не вызывать реальные функции расчёта: переопределяем
	// ReserveCandidate/NextPrime на свои, которые дополнительно инкрементят
	// локальный счётчик вызовов. NextPrime вдобавок следит за тем, сколько
	// его вызовов идёт одновременно - если решение не уважает Workers,
	// тест это поймает независимо от итогового времени.
	var (
		LocalSalt               int
		EntropyLockCounter      uint32
		EntropyUnlockCounter    uint32
		ReserveCandidateCounter uint32
		NextPrimeCounter        uint32
		concurrentSearching     int32
		peakSearching           int32
	)
	LockEntropy = func() {
		atomic.AddUint32(&EntropyLockCounter, 1)
		entropyMu.Lock()
	}
	UnlockEntropy = func() {
		atomic.AddUint32(&EntropyUnlockCounter, 1)
		entropyMu.Unlock()
	}
	ReserveCandidate = func(data string) string {
		atomic.AddUint32(&ReserveCandidateCounter, 1)
		LockEntropy()
		defer UnlockEntropy()
		n, _ := strconv.Atoi(data)
		const registryBase = 2000003
		const registryStep = 2003
		time.Sleep(10 * time.Millisecond)
		return strconv.Itoa(registryBase + n*registryStep + LocalSalt)
	}
	NextPrime = func(data string) string {
		atomic.AddUint32(&NextPrimeCounter, 1)
		current := atomic.AddInt32(&concurrentSearching, 1)
		defer atomic.AddInt32(&concurrentSearching, -1)
		for {
			prevPeak := atomic.LoadInt32(&peakSearching)
			if current <= prevPeak {
				break
			}
			if atomic.CompareAndSwapInt32(&peakSearching, prevPeak, current) {
				break
			}
		}
		n, _ := strconv.Atoi(data)
		n += LocalSalt
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

	itemIDs := make([]int, 20)
	for i := range itemIDs {
		itemIDs[i] = i
	}

	find := NewPrimePool(Workers)

	batchStages := []Stage{
		Stage(func(in, out chan any) {
			for _, id := range itemIDs {
				out <- id
			}
		}),
		NewCandidateCheck(find),
		NewLineCheck(find),
		Stage(CombineResults),
		Stage(func(in, out chan any) {
			raw := <-in
			data, ok := raw.(string)
			if !ok {
				t.Error("cant convert result to string")
			}
			testResult = data
		}),
	}

	start := time.Now()
	ExecutePipeline(batchStages...)
	elapsed := time.Since(start)

	expectedTime := 3 * time.Second

	if testExpected != testResult {
		t.Errorf("results not match\nGot: %v\nExpected: %v", testResult, testExpected)
	}

	if elapsed > expectedTime {
		t.Errorf("execution too long\nGot: %s\nExpected: <%s", elapsed, expectedTime)
	}

	if peakSearching > Workers {
		t.Errorf("Workers violated: peak concurrent NextPrime calls = %d, limit = %d",
			peakSearching, Workers)
	}

	// 8 = 2 в NewCandidateCheck + 6 в NewLineCheck
	if int(EntropyLockCounter) != len(itemIDs) ||
		int(EntropyUnlockCounter) != len(itemIDs) ||
		int(ReserveCandidateCounter) != len(itemIDs) ||
		int(NextPrimeCounter) != len(itemIDs)*8 {
		t.Errorf("not enough calls: lock=%d unlock=%d reserve=%d find=%d",
			EntropyLockCounter, EntropyUnlockCounter, ReserveCandidateCounter, NextPrimeCounter)
	}
}
