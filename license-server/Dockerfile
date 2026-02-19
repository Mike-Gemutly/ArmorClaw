# ArmorClaw License Server
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o license-server .

FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/license-server .

# Create non-root user
RUN adduser -D -u 10001 license
USER license

EXPOSE 8080

ENV PORT=8080
ENV GRACE_PERIOD_DAYS=3

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost:8080/health || exit 1

CMD ["./license-server"]
