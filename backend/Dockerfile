# Multi-stage build. Targets linux/arm64 (RK3588) by default; use
# `docker buildx --platform` for multi-arch.
FROM golang:1.23-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY . .
# Resolve deps from source imports (generates go.sum); robust when no go.sum is
# committed. For reproducible builds, commit go.sum after a local `go mod tidy`.
RUN go mod tidy
ARG TARGETOS=linux
ARG TARGETARCH=arm64
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o /out/campmenu ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/campmenu /app/campmenu
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/campmenu"]
