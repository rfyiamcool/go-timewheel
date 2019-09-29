package main

import (
	"fmt"
	"time"

	"github.com/rfyiamcool/go-timewheel"
)

func main() {
	tw, err := timewheel.NewTimeWheel(200*time.Millisecond, 10)
	if err != nil {
		panic(err)
	}
	tw.Start()
	defer tw.Stop()

	count := 500000
	queue := make(chan bool, count)

	// loop 3
	for index := 0; index < 3; index++ {
		start := time.Now()
		for index := 0; index < count; index++ {
			tw.Add(time.Duration(1*time.Second), func() {
				queue <- true
			})
		}
		fmt.Println("add timer cost: ", time.Since(start))

		start = time.Now()
		incr := 0
		for {
			if incr == count {
				fmt.Println("recv sig cost: ", time.Since(start))
				break
			}

			<-queue
			incr++
		}
	}
}
