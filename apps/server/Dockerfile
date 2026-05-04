# syntax=docker/dockerfile:1

FROM golang:1.26-alpine3.23 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -buildvcs=false \
    -ldflags="-s -w" \
    -o /out/goalrail-server \
    ./cmd/goalrail-server

FROM alpine:3.23 AS runtime

RUN apk add --no-cache ca-certificates \
    && addgroup -S goalrail \
    && adduser -S -D -H -G goalrail goalrail

WORKDIR /app

COPY --from=build /out/goalrail-server /usr/local/bin/goalrail-server

USER goalrail:goalrail

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/livez >/dev/null || exit 1

ENTRYPOINT ["/usr/local/bin/goalrail-server"]
