# ===== 阶段1：构建前端（Vue3 + Vite） =====
FROM node:22-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci --no-audit --no-fund --registry=https://registry.npmmirror.com
COPY web/ ./
RUN npm run build

# ===== 阶段2：构建后端（Go，SQLite 纯 Go 驱动，无需 CGO） =====
FROM golang:1.26-alpine AS server
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /app
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o jigong .

# ===== 阶段3：运行时 =====
FROM alpine:3.20
RUN sed -i 's#https\?://dl-cdn.alpinelinux.org#https://mirrors.aliyun.com#g' /etc/apk/repositories
RUN apk add --no-cache ca-certificates tzdata curl
ENV TZ=Asia/Shanghai
RUN ln -sf /usr/share/zoneinfo/$TZ /etc/localtime && echo "$TZ" > /etc/timezone

WORKDIR /app
COPY --from=server /app/jigong .
COPY --from=web /web/dist ./web

ENV JIGONG_PORT=8910 \
    JIGONG_DATA_DIR=/data \
    JIGONG_WEB_DIR=/app/web

VOLUME ["/data"]
EXPOSE 8910

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -fs http://127.0.0.1:8910/api/health || exit 1

CMD ["./jigong", "-port", "8910", "-data-dir", "/data", "-web-dir", "/app/web"]
