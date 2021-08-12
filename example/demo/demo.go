package main

import (
	"fmt"
	"time"

	"github.com/fufuok/timewheel"
)

var (
	TW *timewheel.TimeWheel
)

func init() {
	TW, _ = timewheel.NewTimeWheel(100*time.Millisecond, 600)
	TW.Start()
}

func TWStop() {
	TW.Stop()
}

func main() {
	defer TWStop()

	fmt.Println(time.Now())

	TW.Sleep(2 * time.Second)

	fmt.Println("sleep 2s:", time.Now())

	TW.AfterFunc(1*time.Second, func() {
		fmt.Println("after 1s:", time.Now())
	})

	TW.AddCron(1*time.Second, func() {
		fmt.Println("cron 1s:", time.Now())
	})

	ticker := TW.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for i := 0; i < 5; i++ {
		<-ticker.C
		fmt.Println("ticker(1s):", i, time.Now())
	}

	ticker.Reset(2 * time.Second)
	for i := 0; i < 5; i++ {
		<-ticker.C
		fmt.Println("ticker(2s):", i, time.Now())
	}
}
