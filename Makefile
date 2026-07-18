GO ?= go

COVERAGE_MIN ?= 98.0
COVERAGE_OUT ?= coverage.out
COVERAGE_TEXT ?= coverage.txt
COVERAGE_HTML ?= coverage.html
COVERAGE_TOTAL ?= coverage-total.txt
FUZZ_COUNT ?= 100000x
FUZZ_TARGETS ?= FuzzParse FuzzCompactSize

.PHONY: test check fuzz-smoke ci bench

test:
	GOFLAGS=-mod=readonly $(GO) test -count=1 ./...

check:
	@set -e; unformatted="$$(gofmt -l .)"; \
	if [ -n "$$unformatted" ]; then \
		printf 'The following files need gofmt:\n%s\n' "$$unformatted"; \
		exit 1; \
	fi
	$(GO) mod tidy -diff
	GOFLAGS=-mod=readonly $(GO) vet ./...
	GOFLAGS=-mod=readonly $(GO) test -race -shuffle=on -count=1 \
		-covermode=atomic -coverprofile=$(COVERAGE_OUT) ./...
	$(GO) tool cover -func=$(COVERAGE_OUT) > $(COVERAGE_TEXT)
	@cat $(COVERAGE_TEXT)
	$(GO) tool cover -html=$(COVERAGE_OUT) -o=$(COVERAGE_HTML)
	@set -e; \
	total="$$(awk '/^total:/ { gsub(/%/, "", $$3); print $$3 }' $(COVERAGE_TEXT))"; \
	test -n "$$total"; \
	printf '%s\n' "$$total" > $(COVERAGE_TOTAL); \
	awk -v got="$$total" -v min="$(COVERAGE_MIN)" \
		'BEGIN { if (got + 0 < min + 0) exit 1 }'; \
	printf 'Coverage: %s%% (minimum %s%%)\n' "$$total" "$(COVERAGE_MIN)"

fuzz-smoke:
	@set -e; for target in $(FUZZ_TARGETS); do \
		printf 'Fuzzing %s for %s executions\n' "$$target" "$(FUZZ_COUNT)"; \
		GOFLAGS=-mod=readonly CGO_ENABLED=0 $(GO) test -run='^$$' \
			-fuzz="^$${target}$$" -fuzztime=$(FUZZ_COUNT) -parallel=2 .; \
	done

ci:
	$(MAKE) check
	$(MAKE) fuzz-smoke

bench:
	GOFLAGS=-mod=readonly $(GO) test -run='^$$' -bench=. -benchmem .
