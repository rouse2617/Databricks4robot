import { useEffect, useRef } from "react";
import { notification } from "antd";

interface VersionInfo {
	version: string;
	buildRef: string;
	timestamp: number;
}

const CHECK_INTERVAL = 60000; // 每60秒检查一次
const UPDATE_NOTIFICATION_KEY = "app-update-notification";

function showUpdateNotification(): void {
	const btn = (
		<button
			type="button"
			style={{
				color: "#1677ff",
				background: "none",
				border: "none",
				cursor: "pointer",
				padding: "4px 8px",
				fontWeight: 500,
			}}
			onClick={() => {
				notification.destroy(UPDATE_NOTIFICATION_KEY);
				window.location.reload();
			}}
		>
			刷新
		</button>
	);

	notification.info({
		key: UPDATE_NOTIFICATION_KEY,
		message: "有新版本可用",
		description: "应用已更新，点击刷新获取最新版本",
		placement: "topRight",
		duration: 0,
		btn,
	});
}

export function useVersionCheck(): void {
	const lastVersionRef = useRef<VersionInfo | null>(null);
	const notificationShownRef = useRef(false);

	useEffect(() => {
		const checkVersion = async () => {
			try {
				// 加 cache-busting query 参数绕过 CDN 缓存
				const response = await fetch(
					`/version.json?t=${Date.now()}`,
					{
						cache: "no-store",
					},
				);

				if (!response.ok) return;

				const currentVersion: VersionInfo = await response.json();

				if (!lastVersionRef.current) {
					lastVersionRef.current = currentVersion;
					return;
				}

				if (
					currentVersion.buildRef !== lastVersionRef.current.buildRef &&
					!notificationShownRef.current
				) {
					notificationShownRef.current = true;
					showUpdateNotification();
				}
			} catch (err) {
				// 版本检查失败，忽略
			}
		};

		checkVersion();
		const interval = setInterval(checkVersion, CHECK_INTERVAL);

		return () => clearInterval(interval);
	}, []);
}
