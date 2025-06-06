2
3
4
5
copy "c:\Users\USER\Desktop\mon\NCP\project\Clash_Royale-game\frontend\src\services\websocket_new.ts" "c:\Users\USER\Desktop\mon\NCP\project\Clash_Royale-game\frontend\src\services\websocket.ts" /Y
Get-Item -Path "c:\Users\USER\Desktop\mon\NCP\project\Clash_Royale-game\frontend\src\services\websocket.ts" | Select-Object FullName, Length
type "c:\Users\USER\Desktop\mon\NCP\project\Clash_Royale-game\frontend\src\services\websocket.ts"
if (Test-Path "c:\Users\USER\Desktop\mon\NCP\project\Clash_Royale-game\frontend\src\services\websocket.ts") { "File exists" } else { "File does not exist" }
cd "c:\Users\USER\Desktop\mon\NCP\project\Clash_Royale-game\frontend"; npm install uuid @types/uuid --save
$content = @'
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
