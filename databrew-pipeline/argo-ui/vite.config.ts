import { defineConfig, type Plugin } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'
import { syncPublicAssets } from './scripts/sync-public-assets.mjs'

function argoAssetsPlugin(): Plugin {
  return {
    name: 'sync-argo-public-assets',
    buildStart() {
      syncPublicAssets()
    },
    configureServer() {
      syncPublicAssets()
    },
  }
}

export default defineConfig({
  plugins: [argoAssetsPlugin(), react()],
  define: {
    'process.env.DEFAULT_TZ': JSON.stringify('UTC'),
    'process.env.NODE_ENV': JSON.stringify(process.env.NODE_ENV || 'development'),
    SYSTEM_INFO: JSON.stringify({ version: process.env.VERSION || 'latest' }),
  },
  optimizeDeps: {
    include: [
      'react-form',
      'prop-types',
      'react-fast-compare',
      'react-autocomplete',
      'xterm',
      'xterm-addon-fit',
      'object-assign',
      'classnames',
      'history',
      'lodash',
      'hoist-non-react-statics',
      'invariant',
      'warning',
      'react-is',
      'scheduler',
      'dom-scroll-into-view',
      'isarray',
      'loose-envify',
      'js-tokens',
      'path-to-regexp',
      'react-side-effect',
      'tiny-warning',
      'resolve-pathname',
      'value-equal',
    ],
  },
  css: {
    preprocessorOptions: {
      scss: {
        // Absolute font URLs so icons work under /workflows/* routes (not /workflows/assets/...).
        additionalData: `$fa-font-path: "/assets/fonts";\n$argo-icon-fonts-root: "/assets/fonts/";\n`,
        silenceDeprecations: ['legacy-js-api', 'mixed-decls'],
        loadPaths: ['.'],
      },
    },
  },
  server: {
    port: 5174,
    compress: false,
    proxy: {
      '/api/v1': {
        target: 'http://localhost:2746',
        changeOrigin: true,
      },
      '/artifact-files': {
        target: 'http://localhost:2746',
        changeOrigin: true,
      },
      '/artifacts': {
        target: 'http://localhost:2746',
        changeOrigin: true,
      },
      '/oauth2': {
        target: 'http://localhost:2746',
        changeOrigin: true,
      },
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
