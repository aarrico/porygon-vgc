# syntax=docker/dockerfile:1
# Build context is backend/. Two final stages, so compose must name a target.

FROM golang:1.27.0-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/etl ./cmd/etl

# alpine rather than scratch/distroless: the compose healthcheck needs a HTTP
# client inside the container, and busybox wget is already there.
FROM alpine:3.22 AS api
RUN adduser -D -u 10001 app
COPY --from=build /out/api /usr/local/bin/api
USER app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api"]

# The CSV dump is embedded in the binary, so this stage ships no data files.
FROM alpine:3.22 AS etl
RUN adduser -D -u 10001 app
COPY --from=build /out/etl /usr/local/bin/etl
USER app
ENTRYPOINT ["/usr/local/bin/etl"]
