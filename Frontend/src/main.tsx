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
		const toast = document.createElement("div");
		toast.textContent = "版本已更新，正在重新加载…";
		Object.assign(toast.style, {
			position: "fixed",
			top: "16px",
			left: "50%",
			transform: "translateX(-50%)",
			background: "#1677ff",
			color: "#fff",
			padding: "10px 24px",
			borderRadius: "8px",
			fontSize: "14px",
			zIndex: "999999",
			boxShadow: "0 4px 12px rgba(0,0,0,0.15)",
		});
		document.body.appendChild(toast);
		setTimeout(() => window.location.reload(), 2000);
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
