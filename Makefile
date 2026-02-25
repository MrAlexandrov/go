BINARY_NAME=family-tree
BUILD_DIR=build

.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) .

.PHONY: run
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)

.PHONY: deps
deps:
	@echo "Installing dependencies..."
	go mod download

.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...
