package cube

import (
	"runtime"
	"sync"
)

func NewTaskGroup(workers int, queuesize int) (push func(task func()), wait func()) {
	if workers < 1 {
		workers = runtime.NumCPU()
	}

	var (
		wg    sync.WaitGroup
		tasks = make(chan func(), max(0, queuesize))
	)

	push = func(task func()) {
		wg.Add(1)
		tasks <- task
	}

	for range workers {
		FlyRethrow(func() {
			for task := range tasks {
				func() {
					defer wg.Done()
					task()
				}()
			}
		})
	}
	wait = func() {
		close(tasks)
		wg.Wait()
	}
	return
}
