# MCP-DevOps Kubernetes 管理系统

[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/569/badge)](https://bestpractices.coreinfrastructure.org/projects/569) [![Go Report Card](https://goreportcard.com/badge/github.com/kubernetes/kubernetes)](https://goreportcard.com/report/github.com/kubernetes/kubernetes) ![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/kubernetes/kubernetes?sort=semver)

<img src="logo.png" width="300">


MCP-DevOps 是一个基于 Go 语言开发的 Kubernetes 资源管理系统，它提供了简单易用的命令行界面来管理 Kubernetes 集群资源。该系统使用客户端-服务器架构，通过语义化交互提供直观的操作方式。

## 系统架构

系统由两部分组成：

1. **Server**：运行在有权限访问 Kubernetes 集群的环境中，提供 Kubernetes 资源操作的 API 接口
2. **Client**：命令行交互客户端，连接到服务器并提供自然语言交互界面

## 功能特性

### Kubernetes 资源管理
- **Pod 管理**：列出、描述、删除 Pod，查看 Pod 日志
- **Deployment 管理**：列出、描述、扩缩容、重启 Deployment
- **StatefulSet 管理**：列出、描述、扩缩容、重启 StatefulSet
- **Service 管理**：列出、描述、修改 Service
- **Namespace 管理**：列出、描述、创建、删除 Namespace
- **Ingress 管理**：列出、描述、创建、更新、删除 Ingress
- **ConfigMap 管理**：列出、描述、创建、更新、删除 ConfigMap
- **Secret 管理**：列出、描述、创建、更新、删除 Secret

### 故障诊断与告警处理
- **集群健康检查**：获取集群整体健康状态，包括节点、Pod 和命名空间状态
- **Pod 诊断**：深入分析 Pod 问题，检查容器状态、事件和日志，提供解决建议
- **节点诊断**：检查节点状态、资源使用情况和运行的 Pod，识别潜在问题
- **Deployment 诊断**：分析 Deployment 部署和更新问题，检查副本状态和事件
- **告警分析**：处理和分析 Prometheus/Alertmanager 告警，提供根本原因分析和解决方案
- **企业微信通知**：支持发送文本、Markdown 和卡片类型的企业微信消息，用于告警通知和状态报告
- **Alertmanager Webhook 集成**：客户端内置 Webhook 监听器（默认端口 9094），可接收 Alertmanager 告警，交由 AI 分析并通过企业微信发送通知

### Linux 系统排查
- **系统信息**：获取主机基本信息，包括操作系统、内核版本、资源使用情况等
- **进程管理**：查看和分析进程状态，识别高 CPU 或内存使用的进程
- **资源监控**：监控 CPU、内存和磁盘使用情况，识别资源瓶颈
- **网络诊断**：检查网络连接、接口状态和路由配置，排查网络问题
- **日志分析**：分析系统日志文件，查找错误和警告信息
- **服务状态**：检查系统服务运行状态，管理服务启停

### Kubernetes 组件排查
- **Kubelet 状态**：检查 Kubelet 服务状态和日志，排查节点问题
- **容器运行时**：检查 Docker、Containerd 或 CRI-O 状态，排查容器问题
- **Kube-Proxy**：检查 Kube-Proxy 状态和配置，排查服务网络问题
- **CNI 状态**：检查网络插件状态和配置，排查 Pod 网络问题
- **组件日志**：分析 Kubernetes 组件日志，查找错误和警告信息
- **容器检查**：检查容器详情和日志，深入排查应用问题

### 交互与用户体验
- **自然语言交互**：通过自然语言描述你想执行的操作
- **中文支持**：系统默认使用中文进行交互
- **智能故障分析**：AI 自动分析故障模式并提供解决方案
- **告警智能处理**：自动分析告警信息，执行诊断步骤，提供详细报告

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

### Kubernetes 资源管理
- 查看所有命名空间：`查看所有命名空间`
- 查看默认命名空间中的所有 Pod：`查看 default 命名空间中的所有 Pod`
- 查看特定 Pod 的详细信息：`描述 pod-name 这个 Pod`
- 查看 Pod 日志：`查看 pod-name 的日志`
- 扩展 Deployment：`将 deployment-name 扩展到 3 个副本`
- 创建 ConfigMap：`创建一个名为 app-config 的 ConfigMap，包含 key1=value1 和 key2=value2`
- 创建 Ingress：`为 my-service 服务创建一个 Ingress，主机名为 example.com，路径为 /api`
- 更新 Secret：`更新 my-secret，添加 username=admin 和 password=secure123`

### 故障诊断与告警处理
- 检查集群健康状态：`检查集群健康状态`
- 诊断 Pod 问题：`诊断 Pod my-pod 的问题`
- 诊断节点问题：`诊断节点 worker-1 的问题`
- 分析 Deployment 问题：`分析 Deployment my-app 的问题`
- 分析告警：`分析 CPU 使用率高的告警，节点是 worker-1，严重性是 warning`
- 发送企业微信通知：`发送企业微信消息"Kubernetes集群重启完成"`

### Linux 系统排查
- 获取系统信息：`获取节点 worker-1 的系统信息`
- 查看进程信息：`查看节点 worker-1 上的 kubelet 进程`
- 监控资源使用情况：`查看节点 worker-1 的资源使用情况`
- 分析日志：`分析节点 worker-1 上的 /var/log/syslog 日志，查找 error 关键字`
- 检查服务状态：`检查节点 worker-1 上的 docker 服务状态`

### Kubernetes 组件排查
- 检查 Kubelet 状态：`检查节点 worker-1 上的 Kubelet 状态`
- 检查容器运行时：`检查节点 worker-1 上的 containerd 状态`
- 检查 Kube-Proxy：`检查节点 worker-1 上的 Kube-Proxy 状态`
- 检查 CNI 状态：`检查节点 worker-1 上的 calico 状态`
- 查看组件日志：`查看节点 worker-1 上的 kubelet 日志`
- 检查容器详情：`检查节点 worker-1 上的容器 abc123 的详情`

## 安全注意事项

- 服务器应部署在安全的环境中，因为它具有 Kubernetes 集群的访问权限
- 生产环境中应配置合适的认证机制
- 对于危险操作（如删除资源），客户端将提供安全提示和确认机制

## 项目结构

```
mcp-devops/
├── client/                # 客户端代码
│   ├── main.go            # 客户端主程序
│   └── pkg/               # 客户端包
│       ├── model/         # 模型相关代码
│       └── mcp/           # MCP 客户端实现
├── server/                # 服务器代码
│   ├── main.go            # 服务器主程序
│   ├── k8s/               # Kubernetes 操作工具
│   │   ├── client.go      # Kubernetes 客户端
│   │   ├── pod.go         # Pod 相关操作
│   │   ├── deployment.go  # Deployment 相关操作
│   │   ├── service.go     # Service 相关操作
│   │   ├── statefulset.go # StatefulSet 相关操作
│   │   ├── namespace.go   # Namespace 相关操作
│   │   ├── ingress.go     # Ingress 相关操作
│   │   ├── configmap.go   # ConfigMap 相关操作
│   │   ├── secret.go      # Secret 相关操作
│   │   ├── troubleshoot.go # 故障诊断工具
│   │   └── wechat.go      # 企业微信通知
│   ├── linux/             # Linux 系统操作工具
│   │   ├── system.go      # 系统信息和资源监控
│   │   └── kubernetes.go  # Kubernetes 组件排查
│   └── sse/               # SSE 服务实现
│       └── server.go      # SSE 服务器
├── .env                   # 环境配置文件
├── go.mod                 # Go 模块定义
└── go.sum                 # Go 依赖校验
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
