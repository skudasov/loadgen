export GOPATH ?= $(shell go env GOPATH)
export GO111MODULE=on

.PHONY: test
test:
	go test ./...

.PHONY: lint
lint:
	docker run --rm -v $(pwd):/app -w /app golangci/golangci-lint:v1.21.0 golangci-lint run -v
