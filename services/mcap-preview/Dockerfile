# syntax=docker/dockerfile:1
# Mirrors backend/Dockerfile layout: golang builder → alpine runtime, nonroot.
FROM golang:latest AS builder
ENV GOTOOLCHAIN=auto
ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY}
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata ffmpeg && \
    addgroup -S app && adduser -S -G app app
WORKDIR /app
COPY --from=builder /server .
USER app
EXPOSE 8090
CMD ["./server"]
