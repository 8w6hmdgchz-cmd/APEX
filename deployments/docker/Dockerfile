# Stage 1: Build
FROM golang:1.22-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod ./
COPY go.sum* ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /agent-os ./cmd/agent-os/

# Stage 2: Runtime
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /agent-os .
COPY config/config.yaml ./config/
RUN mkdir -p data/blocks
EXPOSE 8080
ENTRYPOINT ["./agent-os"]
