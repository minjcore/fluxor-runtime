package core

type WorkerPool struct {
	tasks chan func()
}

func NewWorkerPool(workerCount int, queueSize int) *WorkerPool {
	if workerCount <= 0 {
		workerCount = 1
	}
	if queueSize <= 0 {
		queueSize = 1024
	}

	wp := &WorkerPool{tasks: make(chan func(), queueSize)}
	for i := 0; i < workerCount; i++ {
		go func() {
			for task := range wp.tasks {
				task()
			}
		}()
	}
	return wp
}

// Submit enqueues a unit of work. Keep tasks short; if you need cancellation,
// close over a context in your task.
func (wp *WorkerPool) Submit(task func()) {
	wp.tasks <- task
}

func (wp *WorkerPool) Shutdown() {
	close(wp.tasks)
}
