# build
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY . .

WORKDIR /app/services/auth
RUN go build -o auth cmd/main.go

# run
FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/services/auth/auth .

CMD ["./auth"]
