package timewheel

import (
	"time"
)

var (
	defaultTimeWheel, _ = NewTimeWheel(time.Second, 120)
)

func Start() {
	defaultTimeWheel.Start()
}

func Stop() {
	defaultTimeWheel.Stop()
}

func ResetDefaultTimeWheel(tw *TimeWheel) {
	tw.Start()
	defaultTimeWheel = tw
}

func Add(delay time.Duration, callback func()) *Task {
	return defaultTimeWheel.Add(delay, callback)
}

func AddCron(delay time.Duration, callback func()) *Task {
	return defaultTimeWheel.AddCron(delay, callback)
}

func Remove(task *Task) error {
	return defaultTimeWheel.Remove(task)
}

func NewTimer(delay time.Duration) *Timer {
	return defaultTimeWheel.NewTimer(delay)
}

func NewTicker(delay time.Duration) *Ticker {
	return defaultTimeWheel.NewTicker(delay)
}

func AfterFunc(delay time.Duration, callback func()) *Timer {
	return defaultTimeWheel.AfterFunc(delay, callback)
}

func After(delay time.Duration) <-chan time.Time {
	return defaultTimeWheel.After(delay)
}

func Sleep(delay time.Duration) {
	defaultTimeWheel.Sleep(delay)
}
