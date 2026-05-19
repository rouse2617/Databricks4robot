import { App as AntdApp, ConfigProvider } from "antd";
import zhCN from "antd/locale/zh_CN";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import React from "react";
import ReactDOM from "react-dom/client";
import "dayjs/locale/zh-cn";
import App from "./App";
import "./index.css";

dayjs.extend(relativeTime);
dayjs.locale("zh-cn");

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
