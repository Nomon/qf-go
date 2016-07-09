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

var ErrFull error = errors.New("filter is at its max capacity")

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
	fmt.Printf("%#v\n", qf)
}

func (qf *QuotientFilter) quotientAndRemainder(h uint64) (uint64, uint64) {
	return (h >> qf.rbits) & qf.qMask, h & qf.rMask
}

func (qf *QuotientFilter) hash(key string) uint64 {
	qf.h.Reset()
	qf.h.Write([]byte(key))
	return qf.h.Sum64()
}

func (qf *QuotientFilter) slotIndex(index uint64) (uint64, uint64, uint64, int) {
	bitIndex := uint64(qf.esize) * index
	bitOffset := bitIndex % 64
	sliceIndex := bitIndex / 64
	bitsInNextSlot := int(bitOffset) + int(qf.esize) - 64
	return bitIndex, sliceIndex, bitOffset, bitsInNextSlot
}

func (qf *QuotientFilter) element(pos uint64) (element uint64) {
	_, sliceIndex, bitOffset, nextBits := qf.slotIndex(pos)
	// align the slot and mask with the element mask
	element = (qf.data[sliceIndex] >> bitOffset) & qf.elemMask
	// does the slot span to next slice index, if so, capture rest of the bits from there
	if nextBits > 0 {
		sliceIndex++
		element |= (qf.data[sliceIndex] & maskLower(uint64(nextBits))) << (uint64(qf.esize) - uint64(nextBits))
	}
	return
}

func (qf *QuotientFilter) setElement(pos, element uint64) {
	// slot starts at bit data[sliceIndex][bitoffset:]
	// if the slot crosses slice boundary, nextBits contains
	// the number of bits the slot spans over to next slice item.
	_, sliceIndex, bitOffset, nextBits := qf.slotIndex(pos)
	// remove everything but remainder and meta bits.
	element &= qf.elemMask
	qf.data[sliceIndex] &= ^(qf.elemMask << bitOffset)
	qf.data[sliceIndex] |= element << bitOffset
	// the slot spans slice boundary, write the rest of the element to next index.
	if nextBits > 0 {
		sliceIndex++
		qf.data[sliceIndex] &= ^maskLower(uint64(nextBits))
		qf.data[sliceIndex] |= element >> (uint64(qf.esize) - uint64(nextBits))
	}
}

// Contains checks if key is present in the filter
// false positive propability is based on q, r and number of added keys
func (qf *QuotientFilter) Contains(key string) bool {
	h := qf.hash(key)
	q, r := qf.quotientAndRemainder(h)

	if !isOccupied(qf.element(q)) {
		return false
	}

	start := qf.findRun(q)
	for {
		remainder := getRemainder(qf.element(start))
		if remainder == r {
			return true
		} else if remainder > r {
			return false
		}
		start = qf.next(start)

		if !isContinuation(qf.element(start)) {
			break
		}
	}
	return false
}

func (qf *QuotientFilter) Add(key string) error {
	if qf.len >= qf.cap {
		return ErrFull
	}
	h := qf.hash(key)
	q, r := qf.quotientAndRemainder(h)
	elem := qf.element(q)
	tmp := (int64(r) << 3) & int64(^7)
	entry := uint64(tmp)

	if isEmpty(elem) {
		qf.setElement(q, setOccupied(entry))
		qf.len++
		return nil
	}
	if !isOccupied(elem) {
		qf.setElement(q, setOccupied(elem))
	}
	start := qf.findRun(q)
	s := start
	if isOccupied(elem) {
		for {
			rem := getRemainder(qf.element(s))
			if r == rem {
				return nil
			} else if rem > r {
				break
			}
			s = qf.next(s)
			if !isContinuation(qf.element(s)) {
				break
			}
		}
		if s == start {
			old := qf.element(start)
			qf.setElement(start, setContinuation(old))
		} else {
			entry = setContinuation(entry)
		}
	}
	if s != q {
		entry = setShifted(entry)
	}
	qf.insert(s, entry)
	qf.len++

	return nil
}

func (qf *QuotientFilter) insert(pos, e uint64) {
	cur := e

	for {
		prev := qf.element(pos)
		empty := isEmpty(prev)
		if !empty {
			prev = setShifted(prev)
			if isOccupied(prev) {
				cur = setOccupied(cur)
				prev = clearOccupied(prev)
			}
		}
		qf.setElement(pos, cur)
		cur = prev
		pos = qf.next(pos)
		if empty {
			break
		}
	}
}

func (qf *QuotientFilter) findRun(q uint64) (run uint64) {
	var elem uint64
	pos := q
	for {
		elem = qf.element(pos)
		if !isShifted(elem) {
			break
		}
		pos = qf.previous(pos)
	}
	run = pos
	for pos != q {
		for {
			run = qf.next(run)
			elem = qf.element(run)
			if !isContinuation(elem) {
				break
			}
		}
		for {
			pos = qf.next(pos)
			elem = qf.element(pos)
			if isOccupied(elem) {
				break
			}
		}
	}
	return
}

// AddAll adds multiple keys to the filter
func (q *QuotientFilter) AddAll(keys []string) error {
	for _, k := range keys {
		if err := q.Add(k); err != nil {
			return err
		}
	}
	return nil
}

func (qf *QuotientFilter) Delete(key string) bool {
	h := qf.hash(key)
	q, r := qf.quotientAndRemainder(h)
	elem := qf.element(q)
	if !isOccupied(elem) || qf.len == 0 {
		return false
	}
	start := qf.findRun(q)
	pos := start
	var rem uint64
	for {
		rem = getRemainder(qf.element(pos))
		if rem == r {
			break
		} else if rem > r {
			return true
		}
		pos = qf.next(pos)
		if !isContinuation(qf.element(pos)) {
			break
		}
	}
	if rem != r {
		return true
	}
	var replaceRunStart bool
	var kill uint64
	if pos == q {
		kill = elem
	} else {
		kill = qf.element(pos)
	}
	if isRunStart(kill) {
		next := qf.element(qf.next(pos))
		if !isContinuation(next) {
			elem = setOccupied(elem)
			qf.setElement(q, elem)
		}
	}
	if replaceRunStart {

	}
	return true
}

func (qf *QuotientFilter) previous(pos uint64) uint64 {
	return (pos - 1) & qf.qMask
}
func (qf *QuotientFilter) next(pos uint64) uint64 {
	return (pos + 1) & qf.qMask
}
