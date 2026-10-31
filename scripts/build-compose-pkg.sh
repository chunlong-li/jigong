#!/bin/bash
# 打包 Docker Compose 安装包（含 Dockerfile + 源码，解压即可 up）
# 用法：bash scripts/build-compose-pkg.sh [版本] [输出目录]
set -e
cd "$(dirname "$0")/.."

VER="${1:-1.0.0}"
OUT="${2:-./安装包}"
STAGE="$(pwd)/.compose-pkg"
NAME="jigong-v$VER-compose"

echo "==> 组装 $STAGE"
rm -rf "$STAGE" && mkdir -p "$STAGE/$NAME"
cd "$STAGE/$NAME"

cp ../../Dockerfile ../../docker-compose.yml ../../.dockerignore ../../.env.example ../../README.md .
cp -r ../../server .
cp -r ../../web .
cp -r ../../scripts .
rm -rf web/node_modules web/dist server/data .git

cat > 部署说明.md <<'EOF'
# 记工 · Docker Compose 安装包

## 一、安装

```bash
# 解压后进入目录
tar -xzf jigong-v1.0.0-compose.tar.gz   # 或 unzip jigong-v1.0.0-compose.zip
cd jigong-v1.0.0-compose

# 可选：改初始密码
cp .env.example .env && vi .env

# 构建并启动（会本地构建镜像，不依赖镜像仓库）
docker compose up -d --build
```

访问 `http://<NAS-IP>:8910`，默认账号 `admin / admin123`，登录后到「设置」改密码。

## 二、目录说明

```
docker-compose.yml   服务编排（8910 端口，数据挂载 ./data）
Dockerfile           三阶段构建：前端 → 后端 → 运行镜像
server/ web/         源码（构建时用）
data/                运行后生成，存放 SQLite（jigong.db）
```

## 三、常用命令

```bash
docker compose ps                 # 状态
docker compose logs -f --tail=100 # 日志
docker compose restart            # 重启
docker compose down               # 停止（数据保留）
docker compose up -d --build      # 改完代码重新构建
```

## 四、数据备份

```bash
tar -czf jigong-backup-$(date +%F).tar.gz -C . data
```

## 五、常见问题

- **拉镜像卡住**：本项目 `pull_policy: build`，只本地构建；若卡在基础镜像，先 `docker pull docker.m.daocloud.io/library/node:22-alpine` 等并打回原名。
- **端口占用**：改 `docker-compose.yml` 里 `ports: "8910:8910"` 左边端口。
- **构建时 apk 慢**：Dockerfile 已把 Alpine 源换成阿里云。
EOF

cd "$STAGE"
tar -czf "$OUT/$NAME.tar.gz" "$NAME"
( command -v zip >/dev/null && zip -qr "$OUT/$NAME.zip" "$NAME" ) || true
chmod 644 "$OUT/$NAME.tar.gz" "$OUT/$NAME.zip" 2>/dev/null || true
rm -rf "$STAGE"

echo "==> 完成："
ls -lh "$OUT/$NAME.tar.gz" "$OUT/$NAME.zip" 2>/dev/null
