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

package example

import (
	"testing"

	"github.com/tochemey/goserde/runtime"
)

// PointPF is Point serialized field-by-field rather than via the blittable
// memmove path, so benchmarks can measure the memmove speedup.
type PointPF struct{ X, Y, Z int32 }

// Size returns the fixed 12-byte encoded length of a PointPF.
func (r *PointPF) Size() int { return 12 }

// Marshal writes the three coordinates as little-endian uint32s into b.
func (r *PointPF) Marshal(b []byte) int {
	runtime.PutU32(b[0:], uint32(r.X))
	runtime.PutU32(b[4:], uint32(r.Y))
	runtime.PutU32(b[8:], uint32(r.Z))

	return 12
}

// Unmarshal reads the three coordinates back from b.
func (r *PointPF) Unmarshal(b []byte) (int, error) {
	r.X = int32(runtime.U32(b[0:]))
	r.Y = int32(runtime.U32(b[4:]))
	r.Z = int32(runtime.U32(b[8:]))

	return 12, nil
}

func BenchmarkPointBlitMarshal(b *testing.B) {
	p := &Point{X: -7, Y: 13, Z: 2147483647}
	buf := make([]byte, p.Size())
	b.ReportAllocs()
	for b.Loop() {
		_ = p.Marshal(buf)
	}
}

func BenchmarkPointBlitUnmarshal(b *testing.B) {
	p := &Point{X: -7, Y: 13, Z: 2147483647}
	buf := make([]byte, p.Size())
	p.Marshal(buf)
	b.ReportAllocs()
	for b.Loop() {
		var o Point
		o.Unmarshal(buf)
	}
}

func BenchmarkPointFieldMarshal(b *testing.B) {
	p := &PointPF{X: -7, Y: 13, Z: 2147483647}
	buf := make([]byte, p.Size())
	b.ReportAllocs()
	for b.Loop() {
		_ = p.Marshal(buf)
	}
}

func BenchmarkPointFieldUnmarshal(b *testing.B) {
	p := &PointPF{X: -7, Y: 13, Z: 2147483647}
	buf := make([]byte, p.Size())
	p.Marshal(buf)
	b.ReportAllocs()
	for b.Loop() {
		var o PointPF
		o.Unmarshal(buf)
	}
}

func TestBlitRoundTrip(t *testing.T) {
	in := &Point{X: -7, Y: 13, Z: 2147483647}
	buf := make([]byte, in.Size())

	if n := in.Marshal(buf); n != 12 {
		t.Fatalf("size %d", n)
	}

	var out Point
	out.Unmarshal(buf)

	if out != *in {
		t.Fatalf("got %+v want %+v", out, in)
	}
}
