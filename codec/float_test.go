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

package codec

import (
	"math"
	"testing"
)

// TestFloat64Bits checks F64bits and Bitsf64 are exact inverses across normal,
// extreme, and special values.
func TestFloat64Bits(t *testing.T) {
	vals := []float64{
		0, 1, -1, math.Pi,
		math.MaxFloat64, math.SmallestNonzeroFloat64,
		math.Inf(1), math.Inf(-1),
	}

	for _, v := range vals {
		if got := Bitsf64(F64bits(v)); got != v {
			t.Errorf("Bitsf64(F64bits(%v))=%v", v, got)
		}
	}

	if !math.IsNaN(Bitsf64(F64bits(math.NaN()))) {
		t.Error("NaN did not round-trip through F64bits/Bitsf64")
	}
}

// TestFloat32Bits checks F32bits and Bitsf32 are exact inverses across normal,
// extreme, and special values.
func TestFloat32Bits(t *testing.T) {
	vals := []float32{
		0, 1, -1, 3.1415927,
		math.MaxFloat32, math.SmallestNonzeroFloat32,
		float32(math.Inf(1)), float32(math.Inf(-1)),
	}

	for _, v := range vals {
		if got := Bitsf32(F32bits(v)); got != v {
			t.Errorf("Bitsf32(F32bits(%v))=%v", v, got)
		}
	}

	if !math.IsNaN(float64(Bitsf32(F32bits(float32(math.NaN()))))) {
		t.Error("NaN did not round-trip through F32bits/Bitsf32")
	}
}
