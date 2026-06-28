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

package runtime

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"reflect"
	"testing"
)

// sample returns a representative Record used by the round-trip test and the
// comparative benchmarks.
func sample() *Record {
	return &Record{
		ID:     0xDEADBEEFCAFE,
		Score:  3.14159265358979,
		Active: true,
		Name:   "goserde-serializer-record",
		Tags:   []uint32{1, 2, 3, 5, 8, 13, 21, 34, 55, 89},
		Blob:   []byte("the quick brown fox jumps over the lazy dog 0123456789"),
	}
}

func TestRoundTrip(t *testing.T) {
	in := sample()
	buf := make([]byte, in.Size())

	n := in.Marshal(buf)
	if n != len(buf) {
		t.Fatalf("Size()=%d but Marshal wrote %d", len(buf), n)
	}

	var out Record
	if _, err := out.Unmarshal(buf); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(in, &out) {
		t.Fatalf("round-trip mismatch:\n in=%+v\nout=%+v", in, &out)
	}

	t.Logf("goserde payload size: %d bytes", len(buf))

	j, _ := json.Marshal(in)
	t.Logf("json  payload size: %d bytes", len(j))
}

// TestRecordVariants covers the encoder branches the canonical sample does not:
// a false Active flag and nil/empty collections, which decode back to nil per
// the wire contract.
func TestRecordVariants(t *testing.T) {
	cases := []*Record{
		{ID: 0, Score: -2.5, Active: false, Name: "", Tags: nil, Blob: nil},
		{ID: 1, Score: 0, Active: false, Name: "x", Tags: []uint32{}, Blob: []byte{}},
	}

	for i, in := range cases {
		buf := make([]byte, in.Size())

		if n := in.Marshal(buf); n != len(buf) {
			t.Fatalf("case %d: Marshal wrote %d, Size reported %d", i, n, len(buf))
		}

		var out Record
		if _, err := out.Unmarshal(buf); err != nil {
			t.Fatalf("case %d: Unmarshal: %v", i, err)
		}

		if out.ID != in.ID || out.Score != in.Score || out.Active != in.Active || out.Name != in.Name {
			t.Errorf("case %d: scalar mismatch: %+v", i, out)
		}

		if len(out.Tags) != 0 || len(out.Blob) != 0 {
			t.Errorf("case %d: zero-length collections should decode empty, got %+v", i, out)
		}
	}
}

// BenchmarkGoserdeMarshal measures Marshal into a reused buffer, the intended
// hot-path usage.
func BenchmarkGoserdeMarshal(b *testing.B) {
	in := sample()
	buf := make([]byte, in.Size())
	b.ReportAllocs()
	for b.Loop() {
		_ = in.Marshal(buf)
	}
}

// BenchmarkGoserdeUnmarshal measures Unmarshal from a pre-marshaled buffer.
func BenchmarkGoserdeUnmarshal(b *testing.B) {
	in := sample()
	buf := make([]byte, in.Size())
	in.Marshal(buf)
	b.ReportAllocs()
	for b.Loop() {
		var out Record
		_, _ = out.Unmarshal(buf)
	}
}

// BenchmarkJSONMarshal is the encoding/json marshal baseline for comparison.
func BenchmarkJSONMarshal(b *testing.B) {
	in := sample()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = json.Marshal(in)
	}
}

// BenchmarkJSONUnmarshal is the encoding/json unmarshal baseline for comparison.
func BenchmarkJSONUnmarshal(b *testing.B) {
	in := sample()
	data, _ := json.Marshal(in)
	b.ReportAllocs()
	for b.Loop() {
		var out Record
		_ = json.Unmarshal(data, &out)
	}
}

// BenchmarkGobMarshal is the encoding/gob baseline with the encoder reused,
// gob's best case.
func BenchmarkGobMarshal(b *testing.B) {
	in := sample()

	var buf bytes.Buffer

	enc := gob.NewEncoder(&buf)
	b.ReportAllocs()
	for b.Loop() {
		buf.Reset()
		_ = enc.Encode(in)
	}
}
