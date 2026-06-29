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

package benchcompare

import "github.com/tochemey/goserde/codec"

// Record mirrors codec.Record exactly, so goserde and mus serialize identical data.
type Record struct {
	ID     uint64
	Score  float64
	Active bool
	Name   string
	Tags   []uint32
	Blob   []byte
}

// --- goserde codec (hand-tuned, identical shape to generator output) ---

func (r *Record) Size() int {
	s := 8 + 8 + 1
	s += codec.UvarintSize(uint64(len(r.Name))) + len(r.Name)
	s += codec.UvarintSize(uint64(len(r.Tags))) + 4*len(r.Tags)
	s += codec.UvarintSize(uint64(len(r.Blob))) + len(r.Blob)
	return s
}

func (r *Record) Marshal(b []byte) int {
	codec.PutU64(b[0:], r.ID)
	codec.PutU64(b[8:], codec.F64bits(r.Score))
	if r.Active {
		b[16] = 1
	} else {
		b[16] = 0
	}
	i := 17
	i += codec.PutUvarint(b[i:], uint64(len(r.Name)))
	i += copy(b[i:], codec.S2B(r.Name))
	i += codec.PutUvarint(b[i:], uint64(len(r.Tags)))
	for _, t := range r.Tags {
		codec.PutU32(b[i:], t)
		i += 4
	}
	i += codec.PutUvarint(b[i:], uint64(len(r.Blob)))
	i += copy(b[i:], r.Blob)
	return i
}

func (r *Record) Unmarshal(b []byte) (int, error) {
	r.ID = codec.U64(b[0:])
	r.Score = codec.Bitsf64(codec.U64(b[8:]))
	r.Active = b[16] != 0
	i := 17
	n, c := codec.Uvarint(b[i:])
	i += c
	r.Name = codec.B2S(b[i : i+int(n)])
	i += int(n)
	n, c = codec.Uvarint(b[i:])
	i += c
	r.Tags = make([]uint32, n)
	for j := range r.Tags {
		r.Tags[j] = codec.U32(b[i:])
		i += 4
	}
	n, c = codec.Uvarint(b[i:])
	i += c
	r.Blob = b[i : i+int(n)]
	i += int(n)
	return i, nil
}
