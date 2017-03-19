package vhc

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

var src = rand.NewSource(time.Now().UnixNano())

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
	max := uint64(100000)
	vhll, _ := New()
	r := rand.NewZipf(rand.New(rand.NewSource(0)), 1.1, 1.0, max)
	dict := map[string]uint64{}

	for uint64(len(dict)) < max {
		id := fmt.Sprintf("flow-%d", r.Uint64())
		vhll.Add([]byte(id))
		dict[id]++
	}

	for i := uint64(0); i < 100; i++ {
		id := fmt.Sprintf("flow-%d", i)
		res := vhll.Count([]byte(id))
		exact := dict[string(id)]
		ratio := 100 * estimateError(res, exact)
		if ratio > 3 {
			t.Errorf("%d) VHLL Exact %d, got %d which is %.2f%% error", i, exact, res, ratio)
		}
	}
}
