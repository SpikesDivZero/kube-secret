.PHONY: test
test:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	# To see coverage results, run: open coverage.html
