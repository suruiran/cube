package cube

import "log/slog"

func fly_internal(fnc func(), rethrow bool) {
	go func() {
		defer func() {
			if rv := recover(); rv != nil {
				slog.Error(
					"cube.fly: panic",
					slog.String("func", FuncName(fnc)),
					slog.Any("panic", rv),
					slog.String("stacktrace", ReadStack(2, 20)),
				)
				if rethrow {
					panic(rv)
				}
			}
		}()
		fnc()
	}()
}

func Fly(fnc func()) {
	fly_internal(fnc, false)
}

func FlyRethrow(fnc func()) {
	fly_internal(fnc, true)
}
