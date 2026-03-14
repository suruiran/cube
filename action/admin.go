package action

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/suruiran/cube"
)

type _FsAdminChecker struct {
	secret   []byte
	fsdir    string
	header   string
	filesize int64
	wtest    sync.Once
	haswp    bool
}

func (ac *_FsAdminChecker) Do(ctx context.Context, cli *http.Client, req *http.Request) (*http.Response, error) {
	if cli == nil {
		cli = http.DefaultClient
	}
	cleanup, err := ac.make(req)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	return cli.Do(req)
}

var (
	filenameRegexp = regexp.MustCompile(`^[a-z]{12}$`)
)

func (ac *_FsAdminChecker) make(req *http.Request) (func(), error) {
	filename := ""
	fullpath := ""
	for {
		filename = string(cube.RandChoices(cube.AsciiChars, 12))
		fullpath = filepath.Join(ac.fsdir, filename)
		if _, err := os.Stat(fullpath); os.IsNotExist(err) {
			break
		}
	}

	fobj, err := os.OpenFile(fullpath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return nil, fmt.Errorf("action.FsAdminChecker: create file error: %w", err)
	}
	defer fobj.Close() //nolint:errcheck

	cleanup := func() { _ = os.Remove(fullpath) }

	buf := cube.RandChoices(cube.AsciiChars, int(ac.filesize))
	_, err = fobj.Write(buf)
	if err != nil {
		return nil, fmt.Errorf("action.FsAdminChecker: write file error: %w", err)
	}

	fnhash := hmac.New(sha256.New, ac.secret)
	fnhash.Write([]byte(filename))
	fnhash.Write([]byte(req.URL.Path))

	fchash := hmac.New(sha256.New, ac.secret)
	fchash.Write(buf)

	req.Header.Set(
		ac.header,
		fmt.Sprintf("%s:%s:%s",
			base64.StdEncoding.EncodeToString(fnhash.Sum(nil)),
			filename,
			base64.StdEncoding.EncodeToString(fchash.Sum(nil)),
		),
	)

	return cleanup, nil
}

func equalbytes(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

func (ac *_FsAdminChecker) Check(ctx context.Context, ip string, req *http.Request) error {
	ac.wtest.Do(func() {
		fullpath := filepath.Join(ac.fsdir, ".wtest")
		fobj, err := os.OpenFile(fullpath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
		if err != nil {
			return
		}
		defer fobj.Close()  //nolint:errcheck
		os.Remove(fullpath) //nolint:errcheck
		ac.haswp = true
	})

	if ac.haswp {
		return fmt.Errorf("FsAdminChecker: I has the write permission, so reject all requests")
	}

	header := req.Header.Get(ac.header)
	if header == "" {
		return fmt.Errorf("FsAdminChecker: empty header")
	}
	parts := strings.SplitN(header, ":", 3)
	if len(parts) != 3 {
		return fmt.Errorf("FsAdminChecker: bad header format")
	}
	clifnhval, filename, clifchval := parts[0], parts[1], parts[2]
	if !filenameRegexp.MatchString(filename) {
		return fmt.Errorf("FsAdminChecker: bad filename format")
	}

	// check filename hash
	clifnhvalbytes, err := base64.StdEncoding.DecodeString(clifnhval)
	if err != nil {
		return fmt.Errorf("FsAdminChecker: bad filename hash format")
	}
	fnhash := hmac.New(sha256.New, ac.secret)
	fnhash.Write([]byte(filename))
	fnhash.Write([]byte(req.URL.Path))
	if !equalbytes(fnhash.Sum(nil), clifnhvalbytes) {
		return fmt.Errorf("FsAdminChecker: filename hash not match")
	}
	clifchvalbytes, err := base64.StdEncoding.DecodeString(clifchval)
	if err != nil {
		return fmt.Errorf("FsAdminChecker: bad file content hash format")
	}

	// read file
	fullpath := filepath.Join(ac.fsdir, filename)
	fobj, err := os.Open(fullpath)
	if err != nil {
		return fmt.Errorf("FsAdminChecker: open file error: %w", err)
	}
	defer fobj.Close() //nolint:errcheck

	// check stat
	stat, err := fobj.Stat()
	if err != nil {
		return fmt.Errorf("FsAdminChecker: stat file error: %w", err)
	}
	if time.Since(stat.ModTime()) > time.Minute {
		return fmt.Errorf("FsAdminChecker: file modified time is too old, %s", filename)
	}
	if stat.Size() != ac.filesize {
		return fmt.Errorf("FsAdminChecker: file size is too large, %s", filename)
	}

	// check file content hash
	fchash := hmac.New(sha256.New, ac.secret)
	_, err = io.Copy(fchash, fobj)
	if err != nil {
		return fmt.Errorf("FsAdminChecker: read file error: %w", err)
	}
	if !equalbytes(fchash.Sum(nil), clifchvalbytes) {
		return fmt.Errorf("FsAdminChecker: file content hash not match")
	}
	return nil
}

var _ IAdminChecker = (*_FsAdminChecker)(nil)

// NewFsAdminChecker
// Help dockerized services to verify that admin requests from the host.
// It mandates a read-only mounted fsdir, rejecting all requests if write permission is detected.
func NewFsAdminChecker(secret string, fsdir string, header string, filesize int64) IAdminChecker {
	if filesize <= 0 {
		filesize = 512
	}
	if header == "" {
		header = "X-FsAdmin-Token"
	}
	return &_FsAdminChecker{
		secret:   []byte(secret),
		fsdir:    fsdir,
		header:   header,
		filesize: filesize,
	}
}
