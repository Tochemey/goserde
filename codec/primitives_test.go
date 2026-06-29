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

// TestFixedWidthRoundTrip checks that every fixed-width writer and its matching
// reader are exact inverses across boundary values.
func TestFixedWidthRoundTrip(t *testing.T) {
	for _, v := range []uint16{0, 1, 0x00FF, 0xFF00, 0x1234, math.MaxUint16} {
		b := make([]byte, 2)
		PutU16(b, v)

		if got := U16(b); got != v {
			t.Errorf("U16 round-trip: got %#x, want %#x", got, v)
		}
	}

	for _, v := range []uint32{0, 1, 0x12345678, math.MaxUint32} {
		b := make([]byte, 4)
		PutU32(b, v)

		if got := U32(b); got != v {
			t.Errorf("U32 round-trip: got %#x, want %#x", got, v)
		}
	}

	for _, v := range []uint64{0, 1, 0x0123456789ABCDEF, math.MaxUint64} {
		b := make([]byte, 8)
		PutU64(b, v)

		if got := U64(b); got != v {
			t.Errorf("U64 round-trip: got %#x, want %#x", got, v)
		}
	}
}

// TestFixedWidthLittleEndian pins the on-wire byte order: the least significant
// byte is written first.
func TestFixedWidthLittleEndian(t *testing.T) {
	b16 := make([]byte, 2)
	PutU16(b16, 0x0102)

	if b16[0] != 0x02 || b16[1] != 0x01 {
		t.Errorf("PutU16 not little-endian: %v", b16)
	}

	b32 := make([]byte, 4)
	PutU32(b32, 0x01020304)

	if b32[0] != 0x04 || b32[1] != 0x03 || b32[2] != 0x02 || b32[3] != 0x01 {
		t.Errorf("PutU32 not little-endian: %v", b32)
	}

	b64 := make([]byte, 8)
	PutU64(b64, 0x0102030405060708)

	if b64[0] != 0x08 || b64[7] != 0x01 {
		t.Errorf("PutU64 not little-endian: %v", b64)
	}
}

// TestUvarintRoundTrip checks PutUvarint, Uvarint, and UvarintSize agree across
// values that span every varint length.
func TestUvarintRoundTrip(t *testing.T) {
	vals := []uint64{
		0, 1, 127, 128, 255, 256, 300,
		1<<14 - 1, 1 << 14, 1<<21 - 1, 1 << 21,
		1 << 28, 1 << 35, 1 << 42, 1 << 49, 1 << 56, 1 << 63,
		math.MaxUint64,
	}

	buf := make([]byte, 10)
	for _, v := range vals {
		n := PutUvarint(buf, v)

		if sz := UvarintSize(v); sz != n {
			t.Errorf("v=%d: UvarintSize=%d but PutUvarint wrote %d", v, sz, n)
		}

		got, c := Uvarint(buf)
		if got != v || c != n {
			t.Errorf("v=%d: Uvarint=(%d,%d), want (%d,%d)", v, got, c, v, n)
		}
	}
}

// TestUvarintSizeBoundaries checks the byte count at every length boundary.
func TestUvarintSizeBoundaries(t *testing.T) {
	cases := []struct {
		v    uint64
		size int
	}{
		{0, 1}, {127, 1},
		{128, 2}, {16383, 2},
		{16384, 3},
		{1 << 21, 4},
		{1 << 28, 5},
		{1 << 35, 6},
		{1 << 42, 7},
		{1 << 49, 8},
		{1 << 56, 9},
		{1 << 63, 10}, {math.MaxUint64, 10},
	}

	for _, c := range cases {
		if got := UvarintSize(c.v); got != c.size {
			t.Errorf("UvarintSize(%d)=%d, want %d", c.v, got, c.size)
		}
	}
}

// TestUvarintTruncated checks that a buffer that ends mid-varint reports a zero
// count (the signal generated safe-mode code treats as ErrShortBuffer).
func TestUvarintTruncated(t *testing.T) {
	if v, n := Uvarint([]byte{0x80, 0x80, 0x80}); v != 0 || n != 0 {
		t.Errorf("continuation-only Uvarint=(%d,%d), want (0,0)", v, n)
	}

	if v, n := Uvarint(nil); v != 0 || n != 0 {
		t.Errorf("empty Uvarint=(%d,%d), want (0,0)", v, n)
	}
}

// TestUvarintOverflow checks that an encoding too long for a uint64 reports a
// negative count, hitting both overflow guards.
func TestUvarintOverflow(t *testing.T) {
	// 11 bytes: the terminator lands at index 10, so i > 9.
	over := []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x01}
	if _, n := Uvarint(over); n >= 0 {
		t.Errorf("overflow (i>9): count=%d, want negative", n)
	}

	// 10 bytes where the final byte (index 9) carries more than one high bit.
	over2 := []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x02}
	if _, n := Uvarint(over2); n >= 0 {
		t.Errorf("overflow (i==9, c>1): count=%d, want negative", n)
	}
}

// TestZigZag checks the zigzag transform round-trips and maps small magnitudes
// (either sign) to small unsigned values.
func TestZigZag(t *testing.T) {
	vals := []int64{0, -1, 1, -2, 2, 63, -64, 1 << 40, -(1 << 40), math.MaxInt64, math.MinInt64}
	for _, v := range vals {
		if got := Zag(Zig(v)); got != v {
			t.Errorf("Zag(Zig(%d))=%d", v, got)
		}
	}

	if Zig(0) != 0 || Zig(-1) != 1 || Zig(1) != 2 || Zig(-2) != 3 {
		t.Errorf("Zig mapping: 0=%d -1=%d 1=%d -2=%d, want 0 1 2 3", Zig(0), Zig(-1), Zig(1), Zig(-2))
	}
}

// TestZeroCopyConversions checks B2S and S2B preserve contents, including the
// empty case.
func TestZeroCopyConversions(t *testing.T) {
	s := "goserde zero-copy conversion"

	if got := B2S([]byte(s)); got != s {
		t.Errorf("B2S=%q, want %q", got, s)
	}

	if got := string(S2B(s)); got != s {
		t.Errorf("S2B=%q, want %q", got, s)
	}

	if B2S(nil) != "" {
		t.Error("B2S(nil) should be the empty string")
	}

	if len(S2B("")) != 0 {
		t.Error("S2B(\"\") should be empty")
	}
}
