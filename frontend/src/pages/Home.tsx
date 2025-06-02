import React from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Container,
  Typography,
  Button,
  Paper,
  Grid,
} from '@mui/material'
import { SportsEsports, EmojiEvents, Person } from '@mui/icons-material'

const Home: React.FC = () => {
  const navigate = useNavigate()

  return (
    <Container maxWidth="lg" sx={{ mt: 4 }}>
      <Typography variant="h2" align="center" gutterBottom>
        Welcome to Clash Royale
      </Typography>
      
      <Grid container spacing={4} sx={{ mt: 4 }}>
        <Grid item xs={12} md={4}>
          <Paper
            elevation={3}
            sx={{
              p: 3,
              height: '100%',
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              textAlign: 'center',
            }}
          >
            <SportsEsports sx={{ fontSize: 60, mb: 2 }} />
            <Typography variant="h5" gutterBottom>
              Play Now
            </Typography>
            <Typography variant="body1" sx={{ mb: 3 }}>
              Jump into an exciting battle against other players!
            </Typography>
            <Button
              variant="contained"
              size="large"
              onClick={() => navigate('/game')}
            >
              Start Game
            </Button>
          </Paper>
        </Grid>

        <Grid item xs={12} md={4}>
          <Paper
            elevation={3}
            sx={{
              p: 3,
              height: '100%',
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              textAlign: 'center',
            }}
          >
            <EmojiEvents sx={{ fontSize: 60, mb: 2 }} />
            <Typography variant="h5" gutterBottom>
              Leaderboard
            </Typography>
            <Typography variant="body1" sx={{ mb: 3 }}>
              Check out the top players and their achievements!
            </Typography>
            <Button variant="contained" size="large" disabled>
              Coming Soon
            </Button>
          </Paper>
        </Grid>

        <Grid item xs={12} md={4}>
          <Paper
            elevation={3}
            sx={{
              p: 3,
              height: '100%',
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              textAlign: 'center',
            }}
          >
            <Person sx={{ fontSize: 60, mb: 2 }} />
            <Typography variant="h5" gutterBottom>
              Profile
            </Typography>
            <Typography variant="body1" sx={{ mb: 3 }}>
              View your stats and customize your deck!
            </Typography>
            <Button
              variant="contained"
              size="large"
              onClick={() => navigate('/profile')}
            >
              View Profile
            </Button>
          </Paper>
        </Grid>
      </Grid>
    </Container>
  )
}

export default Home