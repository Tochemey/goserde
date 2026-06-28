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

import bstd "github.com/deneonet/benc/std"

// benc codec for Record, fixed-width numerics + unsafe string, to match goserde.
func bencSize(r *Record) int {
	s := bstd.SizeUint64()              // ID
	s += bstd.SizeFloat64()             // Score
	s += bstd.SizeBool()                // Active
	s += bstd.SizeString(r.Name)        // Name
	s += bstd.SizeFixedSlice(r.Tags, 4) // Tags (uint32 = 4 bytes each)
	s += bstd.SizeBytes(r.Blob)         // Blob
	return s
}
func bencMarshal(r *Record, b []byte) int {
	n := bstd.MarshalUint64(0, b, r.ID)
	n = bstd.MarshalFloat64(n, b, r.Score)
	n = bstd.MarshalBool(n, b, r.Active)
	n = bstd.MarshalString(n, b, r.Name)
	n = bstd.MarshalSlice(n, b, r.Tags, bstd.MarshalUint32)
	n = bstd.MarshalBytes(n, b, r.Blob)
	return n
}
func bencUnmarshal(b []byte) (r Record, err error) {
	n, id, err := bstd.UnmarshalUint64(0, b)
	if err != nil {
		return
	}
	r.ID = id
	var sc float64
	n, sc, err = bstd.UnmarshalFloat64(n, b)
	if err != nil {
		return
	}
	r.Score = sc
	var ac bool
	n, ac, err = bstd.UnmarshalBool(n, b)
	if err != nil {
		return
	}
	r.Active = ac
	var nm string
	n, nm, err = bstd.UnmarshalUnsafeString(n, b)
	if err != nil {
		return
	}
	r.Name = nm
	var tags []uint32
	n, tags, err = bstd.UnmarshalSlice[uint32](n, b, bstd.UnmarshalUint32)
	if err != nil {
		return
	}
	r.Tags = tags
	var blob []byte
	_, blob, err = bstd.UnmarshalBytesCropped(n, b)
	if err != nil {
		return
	}
	r.Blob = blob
	return
}
