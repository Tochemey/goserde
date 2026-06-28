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

import (
	"github.com/mus-format/mus-go/ord"
	"github.com/mus-format/mus-go/unsafe"
	"github.com/mus-format/mus-go/varint"
)

// mus serializer for Record using mus's fastest (unsafe) primitives.
var tagsSer = ord.NewSliceSer[uint32](varint.Uint32)

func musSize(r *Record) int {
	s := unsafe.Uint64.Size(r.ID)
	s += unsafe.Float64.Size(r.Score)
	s += unsafe.Bool.Size(r.Active)
	s += unsafe.String.Size(r.Name)
	s += tagsSer.Size(r.Tags)
	s += unsafe.ByteSlice.Size(r.Blob)
	return s
}

func musMarshal(r *Record, b []byte) int {
	n := unsafe.Uint64.Marshal(r.ID, b)
	n += unsafe.Float64.Marshal(r.Score, b[n:])
	n += unsafe.Bool.Marshal(r.Active, b[n:])
	n += unsafe.String.Marshal(r.Name, b[n:])
	n += tagsSer.Marshal(r.Tags, b[n:])
	n += unsafe.ByteSlice.Marshal(r.Blob, b[n:])
	return n
}

func musUnmarshal(b []byte) (r Record, n int, err error) {
	r.ID, n, err = unsafe.Uint64.Unmarshal(b)
	var c int
	r.Score, c, err = unsafe.Float64.Unmarshal(b[n:])
	n += c
	r.Active, c, err = unsafe.Bool.Unmarshal(b[n:])
	n += c
	r.Name, c, err = unsafe.String.Unmarshal(b[n:])
	n += c
	r.Tags, c, err = tagsSer.Unmarshal(b[n:])
	n += c
	r.Blob, c, err = unsafe.ByteSlice.Unmarshal(b[n:])
	n += c
	return
}
