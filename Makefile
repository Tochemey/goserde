# goserde — common tasks

.PHONY: all build test cover cover-html gen bench bench-shapes vet clean compare

all: gen build test

build:
	go build ./...

test:
	go test ./...

# Coverage for the library packages only: -coverpkg restricts the measured set to
# codec and the generator. The example demo, the test/data fixtures, and the
# separate benchcompare module are excluded.
COVERPKG = ./codec/...,./cmd/goserdegen/...

cover:
	go test ./... -covermode=atomic -coverpkg=$(COVERPKG) -coverprofile=coverage.out
	go tool cover -func=coverage.out | tail -1

cover-html: cover
	go tool cover -html=coverage.out

# Regenerate all codecs from annotated structs.
gen:
	go generate ./...

vet:
	go vet ./...

# Benchmark goserde across the diverse generated shapes.
bench-shapes:
	go test ./cmd/goserdegen/ -bench=. -benchmem -run='^$$'

# Benchmark the core codec package (hand-tuned Record).
bench:
	go test ./codec/ -bench=. -benchmem -run='^$$'

# Head-to-head vs mus and benc. Requires internet (separate module).
compare:
	cd benchcompare && go mod tidy && go test -bench=. -benchmem -run='^$$' -count=3

clean:
	go clean ./...
