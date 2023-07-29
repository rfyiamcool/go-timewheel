package timewheel

import (
	"fmt"
	"time"
)

var (
	DefaultTimeWheel, _ = NewTimeWheel(time.Second, 120)
)

func init() {
	DefaultTimeWheel.Start()
}

func ResetDefaultTimeWheel(tw *TimeWheel) {
	tw.Start()
	DefaultTimeWheel = tw
}

func Add(delay time.Duration, callback func(), async bool) *Task {
	return DefaultTimeWheel.Add(delay, callback, async)
}

func AddCron(delay time.Duration, callback func(), async bool) *Task {
	return DefaultTimeWheel.AddCron(delay, callback, async)
}

func Remove(task *Task) error {
	if ok := DefaultTimeWheel.Remove(task); !ok {
		return fmt.Errorf("fail to remove task:%v", task)
	}
	
	return nil
}

func NewTimer(delay time.Duration) *Timer {
	return DefaultTimeWheel.NewTimer(delay)
}

func NewTicker(delay time.Duration) *Ticker {
	return DefaultTimeWheel.NewTicker(delay)
}

func AfterFunc(delay time.Duration, callback func()) *Timer {
	return DefaultTimeWheel.AfterFunc(delay, callback)
}

func After(delay time.Duration) <-chan time.Time {
	return DefaultTimeWheel.After(delay)
}

func Sleep(delay time.Duration) {
	DefaultTimeWheel.Sleep(delay)
}
