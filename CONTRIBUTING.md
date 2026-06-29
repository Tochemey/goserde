# Contributing to goserde

Thanks for your interest in improving goserde. This guide covers how to set up,
make changes, and get a pull request merged.

By participating, you agree to abide by our
[Code of Conduct](CODE_OF_CONDUCT.md). Contributions are accepted under the
project's [MIT license](LICENSE).

## Getting started

goserde targets the Go version pinned in [`go.mod`](go.mod) and has **zero
external dependencies**, so the core builds and tests entirely offline.

```bash
git clone https://github.com/tochemey/goserde
cd goserde

go build ./...      # build everything
go test ./...       # run the full suite
```

The repository is organized as:

| Path              | What it is                                                          |
|-------------------|--------------------------------------------------------------------|
| `codec/`          | Codec primitives and the convenience API (`Bytes`/`Into`/`From`).   |
| `cmd/goserdegen/` | The code generator and its test suite (unit + integration).        |
| `test/data/`      | Generated codec fixtures (data only) the generator's tests drive.  |
| `example/`        | A standalone usage example, excluded from tests and coverage.      |
| `benchcompare/`   | A **separate** module with the mus/benc comparison benchmarks.     |

## Development workflow

### 1. Follow the coding conventions

goserde follows idiomatic Go style, plus a few house rules:

- Match the style of the surrounding code.
- Keep changes surgical: touch only what the change requires.
- Prefer the simplest solution; no speculative abstractions.
- Document every exported type, function, and field with godoc.
- Put a blank line around multi-line blocks.
- Replace meaningful literals with named constants.

Run the formatters and linters before pushing:

```bash
gofmt -l .          # must print nothing
go vet ./...
golangci-lint run   # config in .golangci.yml
```

### 2. Keep the generator dependency-free

The generator (`cmd/goserdegen`) is deliberately **standard-library only**: it
uses `go/parser` and `go/types`, never `go/packages` (which needs the network).
Please do not add third-party dependencies to the core module or the generator.
The `benchcompare` module is the only place external dependencies (mus, benc)
are allowed, and the core never imports it.

### 3. Regenerate codecs and commit them

If you change an annotated struct, the generator, or anything that affects
emitted code, regenerate and commit the result:

```bash
go generate ./...
```

Generated files (`goserde_gen.go`) are checked in. Regeneration is
deterministic, and CI fails if the committed output differs from a fresh
`go generate` (`git diff --exit-code`). Run it before opening your PR.

When adding support for a new field type or feature, also add a fixture under
`test/data/` and exercise it from the generator's tests in `cmd/goserdegen`, so
it is covered by the round-trip, truncation, and fuzz suites.

## Testing

The generator's tests live in `cmd/goserdegen`: unit tests of the emitter
alongside integration tests that round-trip the codecs generated into the
`test/data/` fixtures.

```bash
go test ./...                 # full suite
go test -race ./...           # with the race detector (CI runs this)
go test ./cmd/goserdegen/ -run '^$' -fuzz FuzzRoundTrip -fuzztime 30s
```

New code needs tests. Some guidelines specific to this project:

- **Round-trip:** every supported type should marshal and unmarshal back to an
  equal value. Watch the documented gotchas: empty collections decode to `nil`,
  and `time.Time` round-trips the instant (compare with `.UnixNano()`).
- **Safe mode:** changes affecting decoding should be exercised in the
  `safeshapes` package, which asserts truncated input returns
  `codec.ErrShortBuffer` instead of panicking.
- **Fuzzing:** the decode path is fuzzed. If you touch it, run the fuzz targets
  locally and commit any failing corpus entries they surface.

## Benchmarks

```bash
make bench          # the codec Record vs the standard library
make bench-shapes   # goserde across struct shapes
make compare        # head-to-head vs mus and benc (separate module, needs network)
```

If a change is performance-motivated, include before/after numbers from the same
machine in your PR description, and prefer `b.Loop()` in new benchmarks so the
compiler does not optimize the measured work away.

## Commit messages

This project uses [Conventional Commits](https://www.conventionalcommits.org).
Each commit message starts with a type, an optional scope, and a short summary:

```
<type>(<optional scope>): <summary>
```

Common types are `feat`, `fix`, `docs`, `test`, `refactor`, `perf`, `build`,
`ci`, and `chore`. Use the imperative mood and keep the summary under ~72
characters. A breaking change is marked with a `!` after the type/scope (for
example `feat!:`) or a `BREAKING CHANGE:` footer. Examples:

```
feat(generator): support time.Time fields
fix(codec): reject oversized varint length in safe mode
docs: clarify the fast vs safe mode trade-offs
test(safeshapes): cover truncated union tags
```

Keep each commit focused; if a change spans several concerns, split it into
multiple commits.

## Opening a pull request

Before you open a PR, make sure:

- [ ] `go build ./...`, `go vet ./...`, and `go test ./...` pass.
- [ ] `gofmt -l .` is clean and `golangci-lint run` reports no issues.
- [ ] `go generate ./...` produces no diff (generated files are committed).
- [ ] New behavior has tests; new types get a `test/data/` fixture exercised from `cmd/goserdegen`.
- [ ] Commits follow Conventional Commits.
- [ ] Exported APIs have godoc, and the README is updated if behavior changed.

Keep pull requests focused on a single change, and write a clear description of
what it does and why. CI runs build, race tests, a fuzz smoke test, and
`golangci-lint` on every push (see [`.github/workflows/ci.yml`](.github/workflows/ci.yml));
all checks must be green before review.

## Releasing

The version goserdegen stamps into every generated file (`// Code generated by
goserdegen <version>`) is the constant `version` in
[`cmd/goserdegen/main.go`](cmd/goserdegen/main.go), the single source of truth.
To cut a release `vX.Y.Z`:

1. Bump `version` to `vX.Y.Z`.
2. Run `go generate ./...` so the committed codecs carry the new header.
3. Commit, then tag and push: `git tag vX.Y.Z && git push origin vX.Y.Z`.

The release workflow rejects any tag whose name does not match `version` (and
re-checks that the generated files are up to date), so the Git release tag and
the generated-file version can never disagree.

## Reporting bugs and proposing features

Open an issue with one of the
[issue forms](https://github.com/tochemey/goserde/issues/new/choose): bug report,
feature request, documentation, or performance. Each form prompts for the details
we need, such as a minimal struct, your Go version and architecture, or
reproducible benchmark numbers.

For "how do I" questions, start a
[discussion](https://github.com/tochemey/goserde/discussions) instead. To report a
security vulnerability, follow the [security policy](SECURITY.md) rather than
opening a public issue.
