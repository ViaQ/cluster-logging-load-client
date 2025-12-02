FROM --platform=$BUILDPLATFORM docker.io/library/golang:1.25.5 AS builder

ARG TARGETOS
ARG TARGETARCH

ENV GOOS=$TARGETOS
ENV GOARCH=$TARGETARCH

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
