import axios from 'axios';
import { useAuthStore } from '../stores/authStore';

// Define message types
export interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  timestamp: Date;
}

// Create an axios instance
const api = axios.create({
  baseURL: '/api',
  headers: {
    'Content-Type': 'application/json',
  },
});

// Add request interceptor to add auth token
api.interceptors.request.use(
  (config) => {
    const { token } = useAuthStore.getState();
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// Chat service functions
export const chatService = {
  // Send a message to the AI
  sendMessage: async (message: string): Promise<Message> => {
    try {
      // In a real implementation, this would be an API call
      // For demo purposes, we'll simulate a response
      
      // Create user message
      const userMessage: Message = {
        id: Date.now().toString(),
        role: 'user',
        content: message,
        timestamp: new Date(),
      };
      
      // Simulate API delay
      await new Promise(resolve => setTimeout(resolve, 500));
      
      // For demo, we'll return a mock response
      // In a real app, this would come from the server
      return userMessage;
    } catch (error) {
      console.error('Error sending message:', error);
      throw error;
    }
  },
  
  // Get chat history
  getChatHistory: async (): Promise<Message[]> => {
    try {
      // In a real implementation, this would be an API call
      // For demo purposes, we'll return mock data
      return [
        {
          id: '1',
          role: 'assistant',
          content: '欢迎使用 MCP-DevOps Kubernetes 管理系统，我可以帮助您管理 Kubernetes 资源、诊断问题和处理告警。请告诉我您需要什么帮助？',
          timestamp: new Date(Date.now() - 60000),
        },
      ];
    } catch (error) {
      console.error('Error getting chat history:', error);
      throw error;
    }
  },
  
  // Create an SSE connection for streaming responses
  createSSEConnection: () => {
    const { token } = useAuthStore.getState();
    const eventSource = new EventSource(`/sse?token=${token}`);
    
    return eventSource;
  },
};
