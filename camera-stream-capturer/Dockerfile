# Start from a Debian-based image because this has dependecnies on C libraries
FROM debian:buster-slim

# Install curl, pkg-config, gcc, ffmpeg and required C libraries
RUN apt-get update && \
    apt-get install -y curl pkg-config gcc ffmpeg libavcodec-dev libswscale-dev && \
    rm -rf /var/lib/apt/lists/*

# Install Go using the latest - WARNING latest now is 1.22.2 which matches my mod version
RUN curl https://dl.google.com/go/go1.22.2.linux-amd64.tar.gz | tar -C /usr/local -xz
ENV PATH="/usr/local/go/bin:${PATH}"

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
# This requires CGO_ENABLED=1 to be set in order to build the app with cgo enabled
RUN CGO_ENABLED=1 GOOS='linux' GOARCH='amd64' GO111MODULE='on'  go build -o main .

# Expose port 8080 to the outside world
# EXPOSE 8080

# Command to run the executable
CMD ["./main"]