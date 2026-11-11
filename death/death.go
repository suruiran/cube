package death

import (
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
)

var (
	deathlock    sync.Mutex
	deathfncs    []func()
	deathsigchan chan os.Signal

	deathing atomic.Bool

	DEATH_SIGNALS = []os.Signal{
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGHUP,
	}
)

func IsDeathing() bool {
	return deathing.Load()
}

func Register(fn func()) {
	deathlock.Lock()
	defer deathlock.Unlock()

	if deathing.Load() {
		go fn()
		return
	}
	deathfncs = append(deathfncs, fn)
}

func ExecHooks() {
	if !deathing.CompareAndSwap(false, true) {
		return
	}

	deathlock.Lock()
	var tmps = make([]func(), 0, len(deathfncs))
	tmps = append(tmps, deathfncs...)
	deathfncs = deathfncs[:0]
	deathlock.Unlock()

	wg := &sync.WaitGroup{}
	wg.Add(len(tmps))
	for _, fn := range tmps {
		go func() {
			defer wg.Done()
			fn()
		}()
	}

	wg.Wait()
}

func init() {
	deathsigchan = make(chan os.Signal, 2)
	signal.Notify(deathsigchan, DEATH_SIGNALS...)

	go func() {
		<-deathsigchan
		go func() {
			<-deathsigchan
			os.Exit(1)
		}()

		ExecHooks()
		os.Exit(0)
	}()
}
