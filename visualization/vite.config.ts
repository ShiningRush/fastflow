import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react' 

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  build: {
    rollupOptions: {
      input: {
        main: 'index.html'
      },
      output: {
        entryFileNames: 'assets/[name].js',
        chunkFileNames: 'assets/[name].js',
        assetFileNames: 'assets/[name].[ext]'
      }
    }
  },
  define: {
    global: 'globalThis'
  },
  optimizeDeps: {
    include: ['monaco-editor']
  },
  worker: {
    format: 'es'
  }
})
