package timewheel

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

func TestAddStopCron(t *testing.T) {
	tw, _ := NewTimeWheel(100*time.Millisecond, 60)
	tw.Start()
	defer tw.Stop()

	queue := make(chan time.Time, 2)
	task := tw.AddCron(time.Second*1, func() {
		queue <- time.Now()
	})

	time.AfterFunc(5*time.Second, func() {
		tw.Remove(task.id)
	})

	exitTimer := time.NewTimer(10 * time.Second)
	lastTs := time.Now()
	c := 0
	for {
		select {
		case <-exitTimer.C:
			if c > 6 {
				t.Error("cron stop failed")
			}
			return

		case now := <-queue:
			c++
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

	exitTimer := time.NewTimer(2 * time.Second)
	timer.Stop()

	select {
	case <-exitTimer.C:
	case <-timer.C:
		t.Error("must not run")
	}
}

func TestStopTicker1(t *testing.T) {
	tw, _ := NewTimeWheel(100*time.Millisecond, 100)
	tw.Start()
	defer tw.Stop()

	timer := tw.NewTicker(time.Millisecond * 500)
	timer.Stop()

	time.Sleep(1 * time.Second)

	select {
	case <-timer.C:
		t.Error("exception")
	default:
	}

	select {
	case <-timer.Ctx.Done():
		return
	case <-time.After(1 * time.Second):
		t.Error("exception")
	}
}

func TestStopTicker2(t *testing.T) {
	tw, _ := NewTimeWheel(100*time.Millisecond, 100)
	tw.Start()
	defer tw.Stop()

	var tickers [1000]*Ticker
	for idx := range tickers {
		ticker := tw.NewTicker(time.Millisecond * 500)
		tickers[idx] = ticker
	}

	for _, ticker := range tickers {
		ticker.Stop()
	}

	time.Sleep(1 * time.Second)
	assert.Equal(t, 0, len(tw.bucketIndexes))
}

func TestTickerBetween(t *testing.T) {
	tw, _ := NewTimeWheel(100*time.Millisecond, 1000)
	tw.Start()
	defer tw.Stop()

	ticker := tw.NewTicker(100 * time.Millisecond)

	time.AfterFunc(5*time.Second, func() {
		ticker.Stop()
	})

	exitTimer := time.After(3 * time.Second)
	last := time.Now()
	for c := 0; c < 6; {
		select {
		case <-ticker.C:
			t.Log("since", time.Since(last))
			last = time.Now()
			c++

		case <-exitTimer:
			return
		}
	}
}

func TestTickerSecond(t *testing.T) {
	tw, err := NewTimeWheel(1*time.Millisecond, 10000)
	assert.Nil(t, err)

	tw.Start()
	defer tw.Stop()

	var (
		timeout = time.After(110 * time.Millisecond)
		ticker  = tw.NewTicker(1 * time.Millisecond)
		incr    int
	)

	for run := true; run; {
		select {
		case <-timeout:
			run = false

		case <-ticker.C:
			incr++
		}
	}

	assert.Greater(t, incr, 100)
}

func TestBatchTicker(t *testing.T) {
	tw, err := NewTimeWheel(100*time.Millisecond, 60)
	assert.Nil(t, err)

	tw.Start()
	defer tw.Stop()

	wg := sync.WaitGroup{}
	for index := 0; index < 100; index++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ticker := tw.NewTicker(1 * time.Second)
			go func() {
				time.Sleep(2 * time.Second)
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

func TestAfterFuncResetStop(t *testing.T) {
	tw, _ := NewTimeWheel(100*time.Millisecond, 10)
	tw.Start()
	defer tw.Stop()

	var incr = 0

	// stop
	timer := tw.AfterFunc(100*time.Millisecond, func() {
		incr++
	})
	timer.Stop()

	time.Sleep(500 * time.Millisecond)
	assert.Equal(t, 0, incr)

	// reset
	incr = 0
	timer = tw.AfterFunc(800*time.Millisecond, func() {
		incr++
	})
	timer.Reset(100 * time.Millisecond)
	time.Sleep(300 * time.Millisecond)
	assert.Equal(t, incr, 1)

	// reset stop
	incr = 0
	timer = tw.AfterFunc(100*time.Millisecond, func() {
		incr++
	})
	timer.Reset(100 * time.Millisecond)
	timer.Reset(100 * time.Millisecond)
	timer.Reset(1000 * time.Millisecond)
	time.Sleep(500 * time.Millisecond)
	assert.Equal(t, 0, incr)
	time.Sleep(700 * time.Millisecond)
	assert.Equal(t, 1, incr)
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
	}
}

func TestRemove(t *testing.T) {
	tw, _ := NewTimeWheel(100*time.Millisecond, 5)
	tw.Start()
	defer tw.Stop()

	queue := make(chan bool, 0)
	task := tw.Add(time.Millisecond*500, func() {
		queue <- true
	})

	// remove action after add action
	time.AfterFunc(time.Millisecond*10, func() {
		tw.Remove(task.id)
	})

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

func TestRunStopFunc(t *testing.T) {
	var (
		t1     = NewTimer(time.Second * 1)
		called bool
	)

	t1.AddStopFunc(func() {
		called = true
	})

	select {
	case <-t1.C:
		t1.Stop()
	}

	assert.Equal(t, called, true)
}

func TestResetStopWithSec(t *testing.T) {
	tw, err := NewTimeWheel(1*time.Second, 1000)
	assert.Nil(t, err)

	tw.Start()
	defer tw.Stop()

	var (
		timers = make([]*Timer, 1000)
		incr   int64
	)

	for i := 0; i < len(timers); i++ {
		timers[i] = tw.AfterFunc(time.Duration(i)*time.Millisecond, func() {
			atomic.AddInt64(&incr, 1)
		})
	}

	for i := 0; i < len(timers); i++ {
		timers[i].Stop()
	}

	time.Sleep(3 * time.Second)
	assert.Equal(t, 0, len(tw.bucketIndexes))
	assert.EqualValues(t, 0, incr)

	for i := 0; i < len(timers); i++ {
		i := i
		go func() {
			tw.AfterFunc(time.Duration(i)*time.Millisecond, func() {
				atomic.AddInt64(&incr, 1)
			})
		}()
	}

	time.Sleep(3 * time.Second)
	assert.EqualValues(t, 1000, incr)
}

func TestResetStop2WithMill(t *testing.T) {
	tw, err := NewTimeWheel(100*time.Millisecond, 1000)
	assert.Nil(t, err)

	tw.Start()
	defer tw.Stop()

	var (
		count = 1000
		incr  int64
	)

	for i := 0; i < count; i++ {
		i := i
		go func() {
			tw.AfterFunc(time.Duration(i)*time.Millisecond, func() {
				atomic.AddInt64(&incr, 1)
			})
		}()
	}

	time.Sleep(2 * time.Second)
	assert.EqualValues(t, count, incr)
}

func TestAddRemove(t *testing.T) {
	tw, err := NewTimeWheel(100*time.Millisecond, 10000, TickSafeMode())
	assert.Equal(t, nil, err)

	tw.Start()
	defer tw.Stop()

	var incr int64
	for i := 0; i < 1000; i++ {
		task := tw.Add(1*time.Second, func() {
			atomic.AddInt64(&incr, 1)
		})

		tw.Remove(task.id)
	}

	time.Sleep(2 * time.Second)
	assert.EqualValues(t, 0, incr)

	for i := 0; i < 1000; i++ {
		tw.Add(1*time.Second, func() {
			atomic.AddInt64(&incr, 1)
		})
	}

	time.Sleep(2 * time.Second)
	assert.EqualValues(t, 1000, incr)
}
