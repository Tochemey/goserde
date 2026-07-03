# Changelog

All notable changes to goserde are documented in this file. The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.2.0] - 2026-07-03

All benchmark numbers below are from Go 1.26 on darwin/arm64 (Apple M1).

### Added

- **Single-pass encoding: a generated `Append` method.** Every codec gains `Append(b []byte) []byte`, which appends the value's encoding in one tree walk, growing the buffer as needed and writing byte-identical output to `Marshal`. `codec.Into` now encodes through it, eliminating the separate `Size()` walk that re-iterated every slice, map, and string: on a reused buffer the map-heavy shape drops from 124 ns to 79 ns (-36%), nested from 13.6 ns to 8.8 ns, and string-heavy from 23.5 ns to 19.6 ns, so `Into` now costs the same as the bare Marshal pass. `codec.Bytes` keeps the two-pass form, since one exactly sized allocation is optimal for one-shot encoding. New append primitives (`AppendU16/32/64`, `AppendUvarint`) join the covered `codec` API.
  - **Breaking (pre-1.0):** the `codec.Marshaler` interface now includes `Append`. Hand-written codecs must add it by mirroring their `Marshal` with the append primitives; `codec/record.go` is the reference.
- **User-owned codecs.** A `//goserde:generate` struct that already declares all four codec methods (`Size`, `Marshal`, `Append`, `Unmarshal`) elsewhere in its package is skipped with a notice instead of causing a redeclaration compile error, so the generated methods can be moved into the struct's own file. Declaring only part of the set is an error. Adopted codecs stop tracking the generator, and slices of user-owned structs keep the per-element loop instead of the bulk copy, since a hand-written codec is a black box. Supporting hardening: `-out` must be a bare non-test `.go` file name (methods must live in the receiver's package, so the file is always written into `-dir`), and the out file is never parsed during generation, so a stale or broken previous run cannot break regeneration.
- **Benchmark coverage and guardrails.** `benchcompare` now runs the mus and benc head-to-head across all six benchmark shapes (the five generated ones plus `Record`), with round-trip tests keeping every hand-written competitor codec honest; goserde leads every cell, including map-heavy decode (README tables updated). The shape suite gains union round-trip and 1000-element map, string-slice, and struct-slice benchmarks. CI gains a report-only `bench-smoke` job that benchmarks a pull request's base and head and prints a `benchstat` comparison in the job summary, so perf regressions surface in review without a flaky hard gate.

### Changed

- **Fast mode bulk-copies slices and arrays of fixed-width elements.** A single raw-memory copy replaces the element loop when elements are explicitly sized basics (`bool`, `u/int8/16/32/64`, `float32/64`), named variants of those, or nested fixed arrays of the same. On the reference `Record`, marshal drops from ~14 ns to ~11 ns and unmarshal from ~28 ns to ~24 ns; a 1000-element `[]uint32` runs at memcpy speed (~60 ns). `[]int` and `[]uint` stay element-wise (platform-dependent in-memory width), as does `[]time.Time` (encodes as UnixNano). Decode fills an owned, capacity-reused backing array, so nothing new aliases the input; safe mode is unaffected.
- **Fast mode bulk-copies slices and arrays of blittable structs.** The same single-memmove treatment when the element is a struct whose fields are all fixed-width, sized via `unsafe.Sizeof` since element padding is platform-dependent. The wire bytes are unchanged (the loop produced the same contiguous block). A 1000-element slice of an 8-byte struct: 1990 ns to 119 ns marshal and 110 ns unmarshal, about 17x; `Nested` improves to 12.3 ns marshal / 32 ns unmarshal (10 ns decoding into a reused destination). Safe mode keeps the portable field-by-field loop.
- **Decode reuses pointer pointees and union members.** A non-nil pointer field is overwritten in place instead of allocating a fresh pointee, and a union member is reused when its dynamic type matches the incoming tag (a mismatch or nil allocates as before). This extends the slice and map reuse contract to pointers and unions in both modes; a wire nil still clears the field. Repeated decode into the same `Nested` destination drops from 36 ns / 2 allocs to 14.6 ns / 0 allocs; fresh destinations are unchanged. The flip sides of in-place reuse, shared with `encoding/json`'s pointer behavior:
  - a retained old pointee is mutated rather than orphaned;
  - pointer fields and union members of a reused destination must not alias each other, since every aliased slot decodes through the one shared pointee;
  - generator-skipped fields (unexported or `goserde:"-"`) keep the reused pointee's previous contents instead of resetting to zero;
  - after a safe-mode decode error the destination is, as before, in an unspecified partial state and must not be used.
- **Marshal and Size iterate struct, array, and `time.Time` slice elements by index.** The previous `for _, e := range` loops copied each element into the iteration variable before encoding it: marshalling 16 wide (~90 B) struct elements drops from 161 ns to 137 ns (~15%). Slim elements such as strings keep the value loop, which is marginally faster for them.
- `codec.UvarintSize` is now branch-free via `bits.Len64` (same results, lower inlining cost). Benchmarks are unchanged; investigation confirmed the varint primitives all inline and are not a bottleneck.

### Fixed

- **Blittable structs no longer leak skipped fields onto the wire.** Since v0.1.0, a struct whose *serialized* fields were all fixed-width took the whole-struct memmove even when it also had an excluded (`goserde:"-"`) or unexported field, so the skipped field's raw bytes were written to the output (a potential secret leak) and overwritten on decode, violating the exclusion contract. The blit fast path now requires the serializable fields to cover the entire struct; partially covered structs, and slices/arrays of them, fall back to the correct field-by-field codec. Structs with every field exported and untagged (the common case) keep the memmove path, and the wire format changes only for the previously mis-encoded structs.

## [v0.1.0] - 2026-06-29

First public release of goserde: a code-generated, zero-reflection binary serializer for Go, built for maximum encode/decode throughput when you own both ends of the wire.

goserde generates type-specific `Size`/`Marshal`/`Unmarshal` methods for your structs at build time. No reflection on the hot path, no schema language, no runtime type registry, and zero external dependencies.

### Highlights

- **Code generation, not reflection.** Annotate a struct with `//goserde:generate`, run `go generate`, and get straight-line `Size`/`Marshal`/`Unmarshal` over a single running offset.
- **Two modes, one API.** Generated method signatures are identical; you pick the trade-off at generation time:
  - **Fast** (default): zero-copy decoding via `unsafe` and a single `memmove` for all-fixed-width structs. For trusted bytes on the same architecture.
  - **Safe** (`-safe`): bounds-checked decoding that returns `codec.ErrShortBuffer` instead of panicking, copies decoded data, and uses a portable little-endian format. For untrusted or cross-machine input.
- **Broad type support:** all integer and float widths, `bool`, `string`, `[]byte`, slices, fixed arrays, maps, pointers, nested annotated structs, `time.Time` (as int64 Unix-nanoseconds, UTC), and tagged unions via `//goserde:union`. Exclude fields with `` `goserde:"-"` ``.
- **Zero dependencies.** The generator is standard-library only (`go/parser` + `go/types`, no `go/packages`, no network). Builds and tests fully offline.
- **A public codec toolkit.** The `codec` package exposes both a convenience API (`Bytes`/`Into`/`From`) and the low-level primitives (varint, zigzag, fixed-width LE read/write, zero-copy conversions, float bit-casts) so you can hand-write a codec that shares the exact wire format.

### Performance

Flat `Record` struct, same machine (Go 1.26, darwin/arm64, Apple M1):

| Operation | goserde         | encoding/json      |
|-----------|-----------------|--------------------|
| Marshal   | 14 ns, 0 allocs | 399 ns, 1 alloc    |
| Unmarshal | 28 ns, 1 alloc  | 2258 ns, 12 allocs |

Roughly 28x faster than JSON on marshal and 81x on unmarshal, and ahead of `benc` and `mus` on the same data. Reproduce with `make bench`, `make bench-shapes`, and `make compare`.

### Install

```bash
# Codec support library (imported by generated code)
go get github.com/tochemey/goserde

# Code generator
go install github.com/tochemey/goserde/cmd/goserdegen@v0.1.0
```

### Compatibility and caveats

goserde is **pre-1.0**: the API and wire format may change between minor versions until 1.0.0.

- **Fast mode is for trusted, same-machine bytes.** It trusts its input (malformed bytes may panic) and decoded `string`/`[]byte` values alias the input buffer, so that buffer must outlive the decoded value and must not be mutated or pooled while it is in use. Decode anything you did not produce yourself with safe mode.
- **No schema evolution.** The schema lives entirely in the generated code: no field names, tags, or type markers on the wire. Adding, removing, or reordering fields breaks compatibility. For evolving schemas, reach for Protobuf, Cap'n Proto, or FlatBuffers.
- **Wire-format portability:** safe-mode bytes are portable across architectures and Go versions; fast-mode bytes are native-layout and should be treated as ephemeral.

Requires Go 1.26.

[v0.2.0]: https://github.com/tochemey/goserde/compare/v0.1.0...v0.2.0
[v0.1.0]: https://github.com/tochemey/goserde/releases/tag/v0.1.0
