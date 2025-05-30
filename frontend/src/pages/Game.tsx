import React, { useEffect, useState } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import {
  Container,
  Typography,
  Button,
  Paper,
} from '@mui/material'
import GameBoard from '../components/Game/GameBoard'
import { gameAPI } from '../services/api'
import { RootState } from '../store/store'
import { setGameStatus, setLobby } from '../store/gameSlice'

const Game: React.FC = () => {
  const dispatch = useDispatch()
  const [loading, setLoading] = useState(false)
  const gameStatus = useSelector((state: RootState) => state.game.gameStatus)

  const handleMatchmaking = async () => {
    setLoading(true)
    try {
      const response = await gameAPI.matchLobby('1v1')
      dispatch(setLobby(response.data.lobby_id))
      dispatch(setGameStatus('inGame'))
    } catch (err) {
      console.error('Failed to join matchmaking:', err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    // Clean up effect
    return () => {
      dispatch(setGameStatus('idle'))
    }
  }, [dispatch])

  if (gameStatus === 'inGame') {
    return (
      <Container maxWidth="lg" sx={{ mt: 4 }}>        <GameBoard />
      </Container>
    )
  }

  return (
    <Container maxWidth="lg" sx={{ mt: 4 }}>
      <Paper sx={{ p: 3, textAlign: 'center' }}>
        <Typography variant="h4" gutterBottom>
          Ready to Play?
        </Typography>
        <Button
          variant="contained"
          size="large"
          onClick={handleMatchmaking}
          disabled={loading || gameStatus === 'searching'}
        >
          {loading ? 'Finding Match...' : 'Start Matchmaking'}
        </Button>
      </Paper>
    </Container>
  )
}

export default Game