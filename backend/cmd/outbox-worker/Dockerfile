FROM golang:latest AS builder
ENV GOTOOLCHAIN=auto
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /outbox-worker ./cmd/outbox-worker

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /outbox-worker .
EXPOSE 8081
CMD ["./outbox-worker"]
