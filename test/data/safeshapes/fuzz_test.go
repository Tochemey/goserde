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
	"testing"

	"github.com/tochemey/goserde/runtime"
)

// fuzzDecode asserts the core safe-mode contract on ARBITRARY input: Unmarshal
// never panics, only ever returns nil or runtime.ErrShortBuffer, and on success
// reports a sane consumed-byte count. (codec, marshal seed via Size/Marshal, and
// sampleAll live in safe_test.go — same package.)
func fuzzDecode(f *testing.F, fresh func() codec, seed codec) {
	buf := make([]byte, seed.Size())
	n := seed.Marshal(buf)
	f.Add(buf[:n])                        // a valid encoding
	f.Add([]byte{})                       // empty
	f.Add([]byte{0})                      // single byte
	f.Add([]byte{0xff, 0xff, 0xff, 0xff}) // truncated varint / garbage
	f.Fuzz(func(t *testing.T, data []byte) {
		v := fresh()
		n, err := v.Unmarshal(data)
		if err != nil && err != runtime.ErrShortBuffer {
			t.Fatalf("unexpected error (want nil or ErrShortBuffer): %v", err)
		}
		if err == nil && (n < 0 || n > len(data)) {
			t.Fatalf("decoded %d bytes of a %d-byte buffer", n, len(data))
		}
	})
}

func FuzzFixedDecode(f *testing.F) {
	fuzzDecode(f, func() codec { return &Fixed{} }, &Fixed{X: 1, Y: -2, Z: 3})
}

func FuzzInnerDecode(f *testing.F) {
	fuzzDecode(f, func() codec { return &Inner{} }, &Inner{A: 7, B: "inner"})
}

func FuzzArraysDecode(f *testing.F) {
	fuzzDecode(f, func() codec { return &Arrays{} }, sampleArrays())
}

func FuzzAllDecode(f *testing.F) {
	fuzzDecode(f, func() codec { return &All{} }, sampleAll())
}
