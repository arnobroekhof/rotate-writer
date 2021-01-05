GOPATH?=$(go env GOPATH)
LINT_VERSION?="v1.34.1"


.PHONY: go/test
go/test: go/lint
	go test -race -coverprofile=coverage.out  -v ./...

.PHONY: go/lint
go/lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin $(LINT_VERSION)
	$(GOPATH)/bin/golangci-lint run -v ./...

.PHONY: go/cover
go/cover:
	go tool cover -html=coverage.out

