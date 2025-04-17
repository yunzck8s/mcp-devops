# MCP-DevOps Kubernetes 管理系统

MCP-DevOps 是一个基于 Go 语言开发的 Kubernetes 资源管理系统，它提供了简单易用的命令行界面来管理 Kubernetes 集群资源。该系统使用客户端-服务器架构，通过语义化交互提供直观的操作方式。

## 系统架构

系统由两部分组成：

1. **Server**：运行在有权限访问 Kubernetes 集群的环境中，提供 Kubernetes 资源操作的 API 接口
2. **Client**：命令行交互客户端，连接到服务器并提供自然语言交互界面

## 功能特性

- **Pod 管理**：列出、描述、删除 Pod，查看 Pod 日志
- **Deployment 管理**：列出、描述、扩缩容、重启 Deployment
- **Service 管理**：列出、描述 Service
- **Namespace 管理**：列出、描述、创建、删除 Namespace
- **企业微信通知**：支持发送文本、Markdown和卡片类型的企业微信消息。默认文本消息会自动转换为带颜色和时间戳的 Markdown 格式。
- **Alertmanager Webhook 集成**：客户端内置 Webhook 监听器（默认端口 9094），可接收 Alertmanager 告警，交由 AI 分析并通过企业微信发送通知。
- **自然语言交互**：通过自然语言描述你想执行的操作
- **中文支持**：系统默认使用中文进行交互

## 环境要求

- Go 1.23.0 或更高版本
- Kubernetes 集群的访问配置（kubeconfig 文件）

## 快速开始

### 环境配置

1. 配置 `.env` 文件（项目根目录已提供示例）：

```
# 服务器配置
MCP_SERVER_ADDRESS=0.0.0.0:12345
API_KEY=your-secret-api-key

# 客户端配置
MCP_SERVER_URL=http://127.0.0.1:12345/sse
OPENAI_API_KEY=your-openai-api-key
OPENAI_BASE_URL=https://dashscope.aliyuncs.com/compatible-mode/v1
OPENAI_MODEL=qwen-max

# 企业微信配置
WECHAT_WEBHOOK_URL=https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=your-webhook-key
```

### 启动服务器

```bash
cd server
go run main.go
```

服务器将监听配置的地址和端口，为客户端提供 Kubernetes 资源管理 API。

### 启动客户端

```bash
cd client
go run main.go
```

客户端启动后，会连接到服务器并提供命令行交互界面。同时，它会在后台启动一个 Webhook 监听器（默认监听 `http://localhost:9094/webhook`）用于接收 Alertmanager 告警。

## 使用示例

客户端启动后，您可以使用自然语言输入以下示例命令：

- 查看所有命名空间：`查看所有命名空间`
- 查看默认命名空间中的所有 Pod：`查看 default 命名空间中的所有 Pod`
- 查看特定 Pod 的详细信息：`描述 pod-name 这个 Pod`
- 查看 Pod 日志：`查看 pod-name 的日志`
- 扩展 Deployment：`将 deployment-name 扩展到 3 个副本`
- 发送企业微信通知：`发送企业微信消息"Kubernetes集群重启完成"`

## 安全注意事项

- 服务器应部署在安全的环境中，因为它具有 Kubernetes 集群的访问权限
- 生产环境中应配置合适的认证机制
- 对于危险操作（如删除资源），客户端将提供安全提示和确认机制

## 项目结构

```
mcp-devops/
├── client/           # 客户端代码
│   ├── main.go       # 客户端主程序
│   └── pkg/          # 客户端包
├── server/           # 服务器代码
│   ├── main.go       # 服务器主程序
│   └── k8s/          # Kubernetes 操作工具
├── .env              # 环境配置文件
├── go.mod            # Go 模块定义
└── go.sum            # Go 依赖校验
```

## 开发与扩展

如需添加新的 Kubernetes 资源管理功能，您可以按照以下步骤进行：

1. 在 `server/k8s/` 目录中添加相应的处理函数
2. 在 `server/main.go` 中注册新的工具
3. 重启服务器和客户端

## 注意事项

- 客户端需要使用大语言模型 API，请确保 OPENAI_API_KEY 有效
- 如果在集群外运行服务器，请确保正确配置了 kubeconfig
- 默认配置使用阿里通义模型 API，可以根据需要更换为其他 API
- 企业微信通知功能需要配置有效的企业微信群机器人 Webhook URL
- 如需使用 Alertmanager 告警集成，请配置 Alertmanager 将告警发送到客户端运行机器的 `http://<client-ip>:9094/webhook` 地址。
