package udshttp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gofrs/flock"
	"github.com/suruiran/cube"
)

type Client struct {
	fp     string
	spwan  func() (*exec.Cmd, error)
	cli    *http.Client
	logger *slog.Logger
}

func (cli *Client) Logger() *slog.Logger {
	if cli.logger == nil {
		return slog.Default()
	}
	return cli.logger
}

func NewClient(fp string, spwan func() (*exec.Cmd, error), logger *slog.Logger) *Client {
	return &Client{fp: fp, spwan: spwan,
		cli: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					if !_IsRunning(fp) {
						return nil, ErrServerIsNotRunning
					}
					return net.DialTimeout("unix", fp, time.Second)
				},
			},
		},
		logger: logger,
	}
}

var (
	ErrServerIsNotRunning = errors.New("the uds server is not running")
)

type _CtkKeyType int

const (
	_CtkKeyForSpwanRetryTimes _CtkKeyType = iota
	_CtkKeyForCancelFunc
)

type ICancelable interface {
	Get() context.CancelFunc
	Set(context.CancelFunc)
}

// WithCancel
// the server will not cancel the request when the handle function is done, it's the client's responsibility.
func WithCancel(ctx context.Context, cancelable ICancelable) context.Context {
	return context.WithValue(ctx, _CtkKeyForCancelFunc, cancelable)
}

type _Ec struct {
	count int
}

func incr_retry_times(ctx context.Context) (context.Context, int) {
	iv := ctx.Value(_CtkKeyForSpwanRetryTimes)
	if iv == nil {
		ctx = context.WithValue(ctx, _CtkKeyForSpwanRetryTimes, &_Ec{count: 1})
		return ctx, 1
	}
	ecv := iv.(*_Ec)
	ecv.count++
	return ctx, ecv.count
}

func spwan(cli *Client) error {
	lockfn := fmt.Sprintf("%s.lock", cli.fp)

	lock := flock.New(lockfn)
	locked, err := lock.TryLock()
	if err != nil {
		cli.logger.Error("try lock failed", slog.Any("error", err), slog.String("lockfile", lockfn))
		return err
	}
	if !locked {
		time.Sleep(time.Millisecond * 200)
		cli.logger.Info("can not require file lock, may be another process is running, exit", slog.String("lockfile", lockfn))
		return nil
	}
	defer lock.Unlock() //nolint:errcheck

	cmd, err := cli.spwan()
	if err != nil {
		return err
	}
	cli.logger.Debug(
		"spwan server process",
		slog.String("dir", cmd.Dir),
		slog.String("cmd", cmd.String()),
	)

	err = cmd.Start()
	if err != nil {
		cli.logger.Error("start server process failed", slog.Any("error", err))
		return err
	}

	cube.Fly(func() {
		cli.logger.Debug(
			"server process started",
			slog.Int("pid", cmd.Process.Pid),
		)
		werr := cmd.Wait()
		if werr != nil {
			cli.logger.Error("server process exited with error", slog.Any("error", werr))
		}
	})

	for range 20 {
		time.Sleep(time.Millisecond * 100)
		if _IsRunning(cli.fp) {
			break
		}
	}
	return nil
}

var (
	reqidseq atomic.Int64
)

func init() {
	reqidseq.Store(10)
}

func Request[Input any, Output any](ctx context.Context, cli *Client, input Input) (Output, error) {
	var out Output
	it := reflect.TypeFor[Input]()
	if it.Kind() == reflect.Pointer {
		it = it.Elem()
	}
	ot := reflect.TypeFor[Output]()
	isptr := false
	if ot.Kind() == reflect.Pointer {
		ot = ot.Elem()
		isptr = true
	}

	if it.Kind() != reflect.Struct {
		panic(fmt.Errorf("cube.udshttp: input is not a struct/*struct"))
	}
	if ot.Kind() != reflect.Struct {
		panic(fmt.Errorf("cube.udshttp: output is not a struct/*struct"))
	}

	scopelog := cli.logger.With(slog.Any("action", it.Name()))

	reqid := reqidseq.Add(1)

	llc := ctx.Value(_CtkKeyForCancelFunc)
	if llc != nil {
		cancel := llc.(ICancelable).Get()
		(llc.(ICancelable)).Set(func() {
			cancel()
			cube.Fly(func() {
				if _, err := Request[_CancelAction, _CancelActionResult](ctx, cli, _CancelAction{ReqId: reqid}); err != nil {
					cli.logger.Error("cancal call failed", slog.String("err", err.Error()))
				}
			})
		})
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("http://localhost/%s", strings.ToLower(it.Name())),
		bytes.NewBuffer(cube.MustMarshalJSON(input)),
	)
	if err != nil {
		scopelog.Error("create request failed", slog.Any("error", err))
		return out, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Os-Pid", fmt.Sprintf("%d", os.Getpid()))
	req.Header.Set(HeaderReqId, fmt.Sprintf("%d", reqid))
	if llc != nil {
		req.Header.Set(HeaderIsCancelable, "true")
	}

	resp, err := cli.cli.Do(req)
	if err != nil {
		err = UnwrapUrlError(err)
		scopelog.Error("do request failed", slog.Any("error", err))
		if errors.Is(err, context.Canceled) {
			return out, err
		}
		if cli.spwan == nil {
			return out, err
		}
		var retrytimes int
		ctx, retrytimes = incr_retry_times(ctx)
		if retrytimes > 10 {
			scopelog.Error("do request failed after retry", slog.Any("error", err))
			return out, err
		}

		if retrytimes > 0 {
			time.Sleep(time.Duration(rand.IntN(80)) * time.Millisecond)
		}

		err := spwan(cli)
		if err != nil {
			scopelog.Error("spwan server failed", slog.Any("error", err), slog.Int("retry", retrytimes))
			return out, err
		}
		if !_IsRunning(cli.fp) {
			ok := false
			for range 10 {
				time.Sleep(time.Millisecond * 100)
				if _IsRunning(cli.fp) {
					ok = true
					break
				}
			}
			if !ok {
				scopelog.Error("server is not running after spwan", slog.Int("retry", retrytimes))
				return out, ErrServerIsNotRunning
			}
		}
		return Request[Input, Output](ctx, cli, input)
	}
	defer resp.Body.Close() //nolint:errcheck
	bs, err := io.ReadAll(resp.Body)
	if err != nil {
		scopelog.Error("read response body failed", slog.Any("error", err))
		return out, err
	}
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusBadRequest {
			err := errors.New(string(bs))
			scopelog.Error("bad request", slog.Any("error", err))
			return out, err
		}
		err := fmt.Errorf("%s, %s", resp.Status, string(bs))
		scopelog.Error("response status code is not ok", slog.Any("error", err))
		return out, err
	}

	if isptr {
		ptrv := reflect.New(ot)
		err = cube.UnmarshalJSON(bs, ptrv.Interface())
		if err == nil {
			out = ptrv.Interface().(Output)
		}
	} else {
		err = cube.UnmarshalJSON(bs, &out)
	}

	if err == nil {
		scopelog.Debug("request succeeded")
	} else {
		scopelog.Error("unmarshal response failed", slog.Any("error", err))
	}
	return out, err
}

func UnwrapUrlError(err error) error {
	if ue, ok := errors.AsType[*url.Error](err); ok {
		return ue.Err
	}
	return err
}
