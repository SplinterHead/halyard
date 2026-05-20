# Stage 1: Build Frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app
COPY ui/package*.json ./
RUN npm install
COPY ui/ ./
RUN npm run build

# Stage 2: Build Backend
FROM golang:1.26-alpine AS backend-builder
WORKDIR /app
COPY . .
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o manager ./cmd/manager/main.go

# Stage 3: Final Image
FROM alpine:latest
RUN apk add --no-cache docker-cli
WORKDIR /app
COPY --from=backend-builder /app/manager .
COPY --from=frontend-builder /app/dist ./ui/dist
EXPOSE 8080
CMD ["./manager"]
