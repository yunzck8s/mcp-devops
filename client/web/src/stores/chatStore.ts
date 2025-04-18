import { create } from 'zustand';
import { Message, chatService } from '../services/chatService';

interface ChatState {
  messages: Message[];
  isLoading: boolean;
  error: string | null;
  eventSource: EventSource | null;
  
  // Actions
  sendMessage: (content: string) => Promise<void>;
  loadChatHistory: () => Promise<void>;
  connectToSSE: () => void;
  disconnectFromSSE: () => void;
  clearMessages: () => void;
}

export const useChatStore = create<ChatState>((set, get) => ({
  messages: [],
  isLoading: false,
  error: null,
  eventSource: null,
  
  sendMessage: async (content: string) => {
    try {
      set({ isLoading: true, error: null });
      
      // Add user message to the chat
      const userMessage = await chatService.sendMessage(content);
      
      set((state) => ({
        messages: [...state.messages, userMessage],
        isLoading: true,
      }));
      
      // In a real app, the assistant's response would come from the SSE connection
      // For demo purposes, we'll simulate a response
      setTimeout(() => {
        const assistantMessage: Message = {
          id: (Date.now() + 1).toString(),
          role: 'assistant',
          content: `我收到了您的消息: "${content}"。正在处理中...`,
          timestamp: new Date(),
        };
        
        set((state) => ({
          messages: [...state.messages, assistantMessage],
          isLoading: false,
        }));
      }, 1000);
      
    } catch (error) {
      console.error('Error sending message:', error);
      set({ 
        error: '发送消息失败，请重试', 
        isLoading: false 
      });
    }
  },
  
  loadChatHistory: async () => {
    try {
      set({ isLoading: true, error: null });
      
      const history = await chatService.getChatHistory();
      
      set({
        messages: history,
        isLoading: false,
      });
    } catch (error) {
      console.error('Error loading chat history:', error);
      set({ 
        error: '加载聊天历史失败', 
        isLoading: false 
      });
    }
  },
  
  connectToSSE: () => {
    try {
      // Close existing connection if any
      get().disconnectFromSSE();
      
      // Create new connection
      const eventSource = chatService.createSSEConnection();
      
      // Set up event listeners
      eventSource.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          
          if (data.type === 'message') {
            const message: Message = {
              id: Date.now().toString(),
              role: 'assistant',
              content: data.content,
              timestamp: new Date(),
            };
            
            set((state) => ({
              messages: [...state.messages, message],
              isLoading: false,
            }));
          }
        } catch (error) {
          console.error('Error processing SSE message:', error);
        }
      };
      
      eventSource.onerror = () => {
        console.error('SSE connection error');
        get().disconnectFromSSE();
        
        // Try to reconnect after a delay
        setTimeout(() => {
          get().connectToSSE();
        }, 5000);
      };
      
      set({ eventSource });
    } catch (error) {
      console.error('Error connecting to SSE:', error);
      set({ error: 'SSE 连接失败' });
    }
  },
  
  disconnectFromSSE: () => {
    const { eventSource } = get();
    
    if (eventSource) {
      eventSource.close();
      set({ eventSource: null });
    }
  },
  
  clearMessages: () => {
    set({ messages: [] });
  },
}));
