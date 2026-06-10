FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o agent-os ./cmd/agent-os/

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/agent-os .
COPY --from=builder /app/docs ./docs
EXPOSE 8090 8096 8103 8110 8119 8124
CMD ["./agent-os"]
