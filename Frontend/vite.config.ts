import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

const appVersion =
	process.env.VITE_APP_VERSION || process.env.npm_package_version || "dev";
const buildRef = process.env.VITE_BUILD_REF || "local";
const DEV_PORT = 5176;
const DEFAULT_LOCAL_API = "http://localhost:8080";
const apiProxyTarget = process.env.VITE_API_BASE_URL || DEFAULT_LOCAL_API;

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

function devServerBanner(): import("vite").Plugin {
	return {
		name: "databrew-dev-banner",
		configureServer(server) {
			server.httpServer?.once("listening", () => {
				const address = server.httpServer?.address();
				const port =
					typeof address === "object" && address !== null
						? address.port
						: DEV_PORT;
				const usingRemoteApi = apiProxyTarget !== DEFAULT_LOCAL_API;
				console.log("");
				console.log("  DataBrew frontend dev");
				console.log(`  UI:  http://127.0.0.1:${port}/`);
				console.log(`  API: ${apiProxyTarget} (via /api proxy)`);
				if (!usingRemoteApi) {
					console.log(
						"  Tip: npm run dev:remote — proxy to Cloud Run dev backend (workflows, pipeline, …)",
					);
				}
				console.log("");
			});
		},
	};
}

export default defineConfig({
	plugins: [react(), devServerBanner()],
	define: {
		__APP_VERSION__: JSON.stringify(appVersion),
		__APP_BUILD_REF__: JSON.stringify(buildRef),
	},
	server: {
		port: DEV_PORT,
		strictPort: true,
		proxy: {
			"/api": {
				target: apiProxyTarget,
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
		testTimeout: 10_000,
		coverage: {
			provider: "v8",
			reporter: ["text", "text-summary", "lcov"],
			include: ["src/**/*.{ts,tsx}"],
			exclude: ["src/main.tsx", "src/**/*.test.{ts,tsx}", "src/**/*.d.ts"],
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
