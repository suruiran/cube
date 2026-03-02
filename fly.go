package cube

import "log/slog"

func Fly(fnc func()) {
	go func() {
		defer func() {
			if rv := recover(); rv != nil {
				slog.Error(
					"rrscpkgs.fly: panic",
					slog.String("func", FuncName(fnc)),
					slog.Any("panic", rv),
					slog.String("stacktrace", ReadStack(1, 15)),
				)
			}
		}()
		fnc()
	}()
}
