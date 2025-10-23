# Multi-stage build for Sentinel
FROM golang:1.21-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o /out/sentinel ./

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /out/sentinel /sbin/sentinel

# Create a non-root user (distroless uses nonroot user already)
USER nonroot:nonroot

# Recommended runtime environment variables:
# - PORT (default 8080)
# - JWT_SECRET (required)
# - TLS_ENABLED (true|false)
# - TLS_CERT_FILE and TLS_KEY_FILE when TLS_ENABLED=true

EXPOSE 8080
ENTRYPOINT ["/sbin/sentinel"]
