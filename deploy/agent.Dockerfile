# Stage 1: Build Agent
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o agent ./cmd/agent/main.go

# Stage 2: Final Image
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/agent .
EXPOSE 9090
CMD ["./agent"]
