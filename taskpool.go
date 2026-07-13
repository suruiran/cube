package cube

import (
	"context"
	"errors"
	"log/slog"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
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

	tasks chan ITaskItem
	wg    sync.WaitGroup

	ctx     context.Context
	cancel  context.CancelFunc
	onpanic func(ctx context.Context, item ITaskItem, err any)
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
	if opts.Workers <= 0 {
		opts.Workers = runtime.NumCPU() / 2
	}
	if opts.MaxQueueSize <= 0 {
		opts.MaxQueueSize = opts.Workers * 2
	}
	pool := &TaskPool{
		tasks:   make(chan ITaskItem, opts.MaxQueueSize),
		onpanic: opts.OnPanic,
	}
	pool.ctx, pool.cancel = context.WithCancel(opts.Context)

	pool.run(opts.Workers)
	return pool
}

func (pool *TaskPool) run(size int) {
	for range size {
		Fly(func() {
			for {
				select {
				case f, ok := <-pool.tasks:
					{
						if !ok {
							return
						}
						pool.exec(f)
					}
				case <-pool.ctx.Done():
					{
						return
					}
				}
			}
		})
	}
}

func (pool *TaskPool) exec(task ITaskItem) {
	defer func() {
		pool.wg.Done()
		if err := recover(); err != nil {
			if pool.onpanic != nil {
				pool.onpanic(pool.ctx, task, err)
				return
			}
			slog.Error("cube.taskpool paniced", slog.Any("paniced", err))
		}
	}()

	if pool.ctx.Err() != nil {
		return
	}
	task.Exec(pool.ctx)
}

var (
	ErrTaskPoolQueueFull = errors.New("cube.taskpool: queue full")
	ErrTaskPoolClosed    = errors.New("cube.taskpool: closed")
)

func (pool *TaskPool) Add(task ITaskItem) (err error) {
	if pool.closed.Load() {
		return ErrTaskPoolClosed
	}

	pool.wg.Add(1)

	select {
	case pool.tasks <- task:
		{
			return nil
		}
	case <-pool.ctx.Done():
		{
			pool.wg.Done()
			return pool.ctx.Err()
		}
	default:
		{
			pool.wg.Done()
			return ErrTaskPoolQueueFull
		}
	}
}

type TaskPoolRetryOptions struct {
	Async   bool
	Times   int
	Step    time.Duration
	OnError func(task ITaskItem, err error)
}

func (pool *TaskPool) AddWithRetry(task ITaskItem, opts *TaskPoolRetryOptions) {
	if opts == nil {
		opts = &TaskPoolRetryOptions{}
	}

	retry := func() {
		i := 0
		for {
			err := pool.Add(task)
			i++
			if err == nil {
				return
			}
			if opts.Times > 0 && i >= opts.Times {
				if opts.OnError == nil {
					slog.Error("")
				}
				opts.OnError(task, err)
				return
			}
			if opts.Step > 0 {
				time.Sleep(opts.Step)
			}
		}
	}

	if opts.Async {
		Fly(retry)
		return
	}

	retry()
}

func (pool *TaskPool) AddFunc(f func(ctx context.Context)) error {
	return pool.Add(TaskFuncType(f))
}

func (pool *TaskPool) AddFuncWithRetry(f func(ctx context.Context), opts *TaskPoolRetryOptions) {
	pool.AddWithRetry(TaskFuncType(f), opts)
}

func (pool *TaskPool) Close(wait bool) {
	if !pool.closed.CompareAndSwap(false, true) {
		return
	}

	if wait {
		pool.wg.Wait()
	}

	close(pool.tasks)
	pool.cancel()
}
