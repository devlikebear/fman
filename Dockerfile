# syntax=docker/dockerfile:1

# Stage 1: Build the Go application
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the fman executable
# CGO_ENABLED=0 is important for static linking, making the binary portable
# -ldflags -s -w reduces the binary size by removing debug info
RUN CGO_ENABLED=0 go build -o fman -ldflags "-s -w" .

# Stage 2: Create the final minimal image
FROM alpine/git:latest # Using alpine/git for git and bash, which might be useful for AI suggestions

WORKDIR /app

# Copy the fman executable from the builder stage
COPY --from=builder /app/fman .

# Create the .fman directory for config and DB
RUN mkdir -p /root/.fman

# Set the entrypoint to the fman executable
ENTRYPOINT ["./fman"]

# Default command (can be overridden)
CMD ["--help"]
