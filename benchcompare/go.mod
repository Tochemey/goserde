// Separate module: optional competitor benchmarks (mus, benc).
// Kept out of the root module so the core has ZERO external dependencies.
//
// To run these on a machine with internet:
//   cd benchcompare && go mod tidy && go test -bench=. -benchmem
module github.com/tochemey/goserde/benchcompare

go 1.26

require (
	github.com/deneonet/benc v1.1.8
	github.com/mus-format/mus-go v0.10.2
	github.com/tochemey/goserde v0.0.0
)

require (
	github.com/mus-format/common-go v0.0.0-20260324174526-3d8f1741b5a2 // indirect
	golang.org/x/exp v0.0.0-20231110203233-9a3e6036ecaa // indirect
)

replace github.com/tochemey/goserde => ../
