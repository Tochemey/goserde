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

// Package safeshapes holds types whose codecs are generated in SAFE mode
// (goserdegen -safe), so Unmarshal is bounds-checked and returns
// codec.ErrShortBuffer on truncated input instead of panicking. Safe mode is
// also the portable mode: no memmove and copying (non-aliasing) decode. It
// exists to exercise every safe-mode guard path; the types deliberately cover
// fixed, variable-length, collection, pointer, and nested shapes.
package safeshapes

//go:generate go run ../../../cmd/goserdegen -dir . -out goserde_gen.go -safe

// Fixed is all-fixed-width: in fast mode it would take the blittable memmove
// path, but safe mode encodes it field-by-field (portable little-endian),
// exercising the per-field length guards.
//
//goserde:generate
type Fixed struct {
	X int32 // fixed-width
	Y int32 // fixed-width
	Z int32 // fixed-width
}

// Inner is a non-blittable nested target (carries a string).
//
//goserde:generate
type Inner struct {
	A uint16 // fixed-width
	B string // length-prefixed bytes
}

// Arrays exercises safe-mode guards on fixed-size arrays: a bulk-copied byte
// array, an array of fixed-width elements (per-element guard), and an array of
// length-prefixed elements.
//
//goserde:generate
type Arrays struct {
	Hash [16]byte  // byte array, bulk-copied with a single length guard
	Quad [4]int32  // array of fixed-width elements, per-element guard
	Tags [2]string // array of length-prefixed elements, per-element guard
}

// All exercises every non-blittable safe-mode guard: each fixed width, the
// varint length prefix, length-prefixed payloads (string/[]byte), variable and
// fixed element slices, a map, a pointer to a basic, a value nested struct, and
// a pointer to a nested struct (error propagation).
//
//goserde:generate
type All struct {
	Flag   bool             // single-byte flag
	N8     int8             // 1-byte integer
	N16    uint16           // 2-byte integer
	N32    int32            // 4-byte integer
	N64    uint64           // 8-byte integer
	F32    float32          // 4-byte IEEE 754
	F64    float64          // 8-byte IEEE 754
	Name   string           // length-prefixed bytes
	Blob   []byte           // length-prefixed raw bytes
	Tags   []string         // slice of length-prefixed strings
	Nums   []int32          // slice of fixed-width elements
	Scores map[string]int32 // map with length-prefixed keys
	Ptr    *uint32          // optional fixed-width value (1-byte nil flag)
	Inner  Inner            // value nested struct
	Nested *Inner           // optional nested struct (error propagation path)
}

// Ping is a Signal union member.
//
//goserde:generate
type Ping struct {
	Seq uint32 // sequence number
}

func (*Ping) isSignal() {}

// Pong is a Signal union member.
//
//goserde:generate
type Pong struct {
	Seq uint32 // sequence number
}

func (*Pong) isSignal() {}

// Signal is a tagged union over Ping and Pong, exercising safe-mode union decode
// (bounds-checked tag, member error propagation, unknown-tag rejection).
//
//goserde:union Ping Pong
type Signal interface {
	isSignal()
}

// Frame carries a fixed-width field and a union field.
//
//goserde:generate
type Frame struct {
	ID  uint16 // fixed-width identifier
	Sig Signal // union field
}
