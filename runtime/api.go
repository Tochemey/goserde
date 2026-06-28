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

// Marshaler is implemented by every generated codec. Size reports the exact
// number of bytes Marshal will write, letting callers size a buffer once.
type Marshaler interface {
	// Size returns the exact encoded length in bytes.
	Size() int
	// Marshal encodes the value into b, which must be at least Size() bytes, and
	// returns the number of bytes written.
	Marshal(b []byte) int
}

// Unmarshaler is implemented by every generated codec.
type Unmarshaler interface {
	// Unmarshal decodes the value from b and returns the number of bytes
	// consumed. Safe-mode codecs return ErrShortBuffer on truncated input.
	Unmarshal(b []byte) (int, error)
}

// Bytes encodes m into a freshly allocated, exactly sized buffer and returns it.
// It is the convenient one-liner for the common "give me the bytes" case; use
// Into when you want to reuse a buffer across calls.
func Bytes(m Marshaler) []byte {
	b := make([]byte, m.Size())
	m.Marshal(b)

	return b
}

// Into encodes m into buf, reusing buf when its capacity is sufficient and
// allocating a new buffer otherwise, and returns the slice holding the encoded
// bytes. Reusing a buffer across calls keeps Marshal allocation-free.
func Into(m Marshaler, buf []byte) []byte {
	n := m.Size()
	if cap(buf) < n {
		buf = make([]byte, n)
	} else {
		buf = buf[:n]
	}

	m.Marshal(buf)

	return buf
}

// From decodes b into u and returns only the error, discarding the consumed-byte
// count for callers that pass a buffer holding exactly one value.
func From(u Unmarshaler, b []byte) error {
	_, err := u.Unmarshal(b)

	return err
}
