FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build both binaries. `server` is the probe service (cmd/server);
# `fitting` is the offline fitting-level microservice (cmd/fitting).
# They share the same image so deployments can pick one via docker-compose
# command override or `docker run <image> ./fitting`.
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server/main.go \
 && CGO_ENABLED=0 GOOS=linux go build -o fitting ./cmd/fitting/main.go


FROM alpine:3.21

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app/

COPY --from=builder /app/server .
COPY --from=builder /app/fitting .

EXPOSE 8080

# Default entrypoint is the probe server. The fitting microservice is opted
# in by docker-compose (profile `fitting`) or by `docker run <image> ./fitting`.
CMD ["./server"]
