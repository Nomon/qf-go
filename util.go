package qf

func maskLower(e uint64) uint64 {
	return (1 << e) - 1
}

func uint64Size(q, r uint8) int {
	var bits int = (1 << q) * int(r+3)
	bytes := bits / 8
	if bits%8 != 0 {
		bytes++
	}
	return int(bytes)
}
