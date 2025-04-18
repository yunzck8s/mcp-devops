import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { jwtDecode } from 'jwt-decode';

interface AuthState {
  token: string | null;
  user: {
    username: string;
    role: string;
  } | null;
  isAuthenticated: boolean;
  login: (username: string, password: string) => Promise<boolean>;
  logout: () => void;
  checkAuth: () => Promise<boolean>;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      token: null,
      user: null,
      isAuthenticated: false,

      login: async (username: string, password: string) => {
        try {
          // In a real app, this would be an API call
          // For demo purposes, we'll just check hardcoded credentials
          if (username === 'admin' && password === 'admin123') {
            // Mock token - in a real app, this would come from the server
            const token = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImFkbWluIiwicm9sZSI6ImFkbWluIiwiaWF0IjoxNjE2MTUyMDAwLCJleHAiOjE2MTYyMzg0MDB9.dKUJxPZK3qPxV2mNhvMMG9QLnFOeQHtKGIWD8v-gc8o';
            
            set({
              token,
              user: {
                username: 'admin',
                role: 'admin',
              },
              isAuthenticated: true,
            });
            return true;
          }
          return false;
        } catch (error) {
          console.error('Login error:', error);
          return false;
        }
      },

      logout: () => {
        set({
          token: null,
          user: null,
          isAuthenticated: false,
        });
      },

      checkAuth: async () => {
        const { token } = get();
        
        if (!token) {
          return false;
        }

        try {
          // Verify token expiration
          const decoded = jwtDecode<{ exp: number }>(token);
          const currentTime = Date.now() / 1000;
          
          if (decoded.exp < currentTime) {
            // Token expired
            set({
              token: null,
              user: null,
              isAuthenticated: false,
            });
            return false;
          }
          
          return true;
        } catch (error) {
          console.error('Token validation error:', error);
          set({
            token: null,
            user: null,
            isAuthenticated: false,
          });
          return false;
        }
      },
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({ token: state.token, user: state.user }),
    }
  )
);
