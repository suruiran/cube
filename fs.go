package cube

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"path"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
)

func ReadDirStream(dir string) (iter.Seq2[os.DirEntry, error], error) {
	d, err := os.Lstat(dir)
	if err != nil {
		return nil, err
	}
	if !d.IsDir() {
		return nil, fmt.Errorf("cube.fs.ReadDirStream: %s is not a directory", dir)
	}

	return func(yield func(os.DirEntry, error) bool) {
		fh, err := os.Open(dir)
		if err != nil {
			yield(nil, err)
			return
		}
		defer fh.Close() //nolint:errcheck

		for {
			files, err := fh.ReadDir(100)
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
}

var (
	CommonIgnorePatterns = []string{
		// vsc
		"**/.git",

		// language
		"**/node_modules",
		"**/__pycache__",
		"**/target",

		// ide
		"**/.vscode",
		"**/.idea",

		// fs
		"**/.DS_Store",
	}
)

func (opts *WalkOptions) match(pattern, filename, filefullpath string) (bool, error) {
	matched, err := doublestar.Match(pattern, filename)
	if err == nil && matched {
		return true, nil
	}
	if opts.OnlyTestName {
		return false, nil
	}
	matched, err = doublestar.Match(pattern, filefullpath)
	if err != nil {
		return false, err
	}
	return matched, nil
}

type FileInfoWithDir struct {
	os.DirEntry
	Dir string
}

func (fi *FileInfoWithDir) Fullpath() string {
	return filepath.Join(fi.Dir, fi.Name())
}

func (fi *FileInfoWithDir) String() string {
	return fmt.Sprintf("FileInfo(%s)", fi.Fullpath())
}

var (
	errEmptyMatchPatterns = errors.New("cube.fs.ScanStream: empty match patterns")
)

func FsScanStream(ctx context.Context, root string, opts *WalkOptions) (iter.Seq2[*FileInfoWithDir, error], error) {
	if len(opts.MatchPatterns) < 1 {
		return nil, errEmptyMatchPatterns
	}
	fullpath, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	var markers = make(Set[string])
	var ic = 0
	return func(yield func(*FileInfoWithDir, error) bool) {
		doWalkStream(ctx, &ic, fullpath, markers, 0, yield, opts)
	}, nil
}

func doWalkStream(ctx context.Context, icptr *int, dir string, markers Set[string], currentDepth int, yield func(*FileInfoWithDir, error) bool, opts *WalkOptions) bool {
	if opts.MaxDepth > 0 && currentDepth > opts.MaxDepth {
		return true
	}
	if _, visited := markers[dir]; visited {
		return true
	}
	markers[dir] = struct{}{}

	if err := ctx.Err(); err != nil {
		yield(nil, err)
		return false
	}

	seq, err := ReadDirStream(dir)
	if err != nil {
		return yield(nil, err)
	}

	subdirs := make([]string, 0, 10)

	mdir := filepath.ToSlash(dir)

loop:
	for item, err := range seq {
		if err != nil {
			if !yield(nil, err) {
				return false
			}
			continue loop
		}

		*icptr++
		if (*icptr)&0x3F == 0 {
			if err = ctx.Err(); err != nil {
				yield(nil, err)
				return false
			}
		}

		filename := item.Name()
		fullpath := filepath.Join(dir, filename)
		mfp := path.Join(mdir, filename)

		for _, ip := range opts.IgnorePatterns {
			matched, err := opts.match(ip, filename, mfp)
			if err != nil && !yield(nil, fmt.Errorf("cube.fs.ScanStream: match error: %s, %s, %w", ip, mfp, err)) {
				return false
			}
			if matched {
				continue loop
			}
		}

	matchloop:
		for _, mp := range opts.MatchPatterns {
			matched, err := opts.match(mp, filename, mfp)
			if err != nil && !yield(nil, fmt.Errorf("cube.fs.ScanStream: match error: %s, %s, %w", mp, mfp, err)) {
				return false
			}
			if matched {
				if !yield(&FileInfoWithDir{DirEntry: item, Dir: dir}, nil) {
					return false
				}
				break matchloop
			}
		}

		if item.IsDir() {
			subdirs = append(subdirs, fullpath)
			continue
		}

		if opts.FollowLink {
			if item.Type()&os.ModeSymlink == 0 {
				continue
			}

			var rlerr error
			var realpath string
			if realpath, rlerr = filepath.EvalSymlinks(fullpath); rlerr == nil {
				var linkstat os.FileInfo
				if linkstat, rlerr = os.Stat(realpath); rlerr == nil && linkstat.IsDir() {
					subdirs = append(subdirs, realpath)
				}
			}
			if rlerr != nil && !yield(nil, rlerr) {
				return false
			}
			continue
		}
	}

	for _, subdir := range subdirs {
		if !doWalkStream(ctx, icptr, subdir, markers, currentDepth+1, yield, opts) {
			return false
		}
	}
	return true
}
