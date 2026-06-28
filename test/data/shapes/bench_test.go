// MIT License
//
// Copyright (c) 2026 Arsene Tochemey Gandote
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package shapes

import "testing"

// benchM benchmarks Marshal for a shape, allocating the destination once.
func benchM[T any](b *testing.B, size func() int, marshal func([]byte) int) {
	buf := make([]byte, size())
	b.ReportAllocs()
	for b.Loop() {
		_ = marshal(buf)
	}
}

func BenchmarkSmallFixed_M(b *testing.B) {
	v := SmallFixed{A: 1, B: 2, C: 3, D: true}
	benchM[SmallFixed](b, v.Size, v.Marshal)
}

func BenchmarkFlatMixed_M(b *testing.B) {
	v := FlatMixed{ID: 1, Ratio: 2, Name: "hello world", Data: []byte("0123456789")}
	benchM[FlatMixed](b, v.Size, v.Marshal)
}

func BenchmarkNested_M(b *testing.B) {
	in := Inner{7, 8}
	v := Nested{Label: "n", Pos: Inner{1, 2}, Opt: &in, Path: []Inner{{3, 4}, {5, 6}, {7, 8}}}
	benchM[Nested](b, v.Size, v.Marshal)
}

func BenchmarkCollection_M(b *testing.B) {
	v := CollectionHeavy{Counts: map[string]int64{"a": 1, "b": 2, "c": 3, "d": 4}, Floats: []float64{1, 2, 3, 4, 5}, Names: []string{"x", "y", "z"}}
	benchM[CollectionHeavy](b, v.Size, v.Marshal)
}

func BenchmarkStringHeavy_M(b *testing.B) {
	v := StringHeavy{Title: "Title", Body: "lorem ipsum dolor sit amet consectetur", Tags: []string{"go", "fast", "serde"}}
	benchM[StringHeavy](b, v.Size, v.Marshal)
}

// benchU benchmarks Unmarshal for a shape against a pre-marshaled buffer.
func benchU(b *testing.B, buf []byte, unmarshal func([]byte)) {
	b.ReportAllocs()
	for b.Loop() {
		unmarshal(buf)
	}
}

func BenchmarkSmallFixed_U(b *testing.B) {
	v := SmallFixed{A: 1, B: 2, C: 3, D: true}
	buf := make([]byte, v.Size())
	v.Marshal(buf)
	benchU(b, buf, func(x []byte) { var o SmallFixed; o.Unmarshal(x) })
}

func BenchmarkFlatMixed_U(b *testing.B) {
	v := FlatMixed{ID: 1, Ratio: 2, Name: "hello world", Data: []byte("0123456789")}
	buf := make([]byte, v.Size())
	v.Marshal(buf)
	benchU(b, buf, func(x []byte) { var o FlatMixed; o.Unmarshal(x) })
}

func BenchmarkNested_U(b *testing.B) {
	in := Inner{7, 8}
	v := Nested{Label: "n", Pos: Inner{1, 2}, Opt: &in, Path: []Inner{{3, 4}, {5, 6}, {7, 8}}}
	buf := make([]byte, v.Size())
	v.Marshal(buf)
	benchU(b, buf, func(x []byte) { var o Nested; o.Unmarshal(x) })
}

func BenchmarkCollection_U(b *testing.B) {
	v := CollectionHeavy{Counts: map[string]int64{"a": 1, "b": 2, "c": 3, "d": 4}, Floats: []float64{1, 2, 3, 4, 5}, Names: []string{"x", "y", "z"}}
	buf := make([]byte, v.Size())
	v.Marshal(buf)
	benchU(b, buf, func(x []byte) { var o CollectionHeavy; o.Unmarshal(x) })
}

// BenchmarkCollection_U_Reuse decodes repeatedly into the SAME destination,
// exercising slice-capacity and map reuse. It should report fewer allocs than
// BenchmarkCollection_U, which decodes into a fresh value every iteration.
func BenchmarkCollection_U_Reuse(b *testing.B) {
	v := CollectionHeavy{Counts: map[string]int64{"a": 1, "b": 2, "c": 3, "d": 4}, Floats: []float64{1, 2, 3, 4, 5}, Names: []string{"x", "y", "z"}}
	buf := make([]byte, v.Size())
	v.Marshal(buf)
	var o CollectionHeavy
	benchU(b, buf, func(x []byte) { o.Unmarshal(x) })
}

func BenchmarkStringHeavy_U(b *testing.B) {
	v := StringHeavy{Title: "Title", Body: "lorem ipsum dolor sit amet consectetur", Tags: []string{"go", "fast", "serde"}}
	buf := make([]byte, v.Size())
	v.Marshal(buf)
	benchU(b, buf, func(x []byte) { var o StringHeavy; o.Unmarshal(x) })
}
