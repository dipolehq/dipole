import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const devToken = env.DIPOLE_DEV_TOKEN
  const apiTarget = env.DIPOLE_API_TARGET || 'http://localhost:8080'

  return {
    plugins: [react(), tailwindcss()],
    resolve: {
      alias: {
        '@': path.resolve(__dirname, './src'),
      },
    },
    server: {
      port: 3000,
      proxy: {
        '/api': {
          target: apiTarget,
          changeOrigin: true,
          configure: (proxy) => {
            if (!devToken) return
            proxy.on('proxyReq', (proxyReq) => {
              if (!proxyReq.getHeader('authorization')) {
                proxyReq.setHeader('authorization', `Bearer ${devToken}`)
              }
            })
          },
        },
      },
    },
  }
})
