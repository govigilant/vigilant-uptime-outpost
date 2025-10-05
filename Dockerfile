FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o outpost ./cmd/outpost

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/outpost .

RUN mkdir -p /var/lib/uptime-outpost

EXPOSE 1000-10000

CMD ["./outpost"]
