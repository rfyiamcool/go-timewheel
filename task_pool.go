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
	task := pool.bp.Get().(*Task)
	task.Reset()
	return task
}

func (pool *taskPool) put(obj *Task) {
	pool.bp.Put(obj)
}
