# Use the official Golang image as the build stage
FROM golang:1.24-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Use a minimal base image for the final stage
FROM debian:stable-slim

# Install ImageMagick, dependencies and CA certificates
RUN apt-get update && \
    apt-get install -y \
    imagemagick \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the executable from the build stage
COPY --from=builder /app/main .

# Command to run the executable
CMD ["./main"]
