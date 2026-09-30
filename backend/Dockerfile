# syntax=docker/dockerfile:1

# Build stage: pure-Go binary (modernc sqlite needs no CGO).
# go.mod pins toolchain go1.23.4, so the image must be at least that version.
FROM golang:1.23.4-alpine AS build
ENV GOTOOLCHAIN=local CGO_ENABLED=0
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# Runtime stage: small image, non-root, wget only for the healthcheck.
FROM alpine:3.20
RUN addgroup -S app && adduser -S -G app app \
    && mkdir -p /var/lib/energy /data \
    && chown app:app /var/lib/energy
COPY --from=build /out/api /usr/local/bin/api
# The seed CSVs are not baked in: compose mounts ./data read-only at /data.
ENV PORT=8080 DB_PATH=/var/lib/energy/energy.db DATA_DIR=/data
USER app
EXPOSE 8080
HEALTHCHECK --interval=5s --timeout=3s --start-period=5s --retries=10 \
    CMD wget -qO- http://127.0.0.1:8080/api/health || exit 1
ENTRYPOINT ["api"]
