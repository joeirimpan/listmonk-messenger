# Use an official Golang runtime as the base image
FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.18 as builder

ARG TARGETOS
ARG TARGETARCH
ARG TARGETPLATFORM
ARG BUILDPLATFORM

# Set the working directory in the container
WORKDIR /app

# Copy the source code to the container
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} make build

# Use a lightweight Alpine image as the base image for the final container
FROM --platform=${TARGETPLATFORM:-linux/amd64} alpine:latest

# Set the working directory in the container
WORKDIR /app

# Copy the binary from the builder stage to the final container
COPY --from=builder /app/listmonk-messenger.bin .

# Copy the config.toml file (adjust the path if necessary)
#COPY config.toml .

# Expose the port that the application listens on (adjust if necessary)
EXPOSE 8082

# Run the application
CMD ["./listmonk-messenger.bin", "--config", "config.toml", "--msgr", "pinpoint", "--msgr", "ses"]
