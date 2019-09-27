package timewheel

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func callback() {
	fmt.Println("callback !!!")
}

func checkTimeCost(t *testing.T, start, end time.Time, before int, after int) bool {
	due := end.Sub(start)
	if due > time.Duration(after)*time.Millisecond {
		t.Error("delay run")
		return false
	}

	if due < time.Duration(before)*time.Millisecond {
		t.Error("run ahead")
		return false
	}

	return true
}

func TestCalcPos(t *testing.T) {
	tw, _ := NewTimeWheel(100*time.Millisecond, 5)
	round := tw.calculateRound(1 * time.Second)
	if round != 2 {
		t.Error("round err")
	}

	idx := tw.calculateIndex(1 * time.Second)
	if idx != 0 {
		t.Error("idx err")
	}
}

func TestAddFunc(t *testing.T) {
	tw, _ := NewTimeWheel(100*time.Millisecond, 5, TickSafeMode())
	tw.Start()
	defer tw.Stop()

	for index := 1; index < 6; index++ {
		queue := make(chan bool, 0)
		start := time.Now()
		tw.Add(time.Duration(index)*time.Second, func() {
			queue <- true
		})

		<-queue

		before := index*1000 - 200
		after := index*1000 + 200
		checkTimeCost(t, start, time.Now(), before, after)
		fmt.Println("time since: ", time.Since(start).String())
	}
}

func TestAddCron(t *testing.T) {
	tw, _ := NewTimeWheel(100*time.Millisecond, 60)
	tw.Start()
	defer tw.Stop()

	queue := make(chan time.Time, 2)
	tw.AddCron(time.Second*1, func() {
		queue <- time.Now()
	})

	exitTimer := time.NewTimer(10 * time.Second)
	lastTs := time.Now()
	for {
		select {
		case <-exitTimer.C:
			return
		case now := <-queue:
			checkTimeCost(t, lastTs, now, 900, 1200)
			fmt.Println("time since: ", now.Sub(lastTs))
			lastTs = now
		}
	}
}

func TestStopTimer(t *testing.T) {
	tw, _ := NewTimeWheel(100*time.Millisecond, 5)
	tw.Start()
	defer tw.Stop()

	timer := tw.NewTimer(time.Millisecond * 500)

	exitTimer := time.NewTimer(1 * time.Second)
	time.Sleep(100 * time.Millisecond)
	timer.Stop()

	select {
	case <-exitTimer.C:
	case <-timer.C:
		t.Error("must not run")
	}
}

func TestStopTicker(t *testing.T) {
	tw, _ := NewTimeWheel(100*time.Millisecond, 5)
	tw.Start()
	defer tw.Stop()

	timer := tw.NewTicker(time.Millisecond * 500)

	exitTimer := time.NewTimer(1 * time.Second)
	time.Sleep(100 * time.Millisecond)
	timer.Stop()

	select {
	case <-exitTimer.C:
	case <-timer.C:
		t.Error("must not run")
	}
}

func TestTicker(t *testing.T) {
	tw, _ := NewTimeWheel(100*time.Millisecond, 5)
	tw.Start()
	defer tw.Stop()

	ticker := tw.NewTicker(time.Second * 1)
	go func() {
		time.Sleep(10 * time.Second)
		ticker.Stop()
	}()

	last := time.Now()
	for {
		select {
		case <-ticker.C:
			checkTimeCost(t, last, time.Now(), 900, 1200)
			last = time.Now()
			fmt.Println(time.Since(last))

		case <-ticker.Ctx.Done():
			return
		}
	}
}

func TestTickerSecond(t *testing.T) {
	tw, _ := NewTimeWheel(time.Second, 5)
	tw.Start()
	defer tw.Stop()

	ticker := tw.NewTicker(time.Second * 1)
	go func() {
		time.Sleep(12 * time.Second)
		ticker.Stop()
		fmt.Println("call stop")
	}()

	last := time.Now()
	for {
		select {
		case <-ticker.C:
			if time.Since(last) > time.Duration(2200*time.Millisecond) {
				fmt.Println("delay run", time.Since(last))
				t.Fatal()
			}

			fmt.Println(time.Since(last))
			last = time.Now()

		case <-ticker.Ctx.Done():
			return
		}
	}
}

func TestBatchTicker(t *testing.T) {
	tw, _ := NewTimeWheel(100*time.Millisecond, 60)
	tw.Start()
	defer tw.Stop()

	wg := sync.WaitGroup{}
	for index := 0; index < 100; index++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ticker := tw.NewTicker(time.Second * 1)
			go func() {
				time.Sleep(5 * time.Second)
				ticker.Stop()
			}()
			for {
				select {
				case <-ticker.C:
				case <-ticker.Ctx.Done():
					return
				}
			}
		}()
	}
	wg.Wait()
}

func TestAfter(t *testing.T) {
	tw, _ := NewTimeWheel(100*time.Millisecond, 10)
	tw.Start()
	defer tw.Stop()

	for index := 1; index < 6; index++ {
		ts := time.Now()
		<-tw.After(time.Duration(index) * time.Second)
		before := index*1000 - 200
		after := index*1000 + 200
		checkTimeCost(t, ts, time.Now(), before, after)
		fmt.Println(time.Since(ts).String())
	}
}

func TestAfterFunc(t *testing.T) {
	tw, _ := NewTimeWheel(100*time.Millisecond, 10)
	tw.Start()
	defer tw.Stop()

	queue := make(chan bool, 1)
	timer := tw.AfterFunc(1*time.Second, func() {
		queue <- true
	})
	<-queue

	for index := 1; index < 6; index++ {
		timer.Reset(time.Duration(index) * time.Second)
		ts := time.Now()
		<-queue
		before := index*1000 - 200
		after := index*1000 + 200
		checkTimeCost(t, ts, time.Now(), before, after)
		fmt.Println(time.Since(ts).String())
	}
}

func TestTimerReset(t *testing.T) {
	tw, _ := NewTimeWheel(100*time.Millisecond, 5)
	tw.Start()
	defer tw.Stop()

	timer := tw.NewTimer(100 * time.Millisecond)
	now := time.Now()
	<-timer.C
	fmt.Println(time.Since(now).String())
	checkTimeCost(t, now, time.Now(), 80, 220)

	for index := 1; index < 6; index++ {
		now := time.Now()
		timer.Reset(time.Duration(index) * time.Second)
		<-timer.C

		before := index*1000 - 200
		after := index*1000 + 200

		checkTimeCost(t, now, time.Now(), before, after)
		fmt.Println(time.Since(now).String())
	}
}

func TestRemove(t *testing.T) {
	tw, _ := NewTimeWheel(100*time.Millisecond, 5)
	tw.Start()
	defer tw.Stop()

	queue := make(chan bool, 0)
	task := tw.Add(time.Millisecond*200, func() {
		queue <- true
	})
	tw.Remove(task)

	exitTimer := time.NewTimer(1 * time.Second)
	select {
	case <-exitTimer.C:
	case <-queue:
		t.Error("must not run")
	}
}

func TestHwTimer(t *testing.T) {
	tw, _ := NewTimeWheel(100*time.Millisecond, 60)
	tw.Start()
	defer tw.Stop()

	worker := 10

	wg := sync.WaitGroup{}
	for index := 0; index < worker; index++ {
		wg.Add(1)
		var (
			htimer = tw.NewTimer(1 * time.Second)
			maxnum = 5
			incr   = 0
		)
		go func(idx int) {
			defer wg.Done()
			for incr < maxnum {
				now := time.Now()
				select {
				case <-htimer.C:
					htimer.Reset(1 * time.Second)
					end := time.Now()
					checkTimeCost(t, now, end, 900, 1200)
					fmt.Println("cost: ", time.Now().Sub(now))
				}
				incr++
			}
		}(index)
	}
	wg.Wait()
}

func BenchmarkAdd(b *testing.B) {
	tw, _ := NewTimeWheel(100*time.Millisecond, 60)
	tw.Start()
	defer tw.Stop()

	for i := 0; i < b.N; i++ {
		tw.Add(time.Second, func() {})
	}
}
