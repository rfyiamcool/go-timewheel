package main

import (
	"fmt"
	"time"

	"github.com/fufuok/timewheel"
)

func main() {
	timewheel.Start()
	defer timewheel.Stop()

	fmt.Println(time.Now())

	timewheel.Sleep(2 * time.Second)

	fmt.Println("sleep 2s:", time.Now())

	timewheel.AfterFunc(1*time.Second, func() {
		fmt.Println("after 1s:", time.Now())
	})

	timewheel.AddCron(1*time.Second, func() {
		fmt.Println("cron 1s:", time.Now())
	})

	ticker := timewheel.NewTicker(1 * time.Second)
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
