SCENARIO ?= scenarios/baseline.yaml
SEED     ?= 42

.PHONY: build test vet fmt check run update-golden milestone1 stability bench

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

# T3.2a stability check: 10 seeds x 100 years, demography + health + work + resources.
stability:
	SILOSIM_STABILITY=1 go test ./internal/modules/resources -run TestStability -v -timeout 60m | grep -E 'seed |^(ok|FAIL|---)'

# T5.4: time one full baseline run (50 + 200 years). Close other programs first;
# on a laptop the CPU governor and other load move this by ±30%.
bench: build
	bin/silosim run --scenario scenarios/baseline.yaml --seed 42
	@grep runtimeSeconds runs/baseline-42/run.json
