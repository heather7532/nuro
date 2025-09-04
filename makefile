APP := nuro
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

all: build

build:
	@echo "==> building $(APP) ($(VERSION))"
	go build -ldflags "$(LDFLAGS)" -o bin/$(APP) .

test:
	go test ./...

clean:
	rm -rf bin

release: clean
	@mkdir -p dist
	GOOS=darwin GOARCH=amd64  go build -ldflags "$(LDFLAGS)" -o dist/$(APP)_$(VERSION)_darwin_amd64 .
	GOOS=darwin GOARCH=arm64  go build -ldflags "$(LDFLAGS)" -o dist/$(APP)_$(VERSION)_darwin_arm64 .
	GOOS=linux  GOARCH=amd64  go build -ldflags "$(LDFLAGS)" -o dist/$(APP)_$(VERSION)_linux_amd64 .
	GOOS=linux  GOARCH=arm64  go build -ldflags "$(LDFLAGS)" -o dist/$(APP)_$(VERSION)_linux_arm64 .
	GOOS=windows GOARCH=amd64  go build -ldflags "$(LDFLAGS)" -o dist/$(APP)_$(VERSION)_windows_amd64.exe .

.PHONY: all build test clean release