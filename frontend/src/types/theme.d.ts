import '@mui/material/styles'
import { Theme } from '@mui/material/styles'

declare module '@mui/material/styles' {
  interface DefaultTheme extends Theme {}
}
