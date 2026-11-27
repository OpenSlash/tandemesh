import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import pkg from './package.json'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  define: {
    // Packaged builds pass their release version; plain web builds fall back
    // to package.json.
    __APP_VERSION__: JSON.stringify(process.env.TANDEMESH_VERSION || pkg.version),
  },
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
