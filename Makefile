.PHONY: build install test clean

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

build:
	go build $(LDFLAGS) -o kubectl-enhanced-cli .

build-dev:
	go build -o kubectl-enhanced-cli .

install: build
	cp kubectl-enhanced-cli ~/.local/bin/ 2>/dev/null || cp kubectl-enhanced-cli /usr/local/bin/
	@echo "Creating symlinks for wrapper mode (kctl) and plugin mode (kubectl-enhanced)..."
	ln -sf ~/.local/bin/kubectl-enhanced-cli ~/.local/bin/kctl 2>/dev/null || ln -sf /usr/local/bin/kubectl-enhanced-cli /usr/local/bin/kctl
	ln -sf ~/.local/bin/kubectl-enhanced-cli ~/.local/bin/kubectl-enhanced 2>/dev/null || ln -sf /usr/local/bin/kubectl-enhanced-cli /usr/local/bin/kubectl-enhanced

test:
	go test ./...

test-verbose:
	go test ./... -v

test-cover:
	go test ./... -cover -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

clean:
	rm -f kubectl-enhanced-cli

version:
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"

init-config:
	@mkdir -p ~/.config/kubectl-enhanced
	@if [ ! -f ~/.config/kubectl-enhanced/config.yaml ]; then \
		cp config.example.yaml ~/.config/kubectl-enhanced/config.yaml; \
		echo "Created default config at ~/.config/kubectl-enhanced/config.yaml"; \
	else \
		echo "Config already exists at ~/.config/kubectl-enhanced/config.yaml"; \
	fi

