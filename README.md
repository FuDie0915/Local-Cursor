# Local-Curosr

本地运行的 Cursor IDE 代理工具。通过 MITM 代理劫持 Cursor 的 AI 请求，转发到自定义模型 API（OpenAI / Anthropic 兼容），支持本地助手模式与直连上游模式。

## 架构概览

```
┌──────────────────────────────────────────────────┐
│                  Cursor IDE                       │
│      (settings.json 注入 http.proxy)              │
└────────────┬─────────────────────────────────────┘
             │ HTTPS CONNECT
             ▼
┌──────────────────────┐     白名单域名(.cursor.sh)
│   MITM Proxy Server  │ ────────────────► ┌──────────────────┐
│   (127.0.0.1:28080)  │                   │  Backend Server  │
│                      │ ◄──────────────── │  (127.0.0.1:28090)│
│  非白名单域名直连回源  │     转发+路由      └────────┬─────────┘
└──────────────────────┘                            │
                                           ┌────────┴────────┐
                                           │    Forwarder    │
                                           │  BidiAppend/SSE │
                                           │  Prompt 编译     │
                                           │  Provider 调用   │
                                           └────────┬────────┘
                                                    │
                                           ┌────────┴────────┐
                                           │   Custom API    │
                                           │ (OpenAI/Anthropic)│
                                           └─────────────────┘

┌──────────────────────────────────────────────────┐
│               CLI 版 cursor-agent                  │
│   (agent.cmd wrapper + tls-bypass.cjs preload)    │
└────────────┬─────────────────────────────────────┘
             │ TCP 重定向 (patch net.connect/tls.connect)
             ▼
┌──────────────────────┐
│  CLI HTTPS Listener   │
│ (127.0.0.1:39092)     │ ← HTTPS+HTTP/2，跳过 MITM
│  复用 Backend 路由     │
└──────────────────────┘
```

### 请求流程

**IDE 模式**：

1. Cursor IDE 通过注入的 `http.proxy` 设置将请求发往本地 MITM 代理（`:28080`）
2. MITM 代理对 `.cursor.sh` 白名单域名进行 MITM 解密，转发到本地后端（`:28090`）
3. 后端 `PolicyMiddleware` 根据 `routing.mode` 选择本地处理或上游转发
4. `BidiAppend` / `RunSSE` 进入 `forwarder`，编译 prompt 并调用配置的模型 provider
5. provider 流式响应经 forwarder 广播回 Cursor IDE

**CLI 模式**：

1. CLI 版 `cursor-agent` 通过 wrapper 脚本（`agent.cmd` / `agent`）启动，注入 `NODE_OPTIONS=--require=tls-bypass.cjs`
2. `tls-bypass.cjs` 在 Node.js 层 patch `net.connect` / `tls.connect`，将 `api2.cursor.sh` / `api3.cursor.sh` 的 TCP 连接重定向到本地 `127.0.0.1:39092`
3. CLI HTTPS 监听器（HTTPS+HTTP/2）接收请求，复用 Backend Server 的路由与 forwarder
4. 跳过 MITM 代理，因为 CLI 的 connect-node 使用 HTTP/2，goproxy 无法 MITM HTTP/2 帧

## 功能

| 功能 | 说明 |
|------|------|
| **请求劫持与转发** | MITM 代理劫持 Cursor AI 请求，转发到自定义 API |
| **多模型渠道** | 支持 OpenAI / Anthropic 兼容 API 同时配置多个渠道 |
| **本地助手模式** | `BidiAppend` / `RunSSE` 协议兼容，支持 agent 模式 |
| **Tab 补全** | 可配置上游 Tab 补全服务地址（可选 Docker 部署） |
| **模型测速** | 内置模型渠道延迟/吞吐量基准测试 |
| **CA 证书管理** | 自动注入 CA 证书到系统信任链，支持多平台 |
| **Cursor 设置注入** | 自动写入 Cursor `settings.json`（代理、HTTP/2、证书） |
| **会话历史** | 本地持久化会话状态与使用量统计 |
| **桌面 GUI** | Wails v3 桌面应用，系统托盘常驻 + 配置界面 |
| **CLI 代理适配** | 自动检测 CLI 版 cursor-agent，生成 wrapper + preload 脚本，重定向 TCP 连接到本地后端 |
| **无广告/无更新** | 移除原版广告与自动更新机制 |
| **网络代理感知** | 自动检测系统/环境变量代理并适配出站连接 |

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端语言 | Go 1.25 |
| 桌面框架 | [Wails v3](https://wails.io/) |
| 前端框架 | Vue 3 + Vite + Tailwind CSS |
| 前端图表 | Chart.js + vue-chartjs |
| MITM 代理 | [goproxy](https://github.com/elazarl/goproxy) |
| RPC 协议 | connectrpc + protobuf |
| 本地存储 | SQLite (modernc.org/sqlite) |
| UI 图标 | Iconify |
| 构建工具 | Taskfile.yml |

## 默认端口

| 服务 | 地址 | 说明 |
|------|------|------|
| Backend Server | `127.0.0.1:28090` | 嵌入式 HTTP 后端 |
| MITM Proxy | `127.0.0.1:28080` | Cursor 流量劫持代理 |
| CLI HTTPS | `127.0.0.1:39092` | CLI 专用 HTTPS+HTTP/2 监听器 |
| Tab 补全 | `18041`（可选） | Docker 部署的 Tab 补全服务 |

## 配置

### 配置文件

首次启动后自动生成于 `~/.cursor-local-assistant-v2/config.yaml`：

```yaml
log: false
providerStreamIdleTimeout: 240
backendListenAddr: "127.0.0.1:28090"
proxyListenAddr: "127.0.0.1:28080"
cliHttpsListenAddr: "127.0.0.1:39092"   # CLI 专用 HTTPS+HTTP/2 监听地址
modelAdapters:
  - displayName: "My Model"
    type: "openai"                    # openai | anthropic
    baseURL: "https://api.example.com"
    apiKey: "sk-xxx"
    modelID: "gpt-4"
    reasoningEffort: "medium"         # low | medium | high | xhigh
    openAIEndpoint: "/v1/chat/completions"  # /v1/chat/completions | /v1/responses
    contextWindowTokens: 131072
    maxCompletionTokens: 65536
    customHeadersEnabled: false
    customHeadersJSON: ""
    openAIExtraParamsEnabled: false
    openAIExtraParamsJSON: ""
routing:
  mode: "local"                       # local | upstream
homeMetrics:
  includeCacheWriteInHitRate: false
tabServerBaseURL: ""                  # 可选 Tab 补全服务地址
tabServerToken: ""                    # 可选 Tab 补全服务 Token
```

### 模型渠道配置

在 GUI 界面中可管理多个模型渠道。每个渠道包含：
- `type`: 仅支持 `openai` 或 `anthropic`
- `baseURL` + `apiKey` + `modelID`: 模型 API 端点与凭证
- `reasoningEffort`: OpenAI 推理强度（`low`/`medium`/`high`/`xhigh`）
- `openAIEndpoint`: OpenAI 端点类型
- `anthropicThinkingEffort`: Anthropic 思考预算（`low`/`medium`/`high`/`xhigh`/`max`）
- 自定义请求头、额外参数

渠道唯一 ID 由 `baseURL + modelID + apiKey + displayName + endpoint` 的 SHA-256 hash 决定。

### Tab 补全服务

可选启用 Tab 补全功能，需自行部署：
```bash
# 构建 Docker 镜像
task cursor-tab-server:docker:amd64

# 或参考 cursor-tab-server/ 目录手动部署
```

### CLI 版 cursor-agent 适配

本工具同时支持 CLI 版 `cursor-agent`（非 IDE 内嵌的 agent），自动检测安装路径并生成代理 wrapper。

**工作原理**：

```
~/.cursor-local-assistant-v2/cli/
├── agent.cmd / agent     # wrapper 脚本，设置环境变量后调用真实 CLI
└── tls-bypass.cjs        # Node.js preload，patch net.connect/tls.connect
```

1. **wrapper 脚本**（`agent.cmd` / `agent`）：清除 `HTTP_PROXY`/`HTTPS_PROXY`，设置 `CLI_HTTPS_PORT`、`NODE_EXTRA_CA_CERTS`、`NODE_OPTIONS=--require=tls-bypass.cjs`、`CURSOR_AUTH_TOKEN`，然后调用真实 CLI
2. **preload 脚本**（`tls-bypass.cjs`）：在 Node.js 层 patch `net.connect` 和 `tls.connect`，将 `api2.cursor.sh` / `api3.cursor.sh` 的 TCP 连接重定向到 `127.0.0.1:39092`，同时设置 `rejectUnauthorized=false` 接受本地 CA 证书
3. **auth.json 注入**：写入 mock `auth.json`（`~/.cursor/auth.json` 或 `%APPDATA%\Cursor\auth.json`），使 CLI 跳过浏览器交互式登录
4. **CLI HTTPS 监听器**（`:39092`）：后端启动独立的 HTTPS+HTTP/2 监听器，复用 Backend Server 的路由与 forwarder，注册 CLI 专属的 mock 端点（`GetUsableModels`、`GetDefaultModelForCli`、`ServerConfig`、`loginWithApiKey` 等）

**为什么 CLI 不走 MITM 代理**：

CLI 的 connect-node 使用 HTTP/2（`http2.connect`），goproxy 无法 MITM HTTP/2 帧。因此跳过 MITM 代理，由 preload 脚本在 TCP 层直接重定向到后端的 HTTPS+HTTP/2 监听器。

**CLI 检测路径**：

| 平台 | 检测路径 |
|------|---------|
| Windows | `%LOCALAPPDATA%\cursor-agent\agent.cmd` |
| macOS | `~/.cursor-agent/agent` 或 `~/.local/bin/cursor-agent` |
| Linux | `~/.cursor-agent/agent` 或 `~/.local/bin/cursor-agent` 或 `/usr/local/bin/cursor-agent` |

**使用方式**：

启动 Local-Curosr GUI 并开启服务后，wrapper 自动生成。将 wrapper 路径加入 `PATH` 或直接调用：

```bash
# Windows
~/.cursor-local-assistant-v2/cli/agent.cmd models
~/.cursor-local-assistant-v2/cli/agent.cmd chat

# macOS / Linux
~/.cursor-local-assistant-v2/cli/agent models
~/.cursor-local-assistant-v2/cli/agent chat
```

关闭服务时 wrapper 和 preload 脚本自动清理。

## 数据目录

所有数据存储在 `~/.cursor-local-assistant-v2/`：

```
~/.cursor-local-assistant-v2/
├── config.yaml          # 用户配置
├── data/
│   ├── ca.crt           # 注入 CA 证书
│   ├── codebase-index/  # 代码库索引
│   └── docs-index/      # 文档索引
├── history/
│   ├── usage.json       # 使用量聚合统计
│   └── <conversation_id>/
│       ├── state.json   # Loop 状态
│       ├── context.json # 语义事件
│       └── conversation.lock
├── logs/                # 运行日志
├── cli/                # CLI wrapper 与 preload 脚本（自动生成）
│   ├── agent.cmd / agent
│   └── tls-bypass.cjs
└── rules/               # 用户规则
```

会话状态语义：
- `idle`: 无活跃 loop
- `running`: 等待/发起模型调用
- `waiting_tool`: 等待工具执行结果
- `completed`: 正常完成
- `canceled`: 已取消
- `provider_error`: LLM 调用失败
- `failed`: 本地内部错误

## 目录结构

```
├── main.go                    # 应用入口（Wails）
├── internal/
│   ├── app/                   # 桌面应用 (窗口、托盘、Wails 服务注册)
│   ├── appdata/               # 数据目录路径管理
│   ├── backend/
│   │   ├── host.go            # 后端唯一组装点
│   │   ├── server/            # HTTP/Connect 路由层
│   │   │   ├── config/        # 配置管理 (Store/Manager/Types)
│   │   │   └── upstream/      # 上游转发与 Mock 动作
│   │   ├── forwarder/         # 本地转发内核 (56 文件)
│   │   │   ├── compiler.go    # Prompt 编译
│   │   │   ├── provider.go    # LLM provider 调用
│   │   │   ├── broker.go      # SSE 广播
│   │   │   ├── tool_catalog.go# 工具注册表
│   │   │   └── ...
│   │   └── agent/             # 本地 Agent 模式
│   │       ├── model/         # OpenAI/Anthropic 适配器
│   │       └── prompt/        # Prompt 引擎
│   ├── bridge/                # Wails 桌面桥接服务
│   ├── buildinfo/             # 构建版本
│   ├── certs/                 # CA 证书管理
│   ├── client/                # 客户端运行时 (生命周期/配置/许可证)
│   ├── cursor/                # Cursor IDE 集成 (设置注入/设备 ID/CLI wrapper)
│   ├── historymetrics/        # 使用量指标
│   ├── logger/                # 日志
│   ├── mitm/                  # MITM 代理服务
│   ├── modelchannel/          # 模型渠道身份/解析
│   ├── netproxy/              # 出站网络代理检测
│   └── runtime/               # 运行时配置快照
├── frontend/                  # Vue 3 桌面 UI
│   └── src/                   # 首页、模型配置、编辑器等页面
├── internal/               # 内部业务包
│   ├── prompt/              # LLM Prompt 模板
│   │   ├── agent/           # Agent 模式 prompt
│   │   ├── plan/            # Plan 模式 prompt
│   │   ├── commit/          # 提交信息 prompt
│   │   ├── ask/             # 问答模式 prompt
│   │   └── ...
├── proto/                     # Protobuf 定义
├── cursor-tab-server/         # Tab 补全服务 (可选 Docker)
├── build/                     # 构建配置与平台脚本
└── scripts/                   # 发布/版本工具
```

## 构建与开发

### 前置依赖

- Go 1.25+
- Node.js 20+
- [Wails v3 CLI](https://wails.io/)

### 开发模式

```bash
# 启动前端 Dev Server + Wails 开发模式
task dev
```

### 构建

```bash
# 构建当前平台分发包
task build

# 构建所有平台 (仅 macOS)
task build:all

# 构建具体平台
task build:darwin:arm64
task build:darwin:amd64
task build:windows:amd64
task build:linux:amd64
```

### 发布

```bash
# 生成发布资产 (需要 gh CLI 和 wails3 CLI)
task release:prepare

# 发布到 GitHub Releases
task release:github
```

## 许可证

MIT
