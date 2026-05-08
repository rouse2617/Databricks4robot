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

const rootElement = document.getElementById("root");

if (!rootElement) {
	throw new Error("Root element not found");
}

ReactDOM.createRoot(rootElement).render(
	<React.StrictMode>
		<ConfigProvider locale={zhCN}>
			<AntdApp>
				<App />
			</AntdApp>
		</ConfigProvider>
	</React.StrictMode>,
);
