.PHONY: build install clean test build-all

# Build for current platform
build:
	go build -o git-wt .

# Install to local bin
install:
	go build -o git-wt .
	mkdir -p ~/bin
	mv git-wt ~/bin/
	@echo "git-wt installed to ~/bin/git-wt"
	@echo "Make sure ~/bin is in your PATH"

# Build for all platforms
build-all:
	@echo "Building for all platforms..."
	GOOS=darwin GOARCH=arm64 go build -o dist/git-wt-darwin-arm64 .
	GOOS=darwin GOARCH=amd64 go build -o dist/git-wt-darwin-amd64 .
	GOOS=linux GOARCH=amd64 go build -o dist/git-wt-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -o dist/git-wt-linux-arm64 .
	GOOS=windows GOARCH=amd64 go build -o dist/git-wt-windows-amd64.exe .
	@echo "Build complete! Binaries in dist/"

# Clean build artifacts
clean:
	rm -f git-wt
	rm -rf dist/

# Run tests
test:
	go test -v ./...

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	go vet ./...
