GO            ?= go
COVER_FILE    ?= cover.out
COVER_HTML    ?= cover.html
COVER_EXCLUDE ?= (^|/)(mocks?|mock_[^/]*)/|_mock\.go:|_gen\.go:|\.pb\.go:

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

.PHONY: test
test:
	$(GO) test ./...

.PHONY: test-verbose
test-verbose:
	$(GO) test -v ./...

.PHONY: cover
cover:
	@echo ">> Collecting coverage..."
	@$(GO) test -covermode=atomic -coverprofile=$(COVER_FILE) ./... > /dev/null
	@echo ">> Raw cover.out lines: $$(wc -l < $(COVER_FILE))"
	@echo ">> Filtering mocks..."
	@head -n1 $(COVER_FILE) > $(COVER_FILE).tmp
	@tail -n +2 $(COVER_FILE) | grep -Ev '$(COVER_EXCLUDE)' >> $(COVER_FILE).tmp || true
	@mv $(COVER_FILE).tmp $(COVER_FILE)
	@echo ">> After filter lines:   $$(wc -l < $(COVER_FILE))"
	@echo ""
	@echo ">> Coverage by package:"
	@$(GO) tool cover -func=$(COVER_FILE) | \
		awk 'NR>1 && $$0 !~ /^total:/ { \
			n = split($$1, parts, "/"); \
			pkg = ""; \
			for (i = 1; i < n; i++) pkg = (pkg == "" ? parts[i] : pkg "/" parts[i]); \
			sub(/%/, "", $$3); \
			sum[pkg] += $$3; cnt[pkg]++; \
		} \
		END { \
			for (key in sum) printf "   %-50s %6.1f%%\n", key, sum[key]/cnt[key]; \
		}' | sort
	@echo ""
	@echo ">> Total coverage:"
	@$(GO) tool cover -func=$(COVER_FILE) | awk '/^total:/ {print "   " $$3}'

.PHONY: cover-html
cover-html: cover
	$(GO) tool cover -html=$(COVER_FILE) -o $(COVER_HTML)
	@echo ">> $(COVER_HTML)"

.PHONY: clean
clean:
	rm -f $(COVER_FILE) $(COVER_FILE).tmp $(COVER_HTML)