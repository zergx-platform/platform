import path from 'node:path'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import tailwindcss from '@tailwindcss/vite'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [
    tailwindcss(),
    svelte({
      compilerOptions: {
        runes: ({ filename }) =>
          filename.split(/[/\\]/).includes('node_modules') ? undefined : true,
      },
    }),
  ],
  resolve: {
    alias: {
      $lib: path.resolve('./src/lib'),
      '@rucoder/schema': path.resolve('./schema/src/index.ts'),
    },
  },
  server: {
    host: '0.0.0.0',
    port: 5601,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/v2': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
