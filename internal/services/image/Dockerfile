# build
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY . .

WORKDIR /app/services/image
RUN go build -o image cmd/main.go

# run
FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/services/image/image .

CMD ["./image"]
