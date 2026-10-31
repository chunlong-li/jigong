#!/bin/bash
# 打包「记工」安装包：Linux(amd64) / ARM(arm64) / Windows(amd64)
# 用法：bash scripts/build-packages.sh [版本号] [输出目录]
#   默认版本 1.0.0，默认输出 ./安装包
set -e
cd "$(dirname "$0")/.."

VER="${1:-1.0.0}"
OUT="${2:-./安装包}"
APP=jigong
BUILD="$(pwd)/.build-pkg"

echo "==> 清理 $BUILD"
rm -rf "$BUILD"
mkdir -p "$BUILD/web" "$OUT"/{Linux,arm,Windows}

echo "==> 1/3 构建前端"
( cd web && ( [ -d node_modules ] || npm ci --no-audit --no-fund >/dev/null ) && npm run build >/dev/null )
cp -r web/dist/* "$BUILD/web/"

echo "==> 2/3 交叉编译后端"
( cd server
  export GOPROXY=https://goproxy.cn,direct CGO_ENABLED=0 GOFLAGS=-mod=mod
  LDFLAGS="-s -w -X main.Version=$VER"
  GOOS=linux  GOARCH=amd64 go build -trimpath -ldflags "$LDFLAGS" -o "$BUILD/$APP-linux-amd64" .
  GOOS=linux  GOARCH=arm64 go build -trimpath -ldflags "$LDFLAGS" -o "$BUILD/$APP-linux-arm64" .
  GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$LDFLAGS" -o "$BUILD/$APP-windows-amd64.exe" .
)

echo "==> 3/3 组装安装包"

# ---------- Linux amd64 ----------
D="$BUILD/Linux"; rm -rf "$D"; mkdir -p "$D/$APP"
cp "$BUILD/$APP-linux-amd64" "$D/$APP/$APP"; chmod 755 "$D/$APP/$APP"
cp -r "$BUILD/web" "$D/$APP/web"
mkdir -p "$D/$APP/data"
cat > "$D/$APP/start.sh" <<'EOF'
#!/bin/sh
cd "$(dirname "$0")"
exec ./jigong -port 8910 -data-dir ./data -web-dir ./web
EOF
chmod 755 "$D/$APP/start.sh"
cat > "$D/$APP/README.txt" <<'EOF'
记工 · Linux x86_64 版

启动：./start.sh          （或 ./jigong -port 8910 -data-dir ./data -web-dir ./web）
访问：http://<本机IP>:8910
账号：admin / admin123   （登录后请到「设置」改密码）
数据：./data/jigong.db   备份这个目录即可
EOF
tar -czf "$OUT/Linux/jigong-v$VER-linux-amd64.tar.gz" -C "$D" "$APP"

# ---------- ARM64 ----------
D="$BUILD/arm"; rm -rf "$D"; mkdir -p "$D/$APP"
cp "$BUILD/$APP-linux-arm64" "$D/$APP/$APP"; chmod 755 "$D/$APP/$APP"
cp -r "$BUILD/web" "$D/$APP/web"
mkdir -p "$D/$APP/data"
cp "$BUILD/Linux/$APP/start.sh" "$D/$APP/start.sh"; chmod 755 "$D/$APP/start.sh"
cat > "$D/$APP/README.txt" <<'EOF'
记工 · ARM64（aarch64）版

适用：飞牛/群晖等 ARM 架构 NAS、树莓派 4/5（64 位系统）
启动：./start.sh
访问：http://<本机IP>:8910
账号：admin / admin123   （登录后请到「设置」改密码）
数据：./data/jigong.db
EOF
tar -czf "$OUT/arm/jigong-v$VER-linux-arm64.tar.gz" -C "$D" "$APP"

# ---------- Windows ----------
D="$BUILD/Windows"; rm -rf "$D"; mkdir -p "$D/$APP"
cp "$BUILD/$APP-windows-amd64.exe" "$D/$APP/$APP.exe"
cp -r "$BUILD/web" "$D/$APP/web"
mkdir -p "$D/$APP/data"
cat > "$D/$APP/启动.bat" <<'EOF'
@echo off
cd /d %~dp0
start "" http://127.0.0.1:8910
jigong.exe -port 8910 -data-dir .\data -web-dir .\web
pause
EOF
cat > "$D/$APP/README.txt" <<'EOF'
记工 · Windows 64 位版

启动：双击「启动.bat」（会同时打开浏览器）
访问：http://127.0.0.1:8910
账号：admin / admin123   （登录后请到「设置」改密码）
数据：.\data\jigong.db
关闭：黑窗口关掉即停止服务
EOF
( cd "$D" && zip -qr "$OUT/Windows/jigong-v$VER-windows-amd64.zip" "$APP" )

echo "==> 完成，产物："
ls -lh "$OUT"/Linux/*.tar.gz "$OUT"/arm/*.tar.gz "$OUT"/Windows/*.zip
rm -rf "$BUILD"
