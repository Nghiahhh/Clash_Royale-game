import React, { useState, useRef, useCallback } from 'react'
import { Box, Paper } from '@mui/material'
import { DragDropContext, Droppable } from 'react-beautiful-dnd'
import { useSelector } from 'react-redux'
import Card from './Card'
import { RootState } from '../../store/store'
import { wsService } from '../../services/websocket'

const GameBoard: React.FC = () => {
  const gameState = useSelector((state: RootState) => state.game)
  const boardRef = useRef<HTMLDivElement>(null)
  const [draggedCard, setDraggedCard] = useState<number | null>(null)

  const handleDragStart = (result: any) => {
    const cardId = parseInt(result.draggableId.split('-')[1])
    setDraggedCard(cardId)
  }

  const handleDragEnd = useCallback((result: any) => {
    setDraggedCard(null)

    if (!result.destination || !boardRef.current) return

    const boardRect = boardRef.current.getBoundingClientRect()
    const dropX = (result.destination.x - boardRect.left) / boardRect.width
    const dropY = (result.destination.y - boardRect.top) / boardRect.height

    if (draggedCard !== null) {
      wsService.releaseCard(draggedCard, dropX, dropY)
    }
  }, [draggedCard])

  return (
    <DragDropContext onDragStart={handleDragStart} onDragEnd={handleDragEnd}>
      <Box sx={{ display: 'flex', flexDirection: 'column', height: '100vh' }}>
        {/* Game Arena */}
        <Paper
          ref={boardRef}
          sx={{
            flex: 1,
            m: 2,
            backgroundColor: '#1a237e',
            position: 'relative',
            minHeight: '60vh',
          }}
        >
          {/* Game elements will be rendered here */}
        </Paper>

        {/* Card Deck */}
        <Droppable droppableId="deck" direction="horizontal">
          {(provided) => (
            <Paper
              ref={provided.innerRef}
              {...provided.droppableProps}
              sx={{
                p: 2,
                m: 2,
                display: 'flex',
                gap: 2,
                backgroundColor: '#263238',
              }}
            >
              {gameState.userDeck?.cards.map((card, index) => (
                <Card
                  key={card.index}
                  id={card.index}
                  name={card.name}
                  level={card.level}
                  index={index}
                />
              ))}
              {provided.placeholder}
            </Paper>
          )}
        </Droppable>
      </Box>
    </DragDropContext>
  )
}

export default GameBoard