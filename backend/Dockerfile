## Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git make

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o bin/mypaas-api cmd/api/main.go

## Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates docker-cli git tzdata

WORKDIR /app

COPY --from=builder /build/bin/mypaas-api /app/mypaas-api

EXPOSE 8080

ENTRYPOINT ["/app/mypaas-api"]
