#!/bin/bash
# 打包「记工」飞牛 fnOS 原生安装包（.fpk）
# 用法：bash scripts/build-fpk.sh [版本] [架构: amd64|arm64] [输出目录]
set -e
cd "$(dirname "$0")/.."

VER="${1:-1.0.0}"
ARCH="${2:-amd64}"
OUT="${3:-./安装包}"
ROOT="$(pwd)"
STAGE="$ROOT/.fpk-build"
PKG="$STAGE/pkg"

case "$ARCH" in
  amd64) GOARCH=amd64; PLATFORM=x86 ;;
  arm64) GOARCH=arm64; PLATFORM=arm ;;
  *) echo "架构只支持 amd64 / arm64"; exit 1 ;;
esac

echo "==> 1/5 构建前端"
( cd web && ( [ -d node_modules ] || npm ci --no-audit --no-fund >/dev/null ) && npm run build >/dev/null )

echo "==> 2/5 编译 $ARCH 二进制"
( cd server
  CGO_ENABLED=0 GOOS=linux GOARCH=$GOARCH GOPROXY=https://goproxy.cn,direct GOFLAGS=-mod=mod \
  go build -trimpath -ldflags "-s -w -X main.Version=$VER" -o "$STAGE/jigong" . )

echo "==> 3/5 组装运行目录（app.tgz）"
rm -rf "$STAGE/app" && mkdir -p "$STAGE/app/ui/images"
cp "$STAGE/jigong" "$STAGE/app/jigong"; chmod 755 "$STAGE/app/jigong"
cp -r web/dist "$STAGE/app/web"
cat > "$STAGE/app/ui/config" <<'EOF'
{
  ".url": {
    "jigong.Application": {
      "title": "记工",
      "desc": "工地记工与结算（记工 / 进料 / 报价单 / 结算单）",
      "icon": "images/icon_{0}.png",
      "type": "iframe",
      "protocol": "http",
      "port": "${wizard_port}",
      "url": "/",
      "allUsers": true
    }
  }
}
EOF

echo "==> 4/5 生成图标"
python3 - "$STAGE/app/ui/images" "$STAGE" <<'PY'
import sys, os
from PIL import Image, ImageDraw

outdir = sys.argv[1]
rootdir = sys.argv[2]

def make(size):
    S = size * 4  # 4x 超采样
    img = Image.new('RGBA', (S, S), (0, 0, 0, 0))
    d = ImageDraw.Draw(img)
    r = int(S * 0.22)
    # 圆角背景（深蓝渐变）
    for i in range(S):
        t = i / S
        col = (int(30 + 20 * t), int(70 + 40 * t), int(150 + 60 * t), 255)
        d.line([(0, i), (S, i)], fill=col)
    mask = Image.new('L', (S, S), 0)
    ImageDraw.Draw(mask).rounded_rectangle([0, 0, S - 1, S - 1], radius=r, fill=255)
    img.putalpha(mask)
    # 白色记录板
    bx0, by0, bx1, by1 = int(S * .24), int(S * .20), int(S * .76), int(S * .80)
    d.rounded_rectangle([bx0, by0, bx1, by1], radius=int(S * .06), fill=(255, 255, 255, 255))
    # 顶部夹子
    d.rounded_rectangle([int(S * .38), int(S * .15), int(S * .62), int(S * .25)],
                        radius=int(S * .04), fill=(255, 200, 90, 255))
    # 三条记录线
    for i, w in enumerate([0.62, 0.50, 0.56]):
        y = int(S * (.40 + i * .13))
        d.rounded_rectangle([int(S * .32), y, int(S * (.32 + w)), y + int(S * .055)],
                            radius=int(S * .03), fill=(46, 96, 180, 255))
    # 右下角对勾
    cx, cy = int(S * .66), int(S * .70)
    d.line([(cx - int(S * .07), cy), (cx - int(S * .02), cy + int(S * .06))],
           fill=(30, 160, 90, 255), width=int(S * .035))
    d.line([(cx - int(S * .02), cy + int(S * .06)), (cx + int(S * .07), cy - int(S * .06))],
           fill=(30, 160, 90, 255), width=int(S * .035))
    return img.resize((size, size), Image.LANCZOS)

os.makedirs(outdir, exist_ok=True)
for sz in (64, 256):
    make(sz).save(os.path.join(outdir, f'icon_{sz}.png'))
make(64).save(os.path.join(rootdir, 'ICON.PNG'))
make(256).save(os.path.join(rootdir, 'ICON_256.PNG'))
print('  icons ok')
PY

echo "==> 5/5 打包 .fpk"
rm -rf "$PKG" && mkdir -p "$PKG/cmd" "$PKG/config" "$PKG/wizard"
mv "$STAGE/app" "$STAGE/app.tgz-src"
tar -czf "$PKG/app.tgz" -C "$STAGE/app.tgz-src" jigong web ui
CHECK=$(md5sum "$PKG/app.tgz" | awk '{print $1}')
cp "$STAGE/ICON.PNG" "$STAGE/ICON_256.PNG" "$PKG/"
rm -f "$STAGE/app.tgz-src/ICON.PNG" "$STAGE/app.tgz-src/ICON_256.PNG"

cat > "$PKG/config/privilege" <<'EOF'
{
    "defaults":
    {
        "run-as": "package"
    }
}
EOF
echo '{}' > "$PKG/config/resource"

cat > "$PKG/cmd/main" <<'EOF'
#!/bin/bash
PID_FILE="${TRIM_PKGVAR}/app.pid"
DATA_DIR="${TRIM_PKGVAR}/data"
WEB_DIR="${TRIM_APPDEST}/web"
BIN="${TRIM_APPDEST}/jigong"
LOG="${TRIM_PKGVAR}/info.log"
PORT=8910
if [ -r "${TRIM_PKGETC}/config.env" ]; then
  . "${TRIM_PKGETC}/config.env"
  [ -n "${wizard_port:-}" ] && PORT="$wizard_port"
fi
check_process() { kill -0 "$1" 2>/dev/null; }
status() { [ -f "${PID_FILE}" ] && check_process "$(cat "${PID_FILE}")"; }
start_process() {
  status && return 0
  mkdir -p "${DATA_DIR}"
  echo "$(date '+%F %T') start on ${PORT}" >> "${LOG}"
  nohup "${BIN}" -port "${PORT}" -data-dir "${DATA_DIR}" -web-dir "${WEB_DIR}" >> "${LOG}" 2>&1 &
  echo $! > "${PID_FILE}"
}
stop_process() {
  [ -r "${PID_FILE}" ] || return 0
  pid=$(cat "${PID_FILE}")
  check_process "$pid" && kill -TERM "$pid"
  for i in $(seq 1 10); do check_process "$pid" || break; sleep 1; done
  check_process "$pid" && kill -KILL "$pid" 2>/dev/null
  rm -f "${PID_FILE}"
}
case "$1" in
  start) start_process ;;
  stop) stop_process ;;
  status) if status; then exit 0; else exit 3; fi ;;
  *) exit 1 ;;
esac
EOF
chmod 755 "$PKG/cmd/main"

cat > "$PKG/cmd/install_init" <<'EOF'
#!/bin/bash
mkdir -p "${TRIM_PKGETC}" "${TRIM_PKGVAR}/data"
CONF="${TRIM_PKGETC}/config.env"
[ -f "$CONF" ] || { echo 'wizard_port=8910' > "$CONF"; chmod 644 "$CONF"; }
exit 0
EOF
chmod 755 "$PKG/cmd/install_init"

cat > "$PKG/cmd/config_callback" <<'EOF'
#!/bin/bash
[ -n "${wizard_port:-}" ] && echo "wizard_port=${wizard_port}" > "${TRIM_PKGETC}/config.env"
exit 0
EOF
chmod 755 "$PKG/cmd/config_callback"

for f in install_callback upgrade_init upgrade_callback uninstall_init uninstall_callback config_init; do
  printf '#!/bin/bash\nexit 0\n' > "$PKG/cmd/$f"; chmod 755 "$PKG/cmd/$f"
done

cat > "$PKG/wizard/install" <<'EOF'
[
  {
    "stepTitle": "服务设置",
    "items": [
      {
        "type": "text",
        "field": "wizard_port",
        "label": "服务端口",
        "initValue": "8910",
        "placeholder": "1024-65535，默认 8910",
        "rules": [
          { "required": true, "message": "请填写服务端口" },
          { "pattern": "^(102[4-9]|10[3-9][0-9]|1[1-9][0-9]{2}|[2-9][0-9]{3}|[1-5][0-9]{4}|6[0-4][0-9]{3}|65[0-4][0-9]{2}|655[0-2][0-9]|6553[0-5])$", "message": "端口必须是 1024-65535" }
        ]
      }
    ]
  },
  {
    "stepTitle": "数据说明",
    "items": [
      { "type": "tips", "helpText": "数据保存在应用数据目录（TRIM_PKGVAR/data/jigong.db），卸载前请自行备份。" }
    ]
  }
]
EOF
cp "$PKG/wizard/install" "$PKG/wizard/config"
cat > "$PKG/wizard/uninstall" <<'EOF'
[
  {
    "stepTitle": "卸载提示",
    "items": [
      { "type": "tips", "helpText": "卸载不会删除数据目录（TRIM_PKGVAR/data），需要彻底清理请手动删除。" }
    ]
  }
]
EOF

cat > "$PKG/manifest" <<EOF
appname               = jigong
version               = $VER
display_name          = 记工
desc                  = 工地记工与结算：记工（一天/半天/加班）、进料登记、价格明细、报价单与结算单（Excel 导出）。纯本地部署，数据只存在您自己的 NAS 上。
platform              = $PLATFORM
source                = thirdparty
maintainer            = 东坑木作小李
maintainer_url        = https://github.com/chunlong-li/jigong
distributor           = 记工
distributor_url       = https://github.com/chunlong-li/jigong
service_port          = 8910
ctl_stop              = true
checkport             = false
desktop_uidir         = ui
desktop_applaunchname = jigong.Application
checksum              = $CHECK
EOF

mkdir -p "$OUT"
FPK="$OUT/jigong-v$VER-$ARCH-fnos.fpk"
( cd "$PKG" && tar -czf "$FPK" . )
chmod 644 "$FPK"
rm -rf "$STAGE"
echo "==> 完成：$FPK"
ls -lh "$FPK"
