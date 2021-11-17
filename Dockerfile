FROM golang:1.16 as builder

WORKDIR /app

# Copy go modules
COPY go.mod .
COPY go.sum .

# Download modules
RUN go mod download

# Copy source code
COPY main.go main.go
COPY loadclient/ loadclient/

# Build
RUN CGO_ENABLED=0 GOOS=linux GO111MODULE=on GOARCH=amd64 go build -o logger main.go

# final stage
FROM scratch
COPY --from=builder /app/logger /app/
EXPOSE 8080

ENTRYPOINT ["/app/logger"]
