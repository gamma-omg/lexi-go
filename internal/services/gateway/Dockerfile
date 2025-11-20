# build
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY . .
RUN go build -o gateway cmd/main.go

# run
FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/gateway .

CMD ["./gateway"]