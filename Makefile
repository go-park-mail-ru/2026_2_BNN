GO         ?= go
COVER_FILE ?= cover.out
COVER_HTML ?= cover.html

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'

.PHONY: test
test:
	$(GO) test ./...

.PHONY: cover
cover:
	$(GO) test -covermode=atomic -coverprofile=$(COVER_FILE) ./...
	$(GO) tool cover -func=$(COVER_FILE)

.PHONY: cover-html
cover-html: cover
	$(GO) tool cover -html=$(COVER_FILE) -o $(COVER_HTML)
	@echo ">> $(COVER_HTML)"