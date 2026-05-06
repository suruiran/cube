package cube

import "log/slog"

func fly_internal(fnc func(), rethrow bool, loop bool) {
	go func() {
		for {
			func() {
				defer func() {
					if rv := recover(); rv != nil {
						slog.Error(
							"cube.fly: panic",
							slog.String("func", FuncName(fnc)),
							slog.Any("panic", rv),
							slog.String("stacktrace", ReadStack(3, 20)),
						)
						if rethrow {
							panic(rv)
						}
					}
				}()
				fnc()
			}()
			if !loop {
				break
			}
		}
	}()
}

func Fly(fnc func()) {
	fly_internal(fnc, false, false)
}

func FlyRethrow(fnc func()) {
	fly_internal(fnc, true, false)
}

// Never lands. 😀
func FlyAsSwallow(fnc func()) {
	fly_internal(fnc, false, true)
}
