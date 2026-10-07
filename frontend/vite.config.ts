import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  // POMOFARM_API lets the isolated E2E runner point Vite at its own throwaway backend.
  server: { proxy: { '/api': process.env.POMOFARM_API ?? 'http://localhost:8080' } },
})
