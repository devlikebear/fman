.PHONY: all build run test clean docker-build docker-run

APP_NAME=fman
BUILD_DIR=./bin

all: build

build:
	@echo "Building $(APP_NAME)..."
	@go build -o $(BUILD_DIR)/$(APP_NAME) .
	@echo "Build complete. Executable: $(BUILD_DIR)/$(APP_NAME)"

run:
	@echo "Running $(APP_NAME)..."
	@$(BUILD_DIR)/$(APP_NAME) $(ARGS)

test:
	@echo "Running tests with coverage..."
	@go test ./... -v -coverprofile=coverage.out -p=1
	@echo "Checking test coverage..."
	@go tool cover -func=coverage.out

clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR) coverage.out
	@rm -f $(APP_NAME)
	@echo "Cleanup complete."

# Docker commands
DOCKER_IMAGE_NAME=fman-app

docker-build:
	@echo "Building Docker image $(DOCKER_IMAGE_NAME)..."
	@docker build -t $(DOCKER_IMAGE_NAME) .
	@echo "Docker image built: $(DOCKER_IMAGE_NAME)"

docker-run:
	@echo "Running Docker container $(DOCKER_IMAGE_NAME)..."
	@docker run --rm -it \
		-v $(shell pwd)/.fman:/root/.fman \
		-v $(shell pwd)/test_data:/app/test_data \
		$(DOCKER_IMAGE_NAME) $(ARGS)
	@echo "Docker container stopped."

# Helper for running commands inside docker container
docker-exec:
	@echo "Executing command inside Docker container..."
	@docker run --rm -it \
		-v $(shell pwd)/.fman:/root/.fman \
		-v $(shell pwd)/test_data:/app/test_data \
		$(DOCKER_IMAGE_NAME) $(ARGS)

# Example usage:
# make build
# make run ARGS="scan /app/test_data"
# make test
# make docker-build
# make docker-run ARGS="scan /app/test_data"
# make docker-exec ARGS="organize --ai /app/test_data"
