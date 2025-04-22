import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 3000, // 前端开发服务器端口
    proxy: {
      '/api': { // 保留 API 代理（如果你的项目需要）
        target: 'http://localhost:12345', // 确认 API 服务器地址和端口
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ''),
      },
      // 修改 WebSocket 代理配置
      '/ws': { // 代理路径改为 /ws，以匹配前端连接和后端服务路径
        target: 'ws://localhost:8080', // 目标改为 Go 后端 WebSocket 地址和端口
        changeOrigin: true, // 允许跨域
        ws: true, // 启用 WebSocket 代理
      },
      // 移除或注释掉旧的 /sse 代理
      // '/sse': {
      //   target: 'http://localhost:12345',
      //   changeOrigin: true,
      //   ws: true,
      // },
    },
  },
});
