package cube

import (
	urand "crypto/rand"
	"io"
	"math/rand/v2"
)

var (
	AsciiChars      = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	LowerAsciiChars = []byte("abcdefghijklmnopqrstuvwxyz0123456789")
)

// RandChoices
// pool: 1 < len(pool)
func RandChoices[T any](pool []T, length int) []T {
	var result = make([]T, length)
	var pl = uint(len(pool))
	for i := range length {
		result[i] = pool[rand.UintN(pl)]
	}
	return result
}

// RandChoicesCrypto
// pool: 1 < len(pool) <= 256
func RandChoicesCrypto[T any](pool []T, length int) ([]T, error) {
	var result = make([]T, length)

	tmplen := min(length+min(length, 128), 4096)
	var tmp = make([]byte, tmplen)

	_, err := io.ReadFull(urand.Reader, tmp)
	if err != nil {
		return nil, err
	}
	var pl = len(pool)
	var maxv = 256 - 256%pl

	var tmpidx = 0
	for i := range length {
		for {
			if tmpidx >= tmplen {
				_, err = io.ReadFull(urand.Reader, tmp)
				if err != nil {
					return nil, err
				}
				tmpidx = 0
			}

			var ele = int(tmp[tmpidx])
			tmpidx++
			if ele < maxv {
				result[i] = pool[ele%pl]
				break
			}
		}
	}
	return result, nil
}

// RandBytesCrypto
// pool: 1 < len(pool) <= 256
func RandBytesCrypto(pool []byte, length int) ([]byte, error) {
	tmplen := min(length, 128)
	var result = make([]byte, length+tmplen)
	_, err := io.ReadFull(urand.Reader, result)
	if err != nil {
		return nil, err
	}
	var pl = len(pool)
	var maxv = 256 - 256%pl

	var tmp = result[length:]
	var tmpidx = 0
	for i := range length {
		var ele = int(result[i])
		if ele >= maxv {
			for {
				if tmpidx >= tmplen {
					_, err = io.ReadFull(urand.Reader, tmp)
					tmpidx = 0
					if err != nil {
						return nil, err
					}
					continue
				}
				ele = int(tmp[tmpidx])
				tmpidx++
				if ele < maxv {
					break
				}
			}
		}
		result[i] = pool[ele%pl]
	}
	return result[:length], nil
}
