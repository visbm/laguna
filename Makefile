test:
	go test -v -race -cover ./...
lint:
	golangci-lint run
fmt:
	go fmt ./...
tidy:
	go mod tidy
run:
	go run main.go