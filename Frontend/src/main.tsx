import "@ant-design/v5-patch-for-react-19";
import { App as AntdApp, ConfigProvider } from "antd";
import zhCN from "antd/locale/zh_CN";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import React from "react";
import ReactDOM from "react-dom/client";
import "dayjs/locale/zh-cn";
import App from "./App";
import "./index.css";
import { installConsoleWarningFilter } from "./lib/consoleWarningFilter";

// Catch dynamic import failures (chunk not found after deploy) and auto-reload.
window.addEventListener("error", (event) => {
	const target = event.target as HTMLElement | null;
	const isChunkError =
		event.message?.includes("Failed to fetch dynamically imported module") ||
		event.message?.includes("Importing a module script failed") ||
		target?.localName === "link" ||
		(target as HTMLScriptElement | null)?.localName === "script";
	if (isChunkError) {
		event.preventDefault();
		console.warn("[chunk-load] detected stale chunk, reloading...");
		window.location.reload();
	}
});

dayjs.extend(relativeTime);
dayjs.locale("zh-cn");
installConsoleWarningFilter();

const appTheme = {
	token: {
		colorPrimary: "#2563eb",
		colorInfo: "#2563eb",
		colorSuccess: "#16a34a",
		colorWarning: "#d97706",
		colorError: "#dc2626",
		colorText: "#1e293b",
		colorTextSecondary: "#64748b",
		colorBgLayout: "#f8fafc",
		colorBorder: "#e2e8f0",
		borderRadius: 8,
		fontFamily:
			"Inter, -apple-system, BlinkMacSystemFont, Segoe UI, Roboto, sans-serif",
		fontFamilyCode: "Fira Code, SF Mono, Cascadia Code, monospace",
		fontSize: 13,
	},
};

const rootElement = document.getElementById("root");

if (!rootElement) {
	throw new Error("Root element not found");
}

ReactDOM.createRoot(rootElement).render(
	<React.StrictMode>
		<ConfigProvider locale={zhCN} theme={appTheme}>
			<AntdApp>
				<App />
			</AntdApp>
		</ConfigProvider>
	</React.StrictMode>,
);
