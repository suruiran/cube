package cube

import (
	"context"
	"errors"
	"log/slog"
	"runtime"
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
	counter CloseableCounter

	tasks chan ITaskItem

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
	if opts == nil {
		opts = &TaskPoolOptions{}
	}
	if opts.Context == nil {
		opts.Context = context.Background()
	}
	if opts.Workers <= 0 {
		opts.Workers = max(runtime.NumCPU()/2, 1)
	}
	if opts.MaxQueueSize <= 0 {
		opts.MaxQueueSize = opts.Workers * 8
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
		pool.counter.Release()
		if err := recover(); err != nil {
			if pool.onpanic != nil {
				pool.onpanic(pool.ctx, task, err)
				return
			}
			slog.Error("cube.taskpool panicked", slog.Any("panic", err))
		}
	}()
	task.Exec(pool.ctx)
}

var (
	ErrTaskPoolQueueFull = errors.New("cube.taskpool: queue full")
	ErrTaskPoolClosed    = errors.New("cube.taskpool: closed")
)

func (pool *TaskPool) Add(task ITaskItem) (err error) {
	if !pool.counter.Acquire() {
		return ErrTaskPoolClosed
	}

	select {
	case pool.tasks <- task:
		{
			return nil
		}
	case <-pool.ctx.Done():
		{
			pool.counter.Release()
			return pool.ctx.Err()
		}
	default:
		{
			pool.counter.Release()
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

	onerr := func(e error) {
		if opts.OnError == nil {
			slog.Error("cube.TaskPool: submit task failed", slog.Any("err", e))
		} else {
			opts.OnError(task, e)
		}
	}

	retry := func() {
		i := 0
		for {
			err := pool.Add(task)
			i++
			if err == nil {
				return
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, ErrTaskPoolClosed) {
				return
			}
			if opts.Times > 0 && i >= opts.Times {
				onerr(err)
				return
			}
			if opts.Step > 0 {
				time.Sleep(opts.Step)
			} else {
				time.Sleep(time.Millisecond)
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

type CloseMode int

const (
	CloseModeGraceful CloseMode = iota
	CloseModeImmediate
	CloseModeDrain
)

func (pool *TaskPool) Close(mode CloseMode) {
	if !pool.counter.Close() {
		return
	}

	switch mode {
	case CloseModeImmediate:
		{
			pool.cancel()
			return
		}
	case CloseModeGraceful:
		{
			pool.cancel()
		drain:
			for {
				select {
				case _, ok := <-pool.tasks:
					{
						if !ok {
							break drain
						}
						pool.counter.Release()
					}
				default:
					{
						break drain
					}
				}
			}
			pool.counter.Wait(0)
			return
		}
	case CloseModeDrain:
		{
			pool.counter.Wait(0)
			pool.cancel()
			return
		}
	default:
		{
			panic(errors.New("cube.taskpool: unknonwn waitkind"))
		}
	}
}

func (pool *TaskPool) CloseImmediate() {
	pool.Close(CloseModeImmediate)
}

func (pool *TaskPool) CloseDrain() {
	pool.Close(CloseModeDrain)
}
