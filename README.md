# 记工

工地记工与结算系统。一个单文件就能跑起来的小系统，专治"工地上记工、报给甲方、月底结算"这一套账。

- 后端：Go + SQLite（纯 Go 驱动，无需 CGO）+ excelize
- 前端：Vue 3 + TypeScript + Vite + Element Plus
- 部署：单二进制 / Docker Compose，自带 ARM 与 Windows 构建

---

## 功能

| 模块 | 说明 |
|---|---|
| 工地项目 | 名称、地址、客户、合同金额、开工日期；**承包方式**：包工包料 / 包工 / 点工；状态在列表行内直接切换（暂停 / 恢复 / 完工） |
| 工人管理 | 姓名、电话、身份证、**内部单价**（日薪，记工成本）、**对外单价**（报给客户）、**加班单价**（元/小时） |
| 记工 | 同一工人 + 同一工地合并为一条，按总天数统计；**一天 / 半天 / 加班** 按钮选择（一天=1、半天=0.5、加班按小时单独统计不计天数）；同一天可「一天+加班」叠加；保留按时 / 计件；支持**批量记工**（多选工人 × 多选日期） |
| 工资结算 | 记工页按「工地 / 月份」查看，一键结算：归档冻结，新记工重新累计；可导出**工资单 Excel** |
| 价格明细 | 三套价目表：**材料价格**（可自定义分类、可按分类筛选）、**施工价格（包工包料）**、**施工价格（包工）**，施工价含**同行价格**与**客户价格** |
| 进料登记 | 在工地项目行内点「进料」，材料名称从价目表下拉带价（可手改），新材料自动入价目表；可查看该工地全部进料明细 |
| 报价单 | 项目名称自由填写 + 承包方式 + **报价类型（客户报价 / 同行报价）**；明细支持 **材料 / 包工包料 / 包工** 三级选择、**尺寸算式**（5*6=30 自动算面积）、房号；导出带大写金额的 Excel |
| 结算单（对甲方） | 项目可选已有或自定义，明细三级选择，点工可从记工自动汇总工数（总工数行带价、按日明细不带价）；登记实收金额；导出 Excel |

Excel 全部服务端生成（excelize）：表头、边框、数字格式、人民币大写、盖章签字行都是成品格式。

## 界面

- 桌面端：左侧固定菜单
- 手机端：侧边栏收起，左上角三横线打开抽屉菜单；弹窗自适应宽度

## 下载

预编译安装包见 [Releases](https://github.com/chunlong-li/jigong/releases)：

| 平台 | 文件 |
|---|---|
| Linux x86_64 | `jigong-v1.0.0-linux-amd64.tar.gz` |
| Linux ARM64 | `jigong-v1.0.0-linux-arm64.tar.gz` |
| Windows x64 | `jigong-v1.0.0-windows-amd64.zip` |

## 快速开始

### 方式一：直接跑二进制

```bash
# Linux x86_64
tar -xzf jigong-v1.0.0-linux-amd64.tar.gz
cd jigong && ./start.sh

# ARM64（飞牛/群晖等 ARM NAS、树莓派 64 位）
tar -xzf jigong-v1.0.0-linux-arm64.tar.gz
cd jigong && ./start.sh

# Windows
解压后双击「启动.bat」
```

浏览器打开 `http://<本机IP>:8910`，默认账号 `admin / admin123`，**登录后请到「设置」改密码**。

### 方式二：Docker Compose

```bash
docker compose up -d --build
```

`docker-compose.yml` 已配好：端口 `8910`、数据挂载 `./data`、健康检查、`pull_policy: build`（本地构建，不去拉不存在的镜像）。

### 方式三：源码运行

```bash
# 后端
cd server && go build -o jigong . && ./jigong -port 8910 -data-dir ./data -web-dir ../web/dist

# 前端（开发模式，带 /api 代理）
cd web && npm install && npm run dev
```

## 数据

- 单文件 SQLite：`data/jigong.db`（WAL 模式），**备份这个目录即可**
- 金额以元为单位存 REAL，写入前统一四舍五入到分

## 打包自己的安装包

```bash
bash scripts/build-packages.sh 1.0.0 [输出目录]
```

会一次产出 Linux(amd64) / ARM64 / Windows(amd64) 三个压缩包。

## 目录结构

```
server/            Go 后端（net/http + modernc sqlite + excelize）
  internal/db      建表与迁移
  internal/api     路由、业务、Excel 生成
web/               Vue 3 前端
  src/views        各业务页
  src/components   通用组件（含赞助组件）
scripts/           打包脚本
docs/              导出的 Excel 示例
Dockerfile / docker-compose.yml
```

## 说明

- 单用户登录，会话存内存（重启需重新登录）
- 删除项目会级联删除其报价单、结算单、进料；有记工记录时会阻止删除
- 删除工人时若有记工记录会被阻止

## 开源协议

[MIT](LICENSE)

## 赞助

项目免费使用。如果你觉得有帮助，欢迎在页面右侧点「赞助作者」扫码支持，或加微信交流。
