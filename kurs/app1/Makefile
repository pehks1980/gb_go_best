# linter vet
check: lint vet
# golangci-lint (must be installed and setup)
lint:
	golangci-lint run
# exe
build:
	go build ./cmd/main.go
# test filter
test:
	go test ./filter/...
# vet
vet:
	go vet $$(go list ./... | grep -v native)

hooks:
	pre-commit run --all-files


.PHONY: test check build vet lint hooks
