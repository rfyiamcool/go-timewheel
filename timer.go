package timewheel

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

const (
	typeTimer taskType = iota
	typeTicker
)

type taskType int64

type taskID int64

type Task struct {
	delay    time.Duration
	id       taskID
	round    int
	callback func()

	async  bool
	stop   bool
	circle bool
	// circleNum int
}

type TimeWheel struct {
	randomID int64
	tick     time.Duration
	ticker   *time.Ticker

	bucketsNum    int
	buckets       []map[taskID]*Task // key: added item, value: *Task
	bucketIndexes map[taskID]int     // key: added item, value: bucket position

	currentIndex int

	onceStart sync.Once

	addC    chan *Task
	removeC chan taskID
	stopC   chan struct{}
}

// NewTimeWheel create new time wheel
func NewTimeWheel(tick time.Duration, bucketsNum int) (*TimeWheel, error) {
	if tick.Seconds() < 1 || bucketsNum <= 0 {
		return nil, errors.New("invalid params")
	}

	tw := &TimeWheel{
		tick:          tick,
		bucketsNum:    bucketsNum,
		bucketIndexes: make(map[taskID]int, 1024*10),
		buckets:       make([]map[taskID]*Task, bucketsNum),
		currentIndex:  0,
		addC:          make(chan *Task, 1024*5),
		removeC:       make(chan taskID, 1024*2),
		stopC:         make(chan struct{}),
	}

	for i := 0; i < bucketsNum; i++ {
		tw.buckets[i] = make(map[taskID]*Task, 16)
	}

	return tw, nil
}

// Start start the time wheel
func (tw *TimeWheel) Start() {
	// onlye once start
	tw.onceStart.Do(
		func() {
			tw.ticker = time.NewTicker(tw.tick)
			go tw.schduler()
		},
	)
}

func (tw *TimeWheel) schduler() {
	for {
		select {
		case <-tw.ticker.C:
			tw.handleTick()
		case task := <-tw.addC:
			tw.add(task)
		case key := <-tw.removeC:
			tw.remove(key)
		case <-tw.stopC:
			tw.ticker.Stop()
			return
		}
	}
}

// Stop stop the time wheel
func (tw *TimeWheel) Stop() {
	tw.stopC <- struct{}{}
}

func (tw *TimeWheel) collectTask(task *Task) {
	delete(tw.buckets[tw.currentIndex], task.id)
	delete(tw.bucketIndexes, task.id)
}

func (tw *TimeWheel) handleTick() {
	bucket := tw.buckets[tw.currentIndex]
	for k, task := range bucket {
		if task.stop {
			tw.collectTask(task)
			continue
		}

		if bucket[k].round > 0 {
			bucket[k].round--
			continue
		}

		if task.async {
			go task.callback()
		} else {
			// optimize gopool
			task.callback()
		}

		// circle
		if task.circle == true {
			tw.add(task)
			continue
		}

		// gc
		tw.collectTask(task)
	}

	if tw.currentIndex == tw.bucketsNum-1 {
		tw.currentIndex = 0
		return
	}

	tw.currentIndex++
}

// Add add an item into time wheel
func (tw *TimeWheel) Add(delay time.Duration, callback func()) *Task {
	if delay <= 0 {
		delay = tw.tick
	}

	id := tw.genUniqueID()
	task := &Task{delay: delay, id: id, callback: callback}
	tw.addC <- task

	return task
}

func (tw *TimeWheel) AddCron(delay time.Duration, callback func()) *Task {
	if delay <= 0 {
		delay = tw.tick
	}

	id := tw.genUniqueID()
	task := &Task{
		delay:    delay,
		id:       id,
		callback: callback,
		circle:   true,
	}
	tw.addC <- task
	return task
}

func (tw *TimeWheel) add(task *Task) {
	round := tw.calculateRound(task.delay)
	index := tw.calculateIndex(task.delay)
	task.round = round
	tw.bucketIndexes[task.id] = index
	tw.buckets[index][task.id] = task
}

func (tw *TimeWheel) calculateRound(delay time.Duration) (round int) {
	delaySeconds := int(delay.Seconds())
	tickSeconds := int(tw.tick.Seconds())
	round = delaySeconds / tickSeconds / tw.bucketsNum
	return
}

func (tw *TimeWheel) calculateIndex(delay time.Duration) (index int) {
	delaySeconds := int(delay.Seconds())
	tickSeconds := int(tw.tick.Seconds())
	index = (tw.currentIndex + delaySeconds/tickSeconds) % tw.bucketsNum
	return
}

func (tw *TimeWheel) Remove(task *Task) error {
	tw.removeC <- task.id
	return nil
}

func (tw *TimeWheel) remove(id taskID) {
	if index, ok := tw.bucketIndexes[id]; ok {
		delete(tw.bucketIndexes, id)
		delete(tw.buckets[index], id)
	}
	return
}

func (tw *TimeWheel) NewTimer(delay time.Duration) *Timer {
	queue := make(chan bool, 1) // buf = 1, refer to src/time/sleep.go
	task := tw.Add(delay,
		func() {
			notfiyChannel(queue)
		},
	)
	task.async = false // refer to src/runtime/time.go

	// init timer
	ctx, cancel := context.WithCancel(context.Background())
	timer := &Timer{
		tw:     tw,
		C:      queue, // faster
		task:   task,
		Ctx:    ctx,
		cancel: cancel,
	}

	return timer
}

func (tw *TimeWheel) NewTicker(delay time.Duration) *Ticker {
	queue := make(chan bool, 1)
	task := tw.AddCron(delay,
		func() {
			notfiyChannel(queue)
		},
	)
	task.async = false

	// init ticker
	ctx, cancel := context.WithCancel(context.Background())
	ticker := &Ticker{
		task:   task,
		tw:     tw,
		C:      queue,
		Ctx:    ctx,
		cancel: cancel,
	}

	return ticker
}

func (tw *TimeWheel) After(delay time.Duration) chan<- bool {
	queue := make(chan bool, 1)
	tw.Add(delay,
		func() {
			queue <- true
		},
	)
	return queue
}

func (tw *TimeWheel) Sleep(delay time.Duration) {
	queue := make(chan bool, 1)
	tw.Add(delay,
		func() {
			queue <- true
		},
	)
	<-queue
}

// like golang std timer
type Timer struct {
	task *Task
	tw   *TimeWheel
	fn   func()
	C    chan bool

	cancel context.CancelFunc
	Ctx    context.Context
}

func (t *Timer) Reset(delay time.Duration) {
	task := t.tw.Add(delay, func() {
		notfiyChannel(t.C)
	})

	t.task = task
}

func (t *Timer) Stop() {
	t.task.stop = true
	t.cancel()
	t.tw.Remove(t.task)
}

type Ticker struct {
	tw     *TimeWheel
	task   *Task
	cancel context.CancelFunc

	C   chan bool
	Ctx context.Context
}

func (t *Ticker) Stop() {
	t.task.stop = true
	t.cancel()
	t.tw.Remove(t.task)
}

func notfiyChannel(q chan bool) {
	select {
	case q <- true:
	default:
	}
}

func (tw *TimeWheel) genUniqueID() taskID {
	id := atomic.AddInt64(&tw.randomID, 1)
	return taskID(id)
}
