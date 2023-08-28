FROM docker.io/library/golang:1.20.6 as builder

WORKDIR /app

# Copy go modules
COPY go.mod go.sum ./

# Download modules
RUN go mod download

# Copy source code
COPY main.go main.go
COPY internal/ internal/

ENV CGO_ENABLED=0

# Build
RUN go build -o logger main.go

# final stage
FROM docker.io/library/busybox

COPY --from=builder /app/logger /bin/

ENTRYPOINT ["/bin/logger"]
