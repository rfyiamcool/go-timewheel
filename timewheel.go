package timewheel

import (
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

func Add(delay time.Duration, callback func()) *Task {
	return DefaultTimeWheel.Add(delay, callback)
}

func AddCron(delay time.Duration, callback func()) *Task {
	return DefaultTimeWheel.AddCron(delay, callback)
}

func Remove(taskId int64) error {
	return DefaultTimeWheel.Remove(taskId)
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
