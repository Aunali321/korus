# Stage 1: Build frontend
FROM oven/bun:1 AS frontend-base
WORKDIR /app

# Install dependencies into temp directory for caching
FROM frontend-base AS frontend-install
RUN mkdir -p /temp/dev
COPY web/package.json web/bun.lock /temp/dev/
RUN cd /temp/dev && bun install --frozen-lockfile

# Build frontend
FROM frontend-base AS frontend-builder
COPY --from=frontend-install /temp/dev/node_modules node_modules
COPY web/ .
ENV NODE_ENV=production
RUN bun run build

# Stage 2: Build backend
FROM golang:1.24-alpine AS backend-builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend-builder /app/dist ./web/dist

RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o korus ./cmd/server

# Stage 3: Runtime
FROM alpine:3.21

RUN apk add --no-cache ffmpeg ca-certificates tzdata

WORKDIR /app

COPY --from=backend-builder /app/korus .
COPY --from=backend-builder /app/web/dist ./web/dist

RUN mkdir -p /data /media

ENV ADDR=:8080
ENV DB_PATH=/data/korus.db
ENV MEDIA_ROOT=/media
ENV FFMPEG_PATH=ffmpeg
ENV FFPROBE_PATH=ffprobe
ENV COVER_CACHE_PATH=/data/covers

EXPOSE 8080

VOLUME ["/data", "/media"]

CMD ["./korus"]
