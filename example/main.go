package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rfyiamcool/go-timewheel"
)

var (
	counter int64 = 0
	loopNum       = 50
	tw            = newTimeWheel()
	wg            = sync.WaitGroup{}
	log           = new(logger)
)

func incrCounter() {
	atomic.AddInt64(&counter, 1)
}

func main() {
	go printCounter()

	batchRun5s()
	batchRun6s()
	batchRun8s()
	batchRun10s()
	batchRun15s()
	batchRun20s()
	batchRun30s()
	batchRun60s()
	batchRun120s()
	batchRun360s()

	log.info("add finish")
	wg.Wait()
}

func batchRun5s() {
	worker := 10000
	delay := 5
	beforeDiff := 1
	afterDiff := 2
	processTimer(worker, delay, beforeDiff, afterDiff)

	loop := 500
	taskNum := 50000
	go processCallbackLoop(loop, taskNum, delay, beforeDiff, afterDiff)
}

func batchRun6s() {
	worker := 5000
	delay := 6
	beforeDiff := 1
	afterDiff := 2
	processTimer(worker, delay, beforeDiff, afterDiff)

	loop := 500
	taskNum := 50000
	go processCallbackLoop(loop, taskNum, delay, beforeDiff, afterDiff)
}
func batchRun8s() {
	worker := 5000
	delay := 8
	beforeDiff := 1
	afterDiff := 2
	processTimer(worker, delay, beforeDiff, afterDiff)

	loop := 500
	taskNum := 50000
	go processCallbackLoop(loop, taskNum, delay, beforeDiff, afterDiff)
}

func batchRun10s() {
	worker := 10000
	delay := 10
	beforeDiff := 1
	afterDiff := 2
	processTimer(worker, delay, beforeDiff, afterDiff)

	loop := 300
	taskNum := 50000
	go processCallbackLoop(loop, taskNum, delay, beforeDiff, afterDiff)
}

func batchRun15s() {
	worker := 10000
	delay := 15
	beforeDiff := 1
	afterDiff := 2
	processTimer(worker, delay, beforeDiff, afterDiff)

	loop := 200
	taskNum := 50000
	go processCallbackLoop(loop, taskNum, delay, beforeDiff, afterDiff)
}

func batchRun20s() {
	worker := 20000
	delay := 20
	beforeDiff := 1
	afterDiff := 2
	processTimer(worker, delay, beforeDiff, afterDiff)

	loop := 150
	taskNum := 30000
	go processCallbackLoop(loop, taskNum, delay, beforeDiff, afterDiff)
}

func batchRun30s() {
	worker := 5000
	delay := 30
	beforeDiff := 1
	afterDiff := 2
	processTimer(worker, delay, beforeDiff, afterDiff)

	loop := 100
	taskNum := 30000
	go processCallbackLoop(loop, taskNum, delay, beforeDiff, afterDiff)
}

func batchRun60s() {
	worker := 10000
	delay := 60
	beforeDiff := 1
	afterDiff := 2
	processTimer(worker, delay, beforeDiff, afterDiff)

	loop := 50
	taskNum := 30000
	go processCallbackLoop(loop, taskNum, delay, beforeDiff, afterDiff)
}

func batchRun120s() {
	worker := 20000
	delay := 120
	beforeDiff := 1
	afterDiff := 2
	processTimer(worker, delay, beforeDiff, afterDiff)

	loop := 50
	taskNum := 30000
	go processCallbackLoop(loop, taskNum, delay, beforeDiff, afterDiff)
}

func batchRun360s() {
	worker := 20000
	delay := 360
	beforeDiff := 1
	afterDiff := 2
	processTimer(worker, delay, beforeDiff, afterDiff)

	loop := 50
	taskNum := 30000
	go processCallbackLoop(loop, taskNum, delay, beforeDiff, afterDiff)
}

func newTimeWheel() *timewheel.TimeWheel {
	tw, err := timewheel.NewTimeWheel(1*time.Second, 120)
	if err != nil {
		panic(err)
	}
	tw.Start()
	return tw
}

func processTimer(worker, delay, beforeDiff, afterDiff int) {
	for index := 0; index < worker; index++ {
		wg.Add(1)
		var (
			htimer = tw.NewTimer(time.Duration(delay) * time.Second)
			incr   = 0
		)
		go func(idx int) {
			defer wg.Done()
			for incr < loopNum {
				now := time.Now()
				target := now.Add(time.Duration(delay) * time.Second)
				select {
				case <-htimer.C:
					htimer.Reset(time.Duration(delay) * time.Second)
					end := time.Now()
					if end.Before(target.Add(time.Duration(-beforeDiff) * time.Second)) {
						log.error("timer befer: %s, delay: %d", time.Since(now).String(), delay)
					}
					if end.After(target.Add(time.Duration(afterDiff) * time.Second)) {
						log.error("timer after: %s, delay: %d", time.Since(now).String(), delay)
					}

					incrCounter()
					// fmt.Println("id: ", idx, "cost: ", end.Sub(now))
				}
				incr++
			}
		}(index)
	}
}

func processCallbackLoop(loop, taskCount, delay, beforeDiff, afterDiff int) {
	wg.Add(1)
	for index := 0; index < loop; index++ {
		processCallback(taskCount, delay, beforeDiff, afterDiff)
		time.Sleep(time.Duration(delay) * time.Second)
	}
	wg.Done()
}

func processCallback(taskCount, delay, beforeDiff, afterDiff int) {
	for index := 0; index < taskCount; index++ {
		now := time.Now()
		cb := func() {
			target := now.Add(time.Duration(delay) * time.Second)
			end := time.Now()
			if end.Before(target.Add(time.Duration(-beforeDiff) * time.Second)) {
				log.error("cb befer: %s, delay: %d", time.Since(now).String(), delay)
			}
			if end.After(target.Add(time.Duration(afterDiff) * time.Second)) {
				log.error("cb after: %s, delay: %d", time.Since(now).String(), delay)
			}
			incrCounter()
			// log.info("cost: %s, delay: %d", end.Sub(now), delay)
		}
		tw.Add(time.Duration(delay)*time.Second, cb)
	}
}

func printCounter() {
	now := time.Now()
	for {
		n := atomic.LoadInt64(&counter)
		log.info("start_time: %s, since_time: %s, counter %d", now.String(), time.Since(now).String(), n)
		time.Sleep(10 * time.Second)
	}
}

type logger struct{}

func (l *logger) info(format string, args ...interface{}) {
	v := fmt.Sprintf(format, args...)
	fmt.Println(v)
}

func (l *logger) error(format string, args ...interface{}) {
	v := fmt.Sprintf(format, args...)
	color := fmt.Sprintf("%c[%d;%d;%dm %s %c[0m", 0x1B, 5, 40, 31, v, 0x1B)
	fmt.Println(color)
}
