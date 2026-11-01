import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

import '@/ui/styles/base.css'

import { Settings } from './Settings'

const root = document.getElementById('root')
if (root) {
  createRoot(root).render(
    <StrictMode>
      <Settings />
    </StrictMode>,
  )
}
