package cube

import (
	"math/rand/v2"
)

func RandBytes(pool []byte, length int) []byte {
	var result = make([]byte, length)
	var pl = uint(len(pool))
	for i := range length {
		result[i] = pool[rand.UintN(pl)]
	}
	return result
}

var (
	asciipool      = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	lowerasciipool = []byte("abcdefghijklmnopqrstuvwxyz0123456789")
)

func RandAsciiBytes(length int) []byte {
	return RandBytes(asciipool, length)
}

func RandLowerAsciiBytes(length int) []byte {
	return RandBytes(lowerasciipool, length)
}
