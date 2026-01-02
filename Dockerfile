FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server/main.go


FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app/

COPY --from=builder /app/server .
COPY --from=builder /app/config/config.yaml ./config/config.yaml

RUN mkdir -p uploads/csv uploads/img

EXPOSE 8080

CMD ["./server"]