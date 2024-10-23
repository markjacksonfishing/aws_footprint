# Step 1: Use an official Golang image as the base
FROM golang:1.23-alpine AS builder

# Set the current working directory inside the container
WORKDIR /app

# Copy the go.mod and go.sum files to the working directory
COPY go.mod go.sum ./

# Download the Go module dependencies
RUN go mod download

# Copy the rest of the application code to the working directory
COPY . .

# Build the Go application
RUN go build -o aws_footprint aws_footprint.go

# Step 2: Use a smaller base image for the final executable
FROM alpine:latest

# Set the working directory for the final image
WORKDIR /root/

# Copy the compiled Go binary from the builder container
COPY --from=builder /app/aws_footprint .

# Install necessary packages (for AWS CLI if needed)
RUN apk --no-cache add ca-certificates

# Command to run the application
CMD ["./aws_footprint"]
