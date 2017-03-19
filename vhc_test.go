package vhc

import (
	"math/rand"
	"testing"
	"time"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var src = rand.NewSource(time.Now().UnixNano())

func RandStringBytesMaskImprSrc(n uint32) string {
	b := make([]byte, n)
	for i := uint32(0); i < n; i++ {
		b[i] = letterBytes[rand.Int()%len(letterBytes)]
	}
	return string(b)
}

func estimateError(got, exp uint64) float64 {
	var delta uint64
	if got > exp {
		delta = got - exp
	} else {
		delta = exp - got
	}
	return float64(delta) / float64(exp)
}

func TestVHLL(t *testing.T) {
	max := uint64(1000000)
	vhll, _ := New()
	r := rand.NewZipf(rand.New(rand.NewSource(0)), 1.1, 1.1, max)
	dict := map[string]uint64{}

	for i := 0; len(dict) < 100; i++ {
		dict[RandStringBytesMaskImprSrc(10)] = max - r.Uint64()
	}

	for k, v := range dict {
		for i := 0; i < int(v); i++ {
			vhll.Add([]byte(k))
		}
	}

	i := 0
	for k, exact := range dict {
		i++
		res := uint64(vhll.Count([]byte(k)))
		ratio := 100 * estimateError(res, exact)

		if ratio > 5 {
			t.Errorf("%d) VHLL Exact %d, got %d which is %.2f%% error", i, exact, res, ratio)
		}

	}
}
