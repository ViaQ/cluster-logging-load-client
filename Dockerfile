FROM registry.redhat.io/ubi8/go-toolset:1.17.12 as builder

WORKDIR /app

# Copy go modules
COPY go.mod .
COPY go.sum .

# Download modules
RUN go mod download

# Copy source code
COPY main.go main.go
COPY internal/ internal/
USER 0

# Build
RUN CGO_ENABLED=0 GOOS=linux GO111MODULE=on GOARCH=amd64 go build -o logger main.go

# final stage
FROM registry.access.redhat.com/ubi8/ubi:8.6
COPY --from=builder /app/logger /app/
RUN chmod 755 /app/*
EXPOSE 8080

ENTRYPOINT ["/app/logger"]
