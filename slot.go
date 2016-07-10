package qf

type slot uint64

func newSlot(remainder uint64) slot {
	// shift remained right to make room for 3 control bits.
	return slot((int64(remainder) << 3) & ^7)
}
func (s slot) isOccupied() bool {
	return s&1 == 1
}
func (s slot) setOccupied() slot {
	s |= 1
	return s
}
func (s slot) clearOccupied() slot {
	clrBits := int64(^1)
	return s & slot(clrBits)
}
func (s slot) isContinuation() bool {
	return s&2 == 2
}
func (s slot) setContinuation() slot {
	return s | 2
}
func (s slot) clearContinuation() slot {
	clrBits := int64(^2)
	return s & slot(clrBits)
}

func (s slot) isShifted() bool {
	return s&4 == 4
}
func (s slot) setShifted() slot {
	return s | 4
}

func (s slot) clearShifted() slot {
	clrBits := int64(^4)
	return s & slot(clrBits)
}

func (s slot) remainder() uint64 {
	return uint64(s) >> 3
}
func (s slot) isEmpty() bool {
	return s&7 == 0
}
func (s slot) isClusterStart() bool {
	return s.isOccupied() && !s.isContinuation() && !s.isShifted()
}
func (s slot) isRunStart() bool {
	return s.isContinuation() && (s.isOccupied() || s.isShifted())
}
