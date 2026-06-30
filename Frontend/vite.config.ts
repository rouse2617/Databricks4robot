import { execSync } from "node:child_process";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

const previewId = (process.env.VITE_PREVIEW_ID ?? "").trim();
const previewHost = (
	process.env.VITE_PREVIEW_HOST ?? "https://cyber-databrew-dev.cyberorigin.ai"
).replace(/\/$/, "");
const appVersion =
	process.env.VITE_APP_VERSION || process.env.npm_package_version || "dev";
const buildRef =
	(process.env.VITE_BUILD_REF ?? "").trim() ||
	(previewId ? `preview/${previewId}` : getGitCommit());
function getGitCommit(): string {
	try {
		return execSync("git rev-parse --short HEAD", {
			encoding: "utf-8",
			timeout: 3000,
		}).trim();
	} catch {
		return "unknown";
	}
}

const DEV_PORT = 5176;
const DEFAULT_LOCAL_API = "http://localhost:8080";
const apiProxyTarget = process.env.VITE_API_BASE_URL || DEFAULT_LOCAL_API;

function apiProxyConfig(): import("vite").ProxyOptions {
	if (previewId) {
		return {
			target: previewHost,
			changeOrigin: true,
			rewrite: (path) => `/preview/${previewId}/api${path}`,
		};
	}
	return {
		target: apiProxyTarget,
		changeOrigin: true,
	};
}

function apiProxyLabel(): string {
	if (previewId) {
		return `${previewHost}/preview/${previewId}/api`;
	}
	return apiProxyTarget;
}

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
				const usingRemoteApi =
					apiProxyLabel() !== DEFAULT_LOCAL_API || previewId.length > 0;
				console.log("");
				console.log("  DataBrew frontend dev");
				console.log(`  UI:  http://127.0.0.1:${port}/`);
				console.log(`  API: ${apiProxyLabel()} (via /api proxy)`);
				console.log(`  Ver: v${appVersion} (${buildRef})`);
				if (!usingRemoteApi) {
					console.log(
						"  Tip: npm run dev:shared / dev:preview — local UI + GKE backend Pod",
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
			"/api/v1/preview": {
				target:
					process.env.VITE_MCAP_PREVIEW_URL ??
					"https://mcap-preview-dev-wtttm6suaq-uc.a.run.app",
				changeOrigin: true,
				timeout: 300000,
				proxyTimeout: 300000,
			},
			"/api": apiProxyConfig(),
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
				// P2-T-1: 核心组件 coverage baseline (ratchet up as coverage improves)
				"src/components/**": {
					statements: 44,
					branches: 34,
					functions: 44,
					lines: 45,
				},
			},
		},
	},
});
