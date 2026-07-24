# syntax=docker/dockerfile:1

# ---- deps: cached separately so `go mod download` only reruns when go.mod/go.sum change ----
FROM golang:1.25-alpine AS deps
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# ---- builder ----
FROM deps AS builder
WORKDIR /src
COPY . .
ENV CGO_ENABLED=0 GOOS=linux
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/app

# ---- runtime ----
FROM alpine:3.20 AS runtime
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S app -G app
WORKDIR /app

COPY --from=builder /out/app ./app
COPY migrations ./migrations

RUN mkdir -p media/avatars media/posts media/thumbnails && \
    chown -R app:app /app

USER app
EXPOSE 8080
ENTRYPOINT ["./app"]
