import React from 'react'
import { Paper, Typography, Box } from '@mui/material'
import { Draggable, DraggableProvided, DraggableStateSnapshot } from 'react-beautiful-dnd'

interface CardProps {
  id: number
  name: string
  level: number
  index: number
}

const Card: React.FC<CardProps> = ({ id, name, level, index }) => {
  return (
    <Draggable draggableId={`card-${id}`} index={index}>
      {(provided: DraggableProvided, snapshot: DraggableStateSnapshot) => (
        <Paper
          ref={provided.innerRef}
          {...provided.draggableProps}
          {...provided.dragHandleProps}
          elevation={snapshot.isDragging ? 6 : 1}
          sx={{
            p: 1,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            cursor: 'grab',
            '&:active': {
              cursor: 'grabbing',
            },
            backgroundColor: snapshot.isDragging ? 'action.hover' : 'background.paper',
          }}
        >
          <Box
            component="img"
            src={`/cards/${name.toLowerCase()}.png`}
            alt={name}
            sx={{
              width: 60,
              height: 60,
              objectFit: 'contain',
            }}
          />
          <Typography variant="subtitle2" align="center" sx={{ mt: 1 }}>
            {name}
          </Typography>
          <Typography variant="caption" color="text.secondary">
            Level {level}
          </Typography>
        </Paper>
      )}
    </Draggable>
  )
}

export default Card