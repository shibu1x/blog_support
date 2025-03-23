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

# Build the Go app
RUN go build -o main ./main.go

# Use a minimal base image for the final stage
FROM alpine:latest

# Install ImageMagick
RUN apk add --no-cache imagemagick

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the executable from the build stage
COPY --from=builder /app/main .

# Command to run the executable
CMD ["./main"]
