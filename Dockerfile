# Build stage
# Pin to a specific patch version to avoid unexpected base image changes.
# To update: change the version tag and update the SHA256 digest below.
# Get digest: docker pull golang:1.25.4-alpine3.21 && docker inspect --format='{{index .RepoDigests 0}}' golang:1.25.4-alpine3.21
FROM golang:1.25.4-alpine3.21 AS builder

# Build arguments for multi-architecture support
ARG TARGETOS=linux
ARG TARGETARCH=amd64

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with optimizations for the target architecture
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o argazer .

# Final stage
# Pin to a specific patch version. Get digest: docker pull alpine:3.21.3 && docker inspect --format='{{index .RepoDigests 0}}' alpine:3.21.3
FROM alpine:3.21.3

# Install ca-certificates for HTTPS requests and timezone data
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S argazer && \
    adduser -u 1001 -S argazer -G argazer

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /build/argazer .

# Copy example config (optional)
COPY --from=builder /build/config.yaml.example .

# Change ownership to non-root user
RUN chown -R argazer:argazer /app

# Switch to non-root user
USER argazer

# Run the application
# Config can be mounted at /app/config.yaml or passed via environment variables
ENTRYPOINT ["./argazer"]
CMD ["--config", "/app/config.yaml"]

