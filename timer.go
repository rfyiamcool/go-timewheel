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

	modeIsCircle  = true
	modeNotCircle = false

	modeIsAsync  = true
	modeNotAsync = false
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
	randomID  int64
	tickQueue chan time.Time
	tick      time.Duration
	ticker    *time.Ticker

	bucketsNum    int
	buckets       []map[taskID]*Task // key: added item, value: *Task
	bucketIndexes map[taskID]int     // key: added item, value: bucket position

	currentIndex int

	onceStart sync.Once

	addC    chan *Task
	removeC chan taskID
	stopC   chan struct{}

	exited bool
}

// NewTimeWheel create new time wheel
func NewTimeWheel(tick time.Duration, bucketsNum int) (*TimeWheel, error) {
	if tick.Seconds() < 0.1 {
		return nil, errors.New("invalid params, must tick >= 100 ms")
	}
	if bucketsNum <= 0 {
		return nil, errors.New("invalid params, must bucketsNum > 0")
	}

	tw := &TimeWheel{
		tick:          tick,
		tickQueue:     make(chan time.Time, 10),
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
			go tw.tickGenerator()
			go tw.schduler()
		},
	)
}

func (tw *TimeWheel) tickGenerator() {
	for !tw.exited {
		select {
		case <-tw.ticker.C:
			select {
			case tw.tickQueue <- time.Now():
			default:
				panic("raise long time blocking")
			}
		}
	}
}

func (tw *TimeWheel) schduler() {
	for {
		select {
		case <-tw.tickQueue:
			tw.handleTick()
		case task := <-tw.addC:
			tw.put(task)
		case key := <-tw.removeC:
			tw.remove(key)
		case <-tw.stopC:
			tw.exited = true
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
			tw.putCircle(task, modeIsCircle)
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

// Add add an task
func (tw *TimeWheel) Add(delay time.Duration, callback func()) *Task {
	return tw.addAny(delay, callback, modeNotCircle, modeIsAsync)
}

// AddCron add interval task
func (tw *TimeWheel) AddCron(delay time.Duration, callback func()) *Task {
	return tw.addAny(delay, callback, modeIsCircle, modeIsAsync)
}

func (tw *TimeWheel) addAny(delay time.Duration, callback func(), circle, async bool) *Task {
	if delay <= 0 {
		delay = tw.tick
	}

	id := tw.genUniqueID()
	task := &Task{
		delay:    delay,
		id:       id,
		callback: callback,
		circle:   circle,
		async:    async, // refer to src/runtime/time.go
	}
	tw.addC <- task
	return task
}

func (tw *TimeWheel) put(task *Task) {
	tw.store(task, false)
}

func (tw *TimeWheel) putCircle(task *Task, circleMode bool) {
	tw.store(task, circleMode)
}

func (tw *TimeWheel) store(task *Task, circleMode bool) {
	round := tw.calculateRound(task.delay)
	index := tw.calculateIndex(task.delay)

	if round > 0 && circleMode {
		task.round = round - 1
	} else {
		task.round = round
	}

	tw.bucketIndexes[task.id] = index
	tw.buckets[index][task.id] = task
}

func (tw *TimeWheel) calculateRound(delay time.Duration) (round int) {
	delaySeconds := delay.Seconds()
	tickSeconds := tw.tick.Seconds()
	round = int(delaySeconds / tickSeconds / float64(tw.bucketsNum))
	return
}

func (tw *TimeWheel) calculateIndex(delay time.Duration) (index int) {
	delaySeconds := delay.Seconds()
	tickSeconds := tw.tick.Seconds()
	index = (int(float64(tw.currentIndex) + delaySeconds/tickSeconds)) % tw.bucketsNum
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
	task := tw.addAny(delay,
		func() {
			notfiyChannel(queue)
		},
		modeNotCircle,
		modeNotAsync,
	)

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

func (tw *TimeWheel) AfterFunc(delay time.Duration, callback func()) *Timer {
	queue := make(chan bool, 1)
	task := tw.addAny(delay,
		func() {
			callback()
			notfiyChannel(queue)
		},
		modeNotCircle, modeIsAsync,
	)

	// init timer
	ctx, cancel := context.WithCancel(context.Background())
	timer := &Timer{
		tw:     tw,
		C:      queue, // faster
		task:   task,
		Ctx:    ctx,
		cancel: cancel,
		fn:     callback,
	}

	return timer
}

func (tw *TimeWheel) NewTicker(delay time.Duration) *Ticker {
	queue := make(chan bool, 1)
	task := tw.addAny(delay,
		func() {
			notfiyChannel(queue)
		},
		modeIsCircle,
		modeNotAsync,
	)

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

func (tw *TimeWheel) After(delay time.Duration) <-chan bool {
	queue := make(chan bool, 1)
	tw.addAny(delay,
		func() {
			queue <- true
		},
		modeNotCircle, modeNotAsync,
	)
	return queue
}

func (tw *TimeWheel) Sleep(delay time.Duration) {
	queue := make(chan bool, 1)
	tw.addAny(delay,
		func() {
			queue <- true
		},
		modeNotCircle, modeNotAsync,
	)
	<-queue
}

// like golang std timer
type Timer struct {
	task *Task
	tw   *TimeWheel
	fn   func() // external custom func
	C    chan bool

	cancel context.CancelFunc
	Ctx    context.Context
}

func (t *Timer) Reset(delay time.Duration) {
	var task *Task
	if t.fn != nil { // use AfterFunc
		task = t.tw.addAny(delay,
			func() {
				t.fn()
				notfiyChannel(t.C)
			},
			modeNotCircle, modeIsAsync, // must async mode
		)
	} else {
		task = t.tw.addAny(delay,
			func() {
				notfiyChannel(t.C)
			},
			modeNotCircle, modeNotAsync)
	}

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
