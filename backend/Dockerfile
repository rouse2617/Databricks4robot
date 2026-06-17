# Keep the builder version pinned to avoid golang:latest digest churn causing
# surprise cache misses during dev Cloud Run deploys.
ARG BUILDPLATFORM=linux/amd64
FROM --platform=$BUILDPLATFORM mirror.gcr.io/library/golang:1.26.0 AS builder
ENV GOTOOLCHAIN=auto
ENV HTTP_PROXY=
ENV HTTPS_PROXY=
ENV ALL_PROXY=
ENV http_proxy=
ENV https_proxy=
ENV all_proxy=
ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY}
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
ARG COMMIT=unknown
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-X github.com/CyberOrigin2077/cyber-databrew/routes.buildVersion=${VERSION} -X github.com/CyberOrigin2077/cyber-databrew/routes.buildCommit=${COMMIT} -X github.com/CyberOrigin2077/cyber-databrew/routes.buildTime=$(date -u +%FT%TZ)" -o /server ./cmd/server

FROM scratch
WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /server /app/server
COPY config/ ./config/
EXPOSE 8080
CMD ["/app/server"]
