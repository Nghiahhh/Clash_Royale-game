import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useDispatch } from 'react-redux'
import {
  Button,
  TextField,
  Paper,
  Typography,
  Container,
  Box,
} from '@mui/material'
import { authAPI } from '../../services/api'
import { setCredentials } from '../../store/authSlice'
import { wsService } from '../../services/websocket'

// Define the response structure
interface WebSocketLoginResponse {
  data: {
    token: string;
    username: string;
  };
}

const Login: React.FC = () => {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const navigate = useNavigate()
  const dispatch = useDispatch()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')

    try {
      // Connect to WebSocket if not already connected
      wsService.connect()
      
      // Use our WebSocket-based API
      const response = await authAPI.login({ gmail: email, password }) as WebSocketLoginResponse
      
      // The response data structure matches what's coming from the WebSocket
      const { token, username } = response.data
      
      // Save credentials to Redux store and localStorage
      dispatch(setCredentials({ token, username }))
      navigate('/game')
    } catch (err: any) {
      setError(err?.message || 'Login failed')
      console.error('Login error:', err)
    }
  }

  return (
    <Container component="main" maxWidth="xs">
      <Paper
        elevation={3}
        sx={{
          marginTop: 8,
          padding: 4,
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
        }}
      >
        <Typography component="h1" variant="h5">
          Sign in to Clash Royale
        </Typography>
        <Box component="form" onSubmit={handleSubmit} sx={{ mt: 1 }}>
          <TextField
            margin="normal"
            required
            fullWidth
            label="Email Address"
            autoComplete="email"
            autoFocus
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
          <TextField
            margin="normal"
            required
            fullWidth
            label="Password"
            type="password"
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
          {error && (
            <Typography color="error" sx={{ mt: 2 }}>
              {error}
            </Typography>
          )}
          <Button
            type="submit"
            fullWidth
            variant="contained"
            sx={{ mt: 3, mb: 2 }}
          >
            Sign In
          </Button>
          <Button
            fullWidth
            variant="text"
            onClick={() => navigate('/register')}
          >
            Don't have an account? Sign Up
          </Button>
        </Box>
      </Paper>
    </Container>
  )
}

export default Login
