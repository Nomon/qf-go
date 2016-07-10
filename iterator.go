package qf

type Iterator struct {
	qf       *QuotientFilter
	visited  int64
	index    uint64
	quotient uint64
}

func NewIterator(qf *QuotientFilter) Iterator {
	return Iterator{
		qf: qf,
	}
}
func (i Iterator) HasNext() bool {
	return uint64(i.visited) < i.qf.len
}

func (i Iterator) Next() uint64 {
	for {
		slot := i.qf.getSlot(i.index)
		if slot.isClusterStart() {
			i.quotient = i.index
		} else {
			if slot.isRunStart() {
				quot := i.quotient
				for {
					quot = i.qf.next(quot)
					if i.qf.getSlot(quot).isOccupied() {
						break
					}
				}
				i.quotient = quot
			}
		}
		i.index = i.qf.next(i.index)
		if !slot.isEmpty() {
			quot := i.quotient
			remainder := slot.remainder()
			i.visited++
			return (quot << i.qf.rbits) | remainder
		}
	}
}
