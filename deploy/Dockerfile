# ==========================================
# Stage 1: Build Frontend Assets (React 19 + Vite)
# ==========================================
FROM node:22-alpine AS web-builder
WORKDIR /app/web

COPY web/package*.json ./
RUN npm ci

COPY web/ ./
RUN npm run build

# ==========================================
# Stage 2: Compile Cubit Control Plane Daemon
# ==========================================
FROM golang:1.24-alpine AS go-builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/cubitd ./cmd/cubitd

# ==========================================
# Stage 3: Minimal Production Runtime
# ==========================================
FROM alpine:3.21

RUN apk add --no-cache ca-certificates docker-cli curl tzdata

WORKDIR /app

COPY --from=go-builder /app/bin/cubitd /usr/local/bin/cubitd
COPY --from=web-builder /app/web/dist /var/lib/cubit/web/dist

ENV CUBIT_WEB_DIR=/var/lib/cubit/web/dist

RUN mkdir -p /var/lib/cubit/storage /etc/traefik/dynamic

EXPOSE 8000

ENTRYPOINT ["/usr/local/bin/cubitd"]
CMD ["-port=8000", "-db=/var/lib/cubit/cubit.db", "-traefik-config=/etc/traefik/dynamic/cubit.yaml", "-storage-dir=/var/lib/cubit/storage"]
