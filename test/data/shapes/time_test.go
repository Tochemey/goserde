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
	"testing"
	"time"
)

// nanosEqual compares two time.Time at nanosecond precision. We cannot use ==
// or reflect.DeepEqual because the wire format stores only unix-nanoseconds:
// the decoded value is always UTC and carries no monotonic clock reading.
func nanosEqual(a, b time.Time) bool { return a.UnixNano() == b.UnixNano() }

func TestTimeRoundTrip(t *testing.T) {
	// A time with sub-second precision and a non-UTC location, to prove the
	// instant survives even though the location/monotonic parts do not.
	loc := time.FixedZone("UTC+5", 5*3600)
	created := time.Date(2023, 6, 15, 10, 30, 0, 123456789, loc)
	updated := created.Add(48 * time.Hour)

	in := TimeStruct{
		Label:   "event",
		Created: created,
		Updated: &updated,
		Stamps:  []time.Time{created, updated, created.Add(-time.Hour)},
	}

	buf := make([]byte, in.Size())
	n := in.Marshal(buf)
	var out TimeStruct
	rn, err := out.Unmarshal(buf[:n])
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if rn != n {
		t.Fatalf("consumed %d bytes, want %d", rn, n)
	}

	if out.Label != in.Label {
		t.Errorf("Label = %q, want %q", out.Label, in.Label)
	}
	if !nanosEqual(out.Created, in.Created) {
		t.Errorf("Created = %d, want %d (unix nanos)", out.Created.UnixNano(), in.Created.UnixNano())
	}
	if out.Created.Location() != time.UTC {
		t.Errorf("Created location = %v, want UTC", out.Created.Location())
	}
	if out.Updated == nil {
		t.Fatal("Updated decoded as nil, want non-nil")
	}
	if !nanosEqual(*out.Updated, *in.Updated) {
		t.Errorf("Updated = %d, want %d", out.Updated.UnixNano(), in.Updated.UnixNano())
	}
	if len(out.Stamps) != len(in.Stamps) {
		t.Fatalf("Stamps len = %d, want %d", len(out.Stamps), len(in.Stamps))
	}
	for i := range in.Stamps {
		if !nanosEqual(out.Stamps[i], in.Stamps[i]) {
			t.Errorf("Stamps[%d] = %d, want %d", i, out.Stamps[i].UnixNano(), in.Stamps[i].UnixNano())
		}
	}
}

// TestTimeNilAndEmpty covers the nil pointer and nil slice cases.
func TestTimeNilAndEmpty(t *testing.T) {
	in := TimeStruct{
		Label:   "z",
		Created: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		Updated: nil,
		Stamps:  nil,
	}
	buf := make([]byte, in.Size())
	n := in.Marshal(buf)
	var out TimeStruct
	if _, err := out.Unmarshal(buf[:n]); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if out.Updated != nil {
		t.Errorf("Updated = %v, want nil", out.Updated)
	}
	if out.Stamps != nil {
		t.Errorf("Stamps = %v, want nil", out.Stamps)
	}
	if !nanosEqual(out.Created, in.Created) {
		t.Errorf("Created = %d, want %d", out.Created.UnixNano(), in.Created.UnixNano())
	}
}
