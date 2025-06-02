import React from 'react'
import { useNavigate } from 'react-router-dom'
import { useDispatch, useSelector } from 'react-redux'
import {
  AppBar,
  Toolbar,
  Typography,
  Button,
  IconButton,
  Box,
} from '@mui/material'
import { Person, ExitToApp, Home, SportsEsports } from '@mui/icons-material'
import { logout } from '../../store/authSlice'
import { RootState } from '../../store/store'
import { wsService } from '../../services/websocket'

const Navbar: React.FC = () => {
  const navigate = useNavigate()
  const dispatch = useDispatch()
  const username = useSelector((state: RootState) => state.auth.username)

  const handleLogout = () => {
    wsService.disconnect()
    dispatch(logout())
    navigate('/login')
  }

  return (
    <AppBar position="static">
      <Toolbar>
        <Typography variant="h6" component="div" sx={{ flexGrow: 1 }}>
          Clash Royale
        </Typography>
        <Box sx={{ display: 'flex', gap: 1 }}>
          <IconButton color="inherit" onClick={() => navigate('/')}>
            <Home />
          </IconButton>
          <IconButton color="inherit" onClick={() => navigate('/game')}>
            <SportsEsports />
          </IconButton>
          <IconButton color="inherit" onClick={() => navigate('/profile')}>
            <Person />
          </IconButton>
          <Typography
            variant="subtitle1"
            component="div"
            sx={{ display: 'flex', alignItems: 'center', mx: 2 }}
          >
            {username}
          </Typography>
          <Button
            color="inherit"
            startIcon={<ExitToApp />}
            onClick={handleLogout}
          >
            Logout
          </Button>
        </Box>
      </Toolbar>
    </AppBar>
  )
}

export default Navbar