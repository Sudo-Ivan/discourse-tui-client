BINARY_NAME=discourse-tui

GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) mod download

BUILD_DIR=build
MAN_DIR=man

LDFLAGS_RELEASE=-ldflags="-s -w"

LDFLAGS_DEBUG=-ldflags=""

all: release man

release:
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS_RELEASE) -a -installsuffix cgo -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd

debug:
	$(GOBUILD) $(LDFLAGS_DEBUG) -o $(BUILD_DIR)/$(BINARY_NAME)-debug ./cmd

man: $(BUILD_DIR)/$(BINARY_NAME).1.gz

$(BUILD_DIR)/$(BINARY_NAME).1.gz: $(MAN_DIR)/$(BINARY_NAME).1
	mkdir -p $(BUILD_DIR)
	gzip -c $< > $@

run:
	$(GOCMD) run ./cmd

cross-build:
	@if [ -z "$(GOOS)" ] || [ -z "$(GOARCH)" ]; then \
		echo "Error: GOOS and GOARCH must be specified"; \
		echo "Usage: make cross-build GOOS=linux GOARCH=amd64 BUILD_TYPE=release STATIC=1"; \
		exit 1; \
	fi
	@if [ "$(STATIC)" = "1" ]; then \
		BUILD_CMD="CGO_ENABLED=0 $(GOBUILD) -a -installsuffix cgo"; \
		SUFFIX="-static"; \
	else \
		BUILD_CMD="$(GOBUILD)"; \
		SUFFIX=""; \
	fi; \
	if [ "$(BUILD_TYPE)" = "debug" ]; then \
		GOOS=$(GOOS) GOARCH=$(GOARCH) $$BUILD_CMD $(LDFLAGS_DEBUG) -o $(BUILD_DIR)/$(BINARY_NAME)-$(GOOS)-$(GOARCH)$${SUFFIX}-debug ./cmd; \
	else \
		GOOS=$(GOOS) GOARCH=$(GOARCH) $$BUILD_CMD $(LDFLAGS_RELEASE) -o $(BUILD_DIR)/$(BINARY_NAME)-$(GOOS)-$(GOARCH)$${SUFFIX} ./cmd; \
	fi
	@if [ -n "$(GOARM)" ]; then \
		if [ "$(BUILD_TYPE)" = "debug" ]; then \
			mv $(BUILD_DIR)/$(BINARY_NAME)-$(GOOS)-$(GOARCH)$${SUFFIX}-debug $(BUILD_DIR)/$(BINARY_NAME)-$(GOOS)-$(GOARCH)-v$(GOARM)$${SUFFIX}-debug; \
		else \
			mv $(BUILD_DIR)/$(BINARY_NAME)-$(GOOS)-$(GOARCH)$${SUFFIX} $(BUILD_DIR)/$(BINARY_NAME)-$(GOOS)-$(GOARCH)-v$(GOARM)$${SUFFIX}; \
		fi; \
	fi

clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f $(MAN_DIR)/*.gz

test:
	$(GOTEST) -v ./...

scan:
	@if command -v gosec >/dev/null 2>&1; then \
		echo "Running gosec security scan..."; \
		gosec ./...; \
	else \
		echo "gosec not found. Install it from https://github.com/securego/gosec"; \
		exit 1; \
	fi

deps:
	$(GOGET)

install-deps:
	$(GOGET) -u

install: release man
	./install.sh

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

.PHONY: all release debug man run cross-build clean test scan deps install-deps install
