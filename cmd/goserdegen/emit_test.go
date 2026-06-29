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
	"bytes"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/tochemey/goserde/codec"
	"github.com/tochemey/goserde/test/data/safeshapes"
	"github.com/tochemey/goserde/test/data/shapes"
)

const testCodecPkg = "github.com/tochemey/goserde/codec"

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

	g, err := load(dir, testCodecPkg, safe)
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
		`"` + testCodecPkg + `"`,
		`"time"`,
		"codec.PutUvarint",
		"codec.B2S",
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

	if !strings.Contains(out, "codec.ErrShortBuffer") {
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

	for _, want := range []string{"case *A:", "case *B:", "codec.ErrUnknownUnionTag"} {
		if !strings.Contains(out, want) {
			t.Errorf("union output missing %q", want)
		}
	}
}

// TestGenerateNamedByteElement checks that a slice/array whose element is a
// defined byte type takes the element-wise path, not the raw []byte copy that
// would not type-check (regression for the named-element bug).
func TestGenerateNamedByteElement(t *testing.T) {
	src := `package fixture

type Hue uint8

//goserde:generate
type T struct {
	Label string
	Pal   []Hue
	Sw    [3]Hue
}
`
	out := genFixture(t, src, false)

	if !strings.Contains(out, "make([]Hue,") {
		t.Errorf("expected element-wise slice decode (make([]Hue, ...)), got:\n%s", out)
	}

	if !strings.Contains(out, "Hue(b[i])") {
		t.Errorf("expected per-element Hue conversion, got:\n%s", out)
	}

	// The broken fast path aliased the buffer straight into the field.
	if strings.Contains(out, "r.Pal = b[i") {
		t.Error("named-byte slice must not alias the input buffer as []byte")
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
		{
			name:    "unsupported basic type",
			src:     "package f\n//goserde:generate\ntype T struct{ C complex128 }\n",
			wantErr: "unsupported basic type",
		},
		{
			name:    "unsupported map key",
			src:     "package f\n//goserde:generate\ntype T struct{ M map[complex128]int }\n",
			wantErr: "unsupported basic type",
		},
		{
			name:    "anonymous struct field",
			src:     "package f\n//goserde:generate\ntype T struct{ A struct{ X int } }\n",
			wantErr: "anonymous struct",
		},
		{
			name:    "inline interface field",
			src:     "package f\n//goserde:generate\ntype T struct{ E interface{ Foo() } }\n",
			wantErr: "must be declared",
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
	if _, err := load(t.TempDir(), testCodecPkg, false); err == nil {
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

// TestCodecPkgPath checks the codec import path is derived from the building
// module.
func TestCodecPkgPath(t *testing.T) {
	if got := codecPkgPath(); !strings.HasSuffix(got, "/codec") {
		t.Errorf("codecPkgPath()=%q, want a path ending in /codec", got)
	}
}

// TestCodecPkgPathFallback covers codecPkgPath's derived and fallback branches by
// stubbing the build-info reader.
func TestCodecPkgPathFallback(t *testing.T) {
	old := readBuildInfo
	defer func() { readBuildInfo = old }()

	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Path: "example.com/forked"}}, true
	}

	if got, want := codecPkgPath(), "example.com/forked/codec"; got != want {
		t.Errorf("codecPkgPath() = %q, want %q", got, want)
	}

	readBuildInfo = func() (*debug.BuildInfo, bool) { return nil, false }

	if got := codecPkgPath(); got != defaultCodecPkg {
		t.Errorf("fallback codecPkgPath() = %q, want %q", got, defaultCodecPkg)
	}
}

// writePkg writes src as types.go into a fresh temp directory and returns the dir.
func writePkg(t *testing.T, src string) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "types.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	return dir
}

// loadGen loads src into a generator without emitting, failing the test on error.
func loadGen(t *testing.T, src string, safe bool) *generator {
	t.Helper()

	g, err := load(writePkg(t, src), testCodecPkg, safe)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	return g
}

// checkPkg type-checks src and returns a bare generator plus the package scope,
// for unit-testing helpers that take a *types.Scope directly.
func checkPkg(t *testing.T, src string) (*generator, *types.Scope) {
	t.Helper()

	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, "types.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	conf := types.Config{Importer: importer.Default()}
	info := &types.Info{Defs: map[*ast.Ident]types.Object{}}

	pkg, err := conf.Check(f.Name.Name, fset, []*ast.File{f}, info)
	if err != nil {
		t.Fatal(err)
	}

	return &generator{pkgName: f.Name.Name, pkg: pkg}, pkg.Scope()
}

// TestRun exercises the run entry point across its success and failure paths.
func TestRun(t *testing.T) {
	valid := "package fixture\n//goserde:generate\ntype T struct{ X int32 }\n"

	t.Run("version", func(t *testing.T) {
		var out bytes.Buffer
		if err := run([]string{"goserdegen", "-version"}, &out); err != nil {
			t.Fatalf("run: %v", err)
		}

		if strings.TrimSpace(out.String()) != version {
			t.Errorf("version output = %q, want %q", out.String(), version)
		}
	})

	t.Run("success", func(t *testing.T) {
		dir := writePkg(t, valid)

		var out bytes.Buffer
		if err := run([]string{"goserdegen", "-dir", dir, "-out", "gen.go"}, &out); err != nil {
			t.Fatalf("run: %v", err)
		}

		if _, err := os.Stat(filepath.Join(dir, "gen.go")); err != nil {
			t.Errorf("expected output file: %v", err)
		}

		if !strings.Contains(out.String(), "wrote") {
			t.Errorf("expected progress on stdout, got %q", out.String())
		}
	})

	t.Run("bad flag", func(t *testing.T) {
		if err := run([]string{"goserdegen", "-nope"}, io.Discard); err == nil {
			t.Error("expected an error for an unknown flag")
		}
	})

	t.Run("load error", func(t *testing.T) {
		if err := run([]string{"goserdegen", "-dir", filepath.Join(t.TempDir(), "missing")}, io.Discard); err == nil {
			t.Error("expected an error for a missing directory")
		}
	})

	t.Run("write error", func(t *testing.T) {
		dir := writePkg(t, valid)

		// -out into a nonexistent subdirectory makes WriteFile fail.
		if err := run([]string{"goserdegen", "-dir", dir, "-out", "nope/gen.go"}, io.Discard); err == nil {
			t.Error("expected an error writing into a missing subdirectory")
		}
	})

	t.Run("generate error", func(t *testing.T) {
		old := formatSource
		defer func() { formatSource = old }()

		formatSource = func(b []byte) ([]byte, error) { return nil, fmt.Errorf("boom") }

		if err := run([]string{"goserdegen", "-dir", writePkg(t, valid), "-out", "gen.go"}, io.Discard); err == nil {
			t.Error("expected a generate error")
		}
	})
}

// TestMainEntry drives main in-process with the exit hook stubbed, covering both
// its success path and its fatal-error path.
func TestMainEntry(t *testing.T) {
	oldArgs, oldExit := os.Args, osExit
	defer func() { os.Args, osExit = oldArgs, oldExit }()

	code := 0
	osExit = func(c int) { code = c }

	dir := writePkg(t, "package fixture\n//goserde:generate\ntype T struct{ X int32 }\n")
	os.Args = []string{"goserdegen", "-dir", dir, "-out", "gen.go"}
	main()

	if code != 0 {
		t.Errorf("success path called exit(%d)", code)
	}

	os.Args = []string{"goserdegen", "-dir", filepath.Join(t.TempDir(), "missing")}
	main()

	if code != 1 {
		t.Errorf("error path exit = %d, want 1", code)
	}
}

// TestGenerateGofmtError covers generate's gofmt-failure branch via the
// formatSource seam.
func TestGenerateGofmtError(t *testing.T) {
	old := formatSource
	defer func() { formatSource = old }()

	formatSource = func(b []byte) ([]byte, error) { return nil, fmt.Errorf("boom") }

	_, err := genFixtureErr(t, "package fixture\n//goserde:generate\ntype T struct{ X int32 }\n", false)
	if err == nil || !strings.Contains(err.Error(), "gofmt") {
		t.Fatalf("got err=%v, want a gofmt error", err)
	}
}

// TestEmitBlitUnmarshalSafe covers the safe-mode length guard in emitBlitUnmarshal.
// The generator only wires the blit path in fast mode, so this exercises the
// method directly.
func TestEmitBlitUnmarshalSafe(t *testing.T) {
	g := loadGen(t, "package fixture\n//goserde:generate\ntype T struct{ X int32 }\n", true)

	var b bytes.Buffer
	g.emitBlitUnmarshal(&b, g.targets[0])

	if !strings.Contains(b.String(), "ErrShortBuffer") {
		t.Errorf("safe blit Unmarshal missing length guard:\n%s", b.String())
	}
}

// TestResolveUnionsErrors covers resolveUnions' interface-validation branches,
// which the normal directive-collection path (collectUnions only emits for real
// interface declarations) cannot reach.
func TestResolveUnionsErrors(t *testing.T) {
	g, scope := checkPkg(t, "package f\ntype S struct{ X int }\ntype Basic = int\n")

	cases := []struct {
		name  string
		decls map[string][]string
		want  string
	}{
		{"unknown interface", map[string][]string{"Ghost": {"S"}}, "unknown type"},
		{"not a named type", map[string][]string{"Basic": {"S"}}, "not a named type"},
		{"not an interface", map[string][]string{"S": {"S"}}, "not an interface"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := g.resolveUnions(tc.decls, scope)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got err=%v, want containing %q", err, tc.want)
			}
		})
	}
}

// TestLoadSkipsNonSourceFiles checks load ignores test files, generated output,
// non-Go files, and subdirectories while still finding the package.
func TestLoadSkipsNonSourceFiles(t *testing.T) {
	dir := writePkg(t, "package f\n//goserde:generate\ntype T struct{ X int32 }\n")

	extra := map[string]string{
		"helper_test.go": "package f\n",
		"goserde_gen.go": "package f\n",
		"README.md":      "not go\n",
	}
	for name, body := range extra {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	g, err := load(dir, testCodecPkg, false)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if len(g.targets) != 1 {
		t.Errorf("targets = %d, want 1", len(g.targets))
	}
}

// TestLoadParseError checks a syntactically invalid source file is reported.
func TestLoadParseError(t *testing.T) {
	if _, err := load(writePkg(t, "package f\nfunc {\n"), testCodecPkg, false); err == nil {
		t.Error("expected a parse error")
	}
}

// TestLoadTypeCheckError checks a file that parses but fails type-checking is
// reported with the type-check prefix.
func TestLoadTypeCheckError(t *testing.T) {
	_, err := load(writePkg(t, "package f\n//goserde:generate\ntype T struct{ X Undefined }\n"), testCodecPkg, false)
	if err == nil || !strings.Contains(err.Error(), "type-check") {
		t.Fatalf("got err=%v, want a type-check error", err)
	}
}

// TestLoadMissingDir checks a nonexistent directory is reported (ReadDir error).
func TestLoadMissingDir(t *testing.T) {
	if _, err := load(filepath.Join(t.TempDir(), "missing"), testCodecPkg, false); err == nil {
		t.Error("expected an error for a missing directory")
	}
}

// TestLoadMarkedAliasSkipped checks a marked type alias to a non-struct is skipped
// rather than treated as a generation target.
func TestLoadMarkedAliasSkipped(t *testing.T) {
	src := "package f\n//goserde:generate\ntype Alias = int\n//goserde:generate\ntype T struct{ X int32 }\n"
	g := loadGen(t, src, false)

	if len(g.targets) != 1 || g.targets[0].Obj().Name() != "T" {
		t.Errorf("targets = %v, want exactly [T]", g.targets)
	}
}

// TestGenerateFieldPathBranches generates a struct that forces the field-by-field
// path through the remaining size/marshal/unmarshal and fixed-size branches: an
// unexported field, a leading time.Time, an array of strings, a byte array in a
// non-blittable struct, an array of time.Time, and slices of fixed and nested
// fixed-width elements.
func TestGenerateFieldPathBranches(t *testing.T) {
	src := `package fixture

import "time"

//goserde:generate
type T struct {
	Stamp  time.Time
	X      int32
	hidden int
	Names  [2]string
	Blob   [4]byte
	Stamps [2]time.Time
	Longs  []int64
	Grid   [][2]int32
}
`
	out := genFixture(t, src, false)

	if strings.Contains(out, "hidden") {
		t.Error("unexported field leaked into generated code")
	}
}

// TestGenerateEmptyStruct covers the blittable check for a struct with no
// serializable fields.
func TestGenerateEmptyStruct(t *testing.T) {
	genFixture(t, "package fixture\n//goserde:generate\ntype Empty struct{}\n", false)
}

// TestGenerateSafeNestedAndUnion covers the safe-mode nested-struct and union
// decode branches, where a delegated Unmarshal's error is propagated.
func TestGenerateSafeNestedAndUnion(t *testing.T) {
	src := `package fixture

//goserde:generate
type Inner struct{ X int32 }

//goserde:generate
type A struct{ Y int32 }

func (*A) isU() {}

//goserde:union A
type U interface{ isU() }

//goserde:generate
type Holder struct {
	N Inner
	V U
}
`
	out := genFixture(t, src, true)

	if !strings.Contains(out, "ErrShortBuffer") {
		t.Error("safe mode should emit bounds checks")
	}
}

// TestGenerateUnionDirectiveForms covers union-directive collection from a grouped
// TypeSpec doc (the GenDecl carries no doc) alongside a documented non-union
// interface (unionMembers scans its comments and finds no directive).
func TestGenerateUnionDirectiveForms(t *testing.T) {
	src := `package fixture

// Plain is a documented interface that is not a union.
type Plain interface{ M() }

//goserde:generate
type A struct{ X int32 }

func (*A) isU() {}

type (
	//goserde:union A
	U interface{ isU() }
)
`
	out := genFixture(t, src, false)

	if !strings.Contains(out, "func (r *A) Size()") {
		t.Errorf("expected A codec, got:\n%s", out)
	}
}

// --- Integration tests -------------------------------------------------------
//
// The tests below exercise codecs the generator actually emitted, end-to-end.
// Their inputs are the annotated structs in the test/data/shapes fixture
// package; round-tripping the generated Marshal/Unmarshal verifies emit.go's
// output for every supported shape and mode.

// rt marshals in, decodes into a fresh value, and asserts the result
// deep-equals the original, failing the test on any mismatch.
func rt[T any](t *testing.T, in T, marshal func([]byte) int, size func() int, unmarshal func(*T, []byte)) {
	buf := make([]byte, size())

	n := marshal(buf)
	if n != len(buf) {
		t.Fatalf("%T: Size=%d Marshal wrote %d", in, len(buf), n)
	}

	var out T
	unmarshal(&out, buf)

	if !reflect.DeepEqual(in, out) {
		t.Fatalf("%T mismatch:\n in=%+v\nout=%+v", in, in, out)
	}
}

func TestAllShapes(t *testing.T) {
	sf := shapes.SmallFixed{A: -5, B: 99, C: 2.71828, D: true}
	rt(t, sf, sf.Marshal, sf.Size, func(o *shapes.SmallFixed, b []byte) { o.Unmarshal(b) })

	fm := shapes.FlatMixed{ID: 1 << 40, Ratio: 0.5, Name: "flat", Data: []byte("xyz")}
	rt(t, fm, fm.Marshal, fm.Size, func(o *shapes.FlatMixed, b []byte) { o.Unmarshal(b) })

	in := shapes.Inner{X: 7, Y: 8}
	ns := shapes.Nested{Label: "n", Pos: shapes.Inner{X: 1, Y: 2}, Opt: &in, Path: []shapes.Inner{{X: 3, Y: 4}, {X: 5, Y: 6}}}
	rt(t, ns, ns.Marshal, ns.Size, func(o *shapes.Nested, b []byte) { o.Unmarshal(b) })

	// nil pointer case
	ns2 := shapes.Nested{Label: "nil", Pos: shapes.Inner{X: 0, Y: 0}, Opt: nil, Path: nil}
	rt(t, ns2, ns2.Marshal, ns2.Size, func(o *shapes.Nested, b []byte) { o.Unmarshal(b) })

	ch := shapes.CollectionHeavy{
		Counts: map[string]int64{"a": 1, "b": 2, "c": 3},
		Floats: []float64{1.1, 2.2, 3.3},
		Names:  []string{"x", "y"},
	}
	rt(t, ch, ch.Marshal, ch.Size, func(o *shapes.CollectionHeavy, b []byte) { o.Unmarshal(b) })

	sh := shapes.StringHeavy{Title: "T", Body: "lorem ipsum dolor", Tags: []string{"go", "fast"}}
	rt(t, sh, sh.Marshal, sh.Size, func(o *shapes.StringHeavy, b []byte) { o.Unmarshal(b) })
}

// TestNamedTypeShapes covers defined types (named uint8/int32) used as slice and
// array elements, the case the raw byte-copy fast path would mis-handle.
func TestNamedTypeShapes(t *testing.T) {
	accent := shapes.Hue(200)
	nsc := shapes.NamedScalars{
		Label:   "named",
		Palette: []shapes.Hue{1, 2, 254, 255},
		Swatch:  [4]shapes.Hue{10, 20, 30, 40},
		Levels:  []shapes.Grade{-7, 0, 1 << 20},
		Accent:  &accent,
	}
	rt(t, nsc, nsc.Marshal, nsc.Size, func(o *shapes.NamedScalars, b []byte) { o.Unmarshal(b) })

	// Nil pointer and nil slices decode back to nil; the zero array stays zero.
	nsc2 := shapes.NamedScalars{Swatch: [4]shapes.Hue{}}
	rt(t, nsc2, nsc2.Marshal, nsc2.Size, func(o *shapes.NamedScalars, b []byte) { o.Unmarshal(b) })

	// All-fixed-width despite defined types: the blittable memmove path.
	nf := shapes.NamedFixed{Tag: 9, Codes: [3]shapes.Hue{1, 2, 3}, Score: -42, Flag: true}
	rt(t, nf, nf.Marshal, nf.Size, func(o *shapes.NamedFixed, b []byte) { o.Unmarshal(b) })
}

func TestArrayShapes(t *testing.T) {
	fa := shapes.FixedArrays{
		Hash: [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
		Quad: [4]int32{-1, 2, -3, 4},
		Flag: true,
	}
	rt(t, fa, fa.Marshal, fa.Size, func(o *shapes.FixedArrays, b []byte) { o.Unmarshal(b) })

	// An all-fixed-width struct (arrays of fixed basics) must take the blittable
	// memmove path, whose wire size is exactly the in-memory size.
	if got, want := fa.Size(), int(unsafe.Sizeof(fa)); got != want {
		t.Errorf("FixedArrays not on memmove path: Size()=%d, unsafe.Sizeof=%d", got, want)
	}

	ma := shapes.MixedArrays{
		Name:   "mixed",
		Words:  [3]string{"a", "bb", "ccc"},
		Points: [2]shapes.Inner{{X: 1, Y: 2}, {X: 3, Y: 4}},
		Bytes:  [8]byte{8, 7, 6, 5, 4, 3, 2, 1},
	}
	rt(t, ma, ma.Marshal, ma.Size, func(o *shapes.MixedArrays, b []byte) { o.Unmarshal(b) })

	// Empty-string array elements still round-trip (arrays carry no length
	// prefix, so every slot is always present on the wire).
	ma2 := shapes.MixedArrays{Words: [3]string{"", "x", ""}}
	rt(t, ma2, ma2.Marshal, ma2.Size, func(o *shapes.MixedArrays, b []byte) { o.Unmarshal(b) })
}

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

	in := shapes.TimeStruct{
		Label:   "event",
		Created: created,
		Updated: &updated,
		Stamps:  []time.Time{created, updated, created.Add(-time.Hour)},
	}

	buf := make([]byte, in.Size())
	n := in.Marshal(buf)

	var out shapes.TimeStruct

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
	in := shapes.TimeStruct{
		Label:   "z",
		Created: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		Updated: nil,
		Stamps:  nil,
	}

	buf := make([]byte, in.Size())
	n := in.Marshal(buf)

	var out shapes.TimeStruct
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

// TestTaggedFieldExcluded verifies that fields tagged `goserde:"-"` are absent
// from the wire and decode back to their zero value.
func TestTaggedFieldExcluded(t *testing.T) {
	a := shapes.Tagged{ID: 7, Secret: "hunter2", Skip: make(chan int), Name: "alice"}
	b := shapes.Tagged{ID: 7, Secret: "a-much-longer-different-secret", Skip: nil, Name: "alice"}

	// Secret is excluded, so changing it must not change the encoded size.
	if a.Size() != b.Size() {
		t.Fatalf("excluded Secret affected Size: %d vs %d", a.Size(), b.Size())
	}

	buf := make([]byte, a.Size())
	a.Marshal(buf)

	var out shapes.Tagged
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

// TestUnionRoundTrip checks that a tagged-union field and a slice of unions
// round-trip for each member variant and for the nil interface.
func TestUnionRoundTrip(t *testing.T) {
	cases := []shapes.Drawing{
		{Name: "circle", Shape: &shapes.Circle{R: 2.5}, Layers: []shapes.Geometry{&shapes.Circle{R: 1}, &shapes.Square{Side: 3}}},
		{Name: "square", Shape: &shapes.Square{Side: 4}, Layers: []shapes.Geometry{&shapes.Square{Side: 9}}},
		{Name: "nil", Shape: nil, Layers: nil},
	}

	for _, in := range cases {
		buf := make([]byte, in.Size())

		n := in.Marshal(buf)
		if n != len(buf) {
			t.Fatalf("%s: Marshal wrote %d bytes, Size reported %d", in.Name, n, len(buf))
		}

		var out shapes.Drawing

		m, err := out.Unmarshal(buf)
		if err != nil {
			t.Fatalf("%s: Unmarshal: %v", in.Name, err)
		}

		if m != len(buf) {
			t.Fatalf("%s: Unmarshal consumed %d bytes, want %d", in.Name, m, len(buf))
		}

		if !reflect.DeepEqual(in, out) {
			t.Fatalf("%s: round-trip mismatch:\n in=%+v\nout=%+v", in.Name, in, out)
		}
	}
}

// TestUnionUnknownTag verifies that decoding a discriminator with no matching
// member returns ErrUnknownUnionTag instead of panicking — the forward-compat
// path for a peer that encoded a newer member.
func TestUnionUnknownTag(t *testing.T) {
	d := shapes.Drawing{Name: "x", Shape: &shapes.Circle{R: 1}}
	buf := make([]byte, d.Size())
	d.Marshal(buf)

	// Name "x" encodes as a 1-byte length prefix + 1 byte, so the Shape tag sits
	// at index 2. Overwrite it with an undeclared member ID.
	buf[2] = 99

	var out shapes.Drawing
	if _, err := out.Unmarshal(buf); err != codec.ErrUnknownUnionTag {
		t.Fatalf("unknown tag: got err=%v, want codec.ErrUnknownUnionTag", err)
	}
}

// roundTripper is satisfied by every generated *T (and by hand-written codecs).
type roundTripper interface {
	Size() int
	Marshal([]byte) int
	Unmarshal([]byte) (int, error)
}

// rtFuzz marshals in, decodes into out, and asserts a faithful round-trip:
// Size matches bytes written, the whole buffer is consumed, no decode error,
// and the decoded value deep-equals the original.
func rtFuzz(t *testing.T, in, out roundTripper) {
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

		fm := &shapes.FlatMixed{ID: u64, Ratio: ff, Name: s1, Data: nilIfEmpty(blob)}
		rtFuzz(t, fm, &shapes.FlatMixed{})

		sh := &shapes.StringHeavy{Title: s1, Body: s2}
		if mask&1 != 0 {
			sh.Tags = []string{s1, s2}
		}
		rtFuzz(t, sh, &shapes.StringHeavy{})

		ns := &shapes.Nested{Label: s1, Pos: shapes.Inner{X: int32(i64), Y: int32(u64)}}
		if mask&2 != 0 {
			ns.Opt = &shapes.Inner{X: int32(u64), Y: int32(i64)}
		}

		if mask&4 != 0 {
			ns.Path = []shapes.Inner{{X: 1, Y: 2}, {X: int32(i64), Y: int32(u64)}}
		}
		rtFuzz(t, ns, &shapes.Nested{})

		ch := &shapes.CollectionHeavy{}
		if mask&8 != 0 {
			ch.Counts = map[string]int64{s1: i64, s2: int64(u64)}
		}

		if mask&16 != 0 {
			ch.Floats = []float64{ff, -ff}
		}

		if mask&32 != 0 {
			ch.Names = []string{s1, s2}
		}
		rtFuzz(t, ch, &shapes.CollectionHeavy{})
	})
}

// benchM benchmarks Marshal for a shape, allocating the destination once.
func benchM(b *testing.B, size func() int, marshal func([]byte) int) {
	buf := make([]byte, size())
	b.ReportAllocs()

	for b.Loop() {
		_ = marshal(buf)
	}
}

// benchU benchmarks Unmarshal for a shape against a pre-marshaled buffer.
func benchU(b *testing.B, buf []byte, unmarshal func([]byte)) {
	b.ReportAllocs()

	for b.Loop() {
		unmarshal(buf)
	}
}

func BenchmarkSmallFixed_M(b *testing.B) {
	v := shapes.SmallFixed{A: 1, B: 2, C: 3, D: true}
	benchM(b, v.Size, v.Marshal)
}

func BenchmarkFlatMixed_M(b *testing.B) {
	v := shapes.FlatMixed{ID: 1, Ratio: 2, Name: "hello world", Data: []byte("0123456789")}
	benchM(b, v.Size, v.Marshal)
}

func BenchmarkNested_M(b *testing.B) {
	in := shapes.Inner{X: 7, Y: 8}
	v := shapes.Nested{Label: "n", Pos: shapes.Inner{X: 1, Y: 2}, Opt: &in, Path: []shapes.Inner{{X: 3, Y: 4}, {X: 5, Y: 6}, {X: 7, Y: 8}}}
	benchM(b, v.Size, v.Marshal)
}

func BenchmarkCollection_M(b *testing.B) {
	v := shapes.CollectionHeavy{Counts: map[string]int64{"a": 1, "b": 2, "c": 3, "d": 4}, Floats: []float64{1, 2, 3, 4, 5}, Names: []string{"x", "y", "z"}}
	benchM(b, v.Size, v.Marshal)
}

func BenchmarkStringHeavy_M(b *testing.B) {
	v := shapes.StringHeavy{Title: "Title", Body: "lorem ipsum dolor sit amet consectetur", Tags: []string{"go", "fast", "serde"}}
	benchM(b, v.Size, v.Marshal)
}

func BenchmarkSmallFixed_U(b *testing.B) {
	v := shapes.SmallFixed{A: 1, B: 2, C: 3, D: true}
	buf := make([]byte, v.Size())
	v.Marshal(buf)
	benchU(b, buf, func(x []byte) { var o shapes.SmallFixed; o.Unmarshal(x) })
}

// smallFixedPF encodes shapes.SmallFixed field-by-field instead of through the
// generated blittable memmove path. Reading BenchmarkSmallFixedField_* against
// BenchmarkSmallFixed_* shows the memmove speedup head-to-head. It is a
// hand-written codec built directly on the codec package primitives.
type smallFixedPF shapes.SmallFixed

func (r *smallFixedPF) Size() int { return 17 }

func (r *smallFixedPF) Marshal(b []byte) int {
	codec.PutU32(b[0:], uint32(r.A))
	codec.PutU32(b[4:], uint32(r.B))
	codec.PutU64(b[8:], codec.F64bits(r.C))

	if r.D {
		b[16] = 1
	} else {
		b[16] = 0
	}

	return 17
}

func (r *smallFixedPF) Unmarshal(b []byte) (int, error) {
	r.A = int32(codec.U32(b[0:]))
	r.B = int32(codec.U32(b[4:]))
	r.C = codec.Bitsf64(codec.U64(b[8:]))
	r.D = b[16] != 0

	return 17, nil
}

func BenchmarkSmallFixedField_M(b *testing.B) {
	v := smallFixedPF{A: 1, B: 2, C: 3, D: true}
	benchM(b, v.Size, v.Marshal)
}

func BenchmarkSmallFixedField_U(b *testing.B) {
	v := smallFixedPF{A: 1, B: 2, C: 3, D: true}
	buf := make([]byte, v.Size())
	v.Marshal(buf)
	benchU(b, buf, func(x []byte) { var o smallFixedPF; o.Unmarshal(x) })
}

func BenchmarkFlatMixed_U(b *testing.B) {
	v := shapes.FlatMixed{ID: 1, Ratio: 2, Name: "hello world", Data: []byte("0123456789")}
	buf := make([]byte, v.Size())
	v.Marshal(buf)
	benchU(b, buf, func(x []byte) { var o shapes.FlatMixed; o.Unmarshal(x) })
}

func BenchmarkNested_U(b *testing.B) {
	in := shapes.Inner{X: 7, Y: 8}
	v := shapes.Nested{Label: "n", Pos: shapes.Inner{X: 1, Y: 2}, Opt: &in, Path: []shapes.Inner{{X: 3, Y: 4}, {X: 5, Y: 6}, {X: 7, Y: 8}}}
	buf := make([]byte, v.Size())
	v.Marshal(buf)
	benchU(b, buf, func(x []byte) { var o shapes.Nested; o.Unmarshal(x) })
}

func BenchmarkCollection_U(b *testing.B) {
	v := shapes.CollectionHeavy{Counts: map[string]int64{"a": 1, "b": 2, "c": 3, "d": 4}, Floats: []float64{1, 2, 3, 4, 5}, Names: []string{"x", "y", "z"}}
	buf := make([]byte, v.Size())
	v.Marshal(buf)
	benchU(b, buf, func(x []byte) { var o shapes.CollectionHeavy; o.Unmarshal(x) })
}

// BenchmarkCollection_U_Reuse decodes repeatedly into the SAME destination,
// exercising slice-capacity and map reuse. It should report fewer allocs than
// BenchmarkCollection_U, which decodes into a fresh value every iteration.
func BenchmarkCollection_U_Reuse(b *testing.B) {
	v := shapes.CollectionHeavy{Counts: map[string]int64{"a": 1, "b": 2, "c": 3, "d": 4}, Floats: []float64{1, 2, 3, 4, 5}, Names: []string{"x", "y", "z"}}
	buf := make([]byte, v.Size())
	v.Marshal(buf)

	var o shapes.CollectionHeavy

	benchU(b, buf, func(x []byte) { o.Unmarshal(x) })
}

func BenchmarkStringHeavy_U(b *testing.B) {
	v := shapes.StringHeavy{Title: "Title", Body: "lorem ipsum dolor sit amet consectetur", Tags: []string{"go", "fast", "serde"}}
	buf := make([]byte, v.Size())
	v.Marshal(buf)
	benchU(b, buf, func(x []byte) { var o shapes.StringHeavy; o.Unmarshal(x) })
}

// --- Integration tests: safe mode --------------------------------------------
//
// The safeshapes fixture package is generated with -safe, so its codecs are
// bounds-checked. These tests assert the safe-mode contract: truncated or
// corrupt input returns codec.ErrShortBuffer (or ErrUnknownUnionTag) instead
// of panicking, and decoded payloads are copied rather than aliased.

func sampleAll() *safeshapes.All {
	p := uint32(99)

	return &safeshapes.All{
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
		Inner:  safeshapes.Inner{A: 7, B: "inner"},
		Nested: &safeshapes.Inner{A: 8, B: "nested"},
	}
}

func sampleArrays() *safeshapes.Arrays {
	return &safeshapes.Arrays{
		Hash: [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
		Quad: [4]int32{-1, 2, -3, 4},
		Tags: [2]string{"a", "bb"},
	}
}

func sampleFrame() *safeshapes.Frame {
	return &safeshapes.Frame{ID: 42, Sig: &safeshapes.Ping{Seq: 7}}
}

// marshalBytes returns the exact wire bytes of v.
func marshalBytes(t *testing.T, v roundTripper) []byte {
	t.Helper()

	buf := make([]byte, v.Size())
	n := v.Marshal(buf)

	return buf[:n]
}

func checkRoundTrip(t *testing.T, in, out roundTripper) {
	t.Helper()

	buf := marshalBytes(t, in)

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

func TestSafeRoundTrip(t *testing.T) {
	checkRoundTrip(t, &safeshapes.Fixed{X: 1, Y: -2, Z: 3}, &safeshapes.Fixed{})
	checkRoundTrip(t, &safeshapes.Inner{A: 7, B: "inner"}, &safeshapes.Inner{})
	checkRoundTrip(t, sampleArrays(), &safeshapes.Arrays{})
	checkRoundTrip(t, sampleAll(), &safeshapes.All{})
	checkRoundTrip(t, sampleFrame(), &safeshapes.Frame{})
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

// TestSafeTruncationNeverPanics is the core safe-mode guarantee: every truncated
// prefix of a valid buffer must return codec.ErrShortBuffer, never panic.
func TestSafeTruncationNeverPanics(t *testing.T) {
	cases := []struct {
		name  string
		fresh func() roundTripper
		val   roundTripper
	}{
		{"Fixed", func() roundTripper { return &safeshapes.Fixed{} }, &safeshapes.Fixed{X: 1, Y: -2, Z: 3}},
		{"Inner", func() roundTripper { return &safeshapes.Inner{} }, &safeshapes.Inner{A: 7, B: "inner"}},
		{"Arrays", func() roundTripper { return &safeshapes.Arrays{} }, sampleArrays()},
		{"All", func() roundTripper { return &safeshapes.All{} }, sampleAll()},
		{"Frame", func() roundTripper { return &safeshapes.Frame{} }, sampleFrame()},
	}

	for _, tc := range cases {
		full := marshalBytes(t, tc.val)

		for k := range len(full) {
			trunc := full[:k:k] // cap=k so an unchecked read can't alias beyond the prefix
			err := callNoPanic(t, tc.name, k, func() error {
				_, e := tc.fresh().Unmarshal(trunc)

				return e
			})

			if err != codec.ErrShortBuffer {
				t.Errorf("%s prefix[:%d]: got err=%v, want codec.ErrShortBuffer", tc.name, k, err)
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
	codec.PutU16(buf[0:], 1)               // Inner.A
	cc := codec.PutUvarint(buf[2:], 1<<50) // Inner.B claims a 1-petabyte string

	if _, err := (&safeshapes.Inner{}).Unmarshal(buf[:2+cc]); err != codec.ErrShortBuffer {
		t.Fatalf("oversized length: got err=%v, want codec.ErrShortBuffer", err)
	}
}

// TestSafeUnknownUnionTag asserts safe-mode union decode rejects an undeclared
// member tag with ErrUnknownUnionTag instead of panicking.
func TestSafeUnknownUnionTag(t *testing.T) {
	buf := marshalBytes(t, sampleFrame())
	buf[2] = 99 // tag sits right after the 2-byte ID; 99 is not a declared member

	if _, err := (&safeshapes.Frame{}).Unmarshal(buf); err != codec.ErrUnknownUnionTag {
		t.Fatalf("unknown tag: got err=%v, want codec.ErrUnknownUnionTag", err)
	}
}

// TestPortableDecodeOwnsBytes verifies that safe (portable) mode copies
// length-prefixed payloads rather than aliasing the input buffer, so a decoded
// value survives mutation of the source bytes.
func TestPortableDecodeOwnsBytes(t *testing.T) {
	in := safeshapes.All{Name: "hello", Blob: []byte("world")}
	buf := make([]byte, in.Size())
	in.Marshal(buf)

	var out safeshapes.All
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

// fuzzDecode asserts the core safe-mode contract on ARBITRARY input: Unmarshal
// never panics, only ever returns nil or codec.ErrShortBuffer, and on success
// reports a sane consumed-byte count.
func fuzzDecode(f *testing.F, fresh func() roundTripper, seed roundTripper) {
	buf := make([]byte, seed.Size())
	n := seed.Marshal(buf)
	f.Add(buf[:n])                        // a valid encoding
	f.Add([]byte{})                       // empty
	f.Add([]byte{0})                      // single byte
	f.Add([]byte{0xff, 0xff, 0xff, 0xff}) // truncated varint / garbage
	f.Fuzz(func(t *testing.T, data []byte) {
		v := fresh()

		n, err := v.Unmarshal(data)
		if err != nil && err != codec.ErrShortBuffer {
			t.Fatalf("unexpected error (want nil or ErrShortBuffer): %v", err)
		}

		if err == nil && (n < 0 || n > len(data)) {
			t.Fatalf("decoded %d bytes of a %d-byte buffer", n, len(data))
		}
	})
}

func FuzzFixedDecode(f *testing.F) {
	fuzzDecode(f, func() roundTripper { return &safeshapes.Fixed{} }, &safeshapes.Fixed{X: 1, Y: -2, Z: 3})
}

func FuzzInnerDecode(f *testing.F) {
	fuzzDecode(f, func() roundTripper { return &safeshapes.Inner{} }, &safeshapes.Inner{A: 7, B: "inner"})
}

func FuzzArraysDecode(f *testing.F) {
	fuzzDecode(f, func() roundTripper { return &safeshapes.Arrays{} }, sampleArrays())
}

func FuzzAllDecode(f *testing.F) {
	fuzzDecode(f, func() roundTripper { return &safeshapes.All{} }, sampleAll())
}
