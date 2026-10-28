SCENARIO ?= scenarios/baseline.yaml
SEED     ?= 42

.PHONY: build test vet fmt check run update-golden milestone1

build:
	go build -o bin/silosim ./cmd/silosim

test:
	go test ./...

vet:
	go vet ./...

fmt:
	@test -z "$$(gofmt -l .)" || { gofmt -l .; echo "run gofmt -w on the files above"; exit 1; }

# The golden test (cmd/silosim TestGolden) runs as part of test.
check: build fmt vet test

# Only with a reason, given in the commit message (CLAUDE.md, Tests).
update-golden:
	go test ./cmd/silosim -run TestGolden -update

run: build
	bin/silosim run --scenario $(SCENARIO) --seed $(SEED) $(ARGS)

# T1.5 milestone check: 10 seeds x 250 years, demography only (minutes).
milestone1:
	SILOSIM_MILESTONE=1 go test ./internal/modules/demography -run TestMilestone1 -v -timeout 60m | grep -E 'seed |^(ok|FAIL|---)'
