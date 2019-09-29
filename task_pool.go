package timewheel

import (
	"sync"
)

var incr = 0

var (
	defaultTaskPool = newTaskPool()
)

type taskPool struct {
	bp *sync.Pool
}

func newTaskPool() *taskPool {
	return &taskPool{
		bp: &sync.Pool{
			New: func() interface{} {
				return &Task{}
			},
		},
	}
}

func (pool *taskPool) get() *Task {
	return pool.bp.Get().(*Task)
}

func (pool *taskPool) put(obj *Task) {
	obj.Reset()
	pool.bp.Put(obj)
}
