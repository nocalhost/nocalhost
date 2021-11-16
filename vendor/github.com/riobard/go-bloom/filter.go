// Package bloom implements Bloom Filter using double hashing
package bloom

import (
	"math"
)

// Filter is a generic Bloom Filter
type Filter interface {
	Add([]byte)       // add an entry to the filter
	Test([]byte) bool // test if an entry is in the filter
	Size() int        // size of the filter in bytes
	Reset()           // reset the filter to initial state
}

// Classic Bloom Filter
type classicFilter struct {
	b []byte
	k int
	h func([]byte) (uint64, uint64)
}

// New creates a classic Bloom Filter that is optimal for n entries and false positive rate of p.
// h is a double hash that takes an entry and returns two different hashes.
func New(n int, p float64, h func([]byte) (uint64, uint64)) Filter {
	k := -math.Log(p) * math.Log2E   // number of hashes
	m := float64(n) * k * math.Log2E // number of bits
	return &classicFilter{b: make([]byte, int(m/8)), k: int(k), h: h}
}

func (f *classicFilter) getOffset(x, y uint64, i int) uint64 {
	return (x + uint64(i)*y) % (8 * uint64(len(f.b)))
}

func (f *classicFilter) Add(b []byte) {
	x, y := f.h(b)
	for i := 0; i < f.k; i++ {
		offset := f.getOffset(x, y, i)
		f.b[offset/8] |= 1 << (offset % 8)
	}
}

func (f *classicFilter) Test(b []byte) bool {
	x, y := f.h(b)
	for i := 0; i < f.k; i++ {
		offset := f.getOffset(x, y, i)
		if f.b[offset/8]&(1<<(offset%8)) == 0 {
			return false
		}
	}
	return true
}

func (f *classicFilter) Size() int { return len(f.b) }

func (f *classicFilter) Reset() {
	for i := range f.b {
		f.b[i] = 0
	}
}
