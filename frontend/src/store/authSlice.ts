import { createSlice, createAsyncThunk } from '@reduxjs/toolkit'
import type { PayloadAction } from '@reduxjs/toolkit'

interface AuthState {
  token: string | null
  username: string | null
  isAuthenticated: boolean
  loading: boolean
  error: string | null
}

const initialState: AuthState = {
  token: localStorage.getItem('token'),
  username: localStorage.getItem('username'),
  isAuthenticated: !!localStorage.getItem('token'),
  loading: false,
  error: null,
}

const authSlice = createSlice({
  name: 'auth',
  initialState,
  reducers: {
    setCredentials: (
      state,
      action: PayloadAction<{ token: string; username: string }>
    ) => {
      const { token, username } = action.payload
      state.token = token
      state.username = username
      state.isAuthenticated = true
      localStorage.setItem('token', token)
      localStorage.setItem('username', username)
    },
    logout: (state) => {
      state.token = null
      state.username = null
      state.isAuthenticated = false
      localStorage.removeItem('token')
      localStorage.removeItem('username')
    },
    setError: (state, action: PayloadAction<string>) => {
      state.error = action.payload
    },
    clearError: (state) => {
      state.error = null
    },
  },
})

export const { setCredentials, logout, setError, clearError } = authSlice.actions
export default authSlice.reducer