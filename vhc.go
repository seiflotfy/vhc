package vhc

import (
	"errors"
	"math"

	"math/rand"

	metro "github.com/dgryski/go-metro"
)

func zeros(registers []uint8) (z float64) {
	for _, val := range registers {
		if val == 0 {
			z++
		}
	}
	return z
}

func beta(ez float64) float64 {
	zl := math.Log(ez + 1)
	return -0.370393911*ez +
		0.070471823*zl +
		0.17393686*math.Pow(zl, 2) +
		0.16339839*math.Pow(zl, 3) +
		-0.09237745*math.Pow(zl, 4) +
		0.03738027*math.Pow(zl, 5) +
		-0.005384159*math.Pow(zl, 6) +
		0.00042419*math.Pow(zl, 7)
}

// Calculate the position of the leftmost 1-bit.
func rho(val uint64, max uint8) (r uint8) {
	for val&0x8000000000000000 == 0 && r < max {
		val <<= 1
		r++
	}
	return r + 1
}

func alpha(m float64) float64 {
	switch m {
	case 16:
		return 0.673
	case 32:
		return 0.697
	case 64:
		return 0.709
	}
	return 0.7213 / (1 + 1.079/m)
}

func round(f float64) float64 {
	return math.Ceil(f - 0.5)
}

func totalFillRate(registers []uint8) float64 {
	return 1 - zeros(registers)/float64(len(registers))
}

// VLogLogBeta is a sketch for cardinality estimation based on LogLog counting
type VLogLogBeta struct {
	registers []uint8
	m         uint64
	s         uint64
	p         uint8
	vp        uint8
	alpha     float64
	mAlpha    float64
	n         uint64
	skip      uint64
}

// NewCounter returns a VLogLogBeta sketch with 2^precision registers, where
// precision must be between 4 and 16
func NewCounter() (*VLogLogBeta, error) {
	p := uint8(20)
	vp := uint8(9)
	m := uint64(1 << p)
	s := uint64(1 << vp)
	return &VLogLogBeta{
		m:         m,
		s:         s,
		p:         p,
		vp:        vp,
		registers: make([]uint8, m),
		alpha:     alpha(float64(m)),
		mAlpha:    alpha(float64(s)),
		n:         rand.Uint64(),
		skip:      1 + rand.Uint64()*2,
	}, nil
}

func (llb *VLogLogBeta) c(f []byte, i uint64) uint64 {
	return metro.Hash64(f, i) % llb.m
}

// Add inserts a value into the sketch
func (llb *VLogLogBeta) Add(f []byte) {
	q := llb.n
	i := q % llb.s
	cfi := llb.c(f, i)

	q1 := rand.Uint64()
	lambda := rho(q1, 32)

	if llb.registers[cfi] < lambda {
		llb.registers[cfi] = lambda
	}
	llb.n += llb.skip
}

// Count returns count in stream
func (llb *VLogLogBeta) Count(f []byte) uint64 {
	cf := make([]uint8, llb.s, llb.s)
	for i := range cf {
		cfi := llb.c(f, uint64(i))
		cf[i] = llb.registers[cfi]
	}

	sum := 0.0
	s := float64(llb.s)
	m := float64(llb.m)
	ez := 0.0

	for _, val := range cf {
		sum += 1.0 / math.Pow(2.0, float64(val))
		if val == 0 {
			ez++
		}
	}

	beta := beta(ez)
	ns := (llb.mAlpha * s * (s - ez) / (beta + sum))

	// estimate error
	N := float64(llb.totalCardinality())
	e := s * N / m
	n := ns - e

	// rounding
	return uint64(n + 0.5)
}

// Merge takes another VLogLogBeta and combines it with llb one, making llb the union of both.
func (llb *VLogLogBeta) Merge(other *VLogLogBeta) error {
	if llb.p != llb.p {
		return errors.New("precisions must be equal")
	}
	for i, v := range llb.registers {
		if v < other.registers[i] {
			llb.registers[i] = other.registers[i]
		}
	}
	return nil
}

func (llb *VLogLogBeta) totalCardinality() uint64 {
	sum := 0.0
	m := float64(llb.m)
	for _, val := range llb.registers {
		sum += 1.0 / math.Pow(2.0, float64(val))
	}
	ez := zeros(llb.registers)
	beta := beta(ez)
	return uint64(llb.alpha * m * (m - ez) / (beta + sum))
}
