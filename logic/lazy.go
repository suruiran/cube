package logic

type LazyBool struct {
	fn       func() bool
	recovery func(val any) bool
}

func (tb *LazyBool) Bool() (tv bool) {
	defer func() {
		if r := recover(); r != nil {
			if tb.recovery != nil {
				tv = tb.recovery(r)
			} else {
				tv = false
			}
		}
	}()
	tv = tb.fn()
	return
}

func Lazy(fn func() bool) *LazyBool { return &LazyBool{fn: fn} }

func LazyWithRecovery(fn func() bool, recovery func(val any) bool) *LazyBool {
	return &LazyBool{fn: fn, recovery: recovery}
}
