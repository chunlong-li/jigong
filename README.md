# 记工

工地记工与结算系统。单二进制 + SQLite，专治"工地上记工、报给甲方、月底结算"这一套账。

- 后端：Go + SQLite（纯 Go 驱动，无需 CGO）+ excelize
- 前端：Vue 3 + TypeScript + Vite + Element Plus
- 部署：单二进制 / Docker Compose / 飞牛 fpk，含 ARM 与 Windows 构建

## 功能

| 模块 | 说明 |
|---|---|
| 工地项目 | 名称、地址、客户、合同金额、开工日期；承包方式：包工包料 / 包工 / 点工；状态行内切换（暂停 / 恢复 / 完工） |
| 工人管理 | 内部单价（记工成本）、对外单价（报给客户）、加班单价（元/小时） |
| 记工 | 一天 / 半天 / 加班（一天=1、半天=0.5、加班按小时单独统计不计天数），同一天可「一天+加班」叠加；支持批量记工 |
| 工资结算 | 按工地 / 月份查看，一键结算归档冻结；可导出工资单 Excel |
| 价格明细 | 三套价目：材料价格、施工价格（包工包料）、施工价格（包工），施工价含同行价与客户价；分类可自定义 |
| 进料登记 | 从价目表下拉带价，新材料自动入价目表；可查该工地全部进料明细 |
| 报价单 | 项目名自由填写 + 承包方式 + 报价类型（客户 / 同行）；明细支持材料 / 包工包料 / 包工三级选择、尺寸算式；导出带大写金额的 Excel |
| 结算单 | 项目可选已有或自定义，点工可从记工汇总工数；登记实收；**编辑 / 已结**（已结后不可改） |

## 下载

预编译安装包见 [Releases](https://github.com/chunlong-li/jigong/releases)。当前正式版 **v1.0.1**。

| 平台 | 文件（把版本号换成对应版本） |
|---|---|
| Linux x86_64 | `jigong-v1.0.1-linux-amd64.tar.gz` |
| Linux ARM64 | `jigong-v1.0.1-linux-arm64.tar.gz` |
| Windows x64 | `jigong-v1.0.1-windows-amd64.zip` |
| 飞牛 fnOS（x86） | `jigong-v1.0.1-amd64-fnos.fpk` |
| 飞牛 fnOS（ARM） | `jigong-v1.0.1-arm64-fnos.fpk` |
| Docker Compose | `jigong-v1.0.1-compose.tar.gz` / `.zip` |

**飞牛 fnOS 安装**：应用中心 → 右上角「手动安装」→ 选 `.fpk`（x86 用 amd64，ARM 用 arm64）。

**Docker Compose 安装**：解压 compose 包，进目录执行 `docker compose up -d --build`。

## 在线更新

实现细节与常见坑见技能 **`selfhosted-update`**。记工的配置：

| 项 | 值 |
|---|---|
| 正式版通道 | `https://zupu-update.lijiacore.com:4440/jigong/latest.json` |
| 测试版通道 | `https://zupu-update.lijiacore.com:4440/jigong/beta/latest.json` |
| 覆盖更新源 | 环境变量 `JIGONG_UPDATE_URL` |
| 证书 | 默认对自有网关域名跳过校验；`JIGONG_UPDATE_INSECURE=0` 关闭 |
| 测试版入口 | 设置页连点「当前版本」3 次解锁（存 localStorage） |
| 运行时目录 | `<数据目录>/runtime/{jigong,web}`，更新后 `syscall.Exec` 原地重启 |
| 登录态 | 存 SQLite `sessions` 表，有效期 30 天，更新后不掉登录 |

发版规则见技能 **`version-and-release`**。记工的具体约定：

- 版本号写在根目录 `VERSION`，构建脚本默认读取并注入 `-X main.Version`
- 测试版 `x.y.z.mmdd.n`：**只本机构建 + 只发网关 beta**，代码不推 GitHub
- 正式版 `x.y.z`：改 `VERSION` → 更新 `changelog.md` → 推代码 + `git tag v1.0.x` → CI 自动出 7 个包

## 更新日志

数据源 `server/internal/api/changelog.md`，用 `go:embed` 打进二进制，前端通过 `GET /api/changelog` 读取，设置页以折叠面板展示（默认展开最新一条，只记正式版）。

条目格式：

```markdown
## v1.0.1 | 2026-11-22

### 在线更新

· 一条一个改动
```

## 快速开始

### 方式一：直接跑二进制

```bash
tar -xzf jigong-v1.0.1-linux-amd64.tar.gz
cd jigong && ./start.sh        # Windows 双击「启动.bat」
```

浏览器打开 `http://<本机IP>:8910`，默认账号 `admin / admin123`，**登录后请到「设置」改密码**。

### 方式二：Docker Compose

```bash
docker compose up -d --build
```

### 方式三：源码运行

```bash
# 后端
cd server && go build -o jigong . && ./jigong -port 8910 -data-dir ./data -web-dir ../web/dist

# 前端开发模式（带 /api 代理）
cd web && npm install && npm run dev
```

## 数据

- 单文件 SQLite：`data/jigong.db`（WAL 模式），**备份这个目录即可**
- 金额以元为单位存 REAL，写入前统一四舍五入到分

## 打包

```bash
bash scripts/build-packages.sh    # Linux/ARM/Windows 三平台压缩包
bash scripts/build-fpk.sh         # 飞牛 fpk（默认 amd64，第二个参数传 arm64）
bash scripts/build-compose-pkg.sh # Docker Compose 包
```

默认输出到 `/vol1/1000/安装包/jigong`（可用参数覆盖）。

## 目录结构

```
server/            Go 后端（net/http + modernc sqlite + excelize）
  internal/db      建表与迁移
  internal/api     路由、业务、Excel、在线更新、更新日志
web/               Vue 3 前端
scripts/           打包 / 发布脚本
.github/workflows 推 tag 自动构建并发布
docs/              导出的 Excel 示例
Dockerfile / docker-compose.yml
```

## 说明

- 单用户登录，会话存 SQLite（30 天）
- 删除项目会级联删除其报价单、结算单、进料；有记工记录时会阻止删除
- 删除工人时若有记工记录会被阻止

## 开源协议

[MIT](LICENSE)

## 赞助

项目免费使用。如果觉得有帮助，欢迎在页面右侧点「赞助作者」扫码支持，或加微信交流。
