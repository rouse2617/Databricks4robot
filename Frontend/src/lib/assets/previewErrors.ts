/** Map mcap-preview JSON error codes to user-facing Chinese messages. */
export function buildPreviewErrorMessage(detail: string): string {
	const lowered = detail.toLowerCase();
	if (
		lowered.includes("storage.objects.get") ||
		(lowered.includes("403") && lowered.includes("gcs"))
	) {
		return "当前环境暂时无法读取预览文件，请联系管理员检查对象存储权限。";
	}
	if (lowered.includes("unsupported_preview_codec")) {
		return "该相机的视频编码暂不支持快速预览（仅支持 H.264 / HEVC）。可稍后用 RosView 全屏查看。";
	}
	if (lowered.includes("window_timebase_mismatch")) {
		return "资产时间窗与 MCAP 录制时间不一致，无法生成预览。请联系数据平台校正时间窗。";
	}
	if (lowered.includes("preview_no_sps_pps") || lowered.includes("preview_no_parameter_sets")) {
		return "视频流缺少解码参数（SPS/PPS），无法生成预览。";
	}
	if (lowered.includes("no_preview_topic")) {
		return "该 MCAP 中未找到可预览的 CompressedVideo 话题。";
	}
	if (lowered.includes("asset_not_previewable")) {
		return "该资产当前状态不支持预览。";
	}
	return "当前资产暂时无法预览，请稍后重试或联系管理员。";
}
