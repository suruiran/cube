package logx

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/suruiran/cube"
)

//go:generate go tool stringer -type=RollingKind -trimprefix=RollingKind
type RollingKind int

var (
	ErrInvalidRollingKind = errors.New("rolling kind is invalid")
	bDaily                = []byte(RollingKindDaily.String())
	bHourly               = []byte(RollingKindHourly.String())
	bMinute               = []byte(RollingKindMinutely.String())
	bSize                 = []byte(RollingKindSize.String())
)

func parseRollingKind(text []byte) (RollingKind, error) {
	if len(text) == 0 {
		return RollingKindNone, ErrInvalidRollingKind
	}
	if text[0] == '"' && text[len(text)-1] == '"' {
		text = text[1 : len(text)-1]
	}
	if len(text) == 0 {
		return RollingKindNone, ErrInvalidRollingKind
	}
	if bytes.EqualFold(text, bDaily) {
		return RollingKindDaily, nil
	}
	if bytes.EqualFold(text, bHourly) {
		return RollingKindHourly, nil
	}
	if bytes.EqualFold(text, bMinute) {
		return RollingKindMinutely, nil
	}
	if bytes.EqualFold(text, bSize) {
		return RollingKindSize, nil
	}

	num, err := strconv.Atoi(string(text))
	if err != nil {
		return RollingKindNone, ErrInvalidRollingKind
	}
	if num <= int(_RollingKindMin) || num >= int(_RollingKindMax) {
		return RollingKindNone, ErrInvalidRollingKind
	}
	return RollingKind(num), nil
}

func (i *RollingKind) UnmarshalText(text []byte) error {
	kind, err := parseRollingKind(text)
	if err != nil {
		return err
	}
	*i = kind
	return nil
}

func (i *RollingKind) UnmarshalJSON(text []byte) error {
	return i.UnmarshalText(text)
}

const (
	_RollingKindMin = -1

	RollingKindNone RollingKind = iota
	RollingKindDaily
	RollingKindHourly
	RollingKindMinutely
	RollingKindSize

	_RollingKindMax
)

var _ json.Unmarshaler = (*RollingKind)(nil)
var _ encoding.TextUnmarshaler = (*RollingKind)(nil)

type RollingFile struct {
	lock sync.RWMutex

	kind    RollingKind
	maxsize int64

	backups                 int
	reopencount             int
	ensureBackupsTaskCancel context.CancelFunc

	compressTaskPool  *cube.TaskPool
	useSlowCompress   bool
	slowCompressLimit int64
	slowCompressDir   string

	dir        string
	filepath   string
	namePrefix string
	nameSuffix string

	calcAt     time.Time
	nextRollAt int64

	filesize int64

	open    OpenWriterFnc
	fs      IFile
	buf     *bufio.Writer
	bufsize int
	gzipw   *gzip.Writer
}

func (r *RollingFile) fmt(t time.Time) string {
	switch r.kind {
	case RollingKindDaily:
		Y, _M, D := t.Date()
		M := int(_M)
		return fmt.Sprintf("%d%02d-%02d", Y, M, D)
	case RollingKindHourly:
		Y, _M, D := t.Date()
		H, _, _ := t.Clock()
		M := int(_M)
		return fmt.Sprintf("%d%02d-%02d%02d", Y, M, D, H)
	case RollingKindMinutely:
		Y, _M, D := t.Date()
		H, Min, _ := t.Clock()
		M := int(_M)
		return fmt.Sprintf("%d%02d-%02d%02d%02d", Y, M, D, H, Min)
	case RollingKindSize:
		return t.Format("20060102-150405.000")
	default:
		panic(ErrInvalidRollingKind)
	}
}

func (r *RollingFile) save() error {
	if r.gzipw != nil {
		if err := r.gzipw.Flush(); err != nil {
			return err
		}
	}
	if r.buf != nil {
		if err := r.buf.Flush(); err != nil {
			return err
		}
	}
	return r.fs.Sync()
}

func (r *RollingFile) rename(t time.Time) error {
	if err := r.save(); err != nil {
		return err
	}
	if err := r.fs.Close(); err != nil {
		return err
	}

	term := r.fmt(t)
	filename := filepath.Join(r.dir, fmt.Sprintf("%s.%s%s", r.namePrefix, term, r.nameSuffix))
	if _, err := os.Stat(filename); err == nil {
		filename = filepath.Join(r.dir, fmt.Sprintf("%s.%s(%s)%s", r.namePrefix, term, uuid.New().String(), r.nameSuffix))
	}
	if err := os.Rename(r.filepath, filename); err != nil {
		return err
	}
	if r.compressTaskPool != nil {
		if err := r.compressTaskPool.AddFunc(func(ctx context.Context) {
			docompress(ctx, r.slowCompressDir, filename, r.useSlowCompress, r.slowCompressLimit)
		}); err != nil {
			fmt.Fprintf(
				os.Stderr,
				"rrscpkgs.rolling: add compress task failed, %s, %s\n",
				filename, err,
			)
		}
	}
	return nil
}

func (r *RollingFile) reopen() error {
	defer func() {
		if r.backups < 1 {
			return
		}
		r.reopencount++
		if r.reopencount >= r.backups {
			r.reopencount = 0
			r.ensureBackups()
		}
	}()

	fs, err := r.open(r.filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	r.fs = fs
	r.buf.Reset(fs)
	if r.gzipw != nil {
		r.gzipw.Reset(r.buf)
	}
	return nil
}

func (r *RollingFile) rollByTime(now time.Time) error {
	if err := r.rename(r.calcAt); err != nil {
		return err
	}
	r.calcAt = now
	r.nextRollAt = r.nextrollat(now)
	return r.reopen()
}

type _SlowWriter struct {
	ctx      context.Context
	w        io.Writer
	duration time.Duration
	size     int64
	wc       int64
}

func (s *_SlowWriter) Write(p []byte) (n int, err error) {
	if err := s.ctx.Err(); err != nil {
		return 0, err
	}

	n, err = s.w.Write(p)
	if err != nil {
		return n, err
	}
	s.wc += int64(n)
	if s.wc >= s.size {
		tick := time.NewTimer(s.duration)
		defer tick.Stop()

		select {
		case <-tick.C:
			{
			}
		case <-s.ctx.Done():
			{
				return 0, s.ctx.Err()
			}
		}
		s.wc = 0
	}
	return n, nil
}

func NewSlowWriter(ctx context.Context, w io.Writer, sizeinbytes int64, duration time.Duration) io.Writer {
	return &_SlowWriter{
		ctx:      ctx,
		w:        w,
		duration: duration,
		size:     sizeinbytes,
	}
}

func docompress(ctx context.Context, temp, prevfn string, usesloww bool, limit int64) {
	srcf, err := os.Open(prevfn)
	if err != nil {
		return
	}
	defer srcf.Close() // nolint:errcheck

	targetfp := filepath.Join(temp, uuid.NewString())
	tf, err := os.OpenFile(targetfp, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		_ = srcf.Close()
		return
	}
	defer func() {
		_ = tf.Close()
		_ = os.Remove(targetfp)
	}()

	bufw := bufio.NewWriter(tf)
	cw := gzip.NewWriter(bufw)
	defer cw.Close() // nolint:errcheck

	var w io.Writer = cw
	if usesloww {
		w = NewSlowWriter(ctx, cw, limit, time.Second)
	}
	if _, err := io.Copy(w, srcf); err != nil {
		return
	}

	_ = bufw.Flush()
	_ = cw.Close()

	if err := os.Rename(targetfp, fmt.Sprintf("%s.gz", prevfn)); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"rrscpkgs.rolling: rename compress log failed, %s",
			err,
		)
	} else {
		_ = os.Remove(prevfn)
	}
}

func (r *RollingFile) rollBySize() error {
	if r.maxsize <= 0 {
		return nil
	}
	if r.filesize < r.maxsize {
		return nil
	}
	if err := r.rename(time.Now()); err != nil {
		return err
	}
	r.filesize = 0
	return r.reopen()
}

func (r *RollingFile) Close() error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.fs == nil {
		return nil
	}

	if r.ensureBackupsTaskCancel != nil {
		r.ensureBackupsTaskCancel()
	}

	if r.compressTaskPool != nil {
		r.compressTaskPool.Close(false)
	}

	saveerr := r.save()
	if err := r.fs.Close(); err != nil {
		if saveerr != nil {
			return errors.Join(err, saveerr)
		}
		return err
	}
	r.fs = nil
	r.buf = nil
	r.gzipw = nil
	return saveerr
}

func (r *RollingFile) nextrollat(now time.Time) int64 {
	switch r.kind {
	case RollingKindDaily:
		return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location()).Unix()
	case RollingKindHourly:
		return time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+1, 0, 0, 0, now.Location()).Unix()
	case RollingKindMinutely:
		return now.Truncate(time.Minute).Add(time.Minute).Unix()
	default:
		return 0
	}
}

func (r *RollingFile) prevrollat(now time.Time) int64 {
	switch r.kind {
	case RollingKindDaily:
		return time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location()).Unix()
	case RollingKindHourly:
		return time.Date(now.Year(), now.Month(), now.Day(), now.Hour()-1, 0, 0, 0, now.Location()).Unix()
	case RollingKindMinutely:
		return now.Truncate(time.Minute).Unix()
	default:
		return 0
	}
}

func (r *RollingFile) Write(p []byte) (n int, err error) {
	now := time.Now()
	nowunix := now.Unix()
	rollbytime := r.kind != RollingKindSize

	r.lock.Lock()
	defer r.lock.Unlock()

	if r.fs == nil {
		return 0, io.ErrClosedPipe
	}

	if rollbytime && nowunix >= r.nextRollAt {
		if err := r.rollByTime(now); err != nil {
			return n, err
		}
	}

	var w io.Writer = r.buf
	if r.gzipw != nil {
		w = r.gzipw
	}

	n, err = w.Write(p)
	if err != nil {
		return n, err
	}

	if !rollbytime {
		r.filesize += int64(len(p))
		if err := r.rollBySize(); err != nil {
			return n, err
		}
	}
	return n, nil
}

var _ io.WriteCloser = (*RollingFile)(nil)

type SlowCompressOptions struct {
	Workers        int    `json:"workers" toml:"workers"`
	BytesPerSecond int64  `json:"limit" toml:"limit"`
	TempDir        string `json:"temp" toml:"temp"`
}

type CompressOptions struct {
	Enable   bool                 `json:"enable" toml:"enable"`
	Directly bool                 `json:"directly" toml:"directly"`
	Slow     *SlowCompressOptions `json:"slow" toml:"slow"`
}

type IFile interface {
	io.WriteCloser
	Sync() error
	Stat() (os.FileInfo, error)
}

type OpenWriterFnc func(string, int, os.FileMode) (IFile, error)

type RollingOptions struct {
	Kind       RollingKind      `json:"kind" toml:"rolling_kind"`
	FileSize   int64            `json:"size" toml:"size"`
	Backups    int              `json:"backups" toml:"backups"`
	Compress   *CompressOptions `json:"compress" toml:"compress"`
	BufferSize int              `json:"buffer_size" toml:"buffer_size"`
	OpenFile   OpenWriterFnc    `json:"-" toml:"-"`
}

var (
	ErrInvalidRollingSize     = errors.New("rrscpkgs.logx: rolling size must be greater than 0")
	ErrInvalidCompressWorkers = errors.New("rrscpkgs.logx: compress workers must be greater than 0")
)

func NewRollingFile(fp string, opts *RollingOptions) (*RollingFile, error) {
	if opts.Kind == RollingKindNone {
		return nil, ErrInvalidRollingKind
	}

	if opts.Compress != nil && opts.Compress.Enable && opts.Compress.Directly {
		if !strings.HasSuffix(fp, ".gz") {
			fp += ".gz"
		}
	}

	absfp, err := filepath.Abs(fp)
	if err != nil {
		return nil, err
	}
	fp = absfp

	dir := filepath.Dir(fp)
	base := filepath.Base(fp)
	nameSuffix := filepath.Ext(fp)
	namePrefix := base[:len(base)-len(nameSuffix)]

	fs, err := opts.OpenFile(fp, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	filesize := int64(0)

	if opts.Kind == RollingKindSize {
		if opts.FileSize <= 0 {
			return nil, ErrInvalidRollingSize
		}

		stat, err := fs.Stat()
		if err != nil {
			return nil, err
		}
		filesize = stat.Size()
	}

	rf := &RollingFile{
		kind: opts.Kind,

		filepath:   fp,
		dir:        dir,
		namePrefix: namePrefix,
		nameSuffix: nameSuffix,

		open:    opts.OpenFile,
		fs:      fs,
		buf:     bufio.NewWriterSize(fs, opts.BufferSize),
		bufsize: opts.BufferSize,

		filesize: filesize,

		backups: opts.Backups,
	}

	if opts.Kind == RollingKindSize {
		rf.maxsize = opts.FileSize
	} else {
		rf.calcAt = time.Now()
		rf.nextRollAt = rf.nextrollat(rf.calcAt)
		stat, err := fs.Stat()
		if err != nil {
			return nil, err
		}
		prevTermEnd := rf.prevrollat(rf.calcAt)
		if stat.ModTime().Unix() < prevTermEnd {
			rf.rename(stat.ModTime()) //nolint:errcheck
			rf.reopen()               //nolint:errcheck
		}
	}

	if opts.Compress != nil && opts.Compress.Enable {
		if opts.Compress.Directly {
			rf.gzipw = gzip.NewWriter(rf.buf)
		} else {
			slowopts := opts.Compress.Slow
			if slowopts == nil {
				slowopts = &SlowCompressOptions{}
			}
			if slowopts.Workers < 1 {
				slowopts.Workers = 2
			}
			if slowopts.BytesPerSecond < 0 {
				rf.useSlowCompress = false
			} else {
				rf.useSlowCompress = true
				rf.slowCompressLimit = max(slowopts.BytesPerSecond, 1024*1024*64)
			}
			if slowopts.TempDir == "" {
				slowopts.TempDir = filepath.Join(dir, ".temp")
			}
			err := os.MkdirAll(slowopts.TempDir, 0o0755)
			if err != nil {
				return nil, fmt.Errorf("rrscpkgs.logx.rolling: create compress temp dir failed, %s, %s", dir, err)
			}
			rf.slowCompressDir = slowopts.TempDir

			rf.compressTaskPool = cube.NewTaskPool(&cube.TaskPoolOptions{
				Workers: slowopts.Workers,
				OnPanic: func(ctx context.Context, _ cube.ITaskItem, err any) {
					fmt.Fprintf(
						os.Stderr,
						"rrscpkgs.logx: compress task panic: %s, %v\n",
						fp, err,
					)
				},
			})
		}
	}

	if rf.backups > 0 {
		rf.ensureBackups()
	}
	return rf, nil
}

func (r *RollingFile) ensureBackups() {
	if r.ensureBackupsTaskCancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.ensureBackupsTaskCancel = cancel
	cube.Fly(func() {
		defer func() {
			r.lock.Lock()
			defer r.lock.Unlock()

			r.ensureBackupsTaskCancel = nil
		}()
		r.doEnsureBackups(ctx, r.backups)
	})
}

func (r *RollingFile) doEnsureBackups(ctx context.Context, count int) {
	defer func() {
		if rv := recover(); rv != nil {
			fmt.Fprintf(
				os.Stderr,
				"rrscpkgs.rolling: ensure backups panic, %s, %v\n",
				r.filepath, rv,
			)
		}
	}()

	stream, err := cube.ReadDirStream(r.dir)
	if err != nil {
		return
	}

	type _FileInfoWithMtime struct {
		Info  os.DirEntry
		Mtime time.Time
	}

	cwname := filepath.Base(r.filepath)
	var files = make([]_FileInfoWithMtime, 0, 32)
	for item, err := range stream {
		if err != nil || ctx.Err() != nil {
			break
		}
		if item.IsDir() {
			continue
		}
		name := item.Name()
		if name == cwname {
			continue
		}
		if !strings.HasPrefix(name, r.namePrefix) {
			continue
		}
		if strings.HasSuffix(name, ".gz") {
			unzipname := name[:len(name)-len(".gz")]
			if !strings.HasSuffix(unzipname, r.nameSuffix) {
				continue
			}
		} else {
			if !strings.HasSuffix(name, r.nameSuffix) {
				continue
			}
		}
		iteminfo, err := item.Info()
		if err != nil {
			continue
		}

		files = append(files, _FileInfoWithMtime{
			Info:  item,
			Mtime: iteminfo.ModTime(),
		})
	}
	if len(files) <= count {
		return
	}

	sort.Slice(files, func(i, j int) bool {
		return (&files[i]).Mtime.Before((&files[j]).Mtime)
	})

	for i := range len(files) - count {
		os.Remove(filepath.Join(r.dir, files[i].Info.Name())) //nolint:errcheck
	}
}
