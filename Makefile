GO ?= go
PKG ?= ./...

.PHONY: test test-verbose testv

test:
	$(GO) test $(PKG)

test-verbose:
	$(GO) test -v $(PKG)

testv:test-verbose
