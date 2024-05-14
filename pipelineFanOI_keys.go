package main

import (
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sync"
	"time"
)

type KeyE struct {
	privateKey1 int
	privateKey2 int
	publicKey   int
}

func randomNumbers() int {

	return rand.Intn(1000000000)
}

func repeatFct(wg *sync.WaitGroup, stop <-chan bool, fct func() int) <-chan int {

	intStream := make(chan int)

	go func() {
		var count int
		defer func() { wg.Done() }()
		defer close(intStream)

		for {
			select {
			case <-stop:
				fmt.Printf("\nFin de repeat (%d)...\n", count)
				return
			case intStream <- fct():
				count++
			}
		}
	}()

	return intStream
}

func takeN(wg *sync.WaitGroup, stop <-chan bool, inputIntstream <-chan int, n int) <-chan int {

	outputIntStream := make(chan int)

	go func() {
		defer func() { wg.Done() }()
		defer close(outputIntStream)
		defer fmt.Println("\nFin de takeN...")
		for i := 0; i < n; i++ {
			select {
			case <-stop:
				break
			case outputIntStream <- <-inputIntstream:
			}
		}
	}()

	return outputIntStream
}

func oddify(x int) int {

	if x%2 == 0 {
		x += 1
	}

	return x
}

func notDiv3(x int) int {

	if x%3 == 0 {
		x += 2
	}

	return x
}

func fctPipeline(wg *sync.WaitGroup, stop <-chan bool, inputIntstream <-chan int, fct func(int) int) <-chan int {

	outputIntStream := make(chan int)

	go func() {
		defer func() { wg.Done() }()
		defer close(outputIntStream)
		defer fmt.Println("\nFin de fctPipeline...")
		for i := range inputIntstream {
			select {
			case <-stop:
				return
			case outputIntStream <- fct(i):
			}
		}
	}()

	return outputIntStream
}

func isPrime(v int) bool {

	sq := int(math.Sqrt(float64(v))) + 1
	var i int

	for i = 2; i < sq; i++ {

		if v%i == 0 {
			return false
		}
	}

	return true
}

func filter(wg *sync.WaitGroup, stop <-chan bool, inputIntstream <-chan int, filter func(int) bool) <-chan int {

	outputIntStream := make(chan int)

	go func() {
		defer func() { wg.Done() }()
		defer close(outputIntStream)
		defer fmt.Println("\nFin de filter...")
		for i := range inputIntstream {

			if !filter(i) {
				continue
			}

			select {
			case <-stop:
				return
			case outputIntStream <- i:
			}
		}
	}()

	return outputIntStream
}

func keyGenerator(wg *sync.WaitGroup, stop <-chan bool, inputIntstream <-chan int) <-chan KeyE {

	outputKeyStream := make(chan KeyE)

	go func() {
		defer func() { wg.Done() }()
		defer close(outputKeyStream)
		defer fmt.Println("\nFin de keyGenerator...")

		var k1 int
		var k2 int
		var ok bool

		for {
			select {
			case <-stop:
				return
			case k1, ok = <-inputIntstream:
				if !ok {
					return
				}
			}

			select {
			case <-stop:
				return
			case k2, ok = <-inputIntstream:
				if !ok {
					return
				}
				outputKeyStream <- KeyE{k1, k2, k1 * k2}
			}
		}
	}()

	return outputKeyStream
}

func fanIn(wg *sync.WaitGroup, stop <-chan bool, channels []<-chan int) <-chan int {

	var multiplexGroup sync.WaitGroup
	outputIntStream := make(chan int)

	reader := func(ch <-chan int) {
		defer func() { multiplexGroup.Done() }()
		for i := range ch {

			select {
			case <-stop:
				return
			case outputIntStream <- i:
			}
		}
	}

	// all goroutine must return before
	// the output channel is closed
	multiplexGroup.Add(len(channels))
	for _, ch := range channels {

		go reader(ch)
	}

	go func() {

		defer func() { wg.Done() }()
		defer close(outputIntStream)
		defer fmt.Println("\n end of fanIn...")
		multiplexGroup.Wait()
	}()

	return outputIntStream
}

func main() {

	rand.Seed(time.Now().UnixNano())
	debut := time.Now() // timer
	fmt.Println("Debut")

	stop := make(chan bool)

	var wg sync.WaitGroup

	wg.Add(3)
	randomStream := fctPipeline(&wg, stop, fctPipeline(&wg, stop, repeatFct(&wg, stop, randomNumbers), oddify), notDiv3)

	fanOut := 8
	wg.Add(fanOut)
	filterStreams := make([]<-chan int, fanOut)
	for i := 0; i < fanOut; i++ {
		filterStreams[i] = filter(&wg, stop, randomStream, isPrime)
	}

	nKeys := 500
	keys := make([]KeyE, nKeys)
	wg.Add(3)
	for key := range keyGenerator(&wg, stop, takeN(&wg, stop, fanIn(&wg, stop, filterStreams), nKeys*2)) {

		keys = append(keys, key)
		//		   fmt.Printf("%v \n", key)
	}
	close(stop) // stopping the threads
	wg.Wait()

	fmt.Println("\nEnd\n")
	fmt.Println(keys[len(keys)-1])
	fin := time.Now()
	fmt.Println(runtime.NumCPU())
	fmt.Printf("Execution time: %s", fin.Sub(debut))
}
