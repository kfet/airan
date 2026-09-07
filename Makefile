.PHONY: all check fmt fmtcheck vet staticcheck run-tests build open_coverage clean \
	check-installsh

# install.sh is GENERATED from install.sh.json by the canonical distkit
# template (github.com/kfet/distkit/installsh) — never hand-edited. Pinned by
# version and run with `go run`, so airan's own module stays stdlib-only.
INSTALLSH := go run github.com/kfet/distkit/cmd/distkit-installsh@v0.1.4

BINDIR := bin

# Quiet runner: $(call RUN,label,cmd) — runs cmd silently, prints "✓ label" on
# success, dumps captured output and exits non-zero on failure. Set V=1 for
# verbose output.
ifdef V
  define RUN
	@echo "→ $(1)"
	@$(2)
  endef
else
  define RUN
	@_log=$$(mktemp); \
	if ( $(2) ) > $$_log 2>&1; then \
		echo "✓ $(1)"; rm -f $$_log; \
	else \
		rc=$$?; cat $$_log; rm -f $$_log; exit $$rc; \
	fi
  endef
endif

# Default target. gofmt + go vet + staticcheck + unit tests with the race
# detector, shuffled order, fresh cache, and a 100% coverage gate, then the
# binary build, then the install.sh drift gate. CI runs this same target
# verbatim — no separate "fast" mode. To iterate faster locally, run
# `go test ./...` directly.
all: run-tests build check-installsh
	@echo "✓ all green"

# Static gates (gofmt + go vet + staticcheck if installed).
check: fmtcheck vet staticcheck

# Regenerate the root install.sh from install.sh.json. The Makefile is a
# prerequisite too: bumping the pinned distkit version changes the output.
install.sh: install.sh.json Makefile
	$(call RUN,generate install.sh,$(INSTALLSH) -o $@)

# Dev-only drift gate: fails the build when the checked-in install.sh no
# longer matches the template + spec, so a stale copy never reaches a user.
check-installsh:
	$(call RUN,install.sh not drifted,$(INSTALLSH) -check)

fmt:
	@gofmt -w .

fmtcheck:
	$(call RUN,gofmt clean,out=$$(gofmt -l .); test -z "$$out" || { echo "gofmt offenders (run 'make fmt'):"; echo "$$out"; exit 1; })

vet:
	$(call RUN,go vet clean,go vet ./...)

# staticcheck and covgate run via the go.mod `tool` directive — no manual
# install, version pinned in go.mod/go.sum. See `go tool` (Go 1.24+).
COVGATE := go tool covgate
staticcheck:
	$(call RUN,staticcheck clean,go tool staticcheck ./...)

# Run unit tests with race + shuffle + fresh cache + 100% coverage gate.
run-tests: check
	@go clean -testcache
	$(call RUN,tests pass,go test -race -shuffle=on -cover ./... -coverprofile=coverage.tmp.out)
	$(call RUN,coverage clean,$(COVGATE) -profile=coverage.tmp.out -out=coverage.out -ignore=.covignore -min=100)
	@rm -f coverage.tmp.out

build: | $(BINDIR)
	$(call RUN,build airan,go build -trimpath -o $(BINDIR)/airan ./cmd/airan)

# Build and install into PREFIX (default ~/.local).
PREFIX ?= $(HOME)/.local
install: build
	@mkdir -p $(PREFIX)/bin
	@install -m 0755 $(BINDIR)/airan $(PREFIX)/bin/airan
	@echo "installed: $(PREFIX)/bin/airan"

# Build binaries for all supported platforms
build-all:
	@mkdir -p dist
	@for platform in darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 linux/386 linux/arm freebsd/amd64 freebsd/386 windows/amd64 windows/386 windows/arm64; do \
		GOOS=$${platform%/*}; \
		GOARCH=$${platform#*/}; \
		ext=""; \
		if [ "$$GOOS" = "windows" ]; then ext=".exe"; fi; \
		echo "Building $$GOOS-$$GOARCH..."; \
		GOOS=$$GOOS GOARCH=$$GOARCH go build -trimpath -ldflags="-s -w" -o dist/airan-$$GOOS-$$GOARCH$$ext ./cmd/airan || exit 1; \
	done

$(BINDIR):
	@mkdir -p $(BINDIR)

open_coverage:
	go tool cover -html=coverage.out

clean:
	rm -rf $(BINDIR) coverage.out coverage.tmp.out
