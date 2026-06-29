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

// Package codec is the support library that goserde codecs are built on. It
// serves two audiences, both supported and both covered by the module's
// semantic-versioning promise:
//
//   - Convenience helpers for everyday use: [Bytes], [Into], and [From], plus
//     the [Marshaler] and [Unmarshaler] interfaces that every codec implements.
//   - Low-level encoding primitives for writing a codec by hand when you want
//     full control instead of running goserdegen: the fixed-width readers and
//     writers ([U32]/[PutU32] and friends), the varint codec ([Uvarint],
//     [PutUvarint], [UvarintSize]), the zigzag transform ([Zig]/[Zag]), the
//     zero-copy string conversions ([B2S]/[S2B]), and the float bit-casts. Code
//     emitted by goserdegen calls these same primitives, so a hand-written codec
//     and a generated one share an identical wire format.
//
// Everything here is written to be inlined by the Go compiler and to avoid
// bounds checks and allocations on the hot path.
package codec

import (
	"errors"
	"unsafe"
)

// ErrShortBuffer is returned by codecs generated in safe mode (goserdegen -safe)
// when the input buffer is truncated or otherwise too small to hold the value
// being decoded. Fast-mode (default) codecs do not perform these checks and may
// panic on such input instead.
var ErrShortBuffer = errors.New("goserde: buffer too small")

// ErrUnknownUnionTag is returned by a generated Unmarshal when a tagged-union
// field carries a discriminator that does not match any declared member. It is
// returned in both fast and safe mode, since an unrecognized tag is a normal
// interoperability condition (e.g. a peer encoded a newer union member) rather
// than buffer corruption.
var ErrUnknownUnionTag = errors.New("goserde: unknown union tag")

// B2S converts a byte slice to a string without copying (trusted-bytes mode,
// uses unsafe). The returned string shares memory with b; mutating b afterwards
// mutates the string, so copy first if the caller needs an independent value.
func B2S(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// S2B returns the bytes backing s without copying (trusted-bytes mode, uses
// unsafe). The result must not be mutated, as Go strings are immutable.
func S2B(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// PutU16 writes v as 2 little-endian bytes at b[0:2]. The caller must guarantee
// len(b) >= 2; in trusted-bytes mode this is ensured by the upfront Size pass.
func PutU16(b []byte, v uint16) {
	_ = b[1] // bounds-check hint: lets the compiler elide the per-store checks
	b[0] = byte(v)
	b[1] = byte(v >> 8)
}

// PutU32 writes v as 4 little-endian bytes at b[0:4]. The caller must guarantee
// len(b) >= 4.
func PutU32(b []byte, v uint32) {
	_ = b[3]
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
}

// PutU64 writes v as 8 little-endian bytes at b[0:8]. The caller must guarantee
// len(b) >= 8.
func PutU64(b []byte, v uint64) {
	_ = b[7]
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
	b[4] = byte(v >> 32)
	b[5] = byte(v >> 40)
	b[6] = byte(v >> 48)
	b[7] = byte(v >> 56)
}

// U16 reads 2 little-endian bytes from b[0:2] as a uint16. It is the inverse of
// [PutU16]; the caller must guarantee len(b) >= 2.
func U16(b []byte) uint16 {
	_ = b[1]
	return uint16(b[0]) | uint16(b[1])<<8
}

// U32 reads 4 little-endian bytes from b[0:4] as a uint32. It is the inverse of
// [PutU32]; the caller must guarantee len(b) >= 4.
func U32(b []byte) uint32 {
	_ = b[3]
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

// U64 reads 8 little-endian bytes from b[0:8] as a uint64. It is the inverse of
// [PutU64]; the caller must guarantee len(b) >= 8.
func U64(b []byte) uint64 {
	_ = b[7]
	return uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
		uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56
}

// PutUvarint writes v as an unsigned LEB128 varint at b[0:] and returns the
// number of bytes written. Varints encode length prefixes and small integers
// compactly; the caller must ensure b has room for UvarintSize(v) bytes.
func PutUvarint(b []byte, v uint64) int {
	i := 0
	for v >= 0x80 {
		b[i] = byte(v) | 0x80
		v >>= 7
		i++
	}
	b[i] = byte(v)
	return i + 1
}

// Uvarint reads an unsigned varint from b, returning the value and bytes read.
// A non-positive count signals an error (overflow or truncation).
func Uvarint(b []byte) (uint64, int) {
	var x uint64
	var s uint
	for i := range len(b) {
		c := b[i]
		if c < 0x80 {
			if i > 9 || (i == 9 && c > 1) {
				return 0, -(i + 1) // overflow
			}
			return x | uint64(c)<<s, i + 1
		}
		x |= uint64(c&0x7f) << s
		s += 7
	}
	return 0, 0 // truncated
}

// UvarintSize returns the number of bytes PutUvarint would write for v.
func UvarintSize(v uint64) int {
	n := 1
	for v >= 0x80 {
		v >>= 7
		n++
	}
	return n
}

// Zig zigzag-encodes a signed integer so that small-magnitude values (positive
// or negative) map to small unsigned values, which a varint then encodes
// compactly. It is the inverse of [Zag].
func Zig(v int64) uint64 { return uint64((v << 1) ^ (v >> 63)) }

// Zag reverses [Zig], recovering the original signed integer from its
// zigzag-encoded form.
func Zag(v uint64) int64 { return int64(v>>1) ^ -int64(v&1) }
