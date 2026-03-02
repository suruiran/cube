package cube

import (
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	deathlock    sync.Mutex
	deathfncs    []func(wg *sync.WaitGroup)
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

func OnDeath(fn func(wg *sync.WaitGroup)) {
	deathlock.Lock()
	defer deathlock.Unlock()
	deathfncs = append(deathfncs, fn)
}

func ExecAllDeathHooks() {
	if !deathing.CompareAndSwap(false, true) {
		return
	}

	deathlock.Lock()
	var tmps = make([]func(wg *sync.WaitGroup), 0, len(deathfncs))
	tmps = append(tmps, deathfncs...)
	deathfncs = deathfncs[:0]
	deathlock.Unlock()

	wg := &sync.WaitGroup{}
	for _, fn := range tmps {
		fn(wg)
	}

	Fly(func() {
		time.Sleep(8 * time.Second)
		os.Exit(1)
	})

	wg.Wait()
}

func init() {
	deathsigchan = make(chan os.Signal, 1)
	signal.Notify(deathsigchan, DEATH_SIGNALS...)

	Fly(func() {
		<-deathsigchan
		ExecAllDeathHooks()
		os.Exit(0)
	})
}
