FROM golang:1.24.4

WORKDIR /usr/src/app

# Pre-copy/cache go.mod for pre-downloading dependencies and only redownloading
# them in subsequent builds if they change.
COPY go.mod go.sum ./
RUN go mod download

# Copy and build the source code.
COPY . .
RUN go build -v -o /usr/local/bin/app ./cmd/gatekeeper/main.go

# Run the program.
CMD ["app"]