package rbc

import (
	"database/sql"
	"sync"
	"time"
)

var (
	builtinPreheatOnce sync.Once
)

func registerBuiltins() {
	builtinPreheatOnce.Do(func() {
		RegisterType[int]()
		RegisterType[byte]()
		RegisterType[int8]()
		RegisterType[int16]()
		RegisterType[rune]()
		RegisterType[int32]()
		RegisterType[int64]()

		RegisterType[uint]()
		RegisterType[uint8]()
		RegisterType[uint16]()
		RegisterType[uint32]()
		RegisterType[uint64]()

		RegisterType[float32]()
		RegisterType[float64]()

		RegisterType[complex64]()
		RegisterType[complex128]()

		RegisterType[bool]()
		RegisterType[string]()

		RegisterType[[]byte]()
		RegisterType[sql.RawBytes]()

		RegisterType[time.Time]()
		RegisterType[time.Duration]()

		RegisterType[sql.NullBool]()
		RegisterType[sql.NullString]()
		RegisterType[sql.NullByte]()
		RegisterType[sql.NullInt16]()
		RegisterType[sql.NullInt32]()
		RegisterType[sql.NullInt64]()
		RegisterType[sql.NullFloat64]()
		RegisterType[sql.NullTime]()
	})
}
