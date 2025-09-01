# syntax=docker/dockerfile:1.7

# ---------- Build stage ----------
ARG GO_VERSION=1.24
FROM golang:${GO_VERSION}-bookworm AS builder

WORKDIR /src

# Cache dependencies first
COPY go.mod go.sum ./
RUN go mod download

# Copy sources
COPY . .

# Build static binary (honor buildx TARGET* for multi-arch)
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags="-s -w" -o /out/gcli2api .

# ---------- Runtime stage ----------
FROM alpine:3.20 AS runtime

WORKDIR /app
ENTRYPOINT ["/app/gcli2api"]
CMD ["server", "-c", "/app/config.json"]
EXPOSE 8085
RUN apk --no-cache add ca-certificates

# Run as non-root by default
RUN adduser -D -H -u 10001 appuser
USER 10001

COPY config.json.example /app/config.json.example
COPY --from=builder /out/gcli2api /app/gcli2api
