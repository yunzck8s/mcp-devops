# MCP-DevOps API 文档

## 概述

MCP-DevOps 基于 Model Context Protocol (MCP) 协议，提供了一套完整的 DevOps 工具集。本文档详细描述了所有可用的工具和 API。

## 目录

- [ClickHouse 工具](#clickhouse-工具)
- [Kubernetes 工具](#kubernetes-工具)
- [Redis 工具](#redis-工具)
- [Linux 系统工具](#linux-系统工具)
- [Loki 日志工具](#loki-日志工具)
- [告警和通知工具](#告警和通知工具)

---

## ClickHouse 工具

ClickHouse 工具集提供了全面的分布式追踪分析能力，帮助诊断和优化微服务系统性能。

### 1. 查找重负载 Trace

**工具名称：** `bb7_find_heavy_traces`

**描述：** 查找 span 数量超过阈值的 trace_id 列表，用于识别过于复杂的请求链路。

**参数：**
- `threshold` (可选): span 数量告警阈值，默认 1000
- `time_range_hours` (可选): 查询时间范围（小时），默认 1
- `limit` (可选): 结果限制条数，默认 100

**使用示例：**
```json
{
  "threshold": 500,
  "time_range_hours": 2,
  "limit": 50
}
```

**返回示例：**
```json
{
  "traces": [
    {
      "trace_id": "abc123def456",
      "span_count": 1250,
      "duration_ms": 5500,
      "service_name": "payment-service"
    }
  ],
  "total": 15
}
```

### 2. 查找最慢 Trace

**工具名称：** `bb7_find_top_slowest_traces`

**描述：** 按最大 span 耗时排序，获取 trace_id 列表，用于性能优化。

**参数：**
- `time_range_hours` (可选): 查询时间范围（小时），默认 1
- `limit` (可选): 结果限制条数，默认 10

**使用示例：**
```json
{
  "time_range_hours": 6,
  "limit": 20
}
```

### 3. 查找错误 Trace

**工具名称：** `bb7_find_error_traces`

**描述：** 查找包含错误 spans 的 trace_id 列表，用于快速定位问题。

**参数：**
- `time_range_hours` (可选): 查询时间范围（小时），默认 1
- `limit` (可选): 结果限制条数，默认 100

### 4. 综合 Trace 报告

**工具名称：** `bb7_report_trace`

**描述：** 生成指定 trace_id 的综合报告，包含完整的链路信息和性能指标。

**参数：**
- `trace_id` (必需): 要分析的 trace_id

**使用示例：**
```json
{
  "trace_id": "abc123def456"
}
```

### 5. 详细 Trace 分析

**工具名称：** `bb7_analyze_trace`

**描述：** 详细分析单个 trace_id，包括 span 列表、时间线、错误信息等。

**参数：**
- `trace_id` (必需): 要分析的 trace_id
- `limit` (可选): 返回的 span 数量限制，默认 50

### 6. 深度 Trace 分析

**工具名称：** `bb7_deep_analyze_trace`

**描述：** 深度分析 trace 中每个 span 节点的行为，识别不合理的模式和性能问题。

**参数：**
- `trace_id` (必需): 要深度分析的 trace_id

### 7. 服务深度分析

**工具名称：** `bb7_analyze_service`

**描述：** 深度分析单个服务的性能指标、调用模式和异常行为。

**参数：**
- `service_name` (必需): 要分析的服务名称
- `time_range_hours` (可选): 分析时间范围（小时），默认 24
- `limit` (可选): 返回的示例数量限制，默认 100

**使用示例：**
```json
{
  "service_name": "payment-service",
  "time_range_hours": 12,
  "limit": 200
}
```

### 8. 服务依赖分析

**工具名称：** `bb7_analyze_service_dependencies`

**描述：** 分析服务间的调用关系和依赖模式。

**参数：**
- `service_name` (必需): 要分析的服务名称
- `time_range_hours` (可选): 分析时间范围（小时），默认 1
- `limit` (可选): 返回的依赖服务数量限制，默认 20

### 9. 服务问题查找

**工具名称：** `bb7_find_service_issues`

**描述：** 查找特定服务的异常 trace，包括错误、慢查询等。

**参数：**
- `service_name` (必需): 要查询的服务名称
- `time_range_hours` (可选): 查询时间范围（小时），默认 1
- `min_duration_ms` (可选): 最小耗时阈值（毫秒），默认 1000
- `limit` (可选): 结果限制条数，默认 50

### 10. 频繁操作分析

**工具名称：** `bb7_find_frequent_operations`

**描述：** 查找指定时间范围内频繁执行的操作和对应的 trace。

**参数：**
- `time_range_hours` (可选): 查询时间范围（小时），默认 1
- `min_frequency` (可选): 最小频率阈值，默认 100 次
- `limit` (可选): 结果限制条数，默认 20

### 11. 异常模式检测

**工具名称：** `bb7_find_anomaly_traces`

**描述：** 查找异常模式的 trace，如超长链路、错误集中等。

**参数：**
- `time_range_hours` (可选): 查询时间范围（小时），默认 1
- `limit` (可选): 结果限制条数，默认 50

---

## Kubernetes 工具

### Pod 管理

#### 列出 Pod

**工具名称：** `bb7_list_pods`

**参数：**
- `namespace` (可选): 命名空间，默认 "default"

#### Pod 详情

**工具名称：** `bb7_describe_pod`

**参数：**
- `pod_name` (必需): Pod 名称
- `namespace` (可选): 命名空间，默认 "default"

#### Pod 日志

**工具名称：** `bb7_pod_logs`

**参数：**
- `pod_name` (必需): Pod 名称
- `namespace` (可选): 命名空间，默认 "default"
- `container` (可选): 容器名称
- `tail` (可选): 日志行数，默认 100

#### Pod 诊断

**工具名称：** `bb7_pod_diagnostic`

**参数：**
- `pod_name` (必需): Pod 名称
- `namespace` (可选): 命名空间，默认 "default"

#### 删除 Pod

**工具名称：** `bb7_delete_pod`

**参数：**
- `pod_name` (必需): Pod 名称
- `namespace` (可选): 命名空间，默认 "default"
- `force` (可选): 是否强制删除，默认 false

### Deployment 管理

#### 列出 Deployment

**工具名称：** `bb7_list_deployments`

#### Deployment 详情

**工具名称：** `bb7_describe_deployment`

#### 扩缩容 Deployment

**工具名称：** `bb7_scale_deployment`

**参数：**
- `deployment_name` (必需): Deployment 名称
- `namespace` (可选): 命名空间，默认 "default"
- `replicas` (必需): 副本数

#### 重启 Deployment

**工具名称：** `bb7_restart_deployment`

#### Deployment 诊断

**工具名称：** `bb7_deployment_diagnostic`

### Service 管理

#### 列出 Service

**工具名称：** `bb7_list_services`

#### Service 详情

**工具名称：** `bb7_describe_service`

#### 修改 Service 类型

**工具名称：** `bb7_modify_service_type`

**参数：**
- `service_name` (必需): Service 名称
- `namespace` (可选): 命名空间，默认 "default"
- `service_type` (必需): Service 类型 (ClusterIP, NodePort, LoadBalancer)

### ConfigMap 管理

#### 创建 ConfigMap

**工具名称：** `bb7_create_configmap`

**参数：**
- `configmap_name` (必需): ConfigMap 名称
- `namespace` (可选): 命名空间，默认 "default"
- `data` (可选): 数据键值对
- `from_file` (可选): 从文件创建
- `from_literal` (可选): 从字面值创建

#### 更新 ConfigMap

**工具名称：** `bb7_update_configmap`

#### 删除 ConfigMap

**工具名称：** `bb7_delete_configmap`

### Secret 管理

#### 创建 Secret

**工具名称：** `bb7_create_secret`

#### 更新 Secret

**工具名称：** `bb7_update_secret`

#### 删除 Secret

**工具名称：** `bb7_delete_secret`

### Ingress 管理

#### 创建 Ingress

**工具名称：** `bb7_create_ingress`

**参数：**
- `ingress_name` (必需): Ingress 名称
- `service_name` (必需): 后端服务名称
- `namespace` (可选): 命名空间，默认 "default"
- `host` (可选): 主机名
- `path` (可选): 路径，默认 "/"
- `service_port` (可选): 服务端口，默认 80
- `tls_enabled` (可选): 是否启用 TLS，默认 false

---

## Redis 工具

### Redis 信息

**工具名称：** `bb7_redis_info`

**参数：**
- `section` (可选): 信息部分，默认 "ALL"

### Redis 延迟测试

**工具名称：** `bb7_redis_latency`

### Redis 延迟历史

**工具名称：** `bb7_redis_latency_history`

**参数：**
- `count` (可选): 测量次数，默认 10
- `interval` (可选): 间隔时间（秒），默认 1

### Redis 慢查询

**工具名称：** `bb7_redis_slowlog`

**参数：**
- `count` (可选): 慢查询条目数，默认 10

### Redis 大键分析

**工具名称：** `bb7_redis_bigkeys`

### Redis 热键分析

**工具名称：** `bb7_redis_hotkeys`

---

## Linux 系统工具

### 系统信息

**工具名称：** `bb7_system_info`

**参数：**
- `hostname` (可选): 主机名

### 资源使用情况

**工具名称：** `bb7_resource_usage`

**参数：**
- `hostname` (可选): 主机名
- `duration` (可选): 监控持续时间（秒），默认 5

### 进程信息

**工具名称：** `bb7_process_info`

**参数：**
- `hostname` (可选): 主机名
- `process_name` (可选): 进程名称
- `top_count` (可选): 显示进程数量，默认 10

### 网络信息

**工具名称：** `bb7_network_info`

**参数：**
- `hostname` (可选): 主机名
- `interface` (可选): 网络接口名称

### 日志分析

**工具名称：** `bb7_log_analysis`

**参数：**
- `hostname` (可选): 主机名
- `log_path` (可选): 日志文件路径，默认 "/var/log/syslog"
- `pattern` (可选): 搜索模式
- `lines` (可选): 显示行数，默认 50

---

## Loki 日志工具

### 服务日志查询

**工具名称：** `bb7_loki_service_logs`

**参数：**
- `service_name` (必需): 服务名称
- `loki_address` (可选): Loki 服务器地址，默认 "http://localhost:3100"

### 时间范围日志查询

**工具名称：** `bb7_loki_time_range_logs`

**参数：**
- `service_name` (必需): 服务名称
- `start_time` (必需): 开始时间 (RFC3339 格式)
- `end_time` (必需): 结束时间 (RFC3339 格式)
- `loki_address` (可选): Loki 服务器地址

---

## 告警和通知工具

### 告警分析

**工具名称：** `bb7_alert_analysis`

**参数：**
- `alert_name` (可选): 告警名称
- `severity` (可选): 告警严重性
- `status` (可选): 告警状态
- `description` (可选): 告警描述
- `namespace` (可选): 相关命名空间
- `pod_name` (可选): 相关 Pod 名称
- `node_name` (可选): 相关节点名称

### 企业微信通知

**工具名称：** `bb7_send_wechat_message`

**参数：**
- `content` (必需): 消息内容
- `msg_type` (可选): 消息类型 (text, markdown, template_card)，默认 "text"
- `title` (可选): 消息标题
- `webhook_url` (可选): Webhook 地址
- `card_type` (可选): 卡片类型，默认 "text_notice"

---

## 错误处理

所有工具都返回标准的 MCP 响应格式：

**成功响应：**
```json
{
  "content": [
    {
      "type": "text",
      "text": "操作结果..."
    }
  ]
}
```

**错误响应：**
```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "错误描述"
  }
}
```

## 通用约定

### 时间格式
- 时间范围通常以小时为单位
- 绝对时间使用 RFC3339 格式 (例: 2023-10-01T00:00:00Z)

### 命名空间
- 默认命名空间为 "default"
- 支持所有 Kubernetes 命名空间

### 限制参数
- `limit` 参数用于控制返回结果数量
- 默认值根据具体工具而定，通常在 10-100 之间

### 主机名
- `hostname` 参数用于指定目标主机
- 如不提供则操作本地系统

---

## 版本信息

- API 版本: v1.0
- 支持的 MCP 协议版本: 2024-11-05
- 最后更新: 2024-12-20

## 许可证

本项目采用 MIT 许可证，详见 [LICENSE](LICENSE) 文件。
