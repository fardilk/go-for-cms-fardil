# Stage 1: Build
FROM golang:1.24-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# CGO stays off so the binary runs on a scratch-like base with no libc.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api .

# Stage 2: Run
FROM alpine:3.20
RUN apk add --no-cache ca-certificates curl tzdata \
 && adduser -D -u 10001 api

WORKDIR /app
COPY --from=builder /out/api /app/api

# Uploads must land on a mounted volume; without one every redeploy would
# destroy them along with the container.
RUN mkdir -p /data/media && chown -R api:api /data/media
VOLUME ["/data/media"]

USER api
EXPOSE 8000

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s \
  CMD curl -fsS http://127.0.0.1:8000/healthz || exit 1

CMD ["/app/api"]
