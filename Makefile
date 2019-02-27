SOURCE_FILES?=$$(go list ./... | grep -v /vendor/)
TEST_PATTERN?=./...
TEST_OPTIONS?=-race

setup: ## Install all the build and lint dependencies
	GO111MODULE=off go get github.com/golangci/golangci-lint/cmd/golangci-lint
	GO111MODULE=off go get github.com/mfridman/tparse
	GO111MODULE=off go get golang.org/x/tools/cmd/cover
	go get ./...

test: ## Run all the tests
	go test $(TEST_OPTIONS) -covermode=atomic -coverprofile=coverage.txt -timeout=1m -cover -json $(SOURCE_FILES) | tparse -all

bench: ## Run the benchmark tests
	go test -bench=. $(TEST_PATTERN)

cover: teste ## Run all the tests and opens the coverage report
	go tool cover -html=coverage.txt

fmt: ## gofmt and goimports all go files
	find . -name '*.go' -not -wholename './vendor/*' | while read -r file; do gofmt -w -s "$$file"; goimports -w "$$file"; done

lint: ## Run all the linters
	golangci-lint run

ci: lint ## Run all the tests and code checks
	go test $(TEST_OPTIONS) -covermode=atomic -coverprofile=coverage.txt -timeout=1m -cover -json $(SOURCE_FILES) | tparse -all -smallscreen

build: ## Build a beta version
	go build -race -o ./dist/message-cannon ./main.go

install: ## Install to $GOPATH/src
	go install ./...

# Absolutely awesome: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
