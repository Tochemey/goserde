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

import (
	"testing"
	"unsafe"
)

func TestArrayShapes(t *testing.T) {
	fa := FixedArrays{
		Hash: [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
		Quad: [4]int32{-1, 2, -3, 4},
		Flag: true,
	}
	rt(t, fa, fa.Marshal, fa.Size, func(o *FixedArrays, b []byte) { o.Unmarshal(b) })

	// An all-fixed-width struct (arrays of fixed basics) must take the blittable
	// memmove path, whose wire size is exactly the in-memory size.
	if got, want := fa.Size(), int(unsafe.Sizeof(fa)); got != want {
		t.Errorf("FixedArrays not on memmove path: Size()=%d, unsafe.Sizeof=%d", got, want)
	}

	ma := MixedArrays{
		Name:   "mixed",
		Words:  [3]string{"a", "bb", "ccc"},
		Points: [2]Inner{{X: 1, Y: 2}, {X: 3, Y: 4}},
		Bytes:  [8]byte{8, 7, 6, 5, 4, 3, 2, 1},
	}
	rt(t, ma, ma.Marshal, ma.Size, func(o *MixedArrays, b []byte) { o.Unmarshal(b) })

	// Empty-string array elements still round-trip (arrays carry no length
	// prefix, so every slot is always present on the wire).
	ma2 := MixedArrays{Words: [3]string{"", "x", ""}}
	rt(t, ma2, ma2.Marshal, ma2.Size, func(o *MixedArrays, b []byte) { o.Unmarshal(b) })
}
