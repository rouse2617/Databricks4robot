/**
 * 共享 UI 设计 token —— CYB-4470 把 BatchJobList / WorkflowExecutionList
 * 状态 Tag 配色统一到同一份语义调色板。优先复用 antd 内置 color
 * （success / error / warning / default / processing / blue），这里集中
 * 映射"业务状态 → antd Tag color"，避免在多处分散硬编码 color 字符串。
 *
 * 约定：
 * - `success`    全部完成 / 资源池 available / 正式发布 等正向终态
 * - `processing` 活动态（运行中 / 排队中 / 处理中）
 * - `warning`    部分失败 / 排队等待 / 暂停 / 降级 / 资源池 unavailable
 * - `error`      失败 / 拒绝 / 资源不可用
 * - `paused`     暂停 / 草稿 / 等待用户操作
 * - `default`    普通 / 辅助 / 中性
 * - `system`     系统 / 版本类（用 antd "blue"，浅蓝底 + 深蓝字）
 */

export const STATUS_TAG_PALETTE = {
	success: "success",
	processing: "processing",
	warning: "warning",
	error: "error",
	paused: "default",
	default: "default",
	system: "blue",
} as const;

export type StatusTagPaletteKey = keyof typeof STATUS_TAG_PALETTE;
