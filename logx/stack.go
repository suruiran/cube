package logx

import (
	"fmt"
	"log/slog"
	"uuid"

	"github.com/suruiran/cube"
)

func WarnStack(label string, opts *StacktraceOptions) {
	if opts == nil {
		opts = &StacktraceOptions{
			Skip: 2,
			Size: 20,
		}
	}
	if opts.Size < 1 {
		opts.Size = 20
	}

	if label == "" {
		label = uuid.NewV7().String()
	}

	lines := cube.ReadStackLines(opts.Skip, opts.Size)

	sepline := fmt.Sprintf(">>>>>>>>>>>>>>>>>>>> %s", label)
	slog.Warn(sepline)
	for _, v := range lines {
		slog.Warn(fmt.Sprintf(">>>> %s %s", label, v))
	}
	slog.Warn(sepline)
}
