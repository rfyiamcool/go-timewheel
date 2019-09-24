package timewheel

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

type A struct {
	a int
	b string
}

func callback() {
	fmt.Println("callback !!!")
}

func newTimeWheel() *TimeWheel {
	tw, err := NewTimeWheel(1*time.Second, 360)
	if err != nil {
		panic(err)
	}
	tw.Start()
	return tw
}

func TestAdd(t *testing.T) {
	tw := newTimeWheel()
	tw.Add(time.Second*1, callback)
	time.Sleep(time.Second * 5)
	tw.Stop()
}

func TestAdd100ms(t *testing.T) {
	tw := newTimeWheel()
	defer tw.Stop()

	now := time.Now()
	q := make(chan bool, 2)
	tw.Add(100*time.Millisecond, func() {
		q <- true
	})

	<-q
	if time.Since(now).Seconds() < 1 {
		t.Fatal("< 1s")
	}
}

func TestCron(t *testing.T) {
	tw := newTimeWheel()
	tw.AddCron(time.Second*1, callback)
	time.Sleep(time.Second * 5)
	tw.Stop()
}

func TestTicker(t *testing.T) {
	tw := newTimeWheel()
	ticker := tw.NewTicker(time.Second * 1)
	go func() {
		time.Sleep(5 * time.Second)
		ticker.Stop()
		fmt.Println("call stop")
	}()
	for {
		select {
		case <-ticker.C:
			callback()
		case <-ticker.Ctx.Done():
			return
		}
	}
}

func TestBatchTicker(t *testing.T) {
	tw := newTimeWheel()
	wg := sync.WaitGroup{}
	for index := 0; index < 100; index++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ticker := tw.NewTicker(time.Second * 1)
			go func() {
				time.Sleep(5 * time.Second)
				ticker.Stop()
				fmt.Println("call stop")
			}()
			for {
				select {
				case <-ticker.C:
					callback()
				case <-ticker.Ctx.Done():
					return
				}
			}
		}()
	}
	wg.Wait()
}

func TestTimerReset(t *testing.T) {
	tw := newTimeWheel()
	timer := tw.NewTimer(1 * time.Second)
	now := time.Now()
	<-timer.C
	t.Logf(time.Since(now).String())

	timer.Reset(2 * time.Second)
	now = time.Now()
	<-timer.C
	t.Logf(time.Since(now).String())

	now = time.Now()
	timer.Reset(3 * time.Second)
	<-timer.C
	t.Logf(time.Since(now).String())

	now = time.Now()
	timer.Reset(5 * time.Second)
	<-timer.C
	t.Logf(time.Since(now).String())
}

func TestRemove(t *testing.T) {
	tw := newTimeWheel()
	task := tw.Add(time.Second*1, callback)
	tw.Remove(task)
	time.Sleep(time.Second * 5)
	tw.Stop()
}

func TestHwTimer(t *testing.T) {
	tw := newTimeWheel()
	worker := 10
	delay := 5

	wg := sync.WaitGroup{}
	for index := 0; index < worker; index++ {
		wg.Add(1)
		var (
			htimer = tw.NewTimer(time.Duration(delay) * time.Second)
			maxnum = 20
			incr   = 0
		)
		go func(idx int) {
			defer wg.Done()
			for incr < maxnum {
				now := time.Now()
				target := time.Now().Add(time.Duration(delay) * time.Second)
				select {
				case <-htimer.C:
					htimer.Reset(time.Duration(delay) * time.Second)
					end := time.Now()
					if end.Before(target.Add(-1 * time.Second)) {
						t.Log("before 1s run")
					}
					if end.After(target.Add(1 * time.Second)) {
						t.Log("delay 1s run")
					}
					fmt.Println("id: ", idx, "cost: ", time.Now().Sub(now))
				}
				incr++
			}
		}(index)
	}
	wg.Wait()
}

func BenchmarkAdd(b *testing.B) {
	tw := newTimeWheel()
	for i := 0; i < b.N; i++ {
		tw.Add(time.Second, callback)
	}
}
