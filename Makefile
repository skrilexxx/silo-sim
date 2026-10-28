SCENARIO ?= scenarios/baseline.yaml
SEED     ?= 42

.PHONY: build test vet fmt check run

build:
	go build -o bin/silosim ./cmd/silosim

test:
	go test ./...

vet:
	go vet ./...

fmt:
	@test -z "$$(gofmt -l .)" || { gofmt -l .; echo "run gofmt -w on the files above"; exit 1; }

# Golden test joins this in T0.5.
check: build fmt vet test

run: build
	bin/silosim run --scenario $(SCENARIO) --seed $(SEED) $(ARGS)
