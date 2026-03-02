package main

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/suruiran/cube/logx"
)

func main() {
	logger, err := logx.New(&logx.Opts{
		Filename:   "./test.log",
		Level:      slog.LevelDebug,
		WithStdout: false,
		Rolling: &logx.RollingOptions{
			Kind:     logx.RollingKindSize,
			FileSize: 1024 * 1024,
			Backups:  6,
		},
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(">>>> ", os.Getpid())

	for range 10 {
		go func() {
			for {
				logger.Debug("debug message")
				logger.Info("info message")
				logger.Warn("warn message")
				logger.Error("error message")
				time.Sleep(time.Millisecond)
			}
		}()
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}
