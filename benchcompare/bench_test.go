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
	"reflect"
	"testing"
)

func sample() *Record {
	return &Record{
		ID: 0xDEADBEEFCAFE, Score: 3.14159265358979, Active: true,
		Name: "goserde-serializer-record",
		Tags: []uint32{1, 2, 3, 5, 8, 13, 21, 34, 55, 89},
		Blob: []byte("the quick brown fox jumps over the lazy dog 0123456789"),
	}
}

func TestBothRoundTrip(t *testing.T) {
	in := sample()

	tb := make([]byte, in.Size())
	in.Marshal(tb)
	var tout Record
	tout.Unmarshal(tb)
	if !reflect.DeepEqual(in, &tout) {
		t.Fatalf("goserde mismatch")
	}

	mb := make([]byte, musSize(in))
	musMarshal(in, mb)
	mout, _, err := musUnmarshal(mb)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, &mout) {
		t.Fatalf("mus mismatch")
	}

	t.Logf("goserde size=%d  mus size=%d", len(tb), len(mb))
}

// ---- goserde ----
func BenchmarkGoserde_Marshal(b *testing.B) {
	in := sample()
	buf := make([]byte, in.Size())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = in.Marshal(buf)
	}
}
func BenchmarkGoserde_Unmarshal(b *testing.B) {
	in := sample()
	buf := make([]byte, in.Size())
	in.Marshal(buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var o Record
		o.Unmarshal(buf)
	}
}

// ---- mus (unsafe mode) ----
func BenchmarkMus_Marshal(b *testing.B) {
	in := sample()
	buf := make([]byte, musSize(in))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = musMarshal(in, buf)
	}
}
func BenchmarkMus_Unmarshal(b *testing.B) {
	in := sample()
	buf := make([]byte, musSize(in))
	musMarshal(in, buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = musUnmarshal(buf)
	}
}

func TestRawAndBencRoundTrip(t *testing.T) {
	in := sample()

	rb := make([]byte, musRawSize(in))
	musRawMarshal(in, rb)
	rout, _, err := musRawUnmarshal(rb)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, &rout) {
		t.Fatalf("mus-raw mismatch:\n%+v\n%+v", in, &rout)
	}

	bb := make([]byte, bencSize(in))
	bencMarshal(in, bb)
	bout, err := bencUnmarshal(bb)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, &bout) {
		t.Fatalf("benc mismatch:\n%+v\n%+v", in, &bout)
	}

	t.Logf("sizes: goserde=%d mus-varint=%d mus-raw=%d benc=%d",
		in.Size(), musSize(in), musRawSize(in), bencSize(in))
}

// ---- mus raw (fixed-width) ----
func BenchmarkMusRaw_Marshal(b *testing.B) {
	in := sample()
	buf := make([]byte, musRawSize(in))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = musRawMarshal(in, buf)
	}
}
func BenchmarkMusRaw_Unmarshal(b *testing.B) {
	in := sample()
	buf := make([]byte, musRawSize(in))
	musRawMarshal(in, buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = musRawUnmarshal(buf)
	}
}

// ---- benc ----
func BenchmarkBenc_Marshal(b *testing.B) {
	in := sample()
	buf := make([]byte, bencSize(in))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bencMarshal(in, buf)
	}
}
func BenchmarkBenc_Unmarshal(b *testing.B) {
	in := sample()
	buf := make([]byte, bencSize(in))
	bencMarshal(in, buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = bencUnmarshal(buf)
	}
}
