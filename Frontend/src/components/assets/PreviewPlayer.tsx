import { Alert, Typography } from "antd";
import { useState } from "react";
import type { PreviewManifest } from "../../lib/assets/assetsDiscoveryTypes";

const { Text } = Typography;

interface PreviewPlayerProps {
	manifest: PreviewManifest | null;
	compact?: boolean;
}

function buildUserFacingError(detail: string): string {
	const lowered = detail.toLowerCase();
	if (
		lowered.includes("storage.objects.get") ||
		(lowered.includes("403") && lowered.includes("gcs"))
	) {
		return "当前环境暂时无法读取预览文件，请联系管理员检查对象存储权限。";
	}
	return "当前资产暂时无法预览，请稍后重试或联系管理员。";
}

// Single source of preview video: the mcap-preview service produces a fragmented
// MP4 (`/api/v1/preview/assets/:id/segment.mp4`) that a native <video> element
// can decode with hardware support. We deliberately removed the in-browser
// MCAP-direct path (fzstd + WebCodecs + canvas) — see commit history if it ever
// needs resurrection — because maintaining two playback pipelines and a
// JS-side decompressor was costing more than it returned.
export default function PreviewPlayer({
	manifest,
	compact,
}: PreviewPlayerProps) {
	// Track <video> load failure so we can replace the silent black box with a
	// real explanation. The browser gives us MEDIA_ERR_* codes but no upstream
	// HTTP status; for finer-grained diagnostics we probe segment.mp4 once and
	// surface the JSON error payload (e.g. UPSTREAM_ERROR for loadtest assets
	// whose MCAP lives in a bucket the preview service can't read).
	const url = manifest?.previewVideoUrl ?? null;
	const [errorState, setErrorState] = useState<{
		url: string;
		detail: string;
	} | null>(null);
	const errorDetail =
		errorState && url && errorState.url === url ? errorState.detail : null;
	const userFacingError = errorDetail
		? buildUserFacingError(errorDetail)
		: null;

	if (!manifest) {
		return <Text type="secondary">暂无预览</Text>;
	}

	if (url) {
		if (errorDetail) {
			return (
				<Alert
					type="warning"
					showIcon
					message="无法预览此资产"
					description={
						<div style={{ display: "grid", gap: 8 }}>
							<Text>{userFacingError}</Text>
							<details>
								<summary>技术详情</summary>
								<Text type="secondary" style={{ whiteSpace: "pre-wrap" }}>
									{errorDetail}
								</Text>
							</details>
						</div>
					}
				/>
			);
		}
		return (
			<div style={{ width: "100%" }}>
				<video
					style={{
						width: "100%",
						maxHeight: compact ? 260 : 360,
						background: "#000",
					}}
					src={url}
					controls
					muted
					playsInline
					preload="none"
					onError={async () => {
						let detail =
							"视频流加载失败（浏览器返回 MEDIA_ELEMENT_ERROR）。常见原因：MCAP 来自预览服务无访问权限的存储桶，或不是可解码的视频载荷。";
						try {
							const resp = await fetch(url, { credentials: "include" });
							if (!resp.ok) {
								const body = await resp.text();
								try {
									const parsed = JSON.parse(body) as {
										code?: string;
										message?: string;
									};
									if (parsed?.message) {
										detail = `${resp.status} ${parsed.code ?? ""}: ${parsed.message}`;
									}
								} catch {
									detail = `${resp.status}: ${body.slice(0, 200)}`;
								}
							}
						} catch {
							// fall back to the generic message above
						}
						setErrorState({ url, detail });
					}}
				/>
			</div>
		);
	}

	return (
		<Alert
			type="warning"
			showIcon
			message="预览流未就绪"
			description="该资产暂无可播放的视频流（segment.mp4）。"
		/>
	);
}
