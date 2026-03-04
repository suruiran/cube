package cube

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

type ITaskItem interface {
	Exec(ctx context.Context)
}

type TaskFuncType func(ctx context.Context)

func (t TaskFuncType) Exec(ctx context.Context) {
	t(ctx)
}

var _ ITaskItem = TaskFuncType(nil)

type TaskPool struct {
	closed atomic.Bool

	tasks  chan ITaskItem
	wg     sync.WaitGroup
	opts   *TaskPoolOptions
	cancel context.CancelFunc
}

type TaskPoolOptions struct {
	Context      context.Context
	OnPanic      func(ctx context.Context, item ITaskItem, err any)
	Workers      int
	MaxQueueSize int
}

func NewTaskPool(opts *TaskPoolOptions) *TaskPool {
	if opts.Context == nil {
		opts.Context = context.Background()
	}
	if opts.MaxQueueSize <= 0 {
		opts.MaxQueueSize = opts.Workers * 2
	}
	pool := &TaskPool{
		tasks: make(chan ITaskItem, opts.MaxQueueSize),
		opts:  opts,
	}
	opts.Context, pool.cancel = context.WithCancel(opts.Context)

	pool.run(opts.Workers)
	return pool
}

func (pool *TaskPool) run(size int) {
	for range size {
		Fly(func() {
			for f := range pool.tasks {
				pool.exec(f)
			}
		})
	}
}

func (pool *TaskPool) exec(task ITaskItem) {
	defer func() {
		pool.wg.Done()
		if err := recover(); err != nil {
			if pool.opts.OnPanic != nil {
				pool.opts.OnPanic(pool.opts.Context, task, err)
			}
		}
	}()

	task.Exec(pool.opts.Context)
}

var (
	ErrTaskPoolQueueFull = errors.New("cube.taskpool: queue full")
	ErrTaskPoolClosed    = errors.New("cube.taskpool: closed")
)

func (pool *TaskPool) Add(task ITaskItem) (err error) {
	if pool.closed.Load() {
		return ErrTaskPoolClosed
	}

	defer func() {
		if r := recover(); r != nil {
			if pool.closed.Load() {
				err = ErrTaskPoolClosed
			} else {
				panic(r)
			}
		}
	}()

	select {
	case pool.tasks <- task:
		{
			pool.wg.Add(1)
		}
	case <-pool.opts.Context.Done():
		{
			return pool.opts.Context.Err()
		}
	default:
		{
			return ErrTaskPoolQueueFull
		}
	}
	return nil
}

func (pool *TaskPool) AddFunc(f func(ctx context.Context)) error {
	return pool.Add(TaskFuncType(f))
}

func (pool *TaskPool) Close(wait bool) {
	if !pool.closed.CompareAndSwap(false, true) {
		return
	}

	pool.cancel()
	close(pool.tasks)
	if wait {
		pool.wg.Wait()
	}
}
