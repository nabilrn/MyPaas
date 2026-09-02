## Build stage
FROM golang:alpine AS builder

WORKDIR /build

RUN apk add --no-cache git make

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o bin/mypaas-api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/mypaas ./cmd/cli
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/mypaas-sqlite-helper ./cmd/sqlite-helper

## Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates docker-cli docker-cli-compose git postgresql-client tzdata

WORKDIR /app

COPY --from=builder /build/bin/mypaas-api /app/mypaas-api
COPY --from=builder /build/bin/mypaas /app/mypaas
COPY --from=builder /build/bin/mypaas-sqlite-helper /app/mypaas-sqlite-helper

EXPOSE 8080

ENTRYPOINT ["/app/mypaas-api"]
