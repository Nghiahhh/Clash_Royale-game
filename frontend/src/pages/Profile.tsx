import React, { useEffect, useState } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import {
  Container,
  Typography,
  Grid,
  Paper,
  Card,
  CardContent,
  CardMedia,
  Button,
  Snackbar,
  Alert,
} from '@mui/material'
import { gameAPI } from '../services/api'
import { RootState } from '../store/store'
import { setUserCards, setUserDeck, setError } from '../store/gameSlice'

const Profile: React.FC = () => {
  const dispatch = useDispatch()
  const userCards = useSelector((state: RootState) => state.game.userCards)
  const userDeck = useSelector((state: RootState) => state.game.userDeck)
  const error = useSelector((state: RootState) => state.game.error)
  const [loading, setLoading] = useState(true)
  const [selectedCard, setSelectedCard] = useState<string | null>(null)
  const [selectedSlot, setSelectedSlot] = useState<number | null>(null)

  useEffect(() => {
    const fetchUserData = async () => {
      try {
        const [cardsResponse, deckResponse] = await Promise.all([
          gameAPI.getUserCards(),
          gameAPI.getUserDeck(),
        ])
        dispatch(setUserCards(cardsResponse.data.cards))
        dispatch(setUserDeck(deckResponse.data))
      } catch (err) {
        console.error('Failed to fetch user data:', err)
        dispatch(setError('Failed to load user data'))
      } finally {
        setLoading(false)
      }
    }

    fetchUserData()
  }, [dispatch])

  const handleCardSelect = (cardName: string) => {
    setSelectedCard(cardName)
  }

  const handleSlotSelect = async (index: number) => {
    if (!selectedCard) return

    try {
      await gameAPI.swapCard(selectedCard, index)
      const deckResponse = await gameAPI.getUserDeck()
      dispatch(setUserDeck(deckResponse.data))
      setSelectedCard(null)
      setSelectedSlot(null)
    } catch (err) {
      console.error('Failed to swap card:', err)
      dispatch(setError('Failed to update deck'))
    }
  }

  if (loading) {
    return (
      <Container maxWidth="lg" sx={{ mt: 4 }}>
        <Typography>Loading...</Typography>
      </Container>
    )
  }

  return (
    <Container maxWidth="lg" sx={{ mt: 4 }}>
      <Grid container spacing={4}>
        <Grid item xs={12}>
          <Paper sx={{ p: 3 }}>
            <Typography variant="h4" gutterBottom>
              Your Deck
            </Typography>
            <Grid container spacing={2}>
              {userDeck?.cards.map((card) => (
                <Grid item xs={6} sm={3} key={card.index}>
                  <Card
                    sx={{
                      cursor: selectedCard ? 'pointer' : 'default',
                      border: selectedSlot === card.index ? 2 : 0,
                      borderColor: 'primary.main',
                    }}
                    onClick={() => selectedCard && handleSlotSelect(card.index)}
                  >
                    <CardMedia
                      component="img"
                      height="140"
                      image={`/cards/${card.name.toLowerCase()}.png`}
                      alt={card.name}
                    />
                    <CardContent>
                      <Typography variant="subtitle1">{card.name}</Typography>
                      <Typography variant="body2">Level {card.level}</Typography>
                    </CardContent>
                  </Card>
                </Grid>
              ))}
            </Grid>
          </Paper>
        </Grid>

        <Grid item xs={12}>
          <Paper sx={{ p: 3 }}>
            <Typography variant="h4" gutterBottom>
              Available Cards
            </Typography>
            <Grid container spacing={2}>
              {userCards.map((card) => (
                <Grid item xs={6} sm={3} md={2} key={card.name}>
                  <Card
                    sx={{
                      cursor: 'pointer',
                      border: selectedCard === card.name ? 2 : 0,
                      borderColor: 'primary.main',
                    }}
                    onClick={() => handleCardSelect(card.name)}
                  >
                    <CardMedia
                      component="img"
                      height="140"
                      image={`/cards/${card.name.toLowerCase()}.png`}
                      alt={card.name}
                    />
                    <CardContent>
                      <Typography variant="subtitle1">{card.name}</Typography>
                      <Typography variant="body2">
                        Level {card.level} ({card.count})
                      </Typography>
                    </CardContent>
                  </Card>
                </Grid>
              ))}
            </Grid>
          </Paper>
        </Grid>
      </Grid>

      <Snackbar
        open={!!error}
        autoHideDuration={6000}
        onClose={() => dispatch(setError(''))}
      >
        <Alert severity="error" onClose={() => dispatch(setError(''))}>
          {error}
        </Alert>
      </Snackbar>
    </Container>
  )
}

export default Profile