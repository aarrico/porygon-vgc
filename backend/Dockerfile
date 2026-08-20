# syntax=docker/dockerfile:1
# Build context is backend/. Story 1.1c adds a second `etl` build target here.

FROM golang:1.26.6-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# alpine rather than scratch/distroless: the compose healthcheck needs a HTTP
# client inside the container, and busybox wget is already there.
FROM alpine:3.22
RUN adduser -D -u 10001 app
COPY --from=build /out/api /usr/local/bin/api
USER app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api"]
