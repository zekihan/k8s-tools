VERSION := $(shell cat VERSION)

.PHONY: lint test build check release-check
lint:
	@unformatted=$$(gofmt -l *.go) || exit $$?; \
	if [ -n "$$unformatted" ]; then \
		printf '%s\n' "$$unformatted"; \
		exit 1; \
	fi
	go vet ./...

test:
	go test -race ./...

build:
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o dist/k8s-tools .

release-check:
	goreleaser check
	goreleaser release --snapshot --clean --skip=publish

check: lint test build
