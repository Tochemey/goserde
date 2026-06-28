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

package safeshapes

import (
	"reflect"
	"testing"

	"github.com/tochemey/goserde/runtime"
)

// codec is satisfied by every generated *T (pointer receiver).
type codec interface {
	Size() int
	Marshal([]byte) int
	Unmarshal([]byte) (int, error)
}

func sampleAll() *All {
	p := uint32(99)
	return &All{
		Flag:   true,
		N8:     -5,
		N16:    600,
		N32:    -70000,
		N64:    1 << 40,
		F32:    3.14,
		F64:    2.718281828,
		Name:   "hello world",
		Blob:   []byte{1, 2, 3, 4, 5},
		Tags:   []string{"a", "bb", "ccc"},
		Nums:   []int32{10, 20, 30},
		Scores: map[string]int32{"x": 1, "y": 2},
		Ptr:    &p,
		Inner:  Inner{A: 7, B: "inner"},
		Nested: &Inner{A: 8, B: "nested"},
	}
}

// marshal returns the exact wire bytes of v.
func marshal(t *testing.T, v codec) []byte {
	t.Helper()
	buf := make([]byte, v.Size())
	n := v.Marshal(buf)
	return buf[:n]
}

func checkRoundTrip(t *testing.T, in, out codec) {
	t.Helper()
	buf := marshal(t, in)
	n, err := out.Unmarshal(buf)
	if err != nil {
		t.Fatalf("Unmarshal returned error on valid buffer: %v", err)
	}
	if n != len(buf) {
		t.Fatalf("Unmarshal consumed %d bytes, want %d", n, len(buf))
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("round-trip mismatch:\n in = %+v\nout = %+v", in, out)
	}
}

func sampleArrays() *Arrays {
	return &Arrays{
		Hash: [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
		Quad: [4]int32{-1, 2, -3, 4},
		Tags: [2]string{"a", "bb"},
	}
}

func sampleFrame() *Frame {
	return &Frame{ID: 42, Sig: &Ping{Seq: 7}}
}

func TestSafeRoundTrip(t *testing.T) {
	checkRoundTrip(t, &Fixed{X: 1, Y: -2, Z: 3}, &Fixed{})
	checkRoundTrip(t, &Inner{A: 7, B: "inner"}, &Inner{})
	checkRoundTrip(t, sampleArrays(), &Arrays{})
	checkRoundTrip(t, sampleAll(), &All{})
	checkRoundTrip(t, sampleFrame(), &Frame{})
}

// TestSafeTruncationNeverPanics is the core safe-mode guarantee: every truncated
// prefix of a valid buffer must return runtime.ErrShortBuffer, never panic.
func TestSafeTruncationNeverPanics(t *testing.T) {
	cases := []struct {
		name  string
		fresh func() codec
		val   codec
	}{
		{"Fixed", func() codec { return &Fixed{} }, &Fixed{X: 1, Y: -2, Z: 3}},
		{"Inner", func() codec { return &Inner{} }, &Inner{A: 7, B: "inner"}},
		{"Arrays", func() codec { return &Arrays{} }, sampleArrays()},
		{"All", func() codec { return &All{} }, sampleAll()},
		{"Frame", func() codec { return &Frame{} }, sampleFrame()},
	}
	for _, tc := range cases {
		full := marshal(t, tc.val)
		for k := range len(full) {
			trunc := full[:k:k] // cap=k so an unchecked read can't alias beyond the prefix
			err := callNoPanic(t, tc.name, k, func() error {
				_, e := tc.fresh().Unmarshal(trunc)
				return e
			})
			if err != runtime.ErrShortBuffer {
				t.Errorf("%s prefix[:%d]: got err=%v, want runtime.ErrShortBuffer", tc.name, k, err)
			}
		}
		// The complete buffer must still decode cleanly.
		if _, err := tc.fresh().Unmarshal(full); err != nil {
			t.Errorf("%s full buffer: unexpected error %v", tc.name, err)
		}
	}
}

// TestSafeCorruptLength feeds an oversized length prefix and asserts the decoder
// rejects it (uint64 comparison stays correct for huge nn) instead of attempting
// a giant slice/make or panicking.
func TestSafeCorruptLength(t *testing.T) {
	buf := make([]byte, 16)
	runtime.PutU16(buf[0:], 1)               // Inner.A
	cc := runtime.PutUvarint(buf[2:], 1<<50) // Inner.B claims a 1-petabyte string
	if _, err := (&Inner{}).Unmarshal(buf[:2+cc]); err != runtime.ErrShortBuffer {
		t.Fatalf("oversized length: got err=%v, want runtime.ErrShortBuffer", err)
	}
}

// TestSafeUnknownUnionTag asserts safe-mode union decode rejects an undeclared
// member tag with ErrUnknownUnionTag instead of panicking.
func TestSafeUnknownUnionTag(t *testing.T) {
	buf := marshal(t, sampleFrame())
	buf[2] = 99 // tag sits right after the 2-byte ID; 99 is not a declared member

	if _, err := (&Frame{}).Unmarshal(buf); err != runtime.ErrUnknownUnionTag {
		t.Fatalf("unknown tag: got err=%v, want runtime.ErrUnknownUnionTag", err)
	}
}

// TestPortableDecodeOwnsBytes verifies that safe (portable) mode copies
// length-prefixed payloads rather than aliasing the input buffer, so a decoded
// value survives mutation of the source bytes. The same test against a fast-mode
// codec would fail, since fast mode decodes string/[]byte zero-copy.
func TestPortableDecodeOwnsBytes(t *testing.T) {
	in := All{Name: "hello", Blob: []byte("world")}
	buf := make([]byte, in.Size())
	in.Marshal(buf)

	var out All
	if _, err := out.Unmarshal(buf); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	for i := range buf {
		buf[i] = 0xFF // scribble the source; an aliasing decode would corrupt out
	}

	if out.Name != "hello" {
		t.Errorf("Name aliased the input buffer: got %q, want %q", out.Name, "hello")
	}

	if string(out.Blob) != "world" {
		t.Errorf("Blob aliased the input buffer: got %q, want %q", out.Blob, "world")
	}
}

func callNoPanic(t *testing.T, name string, k int, f func() error) (err error) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("%s prefix[:%d]: Unmarshal panicked (should return ErrShortBuffer): %v", name, k, r)
		}
	}()
	return f()
}
