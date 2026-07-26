# Build stage
FROM golang:1.26-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /src

# Copy go.mod and go.sum if they exist
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the GPU monitor binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gpu-monitor ./examples/external-plugins/gpu-monitor/

# Runtime stage
FROM nvidia/cuda:12.0-base-ubuntu20.04

# Install nvidia-smi (included in CUDA base image) and ca-certificates
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN groupadd -r npd && useradd -r -g npd npd

# Create directories for socket
RUN mkdir -p /var/run/npd && chown npd:npd /var/run/npd

# Copy binary from builder
COPY --from=builder /src/gpu-monitor /usr/local/bin/

# Set ownership and permissions
RUN chown npd:npd /usr/local/bin/gpu-monitor && \
    chmod +x /usr/local/bin/gpu-monitor

# Switch to non-root user
USER npd

# Expose socket directory as volume
VOLUME ["/var/run/npd"]

# Default command
ENTRYPOINT ["/usr/local/bin/gpu-monitor"]
CMD ["--socket-path=/var/run/npd-gpu-monitor.sock", "--temp-threshold=85", "--memory-threshold=95.0"]