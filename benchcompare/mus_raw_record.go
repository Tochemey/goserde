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
	"github.com/mus-format/mus-go/raw"
	"github.com/mus-format/mus-go/unsafe"
)

// mus in RAW (fixed-width) mode — the fair format match to goserde.
// raw covers numerics; string/byteslice still come from unsafe; slice from ord.
var tagsRawSer = ord.NewSliceSer[uint32](raw.Uint32)

func musRawSize(r *Record) int {
	s := raw.Uint64.Size(r.ID)
	s += raw.Float64.Size(r.Score)
	s += unsafe.Bool.Size(r.Active)
	s += unsafe.String.Size(r.Name)
	s += tagsRawSer.Size(r.Tags)
	s += unsafe.ByteSlice.Size(r.Blob)
	return s
}
func musRawMarshal(r *Record, b []byte) int {
	n := raw.Uint64.Marshal(r.ID, b)
	n += raw.Float64.Marshal(r.Score, b[n:])
	n += unsafe.Bool.Marshal(r.Active, b[n:])
	n += unsafe.String.Marshal(r.Name, b[n:])
	n += tagsRawSer.Marshal(r.Tags, b[n:])
	n += unsafe.ByteSlice.Marshal(r.Blob, b[n:])
	return n
}
func musRawUnmarshal(b []byte) (r Record, n int, err error) {
	r.ID, n, err = raw.Uint64.Unmarshal(b)
	var c int
	r.Score, c, _ = raw.Float64.Unmarshal(b[n:])
	n += c
	r.Active, c, _ = unsafe.Bool.Unmarshal(b[n:])
	n += c
	r.Name, c, _ = unsafe.String.Unmarshal(b[n:])
	n += c
	r.Tags, c, _ = tagsRawSer.Unmarshal(b[n:])
	n += c
	r.Blob, c, _ = unsafe.ByteSlice.Unmarshal(b[n:])
	n += c
	return
}
