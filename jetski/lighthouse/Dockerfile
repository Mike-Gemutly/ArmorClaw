FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o lighthouse ./cmd/lighthouse

FROM alpine:latest

RUN apk --no-cache add ca-certificates sqlite
WORKDIR /root/

COPY --from=builder /app/lighthouse .
COPY --from=builder /app/configs ./configs

EXPOSE 8080

CMD ["./lighthouse"]
