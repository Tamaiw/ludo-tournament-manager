# syntax=docker/dockerfile:1
# Multi-stage build: builder + alpine runtime.
# This Dockerfile produces both the app and seed binaries in a single image;
# the dev/prod compose files select which one to run via the command override.

ARG GO_VERSION=1.23

FROM golang:${GO_VERSION}-alpine AS builder
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY backend/go.mod backend/go.sum ./backend/
WORKDIR /src/backend
RUN go mod download
COPY backend/ .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/seed   ./cmd/seed

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata \
 && addgroup -S app && adduser -S -G app app
WORKDIR /app
COPY --from=builder /out/server /app/server
COPY --from=builder /out/seed   /app/seed
RUN mkdir -p /data && chown -R app:app /data
USER app
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["/app/server"]