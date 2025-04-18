# MCP-DevOps Web Frontend

这是 MCP-DevOps Kubernetes 管理系统的 Web 前端界面，提供了友好的用户界面来与 AI 助手进行交互，管理 Kubernetes 资源，诊断问题和处理告警。

## 功能特性

- 🔐 用户认证 - 安全的登录系统
- 💬 聊天界面 - 与 AI 助手自然语言交互
- 🌓 深色/浅色模式 - 支持两种主题切换
- 📱 响应式设计 - 适配桌面和移动设备
- 🔔 告警管理 - 接收和处理 Kubernetes 告警

## 技术栈

- React 18 - 用户界面库
- TypeScript - 类型安全的 JavaScript
- Vite - 快速的前端构建工具
- Tailwind CSS - 实用优先的 CSS 框架
- DaisyUI - 基于 Tailwind 的组件库
- React Router - 客户端路由
- Zustand - 状态管理
- Axios - HTTP 客户端

## 开发指南

### 安装依赖

```bash
cd client/web
npm install
```

### 启动开发服务器

```bash
npm run dev
```

开发服务器将在 http://localhost:3000 启动。

### 构建生产版本

```bash
npm run build
```

构建后的文件将位于 `dist` 目录中。

## 项目结构

```
client/web/
├── public/              # 静态资源
├── src/                 # 源代码
│   ├── components/      # 可复用组件
│   ├── pages/           # 页面组件
│   ├── services/        # API 服务
│   ├── stores/          # 状态管理
│   ├── App.tsx          # 主应用组件
│   ├── main.tsx         # 应用入口
│   └── index.css        # 全局样式
├── index.html           # HTML 模板
├── package.json         # 项目配置
├── tsconfig.json        # TypeScript 配置
├── vite.config.ts       # Vite 配置
└── README.md            # 项目文档
```

## 使用说明

### 登录

使用以下凭据登录系统：

- 用户名: `admin`
- 密码: `admin123`

### 与 AI 助手交互

登录后，您可以在聊天界面与 AI 助手进行交互。您可以询问关于：

- Kubernetes 资源管理（Pod、Deployment、Service 等）
- 故障诊断与告警处理
- Linux 系统排查与 Kubernetes 组件排查

### 告警管理

系统会自动接收来自 Alertmanager 的告警，并在界面上显示。您可以与 AI 助手讨论这些告警，获取分析和解决方案。

## 与后端集成

前端通过以下方式与后端通信：

1. RESTful API - 用于常规请求
2. Server-Sent Events (SSE) - 用于接收 AI 助手的实时响应

API 端点配置在 `vite.config.ts` 文件中，默认代理到 `http://localhost:12345`。
