# 🤝 贡献指南

感谢您考虑为 MCP-DevOps 项目做出贡献！本指南将帮助您了解如何有效地参与项目开发。

## 📋 目录

- [行为准则](#行为准则)
- [开始贡献](#开始贡献)
- [开发环境设置](#开发环境设置)
- [贡献类型](#贡献类型)
- [提交指南](#提交指南)
- [代码规范](#代码规范)
- [测试要求](#测试要求)
- [Pull Request 流程](#pull-request-流程)

## 🌟 行为准则

我们致力于为每个人提供友好、安全和欢迎的环境。请确保您的参与遵循以下原则：

- 🤝 尊重不同观点和经验
- 🎯 专注于对社区最有益的事情
- 💬 友善和建设性的沟通
- 🚫 不容忍任何形式的骚扰或不当行为

## 🚀 开始贡献

### 1. Fork 和克隆项目

```bash
# Fork 项目到您的 GitHub 账户
# 然后克隆您的 fork
git clone https://github.com/yourusername/mcp-devops.git
cd mcp-devops

# 添加上游仓库
git remote add upstream https://github.com/originalowner/mcp-devops.git
```

### 2. 保持同步

```bash
# 获取最新更改
git fetch upstream
git checkout main
git merge upstream/main
```

## 🛠️ 开发环境设置

### 系统要求

- **Go**: 1.23.0 或更高版本
- **Git**: 最新版本
- **Kubernetes 集群**: 用于测试（可选）
- **ClickHouse**: 用于链路追踪功能测试（可选）
- **Loki**: 用于日志分析功能测试（可选）

### 安装依赖

```bash
# 安装 Go 依赖
go mod tidy

# 验证安装
go version
go mod verify
```

### 配置环境

1. 复制环境配置文件：
```bash
cp .env.example .env
```

2. 根据您的环境修改 `.env` 文件中的配置

## 📝 贡献类型

我们欢迎以下类型的贡献：

### 🐛 Bug 报告
- 使用 GitHub Issues 报告 bug
- 提供详细的重现步骤
- 包含环境信息和错误日志
- 标注相关的标签

### ✨ 功能请求
- 在 GitHub Issues 中提出新功能想法
- 描述使用场景和预期行为
- 考虑向后兼容性

### 📖 文档改进
- 修复文档错误
- 添加缺失的文档
- 改进代码注释
- 翻译文档

### 🔧 代码贡献
- 修复 bug
- 实现新功能
- 性能优化
- 重构代码

## 📋 提交指南

### 提交消息格式

我们使用 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

**类型 (type):**
- `feat`: 新功能
- `fix`: bug 修复
- `docs`: 文档更改
- `style`: 代码格式（不影响代码运行的变动）
- `refactor`: 重构（既不是新增功能，也不是修改bug的代码变动）
- `perf`: 性能优化
- `test`: 添加测试
- `chore`: 构建过程或辅助工具的变动

**示例:**
```bash
git commit -m "feat(k8s): 添加 PersistentVolume 管理功能"
git commit -m "fix(clickhouse): 修复连接超时问题"
git commit -m "docs: 更新 API 文档示例"
```

### 分支命名规范

```bash
# 功能分支
feature/add-persistent-volume-support
feature/improve-error-handling

# Bug 修复分支
fix/clickhouse-connection-timeout
fix/memory-leak-in-pod-monitoring

# 文档分支
docs/update-api-documentation
docs/add-troubleshooting-guide
```

## 🎯 代码规范

### Go 代码规范

1. **格式化**: 使用 `go fmt` 格式化代码
2. **导入**: 按标准库、第三方库、本地包的顺序组织导入
3. **命名**: 遵循 Go 命名约定
4. **注释**: 为公共函数和复杂逻辑添加注释

```go
// 好的示例
package k8s

import (
    "context"
    "fmt"
    
    "k8s.io/client-go/kubernetes"
    "github.com/mark3labs/mcp-go"
    
    "mcp-devops/internal/types"
)

// ListPodsTool 列出指定命名空间中的所有 Pod
// 参数:
//   - namespace: 要查询的命名空间，默认为 "default"
//   - labelSelector: 可选的标签选择器
func ListPodsTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // 实现逻辑...
}
```

### 错误处理

```go
// 好的错误处理
func getPods(namespace string) ([]corev1.Pod, error) {
    client, err := getKubernetesClient()
    if err != nil {
        return nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
    }
    
    pods, err := client.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
    if err != nil {
        return nil, fmt.Errorf("列出 Pod 失败: %w", err)
    }
    
    return pods.Items, nil
}
```

### 日志记录

```go
import "log/slog"

// 使用结构化日志
slog.Info("开始处理请求", 
    "function", "ListPodsTool",
    "namespace", namespace,
    "labelSelector", labelSelector)

slog.Error("操作失败", 
    "error", err,
    "namespace", namespace)
```

## 🧪 测试要求

### 单元测试

每个新功能都应该包含相应的单元测试：

```go
func TestListPodsTool(t *testing.T) {
    tests := []struct {
        name      string
        namespace string
        expected  int
        wantErr   bool
    }{
        {
            name:      "default namespace",
            namespace: "default",
            expected:  3,
            wantErr:   false,
        },
        {
            name:      "invalid namespace",
            namespace: "invalid-ns",
            expected:  0,
            wantErr:   true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 测试实现...
        })
    }
}
```

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./server/k8s/...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 基准测试

对于性能敏感的代码，添加基准测试：

```go
func BenchmarkListPods(b *testing.B) {
    for i := 0; i < b.N; i++ {
        // 基准测试代码...
    }
}
```

## 🔄 Pull Request 流程

### 1. 创建 Pull Request

1. 确保您的分支是基于最新的 `main` 分支
2. 运行所有测试并确保通过
3. 更新相关文档
4. 在 GitHub 上创建 Pull Request

### 2. Pull Request 模板

```markdown
## 📋 变更说明

### 变更类型
- [ ] Bug 修复
- [ ] 新功能
- [ ] 文档更新
- [ ] 重构
- [ ] 性能优化

### 📝 描述
简要描述此 PR 的目的和变更内容。

### 🧪 测试
- [ ] 添加了新的测试用例
- [ ] 所有现有测试通过
- [ ] 手动测试验证

### 📚 文档
- [ ] 更新了相关文档
- [ ] 添加了代码注释

### ✅ 检查清单
- [ ] 代码遵循项目规范
- [ ] 提交消息符合规范
- [ ] 没有引入破坏性变更
```

### 3. 代码审查

- 保持开放和建设性的态度
- 及时响应审查意见
- 根据反馈进行必要的修改

### 4. 合并要求

在 PR 被合并之前，需要满足以下条件：

- ✅ 至少一位维护者的批准
- ✅ 所有自动化检查通过
- ✅ 没有合并冲突
- ✅ 遵循代码规范

## 🎯 特定贡献指南

### 添加新的 Kubernetes 资源支持

1. **创建资源处理文件**
```bash
# 在 server/k8s/ 目录下创建新文件
touch server/k8s/persistentvolume.go
```

2. **实现资源操作函数**
```go
package k8s

import (
    "context"
    "github.com/mark3labs/mcp-go"
)

func ListPersistentVolumesTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // 实现 PV 列表功能
}

func DescribePersistentVolumeTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // 实现 PV 描述功能
}
```

3. **注册工具**
在 `server/sse/server.go` 中添加：
```go
// 注册 PersistentVolume 相关工具
svr.AddTool(mcp.NewTool("list_persistent_volumes",
    mcp.WithDescription("列出集群中的 PersistentVolume"),
), k8s.ListPersistentVolumesTool)
```

4. **添加测试**
```go
func TestListPersistentVolumesTool(t *testing.T) {
    // 测试实现...
}
```

5. **更新文档**
在 README.md 中添加新功能的使用示例。

### 添加新的诊断工具

1. **确定诊断范围**: 明确工具要解决的问题
2. **设计 API 接口**: 定义输入参数和输出格式
3. **实现核心逻辑**: 编写诊断算法
4. **添加错误处理**: 优雅处理各种异常情况
5. **编写测试用例**: 确保功能正确性
6. **更新文档**: 添加使用说明和示例

## 📞 获取帮助

如果您在贡献过程中遇到问题，可以通过以下方式获取帮助：

- 💬 [GitHub Discussions](https://github.com/yourusername/mcp-devops/discussions)
- 📧 发送邮件到 [team@mcp-devops.com](mailto:team@mcp-devops.com)
- 🐛 创建 [GitHub Issue](https://github.com/yourusername/mcp-devops/issues)

## 🎉 感谢

感谢您的贡献！每一个 PR、Issue 报告、文档改进都让项目变得更好。

---

**Happy Contributing! 🚀**
