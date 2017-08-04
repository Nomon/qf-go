package qf

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestNewProbability(t *testing.T) {
	tests := []struct {
		P float64
		S int
	}{{0.001, 10000}, {0.01, 10000}, {0.1, 10000}, {0.3, 10000},
		{0.001, 100000}, {0.01, 100000}, {0.1, 100000}, {0.3, 100000}}
	for _, test := range tests {
		qf := NewProbability(test.S, test.P)
		qf.AddAll(generateItems(test.S))
		if qf.FPProbability() > test.P {
			t.Fatal("False positive rate too high, asked", test.P, "got", qf.FPProbability(), "test", test)
		}
	}
}

func TestAddBasic(t *testing.T) {
	qf := New(8, 3)

	added := generateItems(100) // []string{"brown", "fox", "jump"}
	not := []string{"turbo", "negro"}
	qf.AddAll(added)
	qf.info()
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
		qf := NewProbability(test.S, test.P)
		items := generateItems(test.S / 2)
		qf.AddAll(items)
		for _, item := range items {
			if !qf.Contains(item) {
				t.Fatal("False negative, key:", item, "size", test.S, "probability", test.P)
			}
		}
	}
}

func TestFalsePositives(t *testing.T) {
	tests := []struct {
		P float64
		S int
	}{{0.001, 1000}, {0.01, 1000}, {0.1, 1000}, {0.3, 1000},
		{0.001, 10000}, {0.01, 10000}, {0.1, 10000}, {0.3, 10000},
		{0.001, 100000}, {0.01, 100000}, {0.1, 100000}, {0.3, 100000}}
	for _, test := range tests {
		qf := NewProbability(test.S, test.P)
		items := generateItems(test.S / 2)
		itemsB := generateItems(test.S / 2)
		qf.AddAll(items)
		var positives int
		for _, item := range itemsB {
			if qf.Contains(item) {
				positives++
			}
		}
		for _, item := range items {
			if !qf.Contains(item) {
				t.Fatal("False negative")
			}
		}
		allowed := float64(test.S) * test.P * 2.0
		if float64(positives) > allowed {
			t.Fatal("Too many positives, got", positives, "limit is", allowed, "test:", test)
		}
	}
}

func BenchmarkAdd(b *testing.B) {
	qf := NewProbability(b.N*2, 0.01)
	items := generateItems(b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qf.Add(items[i])
	}
	b.StopTimer()
}

func BenchmarkContains(b *testing.B) {
	qf := NewProbability(b.N*2, 0.01)
	items := generateItems(b.N)
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			qf.Add(items[i])
		}
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
