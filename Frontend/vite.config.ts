import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

const appVersion = process.env.VITE_APP_VERSION || process.env.npm_package_version || "dev";
const buildRef = process.env.VITE_BUILD_REF || "local";

function packageNameFromModuleId(id: string): string | null {
  const marker = "/node_modules/";
  const idx = id.lastIndexOf(marker);
  if (idx === -1) return null;
  const sub = id.slice(idx + marker.length);
  const parts = sub.split("/");
  if (parts.length === 0) return null;
  if (parts[0].startsWith("@") && parts.length >= 2) {
    return `${parts[0]}/${parts[1]}`;
  }
  return parts[0];
}

export default defineConfig({
  plugins: [react()],
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
    chunkSizeWarningLimit: 800,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes("node_modules")) return;
          if (id.includes("echarts")) return "vendor-echarts";

          const pkg = packageNameFromModuleId(id);
          if (!pkg) return;

          if (
            pkg === "react" ||
            pkg === "react-dom" ||
            pkg === "react-router-dom" ||
            pkg === "@remix-run/router" ||
            pkg === "scheduler"
          ) {
            return "vendor-react";
          }

          if (
            pkg === "antd" ||
            pkg === "dayjs" ||
            pkg === "@ant-design/icons" ||
            pkg === "@ant-design/cssinjs" ||
            pkg === "@ant-design/fast-color" ||
            pkg === "@ctrl/tinycolor" ||
            pkg.startsWith("rc-") ||
            pkg.startsWith("@rc-component/") ||
            pkg === "classnames"
          ) {
            return "vendor-antd-core";
          }

          if (pkg === "axios") return "vendor-axios";
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
