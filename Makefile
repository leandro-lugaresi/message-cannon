SOURCE_FILES?=$$(go list ./... | grep -v /vendor/)
TEST_PATTERN?=./...
TEST_OPTIONS?=-race

setup: ## Install all the build and lint dependencies
	go get -u github.com/golangci/golangci-lint/cmd/golangci-lint
	go get -u github.com/golang/dep/...
	go get -u github.com/mfridman/tparse
	go get -u golang.org/x/tools/cmd/cover
	dep ensure

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

ci: lint test ## Run all the tests and code checks

build: ## Build a beta version
	go build -race -o ./dist/message-cannon ./main.go

install: ## Install to $GOPATH/src
	go install ./...

# Absolutely awesome: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
