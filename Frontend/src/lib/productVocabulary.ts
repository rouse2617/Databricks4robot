/**
 * Product-facing Chinese labels for DataBrew UI.
 * Rule: business concepts in Chinese; technical identifiers (Commit, Tag, PG/ES) stay English.
 */

export const COLUMN_LABELS = {
	assetId: "资产 ID",
	owner: "所属方",
	mcapFileId: "MCAP 文件 ID",
	mcapShort: "MCAP",
	runId: "运行 ID",
	eventSeq: "事件序号",
	payload: "载荷",
	algorithm: "算法",
	algorithmVersion: "算法版本",
	runStatus: "运行状态",
	deliveryId: "交付 ID",
	commit: "Commit",
	tag: "Tag",
	gcsPath: "GCS 路径",
	channels: "通道数",
	chunks: "分块数",
	source: "来源",
	publishedAt: "发布时间",
	metricKey: "指标 Key",
	displayName: "展示名",
	dependsOn: "依赖项",
	queryable: "可检索",
} as const;

export const SEARCH_MODE_LABELS = {
	structured: "结构化",
	keyword: "关键词",
	semantic: "语义",
	similar: "相似",
} as const;

export type SearchModeLabelKey = keyof typeof SEARCH_MODE_LABELS;

const BUSINESS_STATUS_LABELS: Record<string, string> = {
	ready: "已就绪",
	pending: "待处理",
	running: "运行中",
	ok: "成功",
	failed: "失败",
	cancelled: "已取消",
	canceled: "已取消",
	delivered: "已交付",
	accepted: "已接收",
	rejected: "已拒绝",
	draft: "草稿",
	published: "已发布",
	summarized: "已汇总",
	blocked: "已阻塞",
	paused: "已暂停",
	completed: "已完成",
	processing: "处理中",
	degraded: "降级",
	retrying: "重试中",
};

const STATUS_TAG_COLORS: Record<string, string> = {
	ok: "success",
	succeeded: "success",
	success: "success",
	completed: "success",
	delivered: "success",
	accepted: "success",
	ready: "success",
	summarized: "success",
	published: "success",
	running: "processing",
	pending: "processing",
	processing: "processing",
	blocked: "warning",
	paused: "warning",
	degraded: "warning",
	retrying: "warning",
	failed: "error",
	error: "error",
	rejected: "error",
	cancelled: "default",
	canceled: "default",
	draft: "default",
	skipped: "default",
	omitted: "default",
};

/** Human-readable business status (Chinese). Falls back to raw value. */
export function formatBusinessStatusLabel(
	status: string | undefined | null,
): string {
	if (!status) return "—";
	const lower = status.toLowerCase();
	if (lower in BUSINESS_STATUS_LABELS) {
		return BUSINESS_STATUS_LABELS[lower];
	}
	if (status in BUSINESS_STATUS_LABELS) {
		return BUSINESS_STATUS_LABELS[status];
	}
	return status;
}

/** Ant Design Tag color for a business/status value. */
export function resolveBusinessStatusTagColor(status: string): string {
	const lower = status.toLowerCase();
	return STATUS_TAG_COLORS[lower] ?? STATUS_TAG_COLORS[status] ?? "default";
}

/** Tooltip hint showing raw technical value under Chinese label. */
export function statusTooltipTitle(
	status: string | undefined | null,
): string | undefined {
	if (!status) return undefined;
	const label = formatBusinessStatusLabel(status);
	if (label === status) return undefined;
	return `${label}（${status}）`;
}
