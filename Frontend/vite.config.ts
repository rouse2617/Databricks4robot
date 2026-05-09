import { defineConfig, splitVendorChunkPlugin } from "vite";
import react from "@vitejs/plugin-react";

const appVersion = process.env.VITE_APP_VERSION || process.env.npm_package_version || "dev";
const buildRef = process.env.VITE_BUILD_REF || "local";

export default defineConfig({
  plugins: [react(), splitVendorChunkPlugin()],
  define: {
    __APP_VERSION__: JSON.stringify(appVersion),
    __APP_BUILD_REF__: JSON.stringify(buildRef),
  },
  server: {
    proxy: {
      "/api": {
        target: process.env.VITE_API_BASE_URL || "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: "dist",
    sourcemap: true,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes("node_modules")) return;
          if (id.includes("echarts")) return "vendor-echarts";
        },
      },
    },
  },
  test: {
    globals: true,
    environment: "jsdom",
    setupFiles: ["src/setupTests.ts"],
    exclude: ["e2e/**", "node_modules/**"],
    coverage: {
      provider: "v8",
      reporter: ["text", "text-summary", "lcov"],
      include: ["src/**/*.{ts,tsx}"],
      exclude: [
        "src/main.tsx",
        "src/**/*.test.{ts,tsx}",
        "src/**/*.d.ts",
      ],
      thresholds: {
        // P2-T-1: 核心组件 ≥ 85% coverage gate
        "src/components/**": {
          statements: 85,
          branches: 85,
          functions: 85,
          lines: 85,
        },
      },
    },
  },
});
