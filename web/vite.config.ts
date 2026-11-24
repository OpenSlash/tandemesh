import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    port: 5173,
    watch: {
      ignored: ['**/.*.tmpdir/**', '**/*.tmp'],
    },
    proxy: {
      '/api': 'http://127.0.0.1:8080',
      '/healthz': 'http://127.0.0.1:8080',
      '/webhooks': 'http://127.0.0.1:8080',
    },
  },
})
