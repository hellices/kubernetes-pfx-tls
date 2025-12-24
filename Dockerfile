# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /workspace

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY main.go ./
COPY pkg/ pkg/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o pfx-converter main.go

# Final stage
FROM alpine:3.18

WORKDIR /

# Copy the binary from builder
COPY --from=builder /workspace/pfx-converter .

# Run as non-root user
USER 65532:65532

ENTRYPOINT ["/pfx-converter"]
