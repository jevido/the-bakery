import { svelte } from '@sveltejs/vite-plugin-svelte'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [svelte()],
  server: {
    host: '127.0.0.1',
    port: 4850,
    strictPort: true,
    proxy: { '/api': 'http://127.0.0.1:4810' },
  },
  preview: { host: '127.0.0.1', port: 4850, strictPort: true },
})
