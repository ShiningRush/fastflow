# must ensure your go version >= 1.16
.PHONY: install
install:
	go install github.com/golang/mock/mockgen@v1.6.0
	go install golang.org/x/tools/cmd/goimports@latest

.PHONY: tidy
tidy:
	go mod tidy
	@$(foreach dir,$(shell go list -f {{.Dir}} ./...),goimports -w $(dir);)
	@$(foreach dir,$(shell go list -f {{.Dir}} ./...),gofmt -s -w $(dir);)

.PHONY: test
test:
	go test -race -coverprofile=coverage.out ./...

# include integration tests
.PHONY: g-test
g-test:
	go test -race -coverprofile=coverage.out ./... -tags=integration

# usage
# you must run `make install` to install necessary tools
# make mock dir=path/to/mock
.PHONY: mock
mock:
	for file in `find . -type d \( -path ./.git -o -path ./.github \) -prune -o -name '*.go' -print | xargs grep --files-with-matches -e '//go:generate mockgen'`; do \
		go generate $$file; \
	done

.PHONY: build
GO     := GO111MODULE=on go
GOBUILD := CGO_ENABLED=0 $(GO) build
build:
	GOARCH=amd64 GOOS=linux $(GOBUILD) -gcflags "all=-N -l" -o dist/fastflow examples/mysql/main.go
