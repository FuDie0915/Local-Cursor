# 项目结构说明

## 目录布局

```
Local-Curosr/
├── cmd/                    # 独立可执行工具入口
│   └── ext-tool/           # proto 提取工具，从 Cursor 扩展源码生成 .proto 定义
├── internal/               # 内部业务包（不对外暴露）
│   ├── app/                # Wails 应用启动与生命周期
│   ├── appdata/            # 用户数据目录管理
│   ├── backend/            # 后端服务核心
│   │   ├── agent/          # Agent 引擎（模型适配、prompt 引擎、协议）
│   │   ├── forwarder/      # 请求转发与 AI 处理
│   │   ├── server/         # HTTP 路由、中间件、配置
│   │   └── host.go         # 后端宿主，管理 HTTP/HTTPS 监听器
│   ├── bridge/             # Wails 前端桥接层
│   ├── buildinfo/          # 构建版本信息
│   ├── certs/              # CA 证书管理
│   ├── client/             # 客户端服务（MITM 代理、CLI wrapper 生成）
│   ├── cursor/             # Cursor 宿主交互（代理设置、证书安装）
│   ├── historymetrics/     # 历史用量指标
│   ├── logger/             # 日志
│   ├── mitm/               # MITM 代理服务
│   ├── modelchannel/       # 模型通道解析
│   ├── netproxy/           # 出站网络代理
│   ├── prompt/             # 静态 prompt 资产与加载
│   └── runtime/            # 运行时注入（auth token、账号信息）
├── proto/                  # Protobuf 定义与扩展提取工具
├── scripts/                # 构建/发布辅助脚本
├── build/                  # 平台构建配置与资源
├── cursor-tab-server/      # 独立子模块：Tab 补全服务
├── main.go                 # 主入口
├── go.mod                  # Go 模块定义
├── Taskfile.yml            # 构建/发布任务
└── build/config.yml        # Wails 构建配置
```

## 数据流

```
Cursor IDE → MITM(38081) → backend(39091) → 路由 → forwarder → agent/model → 上游模型
CLI agent  → tls-bypass → HTTPS(39092) → 同一 backend mux → 同上
```