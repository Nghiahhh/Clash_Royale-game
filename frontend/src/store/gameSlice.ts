import { createSlice, PayloadAction } from '@reduxjs/toolkit'

export type GameStatus = 'idle' | 'searching' | 'inGame'

interface Card {
  name: string
  level: number
  count: number
}

interface DeckCard extends Card {
  index: number
}

interface Tower {
  name: string
  level?: number
  count?: number
}

interface GameState {
  gameStatus: GameStatus
  lobbyId: string | null
  userCards: Card[]
  userDeck: {
    kingTower: Tower
    guardTower: Tower
    cards: DeckCard[]
  } | null
  error: string | null
}

const initialState: GameState = {
  gameStatus: 'idle',
  lobbyId: null,
  userCards: [],
  userDeck: null,
  error: null,
}

const gameSlice = createSlice({
  name: 'game',
  initialState,
  reducers: {
    setGameStatus: (state, action: PayloadAction<GameStatus>) => {
      state.gameStatus = action.payload
    },
    setLobby: (state, action: PayloadAction<string>) => {
      state.lobbyId = action.payload
    },
    setUserCards: (state, action: PayloadAction<Card[]>) => {
      state.userCards = action.payload
    },
    setUserDeck: (state, action: PayloadAction<GameState['userDeck']>) => {
      state.userDeck = action.payload
    },
    setError: (state, action: PayloadAction<string>) => {
      state.error = action.payload
    },
    clearError: (state) => {
      state.error = null
    },
    resetGame: (state) => {
      state.gameStatus = 'idle'
      state.lobbyId = null
      state.error = null
    },
  },
})

export const {
  setGameStatus,
  setLobby,
  setUserCards,
  setUserDeck,
  setError,
  clearError,
  resetGame,
} = gameSlice.actions

export default gameSlice.reducer