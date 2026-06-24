FROM golang:1.21-alpine AS build
WORKDIR /app
COPY go.mod ./
COPY main.go ./
RUN CGO_ENABLED=0 go build -o grace-sync .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /app/grace-sync /grace-sync
COPY config.json /config.json
ENTRYPOINT ["/grace-sync"]
