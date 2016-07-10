package qf

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestNewPropability(t *testing.T) {
	qf := NewPropability(100000, 0.001)
	if qf.cap <= 100000 {
		t.Fatal("Too small filter")
	}
}

func TestAddBasic(t *testing.T) {
	// 2^16 size, 8 bit remainder ~0.7MB
	qf := New(16, 8)
	added := []string{"brown", "fox", "jump"}
	not := []string{"turbo", "negro"}
	qf.AddAll(added)

	for _, s := range added {
		if !qf.Contains(s) {
			t.Fatal("Filter returned false for an added item")
		}
	}

	for _, s := range not {
		if qf.Contains(s) {
			t.Fatal("Filter returned true for not added item")
		}
	}
}

func TestFalseNegatives(t *testing.T) {
	tests := []struct {
		P float64
		S int
	}{{0.01, 1000}, {0.01, 10000}, {0.01, 100000}}
	for _, test := range tests {
		qf := NewPropability(test.S, test.P)
		items := generateItems(test.S / 2)
		qf.AddAll(items)
		for _, item := range items {
			if !qf.Contains(item) {
				t.Fatal("False negative, key:", item, "size", test.S, "propability", test.P)
			}
		}
	}
}

func TestFalsePositives(t *testing.T) {
	tests := []struct {
		P float64
		S int
	}{{0.01, 1000}, {0.01, 10000}, {0.01, 100000}, {0.01, 1000000}}
	for _, test := range tests {
		qf := NewPropability(test.S, test.P)
		items := generateItems(test.S / 2)
		itemsB := generateItems(test.S / 2)
		qf.AddAll(items)
		var positives int
		for _, item := range itemsB {
			if qf.Contains(item) {
				positives++
			}
		}
		allowed := float64(test.S) * test.P * 2.0
		if float64(positives) > allowed {
			t.Fatal("Too many positives, got", positives, "limit is", allowed)
		}
	}
}

func BenchmarkSmall(b *testing.B) {
	qf := New(12, 6)
	setSize := 1000
	items_a := generateItems(setSize)
	items_b := generateItems(setSize)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := qf.Add(items_a[i%setSize]); err != nil {
			b.Fatal(err)
		}
		qf.Contains(items_a[i%setSize])
		qf.Contains(items_b[i%setSize])
	}
	b.StopTimer()
	//qf.info()
	qf.data = nil
	//runtime.GC()
}

func BenchmarkMedium(b *testing.B) {
	qf := New(16, 8)
	setSize := 10000
	items_a := generateItems(setSize)
	items_b := generateItems(setSize)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := qf.Add(items_a[i%setSize]); err != nil {
			b.Fatal(err)
		}
		qf.Contains(items_a[i%setSize])
		qf.Contains(items_b[i%setSize])
	}
	b.StopTimer()

	qf.data = nil
	//	runtime.GC()
}

func BenchmarkLarge(b *testing.B) {
	qf := New(20, 12)
	setSize := 100000
	items_a := generateItems(setSize)
	items_b := generateItems(setSize)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := qf.Add(items_a[i%setSize]); err != nil {
			b.Fatal(err)
		}
		qf.Contains(items_a[i%setSize])
		qf.Contains(items_b[i%setSize])
	}
	b.StopTimer()
}
func BenchmarkAdd(b *testing.B) {
	qf := NewPropability(b.N*2, 0.01)
	items := generateItems(b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qf.Add(items[i])
	}
	b.StopTimer()
}

func BenchmarkContains(b *testing.B) {
	qf := NewPropability(b.N*2, 0.01)
	items := generateItems(b.N)
	for i := 0; i < b.N; i++ {
		qf.Add(items[i])
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qf.Contains(items[i])
	}
	b.StopTimer()
}

var generatedSet int

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
	generatedSet = rand.Int()
}

func generateItems(len int) []string {
	setNum := generatedSet
	generatedSet++
	out := make([]string, 0, len)
	for i := 0; i < len; i++ {
		out = append(out, fmt.Sprintf("item:%d:%d", setNum, i))
	}
	return out
}
