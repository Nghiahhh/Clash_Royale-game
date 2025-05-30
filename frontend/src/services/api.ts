import axios from 'axios'

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

export interface LoginData {
  gmail: string
  password: string
}

export interface RegisterData extends LoginData {
  username: string
}

export const authAPI = {
  login: (data: LoginData) => api.post('/login', data),
  register: (data: RegisterData) => api.post('/register', data),
  relogin: (token: string) => api.post('/relogin', { token }),
}

export const gameAPI = {
  getUserCards: () => api.get('/user/cards'),
  getUserDeck: () => api.get('/user/deck'),
  swapCard: (cardName: string, slotIndex: number) =>
    api.post('/user/deck/swap', { card_name: cardName, slot_index: slotIndex }),
  createLobby: (roomType: string) => api.post('/lobby/create', { room_type: roomType }),
  joinLobby: (lobbyId: string, roomType: string) =>
    api.post('/lobby/join', { lobby_id: lobbyId, room_type: roomType }),
  matchLobby: (roomType: string) => api.post('/lobby/match', { room_type: roomType }),
  leaveLobby: (lobbyId: string) => api.post('/lobby/leave', { lobby_id: lobbyId }),
}

export default api