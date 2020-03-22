clean:
	rm -rvf vendor
	rm -rvf bin/*

test: lint
	go test -race -coverprofile=coverage.out  -v ./...

lint:
	golangci-lint run

cover:
	go tool cover -html=coverage.out

