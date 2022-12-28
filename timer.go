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
	id       int64
	round    int
	callback func()

	async  bool
	stop   bool
	circle bool
	// circleNum int
}

func (t *Task) ID() int64 {
	return t.id
}

// for sync.Pool
func (t *Task) Reset() {
	t.delay = 0
	t.id = 0
	t.round = 0
	t.callback = nil

	t.async = false
	t.stop = false
	t.circle = false
}

type optionCall func(*TimeWheel) error

func TickSafeMode() optionCall {
	return func(o *TimeWheel) error {
		o.tickQueue = make(chan time.Time, 10)
		return nil
	}
}

func SetStartTaskID(taskId int64) optionCall {
	return func(o *TimeWheel) error {
		o.randomID = taskId
		return nil
	}
}

// todo:
// func SetSyncPool(state bool) optionCall {
// 	return func(o *TimeWheel) error {
// 		o.syncPool = state
// 		return nil
// 	}
// }

type TimeWheel struct {
	randomID int64

	tick      time.Duration
	ticker    *time.Ticker
	tickQueue chan time.Time

	bucketsNum    int
	buckets       []map[int64]*Task // key: added item, value: *Task
	bucketIndexes map[int64]int     // key: added item, value: bucket position

	currentIndex int

	onceStart sync.Once

	stopC chan struct{}

	exited bool

	sync.RWMutex
}

// NewTimeWheel create new time wheel
func NewTimeWheel(tick time.Duration, bucketsNum int, options ...optionCall) (*TimeWheel, error) {
	if tick.Milliseconds() < 1 {
		return nil, errors.New("invalid params, must tick >= 1 ms")
	}
	if bucketsNum <= 0 {
		return nil, errors.New("invalid params, must bucketsNum > 0")
	}

	tw := &TimeWheel{
		// tick
		tick:      tick,
		tickQueue: make(chan time.Time, 10),

		// store
		bucketsNum:    bucketsNum,
		bucketIndexes: make(map[int64]int, 1024*100),
		buckets:       make([]map[int64]*Task, bucketsNum),
		currentIndex:  0,

		// signal
		stopC: make(chan struct{}),
	}

	for i := 0; i < bucketsNum; i++ {
		tw.buckets[i] = make(map[int64]*Task, 16)
	}

	for _, op := range options {
		op(tw)
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
			go tw.tickGenerator()
		},
	)
}

func (tw *TimeWheel) StartTaskID() int64 {
	return tw.randomID
}

func (tw *TimeWheel) tickGenerator() {
	if tw.tickQueue != nil {
		return
	}

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
	queue := tw.ticker.C
	if tw.tickQueue == nil {
		queue = tw.tickQueue
	}

	for {
		select {
		case <-queue:
			tw.handleTick()

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

func (tw *TimeWheel) collectTask(taskId int64) {
	index := tw.bucketIndexes[taskId]
	delete(tw.bucketIndexes, taskId)
	delete(tw.buckets[index], taskId)

	// todo:
	// if tw.syncPool {
	// 	defaultTaskPool.put(task)
	// }
}

func (tw *TimeWheel) handleTick() {
	tw.Lock()
	defer tw.Unlock()

	bucket := tw.buckets[tw.currentIndex]
	for k, task := range bucket {
		if task.stop {
			tw.collectTask(task.id)
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
			tw.collectTask(task.id)
			tw.putCircle(task, modeIsCircle)
			continue
		}

		// gc
		tw.collectTask(task.id)
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
	task := new(Task)

	// todo:
	// var task *Task
	// if tw.syncPool {
	// 	task = defaultTaskPool.get()
	// }

	task.delay = delay
	task.id = id
	task.callback = callback
	task.circle = circle
	task.async = async // refer to src/runtime/time.go

	tw.put(task)
	return task
}

func (tw *TimeWheel) put(task *Task) {
	tw.Lock()
	defer tw.Unlock()

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

func (tw *TimeWheel) Remove(taskId int64) error {
	// tw.removeC <- task
	tw.remove(taskId)
	return nil
}

func (tw *TimeWheel) remove(taskId int64) {
	tw.Lock()
	defer tw.Unlock()

	tw.collectTask(taskId)
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

func (tw *TimeWheel) After(delay time.Duration) <-chan time.Time {
	queue := make(chan time.Time, 1)
	tw.addAny(delay,
		func() {
			queue <- time.Now()
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

// similar to golang std timer
type Timer struct {
	task   *Task
	tw     *TimeWheel
	fn     func() // external custom func
	stopFn func() // call function when timer stop

	C chan bool

	cancel context.CancelFunc
	Ctx    context.Context
}

func (t *Timer) Reset(delay time.Duration) {
	// first stop old task
	t.task.stop = true

	// make new task
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
	if t.stopFn != nil {
		t.stopFn()
	}

	t.task.stop = true
	t.cancel()
	t.tw.Remove(t.task.id)
}

func (t *Timer) AddStopFunc(callback func()) {
	t.stopFn = callback
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
	t.tw.Remove(t.task.id)
}

func notfiyChannel(q chan bool) {
	select {
	case q <- true:
	default:
	}
}

func (tw *TimeWheel) genUniqueID() int64 {
	id := atomic.AddInt64(&tw.randomID, 1)
	return id
}
