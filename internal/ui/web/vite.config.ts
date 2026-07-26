import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import { resolve } from 'node:path';

export default defineConfig(() => ({
  base: '/',
  plugins: [svelte()],
  resolve: {
    alias: {
      $lib: resolve(__dirname, 'src/lib'),
      $components: resolve(__dirname, 'src/components'),
      $stores: resolve(__dirname, 'src/stores'),
      $tabs: resolve(__dirname, 'src/tabs')
    },
    conditions: ['browser']
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    assetsDir: 'assets',
    manifest: true,
    sourcemap: false,
    target: 'es2022',
    rollupOptions: {
      output: {
        entryFileNames: 'assets/[name]-[hash].js',
        chunkFileNames: 'assets/[name]-[hash].js',
        assetFileNames: 'assets/[name]-[hash][extname]'
      }
    }
  },
  server: {
    port: 5173,
    proxy: {
      '/api': { target: 'http://localhost:7073', changeOrigin: true, ws: true },
      '/icons': 'http://localhost:7073',
      '/manifest.webmanifest': 'http://localhost:7073',
      '/sw.js': 'http://localhost:7073',
      '/offline.html': 'http://localhost:7073'
    }
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test-setup.ts'],
    // The compiled paraglide bundle is ~8.6 MB across 1167 modules (1163 keys ×
    // 14 locales), and every store that surfaces a message pulls it in. The
    // first `await import()` inside a test therefore pays that transform: ~5 s
    // alone, more when 16 workers transform it concurrently, which tipped a
    // dozen store suites past vitest's 5 s default even on an idle machine.
    // The tests themselves are fast; only the one-off module graph is not.
    testTimeout: 30000
  }
}));
