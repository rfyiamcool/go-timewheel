package timewheel

import (
	"math/rand"
	"sync/atomic"
	"time"
)

type TimeWheelPool struct {
	pool []*TimeWheel
	size int64
	incr int64 // not need for high accuracy
}

func NewTimeWheelPool(size int, tick time.Duration, bucketsNum int, options ...optionCall) (*TimeWheelPool, error) {
	twp := &TimeWheelPool{
		pool: make([]*TimeWheel, size),
		size: int64(size),
	}

	for index := 0; index < bucketsNum; index++ {
		tw, err := NewTimeWheel(tick, bucketsNum, options...)
		if err != nil {
			return twp, err
		}

		twp.pool[index] = tw
	}

	return twp, nil
}

func (tp *TimeWheelPool) Get() *TimeWheel {
	incr := atomic.AddInt64(&tp.incr, 1)
	idx := incr % tp.size
	return tp.pool[idx]
}

func (tp *TimeWheelPool) GetRandom() *TimeWheel {
	rand.Seed(time.Now().UnixNano())
	idx := rand.Int63n(tp.size)
	return tp.pool[idx]
}

func (tp *TimeWheelPool) Start() {
	for _, tw := range tp.pool {
		tw.Start()
	}
}

func (tp *TimeWheelPool) Stop() {
	for _, tw := range tp.pool {
		tw.Stop()
	}
}
