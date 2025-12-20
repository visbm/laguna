fmt:
	go fmt ./...
test:
	go test -v -race -cover ./...
lint:
	golangci-lint run
tidy:
	go mod tidy
run:
	go run main.go