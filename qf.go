package qf

import (
	"errors"
	"fmt"
	"hash"
	"hash/fnv"
	"math"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

// ErrFull is returned when Add is called while the filter is at max capacity.
var ErrFull = errors.New("filter is at its max capacity")

// QuotientFilter is the implementation
type QuotientFilter struct {
	// quotient and remainder bits
	qbits uint8
	rbits uint8
	// total element size, qbits + 3 metadata bits
	esize uint8
	// how many elements does the filter contain and capacity 1 << qbits
	len uint64
	cap uint64
	// data
	data []uint64
	// precalculated masks
	elemMask uint64
	qMask    uint64
	rMask    uint64
	h        hash.Hash64
}

// NewPropability returns a quotient filter that can accomidate capacity number of elements
// with propability passed.
func NewPropability(capacity int, propability float64) *QuotientFilter {
	q := uint8(math.Ceil(math.Log2(float64(capacity))))
	r := uint8(-math.Log2(propability))
	return New(q, r)
}

// NewHash returns a QuotientFilter backed by a different hash function.
// Default hash function is FNV-64a
func NewHash(h hash.Hash64, q, r uint8) *QuotientFilter {
	qf := New(q, r)
	qf.h = h
	return qf
}

// New returns a QuotientFilter with q quotient bits and r remainder bits.
// it can hold 1 << q elements.
func New(q, r uint8) *QuotientFilter {
	if q+r > 64 {
		panic("q + r has to be less than 64 bits")
	}
	qf := &QuotientFilter{
		qbits: q,
		rbits: r,
		esize: r + 3,
		len:   0,
		cap:   1 << q,
		h:     fnv.New64a(),
	}
	qf.qMask = maskLower(uint64(q))
	qf.rMask = maskLower(uint64(r))
	qf.elemMask = maskLower(uint64(qf.esize))
	qf.data = make([]uint64, uint64Size(q, r))
	return qf
}

func (qf *QuotientFilter) info() {
	fmt.Println("slot, is_occopied:is_continuation:is_shifted, remainder")
	for i := uint64(0); i < qf.cap; i++ {
		s := qf.getSlot(i)
		fmt.Printf("%d:\t\t%b%b%b\t%d\n", i, s&1, s&2>>1, s&4>>2, s.remainder())
	}
}

func (qf *QuotientFilter) quotientAndRemainder(h uint64) (uint64, uint64) {
	return (h >> qf.rbits) & qf.qMask, h & qf.rMask
}

func (qf *QuotientFilter) hash(key string) uint64 {
	qf.h.Reset()
	qf.h.Write([]byte(key))
	return qf.h.Sum64()
}

func (qf *QuotientFilter) getSlot(index uint64) slot {
	_, sliceIndex, bitOffset, nextBits := qf.slotIndex(index)
	s := (qf.data[sliceIndex] >> bitOffset) & qf.elemMask
	// does the slot span to next slice index, if so, capture rest of the bits from there
	if nextBits > 0 {
		sliceIndex++
		s |= (qf.data[sliceIndex] & maskLower(uint64(nextBits))) << (uint64(qf.esize) - uint64(nextBits))
	}
	return slot(s)
}

func (qf *QuotientFilter) setSlot(index uint64, s slot) {
	// slot starts at bit data[sliceIndex][bitoffset:]
	// if the slot crosses slice boundary, nextBits contains
	// the number of bits the slot spans over to next slice item.
	_, sliceIndex, bitOffset, nextBits := qf.slotIndex(index)
	// remove everything but remainder and meta bits.
	s &= slot(qf.elemMask)
	qf.data[sliceIndex] &= ^(qf.elemMask << bitOffset)
	qf.data[sliceIndex] |= uint64(s) << bitOffset
	// the slot spans slice boundary, write the rest of the element to next index.
	if nextBits > 0 {
		sliceIndex++
		qf.data[sliceIndex] &^= maskLower(uint64(nextBits))
		qf.data[sliceIndex] |= uint64(s) >> (uint64(qf.esize) - uint64(nextBits))
	}
}

func (qf *QuotientFilter) slotIndex(index uint64) (uint64, uint64, uint64, int) {
	bitIndex := uint64(qf.esize) * index
	bitOffset := bitIndex % 64
	sliceIndex := bitIndex / 64
	bitsInNextSlot := int(bitOffset) + int(qf.esize) - 64
	return bitIndex, sliceIndex, bitOffset, bitsInNextSlot
}

func (qf *QuotientFilter) previous(index uint64) uint64 {
	return (index - 1) & qf.qMask
}
func (qf *QuotientFilter) next(index uint64) uint64 {
	return (index + 1) & qf.qMask
}

// Contains checks if key is present in the filter
// false positive propability is based on q, r and number of added keys
func (qf *QuotientFilter) Contains(key string) bool {
	q, r := qf.quotientAndRemainder(qf.hash(key))

	if !qf.getSlot(q).isOccupied() {
		return false
	}

	index := qf.findRun(q)
	slot := qf.getSlot(index)
	for {
		remainder := slot.remainder()
		if remainder == r {
			return true
		} else if remainder > r {
			return false
		}
		index = qf.next(index)
		slot = qf.getSlot(index)
		if !slot.isContinuation() {
			break
		}
	}
	return false
}

func (qf *QuotientFilter) Add(key string) error {
	if qf.len >= qf.cap {
		return ErrFull
	}
	q, r := qf.quotientAndRemainder(qf.hash(key))
	slot := qf.getSlot(q)
	new := newSlot(r)

	// if slot is empty, just set the new there and occupy it and return.
	if slot.isEmpty() {
		qf.setSlot(q, new.setOccupied())
		qf.len++
		return nil
	}

	if !slot.isOccupied() {
		qf.setSlot(q, slot.setOccupied())
	}

	start := qf.findRun(q)
	index := start

	if slot.isOccupied() {
		runSlot := qf.getSlot(index)
		for {
			remainder := runSlot.remainder()
			if r == remainder {
				return nil
			} else if remainder > r {
				break
			}
			index = qf.next(index)
			runSlot = qf.getSlot(index)
			if !runSlot.isContinuation() {
				break
			}
		}
		if index == start {
			old := qf.getSlot(start)
			qf.setSlot(start, old.setContinuation())
		} else {
			new = new.setContinuation()
		}
	}
	if index != q {
		new = new.setShifted()
	}
	qf.insertSlot(index, new)
	qf.len++

	return nil
}

func (qf *QuotientFilter) insertSlot(index uint64, s slot) {
	curr := s
	for {
		prev := qf.getSlot(index)
		empty := prev.isEmpty()
		if !empty {
			prev = prev.setShifted()
			if prev.isOccupied() {
				curr = curr.setOccupied()
				prev = prev.clearOccupied()
			}
		}
		qf.setSlot(index, curr)
		curr = prev
		index = qf.next(index)
		if empty {
			break
		}
	}
}

func (qf *QuotientFilter) findRun(quotient uint64) (run uint64) {
	var slot slot
	index := quotient
	for {
		slot = qf.getSlot(index)
		if !slot.isShifted() {
			break
		}
		index = qf.previous(index)
	}
	run = index
	for index != quotient {
		for {
			run = qf.next(run)
			slot = qf.getSlot(run)
			if !slot.isContinuation() {
				break
			}
		}
		for {
			index = qf.next(index)
			slot = qf.getSlot(index)
			if slot.isOccupied() {
				break
			}
		}
	}
	return
}

// AddAll adds multiple keys to the filter
func (qf *QuotientFilter) AddAll(keys []string) error {
	for _, k := range keys {
		if err := qf.Add(k); err != nil {
			return err
		}
	}
	return nil
}
