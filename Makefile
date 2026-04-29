# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT
#
# END USERS / AI ASSISTANTS: To upgrade an existing lark-cli install, run
# `lark-cli update`. `make install` here is for CONTRIBUTORS building from
# source — it will NOT match official release artifacts and may break the
# self-update flow.

BINARY   := lark-cli
MODULE   := github.com/larksuite/cli
VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
DATE     := $(shell date +%Y-%m-%d)
LDFLAGS  := -s -w -X $(MODULE)/internal/build.Version=$(VERSION) -X $(MODULE)/internal/build.Date=$(DATE)
PREFIX   ?= /usr/local

.PHONY: build vet test unit-test integration-test install uninstall clean fetch_meta

fetch_meta:
	python3 scripts/fetch_meta.py

build: fetch_meta
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) .

vet: fetch_meta
	go vet ./...

unit-test: fetch_meta
	go test -race -gcflags="all=-N -l" -count=1 ./cmd/... ./internal/... ./shortcuts/...

integration-test: build
	go test -v -count=1 ./tests/...

test: vet unit-test integration-test

install: build
	@if [ -n "$$I_AM_A_CONTRIBUTOR" ]; then \
		: ; \
	elif [ -t 0 ] && [ -t 1 ]; then \
		echo "" ; \
		echo "  make install builds from source — for CONTRIBUTORS only." ; \
		echo "  To upgrade an existing lark-cli, run: lark-cli update" ; \
		echo "" ; \
		printf "  Continue installing from source? (y/N) " ; \
		read ans ; \
		case "$$ans" in y|Y|yes|YES) ;; *) echo "Aborted." ; exit 1 ;; esac ; \
	else \
		echo "make install: refusing in non-interactive mode." ; \
		echo "  To upgrade lark-cli: run \`lark-cli update\`." ; \
		echo "  To install from source non-interactively: I_AM_A_CONTRIBUTOR=1 make install" ; \
		exit 1 ; \
	fi
	install -d $(PREFIX)/bin
	install -m755 $(BINARY) $(PREFIX)/bin/$(BINARY)
	@echo "OK: $(PREFIX)/bin/$(BINARY) ($(VERSION))"

uninstall:
	rm -f $(PREFIX)/bin/$(BINARY)

clean:
	rm -f $(BINARY)
