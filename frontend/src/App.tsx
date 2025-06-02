import { Routes, Route, Navigate } from 'react-router-dom'
import { useSelector } from 'react-redux'
import { CssBaseline, ThemeProvider, createTheme } from '@mui/material'
import { useEffect } from 'react'
import Login from './components/Auth/Login'
import Register from './components/Auth/Register'
import Navbar from './components/Layout/Navbar'
import Home from './pages/Home'
import Game from './pages/Game'
import Profile from './pages/Profile'
import { RootState } from './store/store'
import wsService from './services/websocket'

const theme = createTheme({
  palette: {
    mode: 'dark',
    primary: {
      main: '#2196f3',
    },
    secondary: {
      main: '#f50057',
    },
  },
})

const PrivateRoute: React.FC<{ element: React.ReactElement }> = ({ element }) => {
  const isAuthenticated = useSelector(
    (state: RootState) => state.auth.isAuthenticated
  )
  return isAuthenticated ? element : <Navigate to="/login" replace />
}

const App: React.FC = () => {
  const isAuthenticated = useSelector(
    (state: RootState) => state.auth.isAuthenticated
  )

  // Initialize WebSocket connection when app starts
  useEffect(() => {
    // Connect to WebSocket server
    wsService.connect()

    // Clean up WebSocket connection when component unmounts
    return () => {
      wsService.disconnect()
    }
  }, [])

  // When user gets authenticated (either by login or loaded from localStorage), 
  // try to relogin with token
  useEffect(() => {
    if (isAuthenticated) {
      const token = localStorage.getItem('token')
      if (token) {
        wsService.doReLogin(token)
      }
    }
  }, [isAuthenticated])

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      {isAuthenticated && <Navbar />}
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />
        <Route path="/" element={<PrivateRoute element={<Home />} />} />
        <Route path="/game" element={<PrivateRoute element={<Game />} />} />
        <Route path="/profile" element={<PrivateRoute element={<Profile />} />} />
      </Routes>
    </ThemeProvider>
  )
}

export default App