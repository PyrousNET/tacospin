# === Stage 1: Build the Go binary ===
FROM golang:1.22 AS builder

WORKDIR /build

# Copy the entire project and build it statically
COPY . .

# Build a statically linked Linux AMD64 binary
RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o tacoSpin ./

# === Stage 2: Minimal runtime ===
FROM gcr.io/distroless/static:nonroot

# Set working directory and copy the binary from the builder stage
WORKDIR /app
COPY --from=builder /build/tacoSpin /app/tacoSpin
COPY --from=builder /build/src/static /app/static

# Run as non-root user (provided by distroless)
USER nonroot

# Start the Go service
ENTRYPOINT ["./tacoSpin"]


