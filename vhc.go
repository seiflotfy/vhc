package vhc

import (
	"errors"
	"math"
	"time"

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

// Sketch is a sketch for cardinality estimation based on LogLog counting
type Sketch struct {
	registers []uint8
	m         uint64
	s         uint64
	p         uint8
	vp        uint8
	alpha     float64
	mAlpha    float64
	n         uint64
}

// New returns a VHC sketch with 2^precision registers, where
// precision must be between 4 and 16
func New() (*Sketch, error) {
	rand.Seed(time.Now().UnixNano())
	p := uint8(22)
	vp := uint8(9)
	m := uint64(1 << p)
	s := uint64(1 << vp)
	return &Sketch{
		m:         m,
		s:         s,
		p:         p,
		vp:        vp,
		registers: make([]uint8, m),
		alpha:     alpha(float64(m)),
		mAlpha:    alpha(float64(s)),
	}, nil
}

func (vhc *Sketch) c(f []byte, i uint64) uint64 {
	return metro.Hash64(f, 1337*i) % vhc.m
}

// Add inserts a value into the sketch
func (vhc *Sketch) Add(f []byte) {
	q := rand.Uint64()
	i := q % vhc.s
	cfi := vhc.c(f, i)

	q1 := rand.Uint64()
	lambda := rho(q1, 32)

	if vhc.registers[cfi] < lambda {
		vhc.registers[cfi] = lambda
	}
}

// Count returns count in stream
func (vhc *Sketch) Count(f []byte) uint64 {
	cf := make([]uint8, vhc.s, vhc.s)
	for i := range cf {
		cfi := vhc.c(f, uint64(i))
		cf[i] = vhc.registers[cfi]
	}

	sum := 0.0
	s := float64(vhc.s)
	m := float64(vhc.m)
	ez := 0.0

	for _, val := range cf {
		sum += 1.0 / math.Pow(2.0, float64(val))
		if val == 0 {
			ez++
		}
	}

	beta := beta(ez)
	ns := (vhc.mAlpha * s * (s - ez) / (beta + sum))

	// estimate error
	N := float64(vhc.totalCardinality())
	e := s * N / m
	n := ns - e

	// rounding
	return uint64(n + 0.5)
}

// Merge takes another Sketch and combines it with vhc one, making vhc the union of both.
func (vhc *Sketch) Merge(other *Sketch) error {
	if vhc.p != vhc.p {
		return errors.New("precisions must be equal")
	}
	for i, v := range vhc.registers {
		if v < other.registers[i] {
			vhc.registers[i] = other.registers[i]
		}
	}
	return nil
}

func (vhc *Sketch) totalCardinality() uint64 {
	sum := 0.0
	m := float64(vhc.m)
	for _, val := range vhc.registers {
		sum += 1.0 / math.Pow(2.0, float64(val))
	}
	ez := zeros(vhc.registers)
	beta := beta(ez)
	return uint64(vhc.alpha * m * (m - ez) / (beta + sum))
}
