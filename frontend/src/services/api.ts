import axios from 'axios'
import wsService from './websocket' 
import { LoginRequest, RegisterRequest } from './websocket'

const API_URL = 'http://localhost:8080' // adjust according to your server config

const api = axios.create({
  baseURL: API_URL,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Add token to requests if available
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

export interface LoginData extends LoginRequest {}
export interface RegisterData extends RegisterRequest {}

// Promisify WebSocket operations for better API integration
const promisifyWsResponse = (messageType: string, timeout = 5000) => {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => {
      cleanup();
      reject(new Error(`WebSocket operation timed out after ${timeout}ms`));
    }, timeout);

    const unsubscribe = wsService.onMessage(messageType + '_success', (data) => {
      cleanup();
      resolve({ data });
    });

    const errorUnsubscribe = wsService.onMessage(messageType + '_error', (data) => {
      cleanup();
      reject(new Error(data?.message || 'WebSocket operation failed'));
    });

    function cleanup() {
      clearTimeout(timer);
      unsubscribe();
      errorUnsubscribe();
    }
  });
};

// Auth API with WebSocket implementation
export const authAPI = {
  login: async (data: LoginData) => {
    wsService.doLogin(data.gmail, data.password);
    return promisifyWsResponse('login');
  },
  
  register: async (data: RegisterData) => {
    wsService.doRegister(data.gmail, data.username, data.password);
    return promisifyWsResponse('register');
  },
  
  relogin: async (token: string) => {
    wsService.doReLogin(token);
    return promisifyWsResponse('re_login');
  },
}

// Game API with WebSocket implementation
export const gameAPI = {
  getUserCards: async () => {
    wsService.getUserCards();
    return promisifyWsResponse('get_user_cards');
  },
  
  getUserDeck: async () => {
    wsService.getUserDeck();
    return promisifyWsResponse('get_user_deck');
  },
  
  swapCard: async (cardName: string, slotIndex: number) => {
    wsService.swapCard(cardName, slotIndex);
    return promisifyWsResponse('swap_card');
  },
  
  createLobby: (roomType: string) => api.post('/lobby/create', { room_type: roomType }),
  joinLobby: (lobbyId: string, roomType: string) =>
    api.post('/lobby/join', { lobby_id: lobbyId, room_type: roomType }),
  matchLobby: (roomType: string) => api.post('/lobby/match', { room_type: roomType }),
  leaveLobby: (lobbyId: string) => api.post('/lobby/leave', { lobby_id: lobbyId }),
}

export default api