# Use the official Golang image to build the binary
FROM golang:1.21-alpine AS builder

# Install build dependencies for sqlite3 (CGO)
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application with CGO enabled for sqlite3
RUN CGO_ENABLED=1 GOOS=linux go build -o heat-server .

# Use a small alpine image for the final stage
FROM alpine:latest
RUN apk add --no-cache sqlite-libs ca-certificates

WORKDIR /app

# Copy the binary and static files from the builder stage
COPY --from=builder /app/heat-server .
COPY --from=builder /app/static ./static

# Expose the port
EXPOSE 8080

# Run the server
CMD ["./heat-server"]
