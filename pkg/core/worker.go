package core

import (
	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

type WorkerPool struct {
	tasks chan func()
}

func NewWorkerPool(size int) *WorkerPool {
	failfast.Positive(size, "worker pool size")
	wp := &WorkerPool{tasks: make(chan func(), 1000)}
	for i := 0; i < size; i++ {
		go func() {
			for task := range wp.tasks {
				task()
			}
		}()
	}
	return wp
}

func (wp *WorkerPool) Submit(task func()) {
	// Fail-fast: task cannot be nil
	failfast.NotNil(task, "task")
	wp.tasks <- task
}

func (wp *WorkerPool) Shutdown() {
	close(wp.tasks)
}
