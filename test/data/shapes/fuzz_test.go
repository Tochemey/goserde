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

package shapes

import (
	"reflect"
	"testing"
)

// rtCodec is satisfied by every generated *T.
type rtCodec interface {
	Size() int
	Marshal([]byte) int
	Unmarshal([]byte) (int, error)
}

// rtFuzz marshals in, decodes into out, and asserts a faithful round-trip:
// Size matches bytes written, the whole buffer is consumed, no decode error,
// and the decoded value deep-equals the original.
func rtFuzz(t *testing.T, in, out rtCodec) {
	t.Helper()
	buf := make([]byte, in.Size())
	n := in.Marshal(buf)
	if n != len(buf) {
		t.Fatalf("%T: Size=%d but Marshal wrote %d", in, len(buf), n)
	}
	rn, err := out.Unmarshal(buf)
	if err != nil {
		t.Fatalf("%T: Unmarshal of valid bytes errored: %v", in, err)
	}
	if rn != n {
		t.Fatalf("%T: Unmarshal consumed %d, want %d", in, rn, n)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("%T round-trip mismatch:\n in=%+v\nout=%+v", in, in, out)
	}
}

func nilIfEmpty(b []byte) []byte {
	if len(b) == 0 {
		return nil // wire contract: zero-length collections decode as nil
	}
	return b
}

// FuzzRoundTrip drives the diverse fast-mode shapes with hand-rolled values
// derived from fuzz inputs, deliberately toggling nil vs non-nil collections and
// pointers (where the two historical bugs lived). Floats are derived to stay
// finite so reflect.DeepEqual (which treats NaN != NaN) is reliable.
func FuzzRoundTrip(f *testing.F) {
	f.Add("name", "body", int64(42), uint64(7), []byte("data"), uint8(0xFF))
	f.Add("", "", int64(0), uint64(0), []byte(nil), uint8(0))
	f.Add("k", "k", int64(-1), uint64(1<<63), []byte{0}, uint8(0b101010))
	f.Fuzz(func(t *testing.T, s1, s2 string, i64 int64, u64 uint64, blob []byte, mask uint8) {
		ff := float64(i64%1_000_000) / 3.0 // always finite

		fm := &FlatMixed{ID: u64, Ratio: ff, Name: s1, Data: nilIfEmpty(blob)}
		rtFuzz(t, fm, &FlatMixed{})

		sh := &StringHeavy{Title: s1, Body: s2}
		if mask&1 != 0 {
			sh.Tags = []string{s1, s2}
		}
		rtFuzz(t, sh, &StringHeavy{})

		ns := &Nested{Label: s1, Pos: Inner{X: int32(i64), Y: int32(u64)}}
		if mask&2 != 0 {
			ns.Opt = &Inner{X: int32(u64), Y: int32(i64)}
		}
		if mask&4 != 0 {
			ns.Path = []Inner{{1, 2}, {int32(i64), int32(u64)}}
		}
		rtFuzz(t, ns, &Nested{})

		ch := &CollectionHeavy{}
		if mask&8 != 0 {
			ch.Counts = map[string]int64{s1: i64, s2: int64(u64)}
		}
		if mask&16 != 0 {
			ch.Floats = []float64{ff, -ff}
		}
		if mask&32 != 0 {
			ch.Names = []string{s1, s2}
		}
		rtFuzz(t, ch, &CollectionHeavy{})
	})
}
