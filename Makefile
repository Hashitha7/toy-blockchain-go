.PHONY: build test vet fmt clean run

# Build the binary
build:
	go build -o blockchain.exe .

# Run all tests with verbose output
test:
	go test -v ./...

# Run static analysis
vet:
	go vet ./...

# Check formatting (prints unformatted files)
fmt:
	gofmt -l .

# Format all files in place
fmt-fix:
	gofmt -w .

# Full check: vet + test
check: vet test

# Clean build artifacts and chain data
clean:
	del /Q blockchain.exe chain.json 2>nul || true

# Quick demo: mint, transfer, mine, print, validate
demo:
	go run . add-tx -from coinbase -to Alice -amount 100
	go run . add-tx -from Alice -to Bob -amount 25
	go run . mine
	go run . print
	go run . validate
	go run . balances
