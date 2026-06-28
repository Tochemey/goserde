# goserde — common tasks

.PHONY: all build test cover cover-html gen bench bench-shapes vet clean compare

all: gen build test

build:
	go build ./...

test:
	go test ./...

# Coverage for the root module only. The separate benchcompare module is not
# reached by ./... and is excluded from coverage by design.
cover:
	go test ./... -covermode=atomic -coverprofile=coverage.out
	go tool cover -func=coverage.out | tail -1

cover-html: cover
	go tool cover -html=coverage.out

# Regenerate all codecs from annotated structs.
gen:
	go generate ./...

vet:
	go vet ./...

# Benchmark goserde across the six diverse shapes.
bench-shapes:
	go test ./test/data/shapes/ -bench=. -benchmem -run='^$$'

# Benchmark the core runtime package (hand-tuned Record).
bench:
	go test ./runtime/ -bench=. -benchmem -run='^$$'

# Head-to-head vs mus and benc. Requires internet (separate module).
compare:
	cd benchcompare && go mod tidy && go test -bench=. -benchmem -run='^$$' -count=3

clean:
	go clean ./...
