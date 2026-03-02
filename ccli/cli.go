package ccli

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

type CliReader struct {
	buf *bufio.Reader
}

func NewCliReader() *CliReader {
	return &CliReader{buf: bufio.NewReaderSize(os.Stdin, 1)}
}

func (cr *CliReader) readline() string {
	buf := bytes.NewBuffer(nil)

	for {
		tmp, isprefix, err := cr.buf.ReadLine()
		if err != nil {
			panic(err)
		}
		buf.Write(tmp)
		if isprefix {
			continue
		}
		break
	}

	return strings.TrimSpace(buf.String())
}

func (cr *CliReader) read(typ string, name string) string {
	fmt.Printf("%s <%s>: ", typ, name)
	return cr.readline()
}

func (cr *CliReader) String(name string) string {
	return cr.read("String", name)
}

type ReadStringOpts struct {
	Enums  []string
	Regexp *regexp.Regexp
	MinLen *int
	MaxLen *int
}

var (
	gumavail bool
)

func init() {
	_, err := exec.LookPath("gum")
	gumavail = err == nil
}

func gumfilter(items []string) string {
	args := []string{"filter"}
	args = append(args, items...)
	cmd := exec.Command("gum", args...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	choose := strings.TrimSpace(string(out))
	if choose == "" {
		panic(fmt.Errorf("user canceled"))
	}
	return choose
}

func (cr *CliReader) StringWithOpts(name string, opts *ReadStringOpts) string {
	if opts != nil && len(opts.Enums) > 0 {
		if gumavail {
			return gumfilter(opts.Enums)
		}
		slices.Sort(opts.Enums)
		fmt.Println("\n Enums")
		buf := bytes.NewBuffer(nil)
		for i, v := range opts.Enums {
			fmt.Fprintf(buf, "\t [%d: %s]\n", i, v)
		}
		fmt.Print(buf.String())
	}

	txt := cr.String(name)
	if opts == nil {
		return txt
	}
	if len(opts.Enums) > 0 {
		idx, err := strconv.ParseInt(txt, 10, 64)
		if err == nil && idx > -1 && idx < int64(len(opts.Enums)) {
			txt = opts.Enums[idx]
		}
	}

	if opts.MaxLen != nil && len(txt) > *opts.MaxLen {
		panic(fmt.Errorf("len > max(%d)", len(txt)))
	}

	if opts.MinLen != nil && len(txt) < *opts.MinLen {
		panic(fmt.Errorf("len < min(%d)", len(txt)))
	}

	idx := slices.Index(opts.Enums, txt)
	if idx < 0 {
		panic(fmt.Errorf("%s not in enums", txt))
	}

	if opts.Regexp != nil && !opts.Regexp.MatchString(txt) {
		panic(fmt.Errorf("txt not match regexp"))
	}
	return txt
}

type ReadIntOpts struct {
	Min *int64
	Max *int64
}

func (cr *CliReader) Int(name string) int64 {
	line := cr.read("Int", name)
	num, err := strconv.ParseInt(line, 10, 64)
	if err != nil {
		panic(err)
	}
	return num
}

func (cr *CliReader) IntWithOpts(name string, opts *ReadIntOpts) int64 {
	num := cr.Int(name)
	if opts == nil {
		return num
	}
	if opts.Max != nil && num > *opts.Max {
		panic(fmt.Errorf("%d > max(%d)", num, *opts.Max))
	}
	if opts.Min != nil && num < *opts.Min {
		panic(fmt.Errorf("%d < min(%d)", num, *opts.Min))
	}
	return num
}

func (cr *CliReader) Bool(name string) bool {
	line := cr.read("Bool", name)

	switch line {
	case "y", "Y", "yes", "YES":
		{
			return true
		}
	case "", "n", "no", "N", "NO":
		{
			return false
		}
	}

	bv, err := strconv.ParseBool(line)
	if err != nil {
		panic(err)
	}
	return bv
}

type MultiLinesEnding struct {
	Txt   string
	Count int
}

func (cr *CliReader) MultiLines(name string, ending *MultiLinesEnding) string {
	fmt.Printf("MultiLine <%s>:\n", name)

	if ending == nil {
		ending = &MultiLinesEnding{
			Txt:   "",
			Count: 2,
		}
	}
	if ending.Count < 1 {
		ending.Count = 1
	}

	var lines []string
	ec := 0
	for {
		line := cr.readline()
		if line == ending.Txt {
			ec++
			if ec == ending.Count {
				break
			}
		} else {
			ec = 0
		}
		lines = append(lines, line)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
