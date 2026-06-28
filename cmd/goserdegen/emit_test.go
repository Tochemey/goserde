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

package main

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testRuntimePkg = "github.com/tochemey/goserde/runtime"

// genFixture writes src as a single package file into a fresh temp directory and
// runs the generator over it, returning the generated source. It fails the test
// if loading or generation errors.
func genFixture(t *testing.T, src string, safe bool) string {
	t.Helper()

	out, err := genFixtureErr(t, src, safe)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if _, perr := parser.ParseFile(token.NewFileSet(), "gen.go", out, 0); perr != nil {
		t.Fatalf("generated code does not parse: %v\n%s", perr, out)
	}

	return out
}

// genFixtureErr writes src to a temp package and returns the generator's output
// or the first error from loading or generation.
func genFixtureErr(t *testing.T, src string, safe bool) (string, error) {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "types.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	g, err := load(dir, testRuntimePkg, safe)
	if err != nil {
		return "", err
	}

	out, err := g.generate()

	return string(out), err
}

// TestGenerateCoversAllTypes generates a struct that exercises every supported
// field type and asserts the output is valid Go with the expected methods.
func TestGenerateCoversAllTypes(t *testing.T) {
	src := `package fixture

import "time"

//goserde:generate
type Inner struct {
	X int32
	S string
}

//goserde:generate
type Sample struct {
	A  uint64
	B  int16
	F  float32
	G  float64
	C  bool
	S  string
	Bl []byte
	Ns []int32
	Ss []string
	Ar [3]int16
	M  map[string]int32
	P  *Inner
	N  Inner
	T  time.Time
	Pt *time.Time
}
`
	out := genFixture(t, src, false)

	for _, want := range []string{
		"func (r *Sample) Size() int",
		"func (r *Sample) Marshal(b []byte) int",
		"func (r *Sample) Unmarshal(b []byte) (int, error)",
		"func (r *Inner) Size() int",
		`"` + testRuntimePkg + `"`,
		`"time"`,
		"runtime.PutUvarint",
		"runtime.B2S",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("generated code missing %q", want)
		}
	}
}

// TestGenerateBlittable checks an all-fixed-width struct takes the memmove path
// and pulls in the unsafe import.
func TestGenerateBlittable(t *testing.T) {
	src := `package fixture

//goserde:generate
type Fixed struct {
	A int32
	B int32
	C float64
	D bool
	H [4]byte
}
`
	out := genFixture(t, src, false)

	if !strings.Contains(out, "unsafe.Sizeof") {
		t.Errorf("expected blittable memmove path (unsafe.Sizeof), got:\n%s", out)
	}

	if !strings.Contains(out, `"unsafe"`) {
		t.Error("expected the unsafe import for the blittable path")
	}
}

// TestGenerateSafeMode checks safe mode emits bounds checks, copies length-
// prefixed payloads, and avoids the unsafe memmove path.
func TestGenerateSafeMode(t *testing.T) {
	src := `package fixture

//goserde:generate
type S struct {
	Name  string
	Blob  []byte
	Fixed [2]int32
}
`
	out := genFixture(t, src, true)

	if !strings.Contains(out, "runtime.ErrShortBuffer") {
		t.Error("safe mode should emit ErrShortBuffer guards")
	}

	if strings.Contains(out, "unsafe") {
		t.Error("safe mode must not use unsafe or memmove")
	}

	if !strings.Contains(out, "string(b[") {
		t.Error("safe mode should copy strings rather than alias the buffer")
	}

	if !strings.Contains(out, "append(") {
		t.Error("safe mode should copy []byte via append")
	}
}

// TestGenerateUnion checks a tagged-union field emits a type switch, the nil
// tag, and the unknown-tag error.
func TestGenerateUnion(t *testing.T) {
	src := `package fixture

//goserde:generate
type A struct{ X int32 }

func (*A) isU() {}

//goserde:generate
type B struct{ Y int32 }

func (*B) isU() {}

//goserde:union A B
type U interface{ isU() }

//goserde:generate
type Holder struct {
	V  U
	Vs []U
}
`
	out := genFixture(t, src, false)

	for _, want := range []string{"case *A:", "case *B:", "runtime.ErrUnknownUnionTag"} {
		if !strings.Contains(out, want) {
			t.Errorf("union output missing %q", want)
		}
	}
}

// TestGenerateExcludesTaggedField checks a goserde:"-" field never reaches the
// wire, even when its type is otherwise unsupported.
func TestGenerateExcludesTaggedField(t *testing.T) {
	src := "package fixture\n\n" +
		"//goserde:generate\n" +
		"type T struct {\n" +
		"\tName string\n" +
		"\tSkip chan int `goserde:\"-\"`\n" +
		"}\n"

	out := genFixture(t, src, false)

	if strings.Contains(out, "Skip") {
		t.Errorf("excluded field Skip leaked into generated code:\n%s", out)
	}
}

// TestGenerateIsDeterministic checks two runs over the same input are identical.
func TestGenerateIsDeterministic(t *testing.T) {
	src := `package fixture

//goserde:generate
type T struct {
	A uint32
	S string
}
`
	if a, b := genFixture(t, src, false), genFixture(t, src, false); a != b {
		t.Error("generator output is not deterministic")
	}
}

// TestLoadErrors checks the loader rejects invalid inputs with a clear message
// rather than emitting broken code.
func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantErr string
	}{
		{
			name:    "no marker",
			src:     "package f\ntype T struct{ X int }\n",
			wantErr: "no types marked",
		},
		{
			name:    "non-struct target",
			src:     "package f\n//goserde:generate\ntype T int\n",
			wantErr: "only supports structs",
		},
		{
			name:    "unsupported field type",
			src:     "package f\n//goserde:generate\ntype T struct{ C chan int }\n",
			wantErr: "unsupported",
		},
		{
			name:    "nested struct not a target",
			src:     "package f\ntype Inner struct{ X int32 }\n//goserde:generate\ntype T struct{ N Inner }\n",
			wantErr: "must also have",
		},
		{
			name:    "union unknown member",
			src:     "package f\n//goserde:generate\ntype A struct{ X int32 }\nfunc (*A) i() {}\n//goserde:union A Missing\ntype U interface{ i() }\n",
			wantErr: "unknown member type",
		},
		{
			name:    "union member not a target",
			src:     "package f\ntype A struct{ X int32 }\nfunc (*A) i() {}\n//goserde:generate\ntype B struct{ Y int32 }\nfunc (*B) i() {}\n//goserde:union A B\ntype U interface{ i() }\n",
			wantErr: "must be a",
		},
		{
			name:    "union member does not implement",
			src:     "package f\n//goserde:generate\ntype A struct{ X int32 }\n//goserde:union A\ntype U interface{ i() }\n",
			wantErr: "does not implement",
		},
		{
			name:    "union with no members",
			src:     "package f\n//goserde:generate\ntype A struct{ X int32 }\n//goserde:union\ntype U interface{ i() }\n",
			wantErr: "no member types",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := genFixtureErr(t, tt.src, false)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("got err=%v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

// TestLoadEmptyDir checks an empty directory is reported rather than panicking.
func TestLoadEmptyDir(t *testing.T) {
	if _, err := load(t.TempDir(), testRuntimePkg, false); err == nil {
		t.Fatal("expected an error for a directory with no Go package")
	}
}

// TestUvarintLen checks the compile-time varint width helper against its
// boundaries.
func TestUvarintLen(t *testing.T) {
	cases := []struct {
		v    uint64
		want int
	}{
		{0, 1}, {127, 1}, {128, 2}, {16383, 2}, {16384, 3}, {1 << 63, 10},
	}

	for _, c := range cases {
		if got := uvarintLen(c.v); got != c.want {
			t.Errorf("uvarintLen(%d)=%d, want %d", c.v, got, c.want)
		}
	}
}

// TestRuntimePkgPath checks the runtime import path is derived from the building
// module.
func TestRuntimePkgPath(t *testing.T) {
	if got := runtimePkgPath(); !strings.HasSuffix(got, "/runtime") {
		t.Errorf("runtimePkgPath()=%q, want a path ending in /runtime", got)
	}
}
