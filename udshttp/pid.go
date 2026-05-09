package udshttp

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func _IsRunning(fp string) bool {
	_, err := GetRunningPid(fp)
	return err == nil
}

var (
	GetPidExecPath func(ctx context.Context, pid int) (string, error)
)

func GetRunningPid(fp string) (int, error) {
	pidfile := fmt.Sprintf("%s.pid", fp)
	fcontent, err := os.ReadFile(pidfile)
	if err != nil {
		return 0, err
	}
	pids := strings.TrimSpace(string(fcontent))
	pid, err := strconv.ParseUint(pids, 10, 64)
	if err != nil {
		return 0, err
	}
	if GetPidExecPath != nil {
		_, err = GetPidExecPath(context.Background(), int(pid))
		if err == nil {
			return int(pid), nil
		}
	} else {
		fmt.Printf("cube.udshttp: GetPidExecPath is nil, pid: %d\n", pid)
	}
	return 0, err
}

func writepid(fp string) {
	pidfile := fmt.Sprintf("%s.pid", fp)
	os.WriteFile(pidfile, fmt.Appendf(nil, "%d", os.Getpid()), 0600) //nolint:errcheck
}

func CleanFiles(fp string) {
	os.Remove(fp)                         //nolint:errcheck
	os.Remove(fmt.Sprintf("%s.pid", fp))  //nolint:errcheck
	os.Remove(fmt.Sprintf("%s.lock", fp)) //nolint:errcheck
}
