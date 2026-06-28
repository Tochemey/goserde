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

import "testing"

// TestTaggedFieldExcluded verifies that fields tagged `goserde:"-"` are absent
// from the wire and decode back to their zero value.
func TestTaggedFieldExcluded(t *testing.T) {
	a := Tagged{ID: 7, Secret: "hunter2", Skip: make(chan int), Name: "alice"}
	b := Tagged{ID: 7, Secret: "a-much-longer-different-secret", Skip: nil, Name: "alice"}

	// Secret is excluded, so changing it must not change the encoded size.
	if a.Size() != b.Size() {
		t.Fatalf("excluded Secret affected Size: %d vs %d", a.Size(), b.Size())
	}

	buf := make([]byte, a.Size())
	a.Marshal(buf)

	var out Tagged
	if _, err := out.Unmarshal(buf); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	// Serialized fields round-trip.
	if out.ID != a.ID || out.Name != a.Name {
		t.Errorf("serialized fields mismatch: got ID=%d Name=%q", out.ID, out.Name)
	}

	// Excluded fields stay at their zero value on decode.
	if out.Secret != "" {
		t.Errorf("excluded Secret = %q, want empty", out.Secret)
	}

	if out.Skip != nil {
		t.Error("excluded Skip channel = non-nil, want nil")
	}
}
