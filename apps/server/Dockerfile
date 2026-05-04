# syntax=docker/dockerfile:1

FROM golang:1.26.2-alpine AS build

WORKDIR /src/apps/server

COPY apps/server/go.mod apps/server/go.sum ./
RUN go mod download

COPY apps/server/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/goalrail-server ./cmd/goalrail-server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates \
  && addgroup -S goalrail \
  && adduser -S -D -H -G goalrail goalrail

COPY --from=build /out/goalrail-server /usr/local/bin/goalrail-server

USER goalrail:goalrail
EXPOSE 8080

ENTRYPOINT ["goalrail-server"]

