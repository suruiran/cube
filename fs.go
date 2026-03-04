package cube

import (
	"errors"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
)

func ReadDirStream(dir string) (iter.Seq2[os.FileInfo, error], error) {
	d, err := os.Lstat(dir)
	if err != nil {
		return nil, err
	}
	if !d.IsDir() {
		return nil, fmt.Errorf("cube.fs.ReadDirStream: %s is not a directory", dir)
	}

	return func(yield func(os.FileInfo, error) bool) {
		fh, err := os.Open(dir)
		if err != nil {
			yield(nil, err)
			return
		}
		defer fh.Close() //nolint:errcheck

		for {
			files, err := fh.Readdir(100)
			if err != nil {
				if err == io.EOF {
					return
				}
				yield(nil, err)
				return
			}
			if len(files) == 0 {
				return
			}
			for _, file := range files {
				if !yield(file, nil) {
					return
				}
			}
		}
	}, nil
}

type WalkOptions struct {
	MaxDepth       int
	MatchPatterns  []string
	IgnorePatterns []string
	OnlyTestName   bool
	FollowLink     bool
	OnSlightError  func(error)
}

func (opts *WalkOptions) match(pattern, filename, filefullpath string) bool {
	matched, err := doublestar.Match(pattern, filename)
	if err == nil && matched {
		return true
	}
	if opts.OnlyTestName {
		return false
	}
	matched, err = doublestar.Match(pattern, filefullpath)
	if err != nil {
		opts.OnSlightError(err)
	}
	return matched
}

type FileInfoWithDir struct {
	os.FileInfo
	Dir string
}

func (fi *FileInfoWithDir) Fullpath() string {
	return filepath.Join(fi.Dir, fi.Name())
}

func (fi *FileInfoWithDir) String() string {
	return fmt.Sprintf("FileInfo(%s)", fi.Fullpath())
}

var (
	errEmptyMatchPatterns = errors.New("cube.fs.WalkStream: empty match patterns")
)

func WalkStream(root string, opts *WalkOptions) (iter.Seq2[*FileInfoWithDir, error], error) {
	if len(opts.MatchPatterns) < 1 {
		return nil, errEmptyMatchPatterns
	}
	if opts.OnSlightError == nil {
		opts.OnSlightError = func(err error) {
			slog.Warn(
				"cube.fs.WalkStream: slight error",
				slog.String("root", root), slog.String("error", err.Error()),
			)
		}
	}
	fullpath, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	var markers = make(Set[string])
	return func(yield func(*FileInfoWithDir, error) bool) {
		doWalkStream(fullpath, markers, 0, yield, opts)
	}, nil
}

func doWalkStream(dir string, markers Set[string], currentDepth int, yield func(*FileInfoWithDir, error) bool, opts *WalkOptions) {
	if currentDepth > opts.MaxDepth {
		return
	}
	if _, visited := markers[dir]; visited {
		return
	}
	markers[dir] = struct{}{}

	seq, err := ReadDirStream(dir)
	if err != nil {
		yield(nil, err)
		return
	}

loop:
	for item, err := range seq {
		if err != nil {
			yield(nil, err)
			return
		}

		filename := item.Name()
		fullpath := filepath.Join(dir, filename)
		for _, ip := range opts.IgnorePatterns {
			if opts.match(ip, filename, fullpath) {
				continue loop
			}
		}

	matchloop:
		for _, mp := range opts.MatchPatterns {
			if opts.match(mp, filename, fullpath) {
				if !yield(&FileInfoWithDir{FileInfo: item, Dir: dir}, nil) {
					return
				}
				break matchloop
			}
		}

		if opts.FollowLink && item.Mode()&os.ModeSymlink != 0 {
			var rlerr error
			var realpath string
			if realpath, rlerr = os.Readlink(fullpath); rlerr == nil {
				if !filepath.IsAbs(realpath) {
					realpath = filepath.Join(dir, realpath)
				}
				var linkstat os.FileInfo
				if linkstat, rlerr = os.Stat(realpath); rlerr == nil && linkstat.IsDir() {
					doWalkStream(realpath, markers, currentDepth+1, yield, opts)
				}
			}
			if rlerr != nil {
				opts.OnSlightError(rlerr)
			}
			continue
		}

		if item.IsDir() {
			doWalkStream(fullpath, markers, currentDepth+1, yield, opts)
			continue
		}
	}
}
