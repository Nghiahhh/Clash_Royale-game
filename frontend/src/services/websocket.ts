import { io, Socket } from 'socket.io-client'
import { store } from '../store/store'
import { setGameStatus, setError } from '../store/gameSlice'

class WebSocketService {
  private socket: Socket | null = null
  private readonly url: string = 'ws://localhost:8080/ws'

  connect(token: string) {
    if (this.socket?.connected) return

    this.socket = io(this.url, {
      auth: { token },
      transports: ['websocket'],
    })

    this.setupEventListeners()
  }

  private setupEventListeners() {
    if (!this.socket) return

    this.socket.on('connect', () => {
      console.log('Connected to game server')
    })

    this.socket.on('disconnect', () => {
      console.log('Disconnected from game server')
      store.dispatch(setGameStatus('idle'))
    })

    this.socket.on('error', (error: string) => {
      store.dispatch(setError(error))
    })

    // Game-specific events
    this.socket.on('game_start', (data) => {
      store.dispatch(setGameStatus('inGame'))
      // Handle game start data
    })

    this.socket.on('card_played', (data) => {
      // Handle card being played
    })

    this.socket.on('game_end', (data) => {
      store.dispatch(setGameStatus('idle'))
      // Handle game end data
    })
  }

  releaseCard(cardId: number, x: number, y: number) {
    this.socket?.emit('release_card', { card_id: cardId, x, y })
  }

  disconnect() {
    if (this.socket) {
      this.socket.disconnect()
      this.socket = null
    }
  }
}

export const wsService = new WebSocketService()
export default wsService