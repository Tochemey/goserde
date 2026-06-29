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

// Record is a representative payload: a mix of fixed-width numerics, a string,
// a byte slice, and a slice of fixed-width values — the shape that exercises
// every primitive. The hand-written codec below is the exact shape the
// generator emits and serves as the reference template for cmd/goserdegen.
type Record struct {
	ID     uint64   // fixed-width 8-byte identifier
	Score  float64  // fixed-width 8-byte IEEE 754 value
	Active bool     // single-byte flag
	Name   string   // varint length prefix + raw bytes, decoded zero-copy
	Tags   []uint32 // varint count + fixed-width elements
	Blob   []byte   // varint length prefix + raw bytes, decoded zero-copy
}

// Size returns the exact number of bytes Marshal will write, so the caller can
// allocate the destination buffer once and the codec can skip per-write bounds
// checks.
func (r *Record) Size() int {
	s := 8 + 8 + 1 // ID, Score, Active
	s += UvarintSize(uint64(len(r.Name))) + len(r.Name)
	s += UvarintSize(uint64(len(r.Tags))) + 4*len(r.Tags)
	s += UvarintSize(uint64(len(r.Blob))) + len(r.Blob)
	return s
}

// Marshal writes the record into b in declaration order and returns the number
// of bytes written. b must be at least Size() bytes long.
func (r *Record) Marshal(b []byte) int {
	PutU64(b[0:], r.ID)
	PutU64(b[8:], f64bits(r.Score))

	if r.Active {
		b[16] = 1
	} else {
		b[16] = 0
	}

	i := 17

	i += PutUvarint(b[i:], uint64(len(r.Name)))
	i += copy(b[i:], S2B(r.Name))

	i += PutUvarint(b[i:], uint64(len(r.Tags)))

	for _, t := range r.Tags {
		PutU32(b[i:], t)
		i += 4
	}

	i += PutUvarint(b[i:], uint64(len(r.Blob)))
	i += copy(b[i:], r.Blob)
	return i
}

// Unmarshal reads the record back from b in the same order Marshal wrote it and
// returns the number of bytes consumed. Name and Blob alias b (zero-copy), so
// the caller must copy them if it needs ownership independent of b's lifetime.
// This reference codec is unchecked and assumes trusted, complete input.
func (r *Record) Unmarshal(b []byte) (int, error) {
	r.ID = U64(b[0:])
	r.Score = bitsf64(U64(b[8:]))
	r.Active = b[16] != 0
	i := 17

	n, c := Uvarint(b[i:])
	i += c
	r.Name = B2S(b[i : i+int(n)]) // zero-copy: aliases the input buffer
	i += int(n)

	n, c = Uvarint(b[i:])
	i += c
	r.Tags = make([]uint32, n)

	for j := range r.Tags {
		r.Tags[j] = U32(b[i:])
		i += 4
	}

	n, c = Uvarint(b[i:])
	i += c
	r.Blob = b[i : i+int(n)] // alias; copy if caller needs ownership
	i += int(n)
	return i, nil
}
