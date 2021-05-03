# remove logs
clean:
	@go clean -i ./...
# linter
check: golint vet

testdeps:
	go get -d -v -t ./...
	go get -v $(LINTERS)

golint: testdeps

# exe
build:
	@go build ./...
# test
test: vet
	@go test main_test.go
# vet
vet:
	@go vet $$(go list ./... | grep -v native)
# vendor
vendor:
	@go run _scripts/make.go vendor

.PHONY: vendor test check build vet clean golint errcheck staticcheck testdeps buildandrun