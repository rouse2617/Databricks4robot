# syntax=docker/dockerfile:1
# BuildKit cache mounts persist module download + build action cache across image rebuilds.
FROM --platform=$BUILDPLATFORM golang:latest AS builder
ENV GOTOOLCHAIN=auto
ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY}
WORKDIR /app
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY . .
ARG VERSION=dev
ARG COMMIT=unknown
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-X github.com/CyberOrigin2077/cyber-databrew/routes.buildVersion=${VERSION} -X github.com/CyberOrigin2077/cyber-databrew/routes.buildCommit=${COMMIT} -X github.com/CyberOrigin2077/cyber-databrew/routes.buildTime=$(date -u +%FT%TZ)" -o /server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata postgresql16-client
WORKDIR /app
COPY --from=builder /server .
COPY config/ ./config/
COPY scripts/ ./scripts/
COPY .env.example .env
EXPOSE 8080
CMD ["./server"]
